"""Offline PE32 function execution. All extracted artifacts stay under build/."""
import json
import math
from pathlib import Path
import random
import re
import struct
import sys

from unicorn import (Uc, UcError, UC_ARCH_X86, UC_MODE_32, UC_HOOK_CODE,
    UC_HOOK_MEM_WRITE, UC_HOOK_MEM_UNMAPPED)
from unicorn.x86_const import (UC_X86_REG_EAX, UC_X86_REG_EDX, UC_X86_REG_ECX,
    UC_X86_REG_ESP, UC_X86_REG_EIP, UC_X86_REG_FP0, UC_X86_REG_FPSW,
    UC_X86_REG_FPTAG, UC_X86_REG_FPCW, UC_X86_REG_GDTR, UC_X86_REG_FS,
    UC_X86_REG_SS, UC_X86_REG_DS, UC_X86_REG_ES)

ROOT = Path(__file__).resolve().parents[2]
STACK, ARENA, STUB, STOP = 0x10000000, 0x20000000, 0x30000000, 0x300ff000
HEAP, HEAP_SIZE = 0x50000000, 0x100000  # malloc／operator new 的 bump 配置區，每條重放路徑歸零
# 存檔串流讀取 404ea0(handle, buf, size, *got)：MG 2026-10-09 准樁（decisions/2026-10-09-io-and-presentation-stubs.md），
# case 用 "stream": {"at": <位置>, "size": <bytes>} 依序供位元組；讀滿回 1，不足回 0
IO_STUBS = {0x404ea0}
QCONT = STUB + 0xfe000  # qsort stub's return point after each comparator call
TIB, GDT = 0x40000000, 0x40001000
TEXT_LIMIT = 0x100000  # UTF-16 code units; narrow text uses deterministic Latin-1
REGS = {"eax": UC_X86_REG_EAX, "ecx": UC_X86_REG_ECX}


C_LOCALE_LOWER = {c: c+32 for c in range(65, 91)}
STR_FUNCS = ("wcscat", "wcslen", "wcscspn", "wcscmp", "strchr", "atoi", "_stricmp", "strncpy", "_strnicmp")


def f80(x):
    """binary64 → x87 binary80 暫存器值 (mantissa, sign|exponent)，數值不變。"""
    bits = struct.unpack("<Q", struct.pack("<d", x))[0]
    sign, exp, frac = bits >> 63, (bits >> 52) & 0x7ff, bits & ((1 << 52)-1)
    if exp == 0x7ff:
        mantissa, exp80 = (1 << 63) | (frac << 11) | ((1 << 62) if frac else 0), 0x7fff
    elif exp == 0:
        if frac == 0:
            mantissa, exp80 = 0, 0
        else:  # 次正規數：正規化成 binary80
            n = frac.bit_length()
            mantissa, exp80 = frac << (64-n), n - 1 - 52 - 1022 + 16383
    else:
        mantissa, exp80 = (1 << 63) | (frac << 11), exp - 1023 + 16383
    return mantissa, (sign << 15) | exp80


# SR.DLL 的 void thiscall 方法：emu 當 no-op，值＝callee 清的 bytes（參數皆 4-byte：enum／float／const char*／handle）
SR_VOID_SETTERS = {"?invalidateFrameHandle@srTexture@@IAEXK@Z": 4, "?setName@srRuntimeClass@@QAEXPBD@Z": 4,
                   "?autoRelease@srClass@@QAEXXZ": 0} | {
    f"?set{f}@srTexture@@QAEX{t}@Z": 4 for f, t in (
        ("Correction", "W4e_correction@srTextureIFace@@"), ("MagFilter", "W4e_filter@srTextureIFace@@"),
        ("MinFilter", "W4e_filter@srTextureIFace@@"), ("MipmapBias", "M"), ("Mipmap", "W4e_mipmap@srTextureIFace@@"),
        ("WrapS", "W4e_wrap@srTextureIFace@@"), ("WrapT", "W4e_wrap@srTextureIFace@@"))}


def formula_classes():
    """addr → (類別, 深度)，讀公式登記簿；位址樁的守衛用。"""
    out = {}
    for f in [ROOT/"notes/re/formula-graph.tsv", *sorted((ROOT/"notes/re/formula-graph.d").glob("*.tsv"))]:
        for line in f.read_text(encoding="utf-8").splitlines():
            cols = line.split("\t")
            if len(cols) > 1:
                out[cols[0]] = (cols[1], cols[2] if len(cols) > 2 else "")
    return out


def number(v):
    return int(v, 0) if isinstance(v, str) else v


def u32(data, offset=0):
    return struct.unpack_from("<I", data, offset)[0]


def pack(v):
    return struct.pack("<I", v & 0xffffffff)


class PE:
    def __init__(self, path):
        raw = path.read_bytes()
        pe = u32(raw, 0x3c)
        if raw[:2] != b"MZ" or raw[pe:pe+4] != b"PE\0\0":
            raise ValueError("not a PE file")
        opt = pe + 24
        if struct.unpack_from("<H", raw, opt)[0] != 0x10b:
            raise ValueError("PE32 required")
        self.base = u32(raw, opt + 28)
        self.size = (u32(raw, opt + 56) + 4095) & ~4095
        self.image = bytearray(self.size)
        headers = u32(raw, opt + 60)
        self.image[:headers] = raw[:headers]
        count = struct.unpack_from("<H", raw, pe + 6)[0]
        table = opt + struct.unpack_from("<H", raw, pe + 20)[0]
        for i in range(count):
            off = table + 40*i
            virtual, rva, size, ptr = struct.unpack_from("<IIII", raw, off+8)
            if rva + max(virtual, size) > self.size or ptr+size > len(raw):
                raise ValueError("invalid PE section")
            self.image[rva:rva+size] = raw[ptr:ptr+size]
        self.imports = []
        desc = u32(raw, opt + 104)
        while desc and any(self.image[desc:desc+20]):
            original, _, _, dll, iat = struct.unpack_from("<IIIII", self.image, desc)
            dll = self.string(dll)
            thunk = original or iat
            i = 0
            while (entry := u32(self.image, thunk+4*i)):
                name = f"#{entry & 0xffff}" if entry & 0x80000000 else self.string(entry+2)
                self.imports.append((self.base+iat+4*i, dll, name))
                i += 1
            desc += 20

    def string(self, rva):
        end = self.image.index(0, rva)
        return self.image[rva:end].decode("ascii")


class Branch(Exception):
    def __init__(self, n):
        self.n = n


class Machine:
    def __init__(self, pe, case, inputs, blocks, rng_stub=True, native=False):
        self.uc = u = Uc(UC_ARCH_X86, UC_MODE_32)
        u.reg_write(UC_X86_REG_FPCW, 0x37f)
        u.reg_write(UC_X86_REG_FPTAG, 0xffff)
        u.mem_map(pe.base, pe.size)
        u.mem_write(pe.base, bytes(pe.image))
        u.mem_map(STACK, 0x10000)
        u.mem_map(ARENA, 0x100000)
        u.mem_map(STUB, 0x100000)
        u.mem_map(HEAP, HEAP_SIZE)
        u.mem_map(TIB, 0x2000)
        u.mem_write(TIB, pack(0xffffffff))
        u.mem_write(TIB+0x18, pack(TIB))
        # Byte-granular, writable 32-bit data segment covering the 4 KiB TIB.
        descriptor = struct.pack("<HHBBBB", 0xfff, TIB & 0xffff,
                                 (TIB >> 16) & 0xff, 0x93, 0x40, TIB >> 24)
        u.mem_write(GDT+8, descriptor)
        u.mem_write(GDT+16, struct.pack("<HHBBBB", 0xffff, 0, 0, 0x93, 0xcf, 0))
        u.reg_write(UC_X86_REG_GDTR, (0, GDT, 23, 0))
        # Explicitly retain flat 32-bit data/stack segments after loading FS.
        for reg in (UC_X86_REG_SS, UC_X86_REG_DS, UC_X86_REG_ES):
            u.reg_write(reg, 16)
        u.reg_write(UC_X86_REG_FS, 8)
        u.mem_write(QCONT, b"\xc3")
        self.imports = {}
        for i, imp in enumerate(pe.imports):
            stub = STUB + 16*i
            u.mem_write(imp[0], pack(stub))
            u.mem_write(stub, b"\xc3")  # synthetic stub, never original executable bytes
            self.imports[stub] = imp
        self.case, self.blocks = case, blocks
        # GetTickCount is fixed for all calls and all replay paths of this case.
        self.tick = number(case.get("tick", 0))
        if not isinstance(self.tick, int) or not 0 <= self.tick <= 0xffffffff:
            raise ValueError("tick must be a u32")
        self.hits = set()
        self.objects = {name: ARENA+number(offset) for name, offset in case.get("objects", {}).items()}
        self.case_stubs = {k: v for k, v in case.get("stubs", {}).items() if not k.startswith("0x")}
        # 位址樁（"0x……"）：只准換登記簿裡「排除」類（公式邊界外的引擎流程，例 544630 建彈道），
        # 或深度已 D4 的「公式」類（MG 2026-10-09 裁甲，decisions/2026-10-09-compositional-stubs.md）；
        # 其餘公式類要跑原版，非公式／未登記的 helper 可能藏規則，都不准換
        self.stream_pos = {}
        self.addr_stubs = {int(k, 16): v for k, v in case.get("stubs", {}).items() if k.startswith("0x")}
        if self.addr_stubs:
            cls = formula_classes()
            for a in self.addr_stubs:
                c, d = cls.get(f"{a:06x}", ("未登記", ""))
                if c != "排除" and (c, d) != ("公式", "D4") and a not in IO_STUBS and not (native and c == "公式"):
                    raise ValueError(f"位址樁 {a:#x} 在登記簿是「{c} {d}」：只准換「排除」類或 D4「公式」類")
        removed = {}
        if native:
            removed = {a: spec for a, spec in self.addr_stubs.items()
                       if a not in IO_STUBS and cls.get(f"{a:06x}", ("", ""))[0] == "公式"}
            self.addr_stubs = {a: spec for a, spec in self.addr_stubs.items() if a not in removed}
        self.native = native
        self.sp = STACK + 0x8000
        u.reg_write(UC_X86_REG_ESP, self.sp)
        u.mem_write(self.sp, pack(STOP))
        for i, arg in enumerate(case.get("args", [])):
            u.mem_write(self.sp+4+4*i, pack(self.value(arg)))
        cc = case["cc"]
        if cc not in ("cdecl", "stdcall", "thiscall"):
            raise ValueError(f"unsupported calling convention {cc}")
        if "ecx" in case:
            u.reg_write(UC_X86_REG_ECX, self.value(case["ecx"]))
        for field in case.get("setup", []):
            self.write(field, self.value(field["value"]))
        # index 尋址（at.index）讀同一列的輸入值，先全部解析再寫
        self.named = {f["name"]: v for f, v in zip(case.get("in", []), inputs, strict=True) if "name" in f}
        for field, value in zip(case.get("in", []), inputs, strict=True):
            self.write(field, value)
            for target in field.get("also", []):
                self.write(target, value)
        if native:
            self.compare, self.unused_in = self.native_masks(removed)
        self.context = u.context_save()
        self.dirty = {}
        self.rng_stub = rng_stub
        u.hook_add(UC_HOOK_MEM_WRITE, self.memory_write)
        u.hook_add(UC_HOOK_CODE, self.code)
        u.hook_add(UC_HOOK_MEM_UNMAPPED, self.unmapped)

    def native_masks(self, removed):
        # Resolve byte ranges, including aliases and sequence cells, per input row.
        def span(field, size=None):
            if "reg" in field["at"]:
                return set()
            at = self.address(field)
            size = size or number(field.get("size", 4))
            return set() if at is None else set(range(at, at+size*number(field.get("n", 1))))

        records, sources = set(), set()
        for spec in removed.values():
            for field in spec.get("record", []):
                records.update(span(field))
            if "count" in spec:
                records.update(span(spec["count"], 4))
            for key, size in (("ret", 4), ("st0", 8)):
                if isinstance(spec.get(key), dict):
                    sources.update(span(spec[key], size))
            for field in spec.get("write", []):
                if isinstance(field["value"], dict):
                    sources.update(span(field["value"], number(field["size"])))
        compare = []
        for field in self.case.get("out", []):
            cells = span(field)
            if cells & records and not cells <= records:
                raise ValueError("native output partially overlaps formula record")
            compare.append(not bool(cells & records))
        unused = []
        for i, field in enumerate(self.case.get("in", [])):
            cells = span(field)
            if cells and cells <= sources and all("reg" not in t["at"] and span(t) <= sources for t in field.get("also", [])):
                unused.append(i)
        return compare, unused

    def value(self, v):
        return self.objects[v[1:]] if isinstance(v, str) and v.startswith("@") else number(v)

    def address(self, field):
        where = field["at"]
        if "arg" in where:
            return self.sp+4+4*where["arg"]
        if "object" in where:
            base = self.objects[where["object"]] + number(where.get("offset", 0))
            if "index" not in where:
                return base
            i, count = self.named[where["index"]], where["count"]
            if i < 0:
                return None  # 無記錄（如 itemIndex=-1）：寫入略過，讀取報錯
            if i >= count:
                raise ValueError(f"{where['index']}={i} 超出 count {count}")
            return base + i*number(where["stride"])
        return number(where["global"])

    def write(self, field, value):
        if "reg" in field["at"]:
            self.uc.reg_write(REGS[field["at"]["reg"]], value & 0xffffffff)
        else:
            size, at = field.get("size", 4), self.address(field)
            if at is None:
                return
            self.uc.mem_write(at, int(value).to_bytes(size, "little", signed=field.get("signed", False)))

    def read(self, field):
        size = field.get("size", 4)
        if "reg" in field["at"]:
            if field["at"]["reg"] == "st0":
                if size != 8:
                    raise ValueError("st0 output requires size 8")
                bits = self.st0_bits()
                return bits if bits < 1 << 63 else bits - (1 << 64)
            raw = self.uc.reg_read(REGS[field["at"]["reg"]]).to_bytes(4, "little")[:size]
        else:
            at = self.address(field)
            if at is None:
                raise ValueError(f"{field.get('name')}: index <0，沒有可讀的記錄")
            raw = self.uc.mem_read(at, size)
        return int.from_bytes(raw, "little", signed=field.get("signed", False))

    def st0_bits(self):
        """Read physical TOP without popping; round binary80 to binary64, ties to even."""
        u = self.uc
        top = (u.reg_read(UC_X86_REG_FPSW) >> 11) & 7
        if (u.reg_read(UC_X86_REG_FPTAG) >> (2*top)) & 3 == 3:
            raise ValueError("st0 output: empty x87 stack")
        mantissa, exponent = u.reg_read(UC_X86_REG_FP0 + top)
        sign, exp = (exponent & 0x8000) << 48, exponent & 0x7fff
        if exp == 0x7fff:
            if mantissa == 1 << 63:
                return sign | 0x7ff0000000000000
            # Quiet NaNs, retaining sign and the high payload bits.
            return sign | 0x7ff8000000000000 | ((mantissa >> 11) & ((1 << 51)-1))
        if exp and not mantissa & (1 << 63):
            raise ValueError("st0 output: unsupported unnormal x87 value")
        if not mantissa:
            return sign
        shift = (exp or 1) - 16383 - 63
        power = mantissa.bit_length()-1 + shift

        def rounded(right):
            if right <= 0:
                return mantissa << -right
            value, remainder = divmod(mantissa, 1 << right)
            half = 1 << (right-1)
            return value + (remainder > half or (remainder == half and value & 1))

        if power < -1022:
            # A rounded subnormal may become the smallest normal double.
            return sign | rounded(-1074-shift)
        significand = rounded(mantissa.bit_length()-53)
        if significand == 1 << 53:
            significand >>= 1
            power += 1
        if power > 1023:
            return sign | 0x7ff0000000000000
        return sign | ((power+1023) << 52) | (significand & ((1 << 52)-1))

    def memory_write(self, u, access, address, size, value, user):
        for page in range(address & ~4095, (address+size-1 & ~4095)+1, 4096):
            if page not in self.dirty:
                try:
                    self.dirty[page] = bytes(u.mem_read(page, 4096))
                except UcError:
                    # The unmapped hook reports the actual failed write address.
                    pass

    def unmapped(self, u, access, address, size, value, user):
        eip = u.reg_read(UC_X86_REG_EIP) or self.last_eip
        self.memory_fault = f"unmapped eip=0x{eip:08x} addr=0x{address:08x}"
        return False

    def memory(self, address, size, data=None):
        """Give host-side stub memory accesses the same diagnostics as x86 accesses."""
        try:
            if data is None:
                return self.uc.mem_read(address, size)
            # Host writes do not trigger Unicorn's write hook. Save their pages
            # too, so a formatter output cannot leak into the next replay path.
            self.memory_write(self.uc, None, address, size, 0, None)
            self.uc.mem_write(address, data)
        except UcError:
            # Find the first missing byte, including accesses straddling a mapping.
            regions = list(self.uc.mem_regions())
            end, missing = address+size, address
            while missing < end:
                region = next((r for r in regions if r[0] <= missing <= r[1]), None)
                if region is None:
                    self.unmapped(self.uc, None, missing, size, 0, None)
                    raise RuntimeError(self.memory_fault) from None
                missing = region[1]+1
            raise

    def text(self, address, wide=False):
        unit, raw = 2 if wide else 1, bytearray()
        for i in range(TEXT_LIMIT):
            chunk = self.memory(address+i*unit, unit)
            if not any(chunk):
                return raw.decode("utf-16le" if wide else "latin-1", errors="surrogatepass")
            raw.extend(chunk)
        raise ValueError("stub string exceeds text limit")

    def vswprintf(self, wide_out=True, inline=False):
        """Old MSVCRT cdecl (buf, UTF-16 fmt, va_list), no count argument.

        wide_out=False is vsprintf (narrow fmt/output; s/c default narrow, S wide;
        narrow output must be Latin-1). inline=True is sprintf (varargs on the stack).

        Supports diuxXcsS%, -/0, literal width, h/l. s/c default wide,
        S defaults narrow; h selects narrow and l wide. Integers are 32-bit
        (h truncates to 16 bits). Narrow text is Latin-1, no locale conversion.
        Precision, *, other flags/conversions fail; output capped at TEXT_LIMIT
        UTF-16 units. Caller owns buffer capacity, as with the old CRT API.
        """
        sp = self.uc.reg_read(UC_X86_REG_ESP)
        buf, fmt, va = (u32(self.memory(sp+4*i, 4)) for i in (1, 2, 3))
        if inline:
            va = sp+12
        unit = 2 if wide_out else 1
        enc = (lambda t: t.encode("utf-16le", errors="surrogatepass")) if wide_out else (lambda t: t.encode("latin-1"))
        fmt, pos, parts, total = self.text(fmt, wide_out), 0, [], 0
        while pos < len(fmt):
            start = pos
            if fmt[pos] != "%":
                part = fmt[pos]
                pos += 1
            else:
                pos += 1
                if pos < len(fmt) and fmt[pos] == "%":
                    part, pos = "%", pos+1
                else:
                    flags = ""
                    while pos < len(fmt) and fmt[pos] in "-0":
                        flags += fmt[pos]
                        pos += 1
                    width = 0
                    while pos < len(fmt) and "0" <= fmt[pos] <= "9":
                        width = width*10 + int(fmt[pos])
                        if width > TEXT_LIMIT:
                            raise ValueError("vswprintf: width exceeds text limit")
                        pos += 1
                    length = ""
                    if pos < len(fmt) and fmt[pos] in "hl":
                        length, pos = fmt[pos], pos+1
                    if pos >= len(fmt) or fmt[pos] not in "diuxXcsS":
                        raise ValueError(f"vswprintf: unsupported conversion {fmt[start:pos+1]!r}")
                    conv, pos = fmt[pos], pos+1
                    value = u32(self.memory(va, 4))
                    va += 4
                    numeric = conv in "diuxX"
                    if numeric:
                        bits = 16 if length == "h" else 32
                        value &= (1 << bits)-1
                        if conv in "di" and value & (1 << (bits-1)):
                            value -= 1 << bits
                        part = format(value, conv if conv in "xX" else "d")
                    else:
                        wide = length == "l" or (length != "h" and (conv != "S") == wide_out)
                        part = chr(value & (0xffff if wide else 0xff)) if conv == "c" else self.text(value, wide)
                    pad = max(0, width-len(enc(part))//unit)
                    if "-" in flags:
                        part += " "*pad
                    elif "0" in flags and numeric:
                        part = "-"+"0"*pad+part[1:] if part.startswith("-") else "0"*pad+part
                    else:
                        part = " "*pad+part
            total += len(enc(part))//unit
            if total > TEXT_LIMIT:
                raise ValueError("vswprintf: output exceeds text limit")
            parts.append(part)
        data = enc("".join(parts)) + b"\0"*unit
        self.memory(buf, len(data), data)
        self.ret(total)

    def ret(self, value, pop=0):
        """Return from an import stub; pop = argument bytes the callee cleans (stdcall／thiscall)."""
        u = self.uc
        sp = u.reg_read(UC_X86_REG_ESP)
        u.reg_write(UC_X86_REG_EAX, value & 0xffffffff)
        u.reg_write(UC_X86_REG_ESP, sp+4+pop)
        u.reg_write(UC_X86_REG_EIP, u32(u.mem_read(sp, 4)))

    def push_st0(self, x):
        """像 fld 一樣把 binary64 值推上 x87 堆疊（回 double 的 import 樁用）。"""
        u = self.uc
        sw = u.reg_read(UC_X86_REG_FPSW)
        top = (((sw >> 11) & 7) - 1) & 7
        u.reg_write(UC_X86_REG_FPSW, (sw & ~0x3800) | (top << 11))
        u.reg_write(UC_X86_REG_FPTAG, u.reg_read(UC_X86_REG_FPTAG) & ~(3 << 2*top))
        u.reg_write(UC_X86_REG_FP0 + top, f80(x))

    def ciacos(self, f=math.acos):
        """MSVCRT _CIacos／_CIasin: ST0 = f(ST0), no stack arguments.

        ponytail: computed in binary64 (math.acos) and written back exactly; the real
        x87 routine keeps 80-bit intermediates, so the last binary64 bit may differ."""
        u = self.uc
        x = struct.unpack("<d", self.st0_bits().to_bytes(8, "little"))[0]
        y = f(x) if -1 <= x <= 1 else math.nan
        top = (u.reg_read(UC_X86_REG_FPSW) >> 11) & 7
        u.reg_write(UC_X86_REG_FP0 + top, f80(y))
        self.ret(u.reg_read(UC_X86_REG_EAX))

    def cipow(self):
        """MSVCRT _CIpow: ST1 = ST1 ** ST0, pop; no stack arguments.

        ponytail: binary64 math.pow like ciacos; domain errors (negative base, fractional power) give NaN."""
        u = self.uc
        d = lambda: struct.unpack("<d", self.st0_bits().to_bytes(8, "little"))[0]
        e = d()
        top = (u.reg_read(UC_X86_REG_FPSW) >> 11) & 7
        u.reg_write(UC_X86_REG_FPTAG, u.reg_read(UC_X86_REG_FPTAG) | (3 << 2*top))
        u.reg_write(UC_X86_REG_FPSW, (u.reg_read(UC_X86_REG_FPSW) & ~0x3800) | (((top+1) & 7) << 11))
        b = d()
        try:
            y = math.pow(b, e)
        except (ValueError, OverflowError):
            y = math.nan
        u.reg_write(UC_X86_REG_FP0 + ((top+1) & 7), f80(y))
        self.ret(u.reg_read(UC_X86_REG_EAX))

    def ftol(self):
        u = self.uc
        status = u.reg_read(UC_X86_REG_FPSW)
        top = (status >> 11) & 7
        mantissa, exponent = u.reg_read(UC_X86_REG_FP0 + top)
        exp = exponent & 0x7fff
        if exp == 0x7fff:
            raise ValueError("_ftol: non-finite ST0")
        shift = (exp or 1) - 16383 - 63
        value = mantissa << shift if shift >= 0 else mantissa >> -shift
        if exponent & 0x8000:
            value = -value
        if not -(1 << 63) <= value < (1 << 63):
            raise ValueError("_ftol: int64 overflow")
        u.reg_write(UC_X86_REG_FPTAG, u.reg_read(UC_X86_REG_FPTAG, UC_X86_REG_FPCW) | (3 << (2*top)))
        u.reg_write(UC_X86_REG_FPSW, (status & ~0x3800) | (((top+1) & 7) << 11))
        u.reg_write(UC_X86_REG_EDX, (value >> 32) & 0xffffffff)
        self.ret(value)

    def qsort_order(self, base, num, width):
        """MSVCRT qsort (CUTOFF 8 shortsort + median-of-three quicksort), as a generator of
        comparator calls (a, b) that receives each signed result. Equal-key order follows this
        algorithm; not checked against the user's msvcrt.dll (notes/re/emu.md)."""
        u = self.uc

        def swap(a, b):
            if a != b:
                x, y = bytes(u.mem_read(a, width)), bytes(u.mem_read(b, width))
                u.mem_write(a, y)
                u.mem_write(b, x)
        if num < 2 or width == 0:
            return
        stack, lo, hi = [], base, base+width*(num-1)
        while True:
            size = (hi-lo)//width+1
            if size <= 8:
                while hi > lo:
                    mx, p = lo, lo+width
                    while p <= hi:
                        if (yield p, mx) > 0:
                            mx = p
                        p += width
                    swap(mx, hi)
                    hi -= width
            else:
                mid = lo+(size//2)*width
                if (yield lo, mid) > 0:
                    swap(lo, mid)
                if (yield lo, hi) > 0:
                    swap(lo, hi)
                if (yield mid, hi) > 0:
                    swap(mid, hi)
                loguy, higuy = lo, hi
                while True:
                    if mid > loguy:
                        while True:
                            loguy += width
                            if not (loguy < mid and (yield loguy, mid) <= 0):
                                break
                    if mid <= loguy:
                        while True:
                            loguy += width
                            if not (loguy <= hi and (yield loguy, mid) <= 0):
                                break
                    while True:
                        higuy -= width
                        if not (higuy > mid and (yield higuy, mid) > 0):
                            break
                    if higuy < loguy:
                        break
                    swap(loguy, higuy)
                    if mid == higuy:
                        mid = loguy
                higuy += width
                if mid < higuy:
                    while True:
                        higuy -= width
                        if not (higuy > mid and (yield higuy, mid) == 0):
                            break
                if mid >= higuy:
                    while True:
                        higuy -= width
                        if not (higuy > lo and (yield higuy, mid) == 0):
                            break
                if higuy-lo >= hi-loguy:
                    if lo < higuy:
                        stack.append((lo, higuy))
                    if loguy < hi:
                        lo = loguy
                        continue
                else:
                    if loguy < hi:
                        stack.append((loguy, hi))
                    if lo < higuy:
                        hi = higuy
                        continue
            if not stack:
                return
            lo, hi = stack.pop()

    def qsort_start(self):
        u = self.uc
        if self.qsort is not None:
            raise RuntimeError("qsort: nested call not supported")
        sp = u.reg_read(UC_X86_REG_ESP)
        base, num, width, self.qsort_cmp = (u32(u.mem_read(sp+4*i, 4)) for i in range(1, 5))
        self.qsort, self.qsort_sp = self.qsort_order(base, num, width), sp
        self.qsort_next(None)

    def qsort_next(self, result):
        u, sp = self.uc, self.qsort_sp
        try:
            a, b = self.qsort.send(result)
        except StopIteration:
            self.qsort = None
            self.ret(0)
            return
        u.mem_write(sp-4, pack(b))
        u.mem_write(sp-8, pack(a))
        u.mem_write(sp-12, pack(QCONT))
        u.reg_write(UC_X86_REG_ESP, sp-12)
        u.reg_write(UC_X86_REG_EIP, self.qsort_cmp)

    def apply_stub(self, spec):
        # case 指定的回傳值（物件名或整數）；pop＝callee 清的參數 bytes（stdcall／thiscall）
        u = self.uc
        sp = u.reg_read(UC_X86_REG_ESP)
        for r in spec.get("record", []):  # 把第 arg 個堆疊參數（1 起）的低 size bytes 抄到 to，供 out 讀
            size = number(r.get("size", 4))
            at = self.address(r)
            if "n" in r:  # 第 k 次呼叫寫 at+k*size（每次呼叫的參數都留下，不只末次）；超過 n 次停
                k = self.stream_pos.get(id(r), 0)
                if k >= number(r["n"]):
                    raise ValueError(f"stub {self.last_eip:#x} record sequence full after {k} calls")
                self.stream_pos[id(r)] = k+1
                at += k*size
            self.memory(at, size, bytes(u.mem_read(sp+4*number(r["arg"]), size)))
        if "count" in spec:  # 呼叫次數（u32）
            at = self.address(spec["count"])
            self.memory(at, 4, pack((u32(u.mem_read(at, 4))+1) & 0xffffffff))
        if "stream" in spec:
            st = spec["stream"]
            pos = self.stream_pos.get(id(st), 0)
            want = u32(u.mem_read(sp+12, 4))
            n = max(0, min(want, number(st["size"])-pos))
            if n:
                self.memory(u32(u.mem_read(sp+8, 4)), n, bytes(self.memory(self.address(st)+pos, n)))
            self.stream_pos[id(st)] = pos+n
            got = u32(u.mem_read(sp+16, 4))
            if got:
                self.memory(got, 4, pack(n))
            spec = dict(spec, ret=int(n == want))
        ret = spec["ret"]
        if isinstance(ret, dict) and "n" in ret:  # {"at","n"}：第 k 次呼叫讀 at+4k（依呼叫序的 case 輸入）；超過 n 次停
            k = self.stream_pos.get(id(ret), 0)
            if k >= number(ret["n"]):
                raise ValueError(f"stub ret sequence exhausted after {k} calls")
            self.stream_pos[id(ret)] = k+1
            ret = u32(self.memory(self.address(ret)+4*k, 4))
        elif isinstance(ret, dict):  # {"at": …}：從該位置讀 u32（可當 case 輸入，讓差分涵蓋回傳值）
            ret = u32(self.memory(self.address(ret), 4))
        else:
            ret = self.objects[ret] if isinstance(ret, str) and ret in self.objects else number(ret)
        u.reg_write(UC_X86_REG_EAX, ret & 0xffffffff)
        if "st0" in spec:  # 回 double 的 import：數值，或 {"at": …} 從該位置讀 8 bytes double（可當 case 輸入）
            st = spec["st0"]
            self.push_st0(struct.unpack("<d", self.memory(self.address(st), 8))[0] if isinstance(st, dict) else float(st))
        for w in spec.get("write", []):  # 寫入參數指標指的結構：第 arg 個堆疊參數（1 起）＋offset；或 global＝絕對位址（樁掉的函式改的全域）
            if "global" in w:
                ptr = number(w["global"])
            else:
                ptr = u32(self.memory(sp+4*number(w["arg"]), 4))
            if ptr == 0 and w.get("nullskip"):  # 呼叫端傳 NULL 輸出指標（原函式自己判空）：不寫；需在 case 明示
                continue
            dst = ptr + number(w.get("offset", 0))
            n = number(w["size"])
            v = w["value"]
            if isinstance(v, dict) and "n" in v:  # {"at","n"}：第 k 次呼叫讀 at+k*size（依呼叫序的 case 輸入）；超過 n 次停
                k = self.stream_pos.get(id(v), 0)
                if k >= number(v["n"]):
                    raise ValueError(f"stub write sequence exhausted after {k} calls")
                self.stream_pos[id(v)] = k+1
                self.memory(dst, n, bytes(self.memory(self.address(v)+k*n, n)))
            elif isinstance(v, dict):  # {"at": …}：從該位置讀 size bytes（可當 case 輸入，讓差分涵蓋樁寫出的值）
                self.memory(dst, n, bytes(self.memory(self.address(w["value"]), n)))
            else:
                self.memory(dst, n, (self.value(w["value"]) & (1 << 8*n)-1).to_bytes(n, "little"))
        u.reg_write(UC_X86_REG_ESP, sp+4+number(spec.get("pop", 0)))
        u.reg_write(UC_X86_REG_EIP, u32(u.mem_read(sp, 4)))

    def code(self, u, address, size, user):
        self.last_eip = address
        if address in self.blocks:
            self.run_hits.add(address)
        if address in self.addr_stubs:
            self.apply_stub(self.addr_stubs[address])
            return
        if address in self.imports:
            iat, dll, name = self.imports[address]
            if name in self.case_stubs:
                self.apply_stub(self.case_stubs[name])
                return
            if name == "_ftol":
                self.ftol()
                return
            if name == "qsort":
                self.qsort_start()
                return
            if name == "vswprintf":
                self.vswprintf()
                return
            if name in ("vsprintf", "sprintf"):
                self.vswprintf(wide_out=False, inline=name == "sprintf")
                return
            if name == "swprintf":
                self.vswprintf(inline=True)
                return
            if name == "floor":
                # cdecl double floor(double)：參數在 [esp+4]，結果推上 ST0（binary64 上是精確的）
                x = struct.unpack("<d", self.memory(u.reg_read(UC_X86_REG_ESP)+4, 8))[0]
                self.push_st0(float(math.floor(x)) if math.isfinite(x) else x)
                self.ret(u.reg_read(UC_X86_REG_EAX))
                return
            if name == "_CIpow":
                self.cipow()
                return
            if name in ("_CIacos", "_CIasin"):
                self.ciacos(math.asin if name == "_CIasin" else math.acos)
                return
            if name == "fopen":
                # 檔案 I/O 不模擬：fopen 一律失敗（NULL），呼叫端走「開不了檔」路徑
                self.ret(0)
                return
            if name == "CreateFileA":
                # 同 fopen：一律失敗，回 INVALID_HANDLE_VALUE；stdcall 7 個參數
                self.ret(-1, 28)
                return
            if name == "memmove":
                sp = u.reg_read(UC_X86_REG_ESP)
                dst, src, n = (u32(self.memory(sp+4*i, 4)) for i in (1, 2, 3))
                if n:
                    self.memory(dst, n, bytes(self.memory(src, n)))  # 先整段讀出再寫：重疊也對
                self.ret(dst)
                return
            if name in STR_FUNCS:
                # cdecl 字串函式：wide 用 UTF-16、narrow 用 latin-1；比較照 C 語意回 <0／0／>0
                sp = u.reg_read(UC_X86_REG_ESP)
                a1, a2, a3 = (u32(self.memory(sp+4*i, 4)) for i in (1, 2, 3))
                wide = name.startswith("wcs")
                if name in ("wcslen",):
                    self.ret(len(self.text(a1, True).encode("utf-16le", errors="surrogatepass")) // 2)
                elif name == "wcscspn":  # 第一個落在 a2 字元集的位置（UTF-16 code unit）
                    x, cs = (self.text(v, True).encode("utf-16le", errors="surrogatepass") for v in (a1, a2))
                    units, keep = struct.unpack(f"<{len(x)//2}H", x), set(struct.unpack(f"<{len(cs)//2}H", cs))
                    self.ret(next((i for i, c in enumerate(units) if c in keep), len(units)))
                elif name == "wcscat":
                    tail = self.text(a2, True).encode("utf-16le", errors="surrogatepass") + b"\0\0"
                    self.memory(a1 + len(self.text(a1, True).encode("utf-16le", errors="surrogatepass")), len(tail), tail)
                    self.ret(a1)
                elif name == "strchr":
                    t, c = self.text(a1), a2 & 0xff
                    i = len(t) if c == 0 else t.find(chr(c))
                    self.ret(0 if i < 0 else a1 + i)
                elif name == "atoi":
                    m = re.match(r"[ \t\n\v\f\r]*([+-]?\d+)", self.text(a1))
                    self.ret((int(m.group(1)) if m else 0) & 0xffffffff)  # ponytail: 溢位不飽和（MSVCRT 舊版同樣環繞）
                elif name == "strncpy":
                    src = self.text(a2).encode("latin-1")[:a3]
                    self.memory(a1, a3, src + b"\0" * (a3 - len(src))) if a3 else None
                    self.ret(a1)
                else:  # wcscmp／_stricmp／_strnicmp
                    x, y = self.text(a1, wide), self.text(a2, wide)
                    if name != "wcscmp":
                        x, y = (v.translate(C_LOCALE_LOWER) for v in (x, y))  # C 區域（exe 無 setlocale）只折 A-Z
                    if name == "_strnicmp":
                        x, y = x[:a3], y[:a3]
                    if wide:  # 依 UTF-16 code unit 比
                        x, y = (struct.unpack(f"<{len(v.encode('utf-16le', errors='surrogatepass'))//2}H", v.encode("utf-16le", errors="surrogatepass")) for v in (x, y))
                    self.ret(((x > y) - (x < y)) & 0xffffffff)
                return
            if name == "wcscpy":
                sp = u.reg_read(UC_X86_REG_ESP)
                dst, src = (u32(self.memory(sp+4*i, 4)) for i in (1, 2))
                data = self.text(src, True).encode("utf-16le", errors="surrogatepass") + b"\0\0"
                self.memory(dst, len(data), data)
                self.ret(dst)
                return
            if name == "?getNewFrameHandle@srTexture@@KAKXZ":
                # 靜態、無參數：回新的 frame handle，每條重放路徑從 1 起遞增
                self.frame_handle += 1
                self.ret(self.frame_handle)
                return
            if dll.upper() == "SR.DLL" and name in SR_VOID_SETTERS:
                # SR.DLL 內部狀態（材質取樣設定、快取作廢、執行期類別名、自動釋放登記），exe 記憶體看不到：不模擬
                self.ret(u.reg_read(UC_X86_REG_EAX), SR_VOID_SETTERS[name])
                return
            if name == "FormatMessageA":
                # 錯誤訊息文字不模擬：回 0（失敗、沒寫任何字），stdcall 7 個參數
                self.ret(0, 28)
                return
            if name == "GetLastError":
                # CreateFileA／fopen 一律失敗，對應的錯誤碼是 ERROR_FILE_NOT_FOUND
                self.ret(2)
                return
            if dll.upper() == "SR.DLL" and name in ("??0srClass@@IAE@XZ", "??0srTexture@@IAE@XZ"):
                self.ret(u.reg_read(UC_X86_REG_ECX))
                return
            if dll.upper() == "SR.DLL" and name == "??0srNode@@QAE@PAV0@@Z":
                # ponytail: 不掛進父節點的子串列（節點樹不模擬）；thiscall、1 個參數（父節點）
                self.ret(u.reg_read(UC_X86_REG_ECX), 4)
                return
            if name == "GetTickCount":
                self.ret(self.tick)
                return
            if name in ("malloc", "??2@YAPAXI@Z", "?allocate@srHeap@@QAEPAXK@Z"):
                # ponytail: bump 配置、free 不回收；HEAP_SIZE 用完就報錯。srHeap::allocate 是 thiscall（ecx＝heap，callee 清 4）
                n = u32(self.memory(u.reg_read(UC_X86_REG_ESP)+4, 4))
                at, self.heap = self.heap, self.heap + max(8, n+7 & ~7)
                if self.heap > HEAP+HEAP_SIZE:
                    raise RuntimeError(f"{name}({n}): emu heap exhausted")
                self.ret(at, 4 if name.startswith("?allocate@") else 0)
                return
            if name in ("free", "??3@YAXPAX@Z"):
                self.ret(0)
                return
            if dll.upper() == "SR.DLL" and name == "?getMsTime@srTimer@@UAEKW4e_timerReadControl@1@@Z":
                # thiscall、一個參數（讀取控制），callee 清 4 bytes；回 case 的 tick，與 GetTickCount 同源
                sp = u.reg_read(UC_X86_REG_ESP)
                u.reg_write(UC_X86_REG_EAX, self.tick)
                u.reg_write(UC_X86_REG_ESP, sp+8)
                u.reg_write(UC_X86_REG_EIP, u32(u.mem_read(sp, 4)))
                return
            if name == "?srAssertFail@@YAXPBD0J0ZZ":
                sp = u.reg_read(UC_X86_REG_ESP)
                file = self.text(u32(self.memory(sp+8, 4)))
                line = struct.unpack("<i", self.memory(sp+12, 4))[0]
                raise RuntimeError(f"srAssertFail stop {file}:{line}")
            raise RuntimeError(f"unstubbed import 0x{iat:x} {dll}!{name}")
        if address == QCONT:
            u.reg_write(UC_X86_REG_ESP, self.qsort_sp)
            self.qsort_next(struct.unpack("<i", pack(u.reg_read(UC_X86_REG_EAX)))[0])
            return
        # 40efa0 回 [0,n)；40efe0 是百分比擲骰（rand 取 0..99 與參數比，回 v<pct），同一條亂數軌記 [100, v]
        if address in (0x40efa0, 0x40efe0) and self.rng_stub:
            arg = u32(u.mem_read(u.reg_read(UC_X86_REG_ESP)+4, 4))
            n = arg if address == 0x40efa0 else 100
            pos = len(self.trace)
            if pos < len(self.prefix):
                value = self.prefix[pos]
            elif self.sampler is not None:
                value = self.sampler.randrange(max(1, n))
            else:
                raise Branch(n)
            if not 0 <= value < max(1, n):
                raise ValueError("RNG prefix out of range")
            self.trace.append([n, value])
            self.ret(value if address == 0x40efa0 else int(value < arg))

    def run(self, prefix, sampler=None):
        u = self.uc
        for page, data in self.dirty.items():
            u.mem_write(page, data)
        self.dirty.clear()
        u.context_restore(self.context)
        self.prefix, self.sampler, self.trace = prefix, sampler, []
        self.qsort = None
        self.heap = HEAP
        self.stream_pos = {}  # 每次重放（DFS 前綴、抽樣）串流都從頭讀
        self.frame_handle = 0
        self.run_hits = set()
        self.last_eip = number(self.case["addr"])
        self.memory_fault = None
        try:
            u.emu_start(number(self.case["addr"]), STOP, count=self.case["steps"])
        except UcError:
            if self.memory_fault:
                raise RuntimeError(self.memory_fault) from None
            raise
        if u.reg_read(UC_X86_REG_EIP) != STOP:
            raise RuntimeError(f"instruction limit {self.case['steps']} exceeded")
        expected = self.sp+4
        if self.case["cc"] in ("stdcall", "thiscall"):
            expected += 4*len(self.case.get("args", []))
        if u.reg_read(UC_X86_REG_ESP) != expected:
            raise RuntimeError("calling convention stack imbalance")
        if len(self.trace) < len(prefix):
            raise ValueError("unused RNG prefix")
        self.hits.update(self.run_hits)
        return {**({"compare": self.compare, "unused_in": self.unused_in} if self.native else {}),
                "in": self.inputs, "rng": self.trace,
                "out": [self.read(f) for f in self.case.get("out", [])], "sampled": sampler is not None}


def paths(machine, case, seed):
    pending, result = [()], []
    while pending:
        prefix = pending.pop()
        try:
            result.append(machine.run(prefix))
        except Branch as branch:
            width = max(1, branch.n)  # the original R(0) returns zero
            # Every frontier prefix has at least one leaf. Abort before expanding huge n.
            if len(result)+len(pending)+width > case["path_limit"]:
                break
            pending.extend(prefix+(v,) for v in range(width-1, -1, -1))
    else:
        return result
    machine.hits.clear()  # coverage belongs only to emitted complete paths
    sampler = random.Random(seed)
    return [machine.run((), sampler) for _ in range(case["samples"])]


def selftest_stubs(pe):
    """Synthetic x86 callers only; nothing copied from the executable.

    srAssertFail terminates normal runs (stderr/exit 1 in main); this selftest
    catches its exception. Node cases must avoid assert paths and record their
    assert-only blocks as cov-exempt in their own tickets.
    """
    def machine(code, tick=0):
        case = {"addr": ARENA, "cc": "cdecl", "steps": 1000, "tick": tick}
        m = Machine(pe, case, [], set())
        m.inputs = []
        m.uc.mem_write(ARENA, code)
        return m

    def call(name, args):
        iat = next(i for i, dll, n in pe.imports if n == name)
        code = b"".join(b"\x68"+pack(v) for v in reversed(args))
        code += b"\xff\x15"+pack(iat)
        if args:
            code += b"\x83\xc4"+bytes([4*len(args)])
        return code+b"\xc3"

    buf, fmt, va, narrow, wide = (ARENA+i for i in (0x1000, 0x2000, 0x3000, 0x4000, 0x5000))
    m = machine(call("vswprintf", [buf, fmt, va]))
    m.uc.mem_write(fmt, "hp=%d x=%x n=%S w=%s %04d%%\0".encode("utf-16le"))
    m.uc.mem_write(va, b"".join(pack(v) for v in (-7, 0x1f, narrow, wide, 42)))
    m.uc.mem_write(narrow, b"ab\0")
    m.uc.mem_write(wide, "AB\0".encode("utf-16le"))
    m.run(())
    result = m.text(buf, True)
    if result != "hp=-7 x=1f n=ab w=AB 0042%" or m.uc.reg_read(UC_X86_REG_EAX) != len(result):
        raise RuntimeError(f"vswprintf selftest wrong result: {result!r}")
    print(f'vswprintf "{result}"')

    m = machine(call("sprintf", [buf, fmt, -7, wide, narrow, 0]))
    m.uc.mem_write(fmt, b"hp=%d w=%S n=%s %03x\0")
    m.uc.mem_write(narrow, b"ab\0")
    m.uc.mem_write(wide, "AB\0".encode("utf-16le"))
    m.run(())
    if m.text(buf) != "hp=-7 w=AB n=ab 000" or m.uc.reg_read(UC_X86_REG_EAX) != 19:
        raise RuntimeError(f"sprintf selftest wrong result {m.text(buf)!r}")
    print('sprintf "hp=-7 w=AB n=ab 000"')

    m = machine(call("vsprintf", [buf, fmt, va]))
    m.uc.mem_write(fmt, b"%-4d|\0")
    m.uc.mem_write(va, pack(42))
    m.run(())
    if m.text(buf) != "42  |":
        raise RuntimeError(f"vsprintf selftest wrong result {m.text(buf)!r}")
    print('vsprintf "42  |"')

    case = {"addr": ARENA, "cc": "cdecl", "steps": 100, "objects": {"node": 0x7000},
            "stubs": {"?getMsTime@srTimer@@UAEKW4e_timerReadControl@1@@Z": {"ret": "node", "pop": 4}}}
    m = Machine(pe, case, [], set())
    m.inputs = []
    m.uc.mem_write(ARENA, b"\x6a\x01"+call("?getMsTime@srTimer@@UAEKW4e_timerReadControl@1@@Z", []))
    m.run(())
    if m.uc.reg_read(UC_X86_REG_EAX) != ARENA+0x7000:
        raise RuntimeError("case stub selftest wrong result")
    case["stubs"] = {"GlobalMemoryStatus": {"ret": 0, "pop": 4, "write": [{"arg": 1, "offset": 8, "size": 4, "value": "0x12345678"}, {"arg": 1, "offset": 12, "size": 2, "value": "@node"},
                                                        {"arg": 1, "offset": 14, "size": 2, "value": {"at": {"object": "node", "offset": 0x20}}}]}}
    m = Machine(pe, case, [], set())
    m.inputs = []
    m.uc.mem_write(ARENA+0x7020, pack(0xbeefcafe))
    m.uc.mem_write(ARENA, b"\x6a\x55"+call("GlobalMemoryStatus", [buf])[:-4]+b"\x5a\xc3")
    m.run(())
    if (bytes(m.uc.mem_read(buf+8, 8)), m.uc.reg_read(UC_X86_REG_EDX)) != (pack(0x12345678)+pack(ARENA+0x7000)[:2]+pack(0xcafe)[:2], 0x55):
        raise RuntimeError("case stub write selftest wrong")
    case["stubs"] = {"?getLocationX@srNode@@QBENXZ": {"ret": 0, "st0": {"at": {"object": "node", "offset": 8}}},
                     "?getLocationY@srNode@@QBENXZ": {"ret": 0, "st0": "-2.5"}}
    m = Machine(pe, case, [], set())
    m.inputs = []
    m.uc.mem_write(ARENA+0x7008, struct.pack("<d", 1.25))
    m.uc.mem_write(ARENA, call("?getLocationX@srNode@@QBENXZ", [])[:-1]+call("?getLocationY@srNode@@QBENXZ", [])[:-1]
                   + b"\xde\xc1\xdd\x1d"+pack(buf)+b"\xc3")  # faddp; fstp qword [buf]
    m.run(())
    if struct.unpack("<d", bytes(m.uc.mem_read(buf, 8)))[0] != -1.25 or (m.uc.reg_read(UC_X86_REG_FPSW) >> 11) & 7:
        raise RuntimeError("case stub st0 selftest wrong")
    print("case stub ok")

    # 位址樁（544630 登記為排除）：push 7; push 9; call 0x544630; add esp,8; ret → eax=5、記第 2 參數 7、次數 1
    case = {"addr": ARENA, "cc": "cdecl", "steps": 100, "objects": {"node": 0x7000},
            "stubs": {"0x544630": {"ret": 5, "record": [{"arg": 2, "at": {"object": "node"}}],
                                             "count": {"at": {"object": "node", "offset": 4}}}}}
    m = Machine(pe, case, [], set())
    m.inputs = []
    m.uc.mem_write(ARENA, b"\x6a\x07\x6a\x09\xe8"+pack((0x544630-(ARENA+9)) & 0xffffffff)+b"\x83\xc4\x08\xc3")
    m.run(())
    if (m.uc.reg_read(UC_X86_REG_EAX), u32(m.uc.mem_read(ARENA+0x7000, 4)), u32(m.uc.mem_read(ARENA+0x7004, 4))) != (5, 7, 1):
        raise RuntimeError("address stub selftest wrong")
    # record 帶 n：push 7 呼叫、push 8 呼叫 → node+0x10=7、+0x14=8（每次呼叫各留一格）
    rec = {"0x544630": {"ret": 0, "record": [{"arg": 1, "at": {"object": "node", "offset": 0x10}, "n": 2}]}}
    m = Machine(pe, dict(case, stubs=rec), [], set())
    m.inputs = []
    c544 = lambda o: b"\xe8"+pack((0x544630-(ARENA+o+5)) & 0xffffffff)
    m.uc.mem_write(ARENA, b"\x6a\x07"+c544(2)+b"\x6a\x08"+c544(9)+b"\x83\xc4\x08\xc3")
    m.run(())
    if (u32(m.uc.mem_read(ARENA+0x7010, 4)), u32(m.uc.mem_read(ARENA+0x7014, 4))) != (7, 8):
        raise RuntimeError("address stub record sequence selftest wrong")
    # write 帶 nullskip：push 0 呼叫（NULL 輸出指標）不寫不 fault；push buf 呼叫照寫
    ws = {"0x544630": {"ret": 0, "write": [{"arg": 1, "size": 4, "value": "0x77", "nullskip": True}]}}
    m = Machine(pe, dict(case, stubs=ws), [], set())
    m.inputs = []
    m.uc.mem_write(ARENA, b"\x6a\x00"+c544(2)+b"\x68"+pack(ARENA+0x7018)+c544(12)+b"\x83\xc4\x08\xc3")
    m.run(())
    if u32(m.uc.mem_read(ARENA+0x7018, 4)) != 0x77:
        raise RuntimeError("address stub write nullskip selftest wrong")
    # 40efe0 百分比樁：push 30; call 40efe0 → 無前綴時 Branch(100)；前綴 29 → 1、30 → 0
    m = Machine(pe, dict(case, stubs={}), [], set())
    m.inputs = []
    m.uc.mem_write(ARENA, b"\x6a\x1e\xe8"+pack((0x40efe0-(ARENA+7)) & 0xffffffff)+b"\x83\xc4\x04\xc3")
    try:
        m.run(())
        raise RuntimeError("40efe0 stub selftest: expected Branch(100)")
    except Branch as b:
        if b.n != 100:
            raise RuntimeError(f"40efe0 stub selftest: Branch({b.n})")
    for v, want in ((29, 1), (30, 0)):
        m.run((v,))
        if (m.uc.reg_read(UC_X86_REG_EAX), m.trace) != (want, [[100, v]]):
            raise RuntimeError("40efe0 stub selftest wrong")
    print("40efe0 pct ok")
    # write global＋依呼叫序取值：樁 544630 每次呼叫把 node+0x20+4k 寫到全域 node+0x40；迴圈 while(*g) call，g 初值 2 → 依序寫 1、0，呼叫 2 次（第三格 9 不讀）
    g = ARENA+0x7040
    wg = {"0x544630": {"ret": 0, "count": {"at": {"object": "node", "offset": 4}},
                       "write": [{"global": hex(g), "size": 4, "value": {"at": {"object": "node", "offset": 0x20}, "n": 3}}]}}
    m = Machine(pe, dict(case, stubs=wg), [], set())
    m.inputs = []
    m.uc.mem_write(ARENA+0x7020, pack(1)+pack(0)+pack(9))
    m.uc.mem_write(g, pack(2))
    # loop: cmp dword [g],0; je end; call 544630; jmp loop; end: ret
    loop = b"\x83\x3d"+pack(g)+b"\x00\x74\x07"+b"\xe8"+pack((0x544630-(ARENA+14)) & 0xffffffff)+b"\xeb\xf0\xc3"
    m.uc.mem_write(ARENA, loop)
    m.run(())
    if (u32(m.uc.mem_read(g, 4)), u32(m.uc.mem_read(ARENA+0x7004, 4))) != (0, 2):
        raise RuntimeError(f"write global sequence selftest wrong {u32(m.uc.mem_read(g, 4))} {u32(m.uc.mem_read(ARENA+0x7004, 4))}")
    print("write global seq ok")
    # D4 公式類（5194e0）可樁，回傳值從 case 位置讀
    m = Machine(pe, dict(case, stubs={"0x5194e0": {"ret": {"at": {"object": "node", "offset": 8}}}}), [], set())
    m.inputs = []
    m.uc.mem_write(ARENA+0x7008, pack(0x2a))
    m.uc.mem_write(ARENA, b"\x6a\x07\x6a\x09\xe8"+pack((0x5194e0-(ARENA+9)) & 0xffffffff)+b"\x83\xc4\x08\xc3")
    m.run(())
    if m.uc.reg_read(UC_X86_REG_EAX) != 0x2a:
        raise RuntimeError("D4 formula address stub selftest wrong")
    # Native selection and masks use resolved storage, including sequence aliases.
    def field(offset, **kw):
        return {"at": {"object": "node", "offset": offset}, **kw}
    native_case = dict(case, **{"in": [field(8), field(12)],
        "out": [field(16), field(20), field(24), field(28)],
        "stubs": {"0x5194e0": {"ret": field(8, n=2),
                    "record": [field(16, arg=1, n=2)], "count": field(24)},
                  "0x544630": {"ret": 0}, "0x404ea0": {"ret": 0}}})
    m = Machine(pe, native_case, [3, 4], set(), native=True)
    if set(m.addr_stubs) != {0x544630, 0x404ea0} or m.compare != [False, False, False, True] or m.unused_in != [0, 1]:
        raise RuntimeError("native stub/mask selftest wrong")
    m = Machine(pe, native_case, [3, 4], set())
    if 0x5194e0 not in m.addr_stubs:
        raise RuntimeError("normal formula stub unexpectedly removed")
    print("native selection/masks ok")
    # 依呼叫序回傳：兩次呼叫分別回 at、at+4，重放從頭；第三次呼叫超過 n=2 要停
    seq = {"0x5194e0": {"ret": {"at": {"object": "node", "offset": 8}, "n": 2}}}
    m = Machine(pe, dict(case, stubs=seq), [], set())
    m.inputs = []
    m.uc.mem_write(ARENA+0x7008, pack(3)+pack(4))
    call2 = lambda o: b"\xe8"+pack((0x5194e0-(ARENA+o+5)) & 0xffffffff)
    code = call2(0)+b"\xa3"+pack(ARENA+0x7034)
    code += call2(len(code))+b"\xc3"
    m.uc.mem_write(ARENA, code)
    for _ in range(2):
        m.run(())
        if (u32(m.uc.mem_read(ARENA+0x7034, 4)), m.uc.reg_read(UC_X86_REG_EAX)) != (3, 4):
            raise RuntimeError("sequenced ret stub selftest wrong")
    m = Machine(pe, dict(case, stubs=seq), [], set())  # 新機器：unicorn 會快取已翻譯的程式碼塊，不能就地改碼
    m.inputs = []
    m.uc.mem_write(ARENA+0x7008, pack(3)+pack(4))
    m.uc.mem_write(ARENA, code[:-1]+call2(len(code)-1)+b"\xc3")
    try:
        m.run(())
    except ValueError:
        pass
    else:
        raise RuntimeError("sequenced ret stub must stop past n")
    # 404ea0 存檔串流：讀 2＋讀 4（只剩 2）→ 第二次回 0、got=2
    m = Machine(pe, dict(case, stubs={"0x404ea0": {"ret": 0, "stream": {"at": {"object": "node", "offset": 16}, "size": 4}}}), [], set())
    m.inputs = []
    m.uc.mem_write(ARENA+0x7010, b"\x11\x22\x33\x44")
    rd = lambda n, buf: b"\x68"+pack(ARENA+0x7030)+b"\x6a"+bytes([n])+b"\x68"+pack(ARENA+buf)+b"\x6a\x05"
    code = rd(2, 0x7020)
    code += b"\xe8"+pack((0x404ea0-(ARENA+len(code)+5)) & 0xffffffff)+b"\x83\xc4\x10\xa3"+pack(ARENA+0x7034)
    code += rd(4, 0x7022)
    code += b"\xe8"+pack((0x404ea0-(ARENA+len(code)+5)) & 0xffffffff)+b"\x83\xc4\x10\xc3"
    m.uc.mem_write(ARENA, code)
    m.run(())
    got = (u32(m.uc.mem_read(ARENA+0x7034, 4)), m.uc.reg_read(UC_X86_REG_EAX), bytes(m.uc.mem_read(ARENA+0x7020, 4)), u32(m.uc.mem_read(ARENA+0x7030, 4)))
    if got != (1, 0, b"\x11\x22\x33\x44", 2):
        raise RuntimeError(f"stream stub selftest wrong {got}")
    m.run(())  # 同一台 Machine 重放：串流要從頭讀
    again = (u32(m.uc.mem_read(ARENA+0x7034, 4)), m.uc.reg_read(UC_X86_REG_EAX), bytes(m.uc.mem_read(ARENA+0x7020, 4)), u32(m.uc.mem_read(ARENA+0x7030, 4)))
    if again != got:
        raise RuntimeError(f"stream stub replay wrong {again}")
    for bad in ("0x571660", f"{ARENA+0x100:#x}"):  # 非公式類（D0 公式會陸續升 D4，不拿來當例）、未登記
        try:
            Machine(pe, dict(case, stubs={bad: {"ret": 0}}), [], set())
        except ValueError:
            continue
        raise RuntimeError(f"address stub {bad} must be refused")
    print("address stub ok")

    m = machine(call("fopen", [narrow, narrow]))
    m.uc.reg_write(UC_X86_REG_EAX, 1)
    m.run(())
    if m.uc.reg_read(UC_X86_REG_EAX) != 0:
        raise RuntimeError("fopen selftest: expected NULL")
    print("fopen NULL")

    m = machine(call("wcscpy", [buf, wide]))
    m.uc.mem_write(wide, "hello\0".encode("utf-16le"))
    m.uc.mem_write(buf, b"\xff"*12)
    m.run(())
    if bytes(m.uc.mem_read(buf, 12)) != "hello\0".encode("utf-16le") or m.uc.reg_read(UC_X86_REG_EAX) != buf:
        raise RuntimeError("wcscpy selftest wrong result")
    print('wcscpy "hello" ok')

    m = machine(b"\xb9"+pack(buf)+call("??0srClass@@IAE@XZ", []))
    sentinel = b"\xa5"*32
    m.uc.mem_write(buf, sentinel)
    m.run(())
    if m.uc.reg_read(UC_X86_REG_EAX) != buf or bytes(m.uc.mem_read(buf, len(sentinel))) != sentinel:
        raise RuntimeError("srClass selftest wrong result")
    print("srClass ok")

    rec = {"name": "v", "at": {"object": "r", "offset": 2, "index": "i", "stride": 8, "count": 10}, "size": 2}
    case = {"addr": ARENA, "cc": "cdecl", "steps": 10, "objects": {"r": 0x8000},
            "in": [{"name": "i", "at": {"arg": 0}, "signed": True}, rec]}
    got = []
    for i, v in ((3, 0x1234), (5, 0x5678), (-1, 0x9abc)):
        m = Machine(pe, case, [i, v], set())
        got.append([int.from_bytes(m.uc.mem_read(ARENA+0x8000+2+8*k, 2), "little") for k in (3, 5)])
        if i >= 0 and m.read(rec) != v:
            raise RuntimeError("index selftest read-back wrong")
    try:
        Machine(pe, case, [10, 1], set())
        raise RuntimeError("index selftest: count not enforced")
    except ValueError:
        pass
    if got != [[0x1234, 0], [0, 0x5678], [0, 0]]:
        raise RuntimeError(f"index selftest wrong records: {got}")
    print("index ok")

    m = machine(b"\xdd\x05"+pack(wide)+b"\xc3")
    m.uc.mem_write(wide, struct.pack("<d", 3.5))
    m.case["out"] = [{"at": {"reg": "st0"}, "size": 8}]
    result = m.run(())["out"][0]
    if result != 0x400c000000000000:
        raise RuntimeError("st0 selftest wrong result")
    print(f"st0 0x{result & ((1 << 64)-1):016x}")

    # malloc(5) → edx；operator new(16) → eax；第二塊要在第一塊 8 對齊之後
    m = machine(call("malloc", [5])[:-1]+b"\x8b\xd0"+call("??2@YAPAXI@Z", [16]))
    m.run(())
    a, b = m.uc.reg_read(UC_X86_REG_EDX), m.uc.reg_read(UC_X86_REG_EAX)
    if (a, b) != (HEAP, HEAP+8):
        raise RuntimeError(f"heap selftest wrong: {a:#x} {b:#x}")
    print("heap ok")

    # srHeap::allocate(3) 接在 malloc(5) 後；thiscall 不靠呼叫端清堆疊，esp 要回到原位（push 0x55 當哨兵）
    m = machine(b"\x6a\x55"+call("malloc", [5])[:-1]+b"\xb9"+pack(ARENA)+call("?allocate@srHeap@@QAEPAXK@Z", [3])[:-4]+b"\x5a\xc3")
    m.run(())
    if (m.uc.reg_read(UC_X86_REG_EAX), m.uc.reg_read(UC_X86_REG_EDX)) != (HEAP+8, 0x55):
        raise RuntimeError("srHeap::allocate selftest wrong")
    m = machine(b"\x6a\x55"+call("CreateFileA", [ARENA, 0, 0, 0, 3, 0, 0])[:-4]+b"\x5a\xc3")
    m.run(())
    if (m.uc.reg_read(UC_X86_REG_EAX), m.uc.reg_read(UC_X86_REG_EDX)) != (0xffffffff, 0x55):
        raise RuntimeError("CreateFileA selftest wrong")
    print("srHeap allocate ok, CreateFileA INVALID")

    m = machine(call("GetLastError", [])+b"")
    m.run(())
    if m.uc.reg_read(UC_X86_REG_EAX) != 2:
        raise RuntimeError("GetLastError selftest wrong")
    m = machine(b"\xb9"+pack(ARENA+0x40)+call("??0srTexture@@IAE@XZ", []))
    m.run(())
    if m.uc.reg_read(UC_X86_REG_EAX) != ARENA+0x40:
        raise RuntimeError("srTexture ctor selftest wrong")
    m = machine(b"\xb9"+pack(ARENA+0x40)+b"\x6a\x55"+call("??0srNode@@QAE@PAV0@@Z", [ARENA])[:-4]+b"\x5a\xc3")
    m.run(())
    if (m.uc.reg_read(UC_X86_REG_EAX), m.uc.reg_read(UC_X86_REG_EDX)) != (ARENA+0x40, 0x55):
        raise RuntimeError("srNode ctor selftest wrong")
    print("GetLastError 2, srTexture this, srNode pop 4")

    m = machine(b"\xdd\x05"+pack(wide)+call("_CIacos", []))
    m.uc.mem_write(wide, struct.pack("<d", 0.5))
    m.case["out"] = [{"at": {"reg": "st0"}, "size": 8}]
    got = m.run(())["out"][0] & ((1 << 64)-1)
    if got != struct.unpack("<Q", struct.pack("<d", math.acos(0.5)))[0]:
        raise RuntimeError(f"_CIacos selftest wrong: {got:#x}")
    m = machine(b"\x6a\x55"+call("FormatMessageA", [0x1000, 0, 2, 0, buf, 64, 0])[:-4]+b"\x5a\xc3")
    m.run(())
    if (m.uc.reg_read(UC_X86_REG_EAX), m.uc.reg_read(UC_X86_REG_EDX)) != (0, 0x55):
        raise RuntimeError("FormatMessageA selftest wrong")
    m = machine(call("swprintf", [buf, fmt, 42]))
    m.uc.mem_write(fmt, "n=%d\0".encode("utf-16le"))
    m.run(())
    if bytes(m.uc.mem_read(buf, 10)) != "n=42\0".encode("utf-16le"):
        raise RuntimeError("swprintf selftest wrong")
    print("_CIacos 0.5, FormatMessageA 0, swprintf ok")
    m = machine(b"\x68"+struct.pack("<d", -2.5)[4:]+b"\x68"+struct.pack("<d", -2.5)[:4]+call("floor", [])[:-1]+b"\x83\xc4\x08\xdd\x1d"+pack(buf)+b"\xc3")
    m.run(())
    if struct.unpack("<d", bytes(m.uc.mem_read(buf, 8)))[0] != -3.0 or (m.uc.reg_read(UC_X86_REG_FPSW) >> 11) & 7:
        raise RuntimeError("floor selftest wrong")
    m = machine(b"\xdd\x05"+pack(wide)+call("_CIasin", []))
    m.uc.mem_write(wide, struct.pack("<d", 0.5))
    m.case["out"] = [{"at": {"reg": "st0"}, "size": 8}]
    if m.run(())["out"][0] & ((1 << 64)-1) != struct.unpack("<Q", struct.pack("<d", math.asin(0.5)))[0]:
        raise RuntimeError("_CIasin selftest wrong")
    m = machine(call("memmove", [buf+2, buf, 6]))
    m.uc.mem_write(buf, b"abcdefgh")
    m.run(())
    if (bytes(m.uc.mem_read(buf, 8)), m.uc.reg_read(UC_X86_REG_EAX)) != (b"ababcdef", buf+2):
        raise RuntimeError("memmove selftest wrong")
    m = machine(b"\xdd\x05"+pack(wide)+b"\xdd\x05"+pack(wide+8)+call("_CIpow", []))
    m.uc.mem_write(wide, struct.pack("<dd", 81.0, 0.75))
    m.case["out"] = [{"at": {"reg": "st0"}, "size": 8}]
    if m.run(())["out"][0] & ((1 << 64)-1) != struct.unpack("<Q", struct.pack("<d", 27.0))[0] or (m.uc.reg_read(UC_X86_REG_FPSW) >> 11) & 7 != 7:
        raise RuntimeError("_CIpow selftest wrong")
    print("floor -3, _CIasin 0.5, memmove overlap, _CIpow 27")
    m = machine(call("wcscat", [wide, wide+0x100])[:-1]+b"\x50"+call("wcslen", [wide])[:-1]+b"\x5a\xc3")
    m.uc.mem_write(wide, "ab\0".encode("utf-16le")); m.uc.mem_write(wide+0x100, "cd\0".encode("utf-16le"))
    m.run(())
    if (m.text(wide, True), m.uc.reg_read(UC_X86_REG_EAX)) != ("abcd", 4):
        raise RuntimeError("wcscat/wcslen selftest wrong")
    for a, cs, want in (("ab[cd", "[]", 2), ("abcd", "[]", 4), ("", "x", 0)):
        m = machine(call("wcscspn", [wide, wide+0x100])); m.uc.mem_write(wide, (a+"\0").encode("utf-16le")); m.uc.mem_write(wide+0x100, (cs+"\0").encode("utf-16le")); m.run(())
        if m.uc.reg_read(UC_X86_REG_EAX) != want:
            raise RuntimeError(f"wcscspn selftest wrong {a!r} {cs!r}")
    m.uc.mem_write(narrow, b"  -42x\0"); m.uc.mem_write(buf, b"HeLLo\0"); m.uc.mem_write(buf+0x40, b"help\0")
    got = []
    for nm, args in (("atoi", [narrow]), ("strchr", [buf, ord("L")]), ("_stricmp", [buf, buf+0x40]), ("_strnicmp", [buf, buf+0x40, 3]), ("wcscmp", [wide, wide+0x100])):
        m = machine(call(nm, args)); m.uc.mem_write(narrow, b"  -42x\0"); m.uc.mem_write(buf, b"HeLLo\0"); m.uc.mem_write(buf+0x40, b"help\0")
        m.uc.mem_write(wide, "aB\0".encode("utf-16le")); m.uc.mem_write(wide+0x100, "ab\0".encode("utf-16le"))
        m.run(()); got.append(struct.unpack("<i", struct.pack("<I", m.uc.reg_read(UC_X86_REG_EAX)))[0])
    m = machine(call("_stricmp", [buf, buf+0x40])); m.uc.mem_write(buf, b"\xc0\0"); m.uc.mem_write(buf+0x40, b"\xe0\0"); m.run(())
    if m.uc.reg_read(UC_X86_REG_EAX) != 0xffffffff:
        raise RuntimeError("_stricmp C-locale selftest wrong (0xC0 vs 0xE0 must differ)")
    m = machine(call("strncpy", [buf+0x80, buf, 8])); m.uc.mem_write(buf, b"HeLLo\0"); m.uc.mem_write(buf+0x80, b"\xff"*9); m.run(())
    if got != [-42, buf+2-(1 << 32) if buf+2 >= 1 << 31 else buf+2, -1, 0, -1] or bytes(m.uc.mem_read(buf+0x80, 9)) != b"HeLLo\0\0\0\xff":
        raise RuntimeError(f"str stubs selftest wrong {got}")
    print("wcscat abcd 4, atoi strchr stricmp strnicmp wcscmp strncpy")

    h = "?getNewFrameHandle@srTexture@@KAKXZ"
    m = machine(call(h, [])[:-1]+b"\x8b\xd0"+call(h, []))
    for _ in range(2):  # 第二次 run 要從 1 重新起算
        m.run(())
    if (m.uc.reg_read(UC_X86_REG_EDX), m.uc.reg_read(UC_X86_REG_EAX)) != (1, 2):
        raise RuntimeError("getNewFrameHandle selftest wrong")
    m = machine(b"\x6a\x55"+b"\xb9"+pack(ARENA)+call("?invalidateFrameHandle@srTexture@@IAEXK@Z", [7])[:-4]+b"\x5a\xc3")
    m.run(())
    if m.uc.reg_read(UC_X86_REG_EDX) != 0x55:
        raise RuntimeError("invalidateFrameHandle selftest wrong")
    print("getNewFrameHandle 1 2, invalidateFrameHandle pop 4")

    m = machine(b"\xb9"+pack(buf)+b"\x6a\x01"+call("?getMsTime@srTimer@@UAEKW4e_timerReadControl@1@@Z", []), tick=777)
    m.run(())
    if m.uc.reg_read(UC_X86_REG_EAX) != 777:
        raise RuntimeError("getMsTime selftest wrong result")
    print("getMsTime 777")

    m = machine(call("GetTickCount", []), tick=12345)
    for _ in range(2):
        m.run(())
        if m.uc.reg_read(UC_X86_REG_EAX) != 12345:
            raise RuntimeError("GetTickCount selftest wrong result")
    print("GetTickCount 12345")

    # Save initial FS:[0] and self pointer, then write EBX and read FS:[0] back.
    initial, selfptr = ARENA+0x1000, ARENA+0x1004
    code = b"\x64\xa1"+pack(0)+b"\xa3"+pack(initial)
    code += b"\x64\xa1"+pack(0x18)+b"\xa3"+pack(selfptr)
    code += b"\xbb"+pack(0x12345678)+b"\x64\x89\x1d"+pack(0)
    code += b"\x64\xa1"+pack(0)+b"\xc3"
    m = machine(code)
    for _ in range(2):
        m.run(())
        if (u32(m.uc.mem_read(initial, 4)) != 0xffffffff or
                u32(m.uc.mem_read(selfptr, 4)) != TIB or
                m.uc.reg_read(UC_X86_REG_EAX) != 0x12345678):
            raise RuntimeError("FS selftest wrong result")
    print("fs0 ok")

    cond, file, msg = ARENA+0x1000, ARENA+0x2000, ARENA+0x3000
    m = machine(call("?srAssertFail@@YAXPBD0J0ZZ", [cond, file, 77, msg]))
    for addr, text in ((cond, b"x\0"), (file, b"foo.cpp\0"), (msg, b"msg\0")):
        m.uc.mem_write(addr, text)
    try:
        m.run(())
    except RuntimeError as error:
        if str(error) != "srAssertFail stop foo.cpp:77":
            raise
        print(error)
    else:
        raise RuntimeError("srAssertFail selftest unexpectedly returned")

    # Read, write and fetch faults must all carry the failed address and EIP.
    for code, eip in ((b"\xa1"+pack(4)+b"\xc3", ARENA),
                      (b"\xa3"+pack(4)+b"\xc3", ARENA),
                      (b"\xb8"+pack(4)+b"\xff\xe0", 4)):
        m = machine(code)
        try:
            m.run(())
        except RuntimeError as error:
            if str(error) != f"unmapped eip=0x{eip:08x} addr=0x00000004":
                raise
            if eip == ARENA and code[0] == 0xa1:
                print(error)
        else:
            raise RuntimeError("unmapped selftest unexpectedly returned")


def main():
    native = len(sys.argv) == 3 and sys.argv[1] == "--native"
    if len(sys.argv) != 2 and not native:
        raise ValueError("usage: run.sh <8-digit address>|--native <address>|--selftest-import|--selftest-qsort|--selftest-stubs")
    pe = PE(ROOT / "build/Wiz8.exe")
    if sys.argv[1] == "--selftest-stubs":
        selftest_stubs(pe)
        return
    if sys.argv[1] == "--selftest-import":
        # Execute the original RNG function with its hook disabled; it calls CRT rand.
        case = {"addr": "0x40efa0", "cc": "cdecl", "args": [6], "steps": 1000}
        machine = Machine(pe, case, [], set(), rng_stub=False)
        machine.inputs = []
        machine.run(())
        raise RuntimeError("selftest unexpectedly returned without an import failure")
    if sys.argv[1] == "--selftest-qsort":
        # Synthetic caller and comparator in the arena (not exe bytes): qsort(arr, n, 4, int-compare).
        iat = next(i for i, dll, name in pe.imports if name == "qsort")
        rng = random.Random(7)
        values = [rng.randint(-50, 50) for _ in range(40)]
        assert len(set(values)) > 20, values
        arr, cmp_, caller = ARENA+0x1000, ARENA+0x100, ARENA
        code = b"\x8b\x44\x24\x04\x8b\x00\x8b\x4c\x24\x08\x2b\x01\xc3"
        call = b"".join(b"\x68"+pack(v) for v in (cmp_, 4, len(values), arr)) + b"\xff\x15"+pack(iat) + b"\x83\xc4\x10\xc3"
        for n in (len(values), 7, 1):
            case = {"addr": hex(caller), "cc": "cdecl", "steps": 200000}
            machine = Machine(pe, case, [], set())
            machine.inputs = []
            u = machine.uc
            u.mem_write(cmp_, code)
            u.mem_write(caller, call.replace(b"\x68"+pack(len(values)), b"\x68"+pack(n), 1))
            u.mem_write(arr, b"".join(struct.pack("<i", v) for v in values))
            machine.context = u.context_save()
            machine.run(())
            got = list(struct.unpack(f"<{len(values)}i", u.mem_read(arr, 4*len(values))))
            if got != sorted(values[:n]) + values[n:]:
                raise RuntimeError(f"qsort selftest n={n} wrong order: {got[:n]}")
        print("ok qsort selftest n=40,7,1")
        return
    addr = sys.argv[2].lower().zfill(8) if native else sys.argv[1].lower()
    if len(addr) != 8 or any(c not in "0123456789abcdef" for c in addr):
        raise ValueError("address must be eight hexadecimal digits")
    dest = ROOT / ("build/emu-f" if native else "build/emu")
    dest.mkdir(parents=True, exist_ok=True)
    # Stale success evidence must not survive a failed execution.
    for suffix in ("jsonl", "hit", "d3f" if native else "d3"):
        (dest / f"{addr}.{suffix}").unlink(missing_ok=True)
    case = json.loads((ROOT / f"re/emu/cases/{addr}.json").read_text())
    if native:
        overlay = ROOT / f"re/emu/native/{addr}.json"
        if overlay.exists():
            extra = json.loads(overlay.read_text())
            if set(extra) - {"objects", "setup", "stubs"}:
                raise ValueError("native overlay allows only objects/setup/stubs")
            for key in ("objects", "stubs"):
                if set(extra.get(key, {})) & set(case.get(key, {})):
                    raise ValueError(f"native overlay cannot replace {key}")
                case.setdefault(key, {}).update(extra.get(key, {}))
            case.setdefault("setup", []).extend(extra.get("setup", []))
    case["addr"] = "0x"+addr
    for key in ("steps", "path_limit", "samples"):
        if case[key] <= 0:
            raise ValueError(f"{key} must be positive")
    blocks = None
    for line in (ROOT / "build/re/funcs.tsv").read_text().splitlines():
        columns = line.split("\t")
        if int(columns[0], 16) == int(addr, 16):
            blocks = {int(v, 16) for v in columns[6].split(",") if v}
            break
    if not blocks:
        raise ValueError("function has no basic block metadata")
    rand = case["random"]
    gen = random.Random(rand["seed"])
    inputs = list(case["boundary"])
    inputs += [[gen.randint(*limits) for limits in rand["ranges"]] for _ in range(rand["count"])]
    all_hits, rows = set(), []
    for i, values in enumerate(inputs):
        machine = Machine(pe, case, values, blocks, native=native)
        machine.inputs = values
        try:
            rows.extend(paths(machine, case, rand["seed"]+i))
        except (ValueError, RuntimeError, UcError) as error:
            raise RuntimeError(f"input {i}: {error}") from error
        all_hits.update(machine.hits)
    (dest / f"{addr}.jsonl").write_text("".join(json.dumps(row, separators=(",", ":"))+"\n" for row in rows))
    (dest / f"{addr}.hit").write_text("".join(f"{hit:x}\n" for hit in sorted(all_hits)))
    print(f"{addr}: paths={len(rows)} blocks={len(all_hits)}/{len(blocks)}")


if __name__ == "__main__":
    try:
        main()
    except (ValueError, RuntimeError, OSError, UcError) as error:
        print(error, file=sys.stderr)
        sys.exit(1)

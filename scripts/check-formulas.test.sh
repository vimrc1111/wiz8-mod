#!/usr/bin/env bash
# check-formulas.sh 的夾具測試：基準綠＋注入各自紅（刪一列、重複列、D4 改 D2、刪覆蓋登記、候選多一個、刪欄位命名）。不需要 exe／Ghidra。
# 另測 func-fields.sh：從 asm 夾具抽每個函式的 [reg±off] 記憶體運算元（D1 欄位）。
set -uo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); t=$(mktemp -d); trap 'rm -rf "$t"' EXIT
mkdir -p "$t/iface" "$t/calc" "$t/emu" "$t/fgd"
T=$'\t'; bad=0
printf '%s\n' "401000${T}32${T}00${T}40efa0${T}${T}600000${T}401000,401010" \
  "402000${T}40${T}00${T}401000${T}685174${T}685174${T}402000" \
  "403000${T}10${T}00${T}${T}${T}${T}403000" \
  "40efa0${T}80${T}00${T}${T}${T}${T}40efa0" > "$t/funcs.tsv"
c=$(bash "$here/scripts/formula-candidates.sh" "$t/funcs.tsv")
printf '#candidates\t%s\t%s\naddr\tclass\tdepth\tnode\tcalc\tevidence\n' "$(printf '%s\n' "$c" | grep -c .)" "$(printf '%s\n' "$c" | sha256sum | cut -d' ' -f1)" > "$t/fg.tsv"
printf '%s\n' "401000${T}公式${T}D4${T}rand_pct${T}RandPct${T}夾具" >> "$t/fg.tsv"
printf '%s\n' "addr${T}class${T}depth${T}node${T}calc${T}evidence" "402000${T}非公式${T}-${T}sheet${T}${T}畫角色表 build/wiz8.asm:12" > "$t/fgd/b00.tsv"
printf '%s\n' "call${T}40efa0${T}rng${T}亂數" "data${T}600000${T}table${T}規則表" > "$t/iface/401000.tsv"
echo 'func RandPct(n int) int { return n }' > "$t/calc/a.go"
echo 'cases=5 diff=0' > "$t/emu/401000.d3"; echo 401000 > "$t/emu/401000.hit"
printf '401000\t401010\t錯誤分支，夾具\n' > "$t/covx.tsv"
# asm 夾具（objdump 格式）：esp 基底是堆疊不算；402000 有 ebp 框架，ebp 基底不算；無位移寫成 +0x0；位址外的列不算
printf '%s\n' '  400ff0:	mov    eax,DWORD PTR [edx+0x99]' \
  '  401000:	push   esi' '  401001:	mov    eax,DWORD PTR [esi+0x24]' '  401004:	mov    ecx,DWORD PTR [esp+0x8]' \
  '  401008:	mov    BYTE PTR [ebx+0x1d8],al' '  40100e:	mov    eax,DWORD PTR ds:0x600000' '  401013:	movsx  edi,WORD PTR [esi]' \
  '  401016:	mov    edx,DWORD PTR [eax+ecx*4+0x10]' '  40101a:	lea    ecx,[edi+eax*1+0x1]' '  40101e:	ret' \
  '  402000:	push   ebp' '  402001:	mov    ebp,esp' '  402003:	mov    eax,DWORD PTR [ebp+0x8]' '  402006:	mov    ecx,DWORD PTR [edi-0x4]' '  402009:	ret' > "$t/asm"
want=$(printf '%s\n' "401000${T}eax+ecx*4+0x10,ebx+0x1d8,esi+0x0,esi+0x24" "402000${T}edi-0x4")
if [ -f "$here/scripts/func-fields.sh" ] && got=$(bash "$here/scripts/func-fields.sh" "$t/asm" "$t/funcs.tsv" | grep -v "${T}\$") && [ "$got" = "$want" ]; then echo "PASS fields:scan"
else echo "FAIL fields scan：要"; printf '%s\n' "$want" | sed 's/^/  /'; echo "  得"; printf '%s\n' "${got:-（無或腳本不存在）}" | sed 's/^/  /'; bad=1; fi
printf '%s\n' "$want" > "$t/fields.tsv"
sed 2d "$t/asm" | sha256sum | cut -d' ' -f1 > "$t/asm.sha"
printf '%s\n' "field${T}eax+ecx*4+0x10${T}table${T}夾具" "field${T}ebx+0x1d8${T}flag${T}夾具" "field${T}esi+0x0${T}base${T}夾具" "field${T}esi+0x24${T}hp${T}夾具" >> "$t/iface/401000.tsv"
run() { ASM=$t/asm ASMSHA=$t/asm.sha FIELDS=$t/fields.tsv FUNCS=$t/funcs.tsv FG=$t/fg.tsv FGD=$t/fgd IFACE=$t/iface CALC=$t/calc EMU=$t/emu COVX=$t/covx.tsv bash "$here/scripts/check-formulas.sh" >"$t/out" 2>&1; }
expect() { # 名稱 期望 rc(0/1) 期望輸出含的字
  run; rc=$?
  if [ "$2" = 0 ] && [ $rc = 0 ]; then echo "PASS $1"
  elif [ "$2" = 1 ] && [ $rc != 0 ] && grep -q "$3" "$t/out"; then echo "PASS $1（紅在 $3）"
  else echo "FAIL $1 rc=$rc"; sed 's/^/  /' "$t/out"; bad=1; fi
}
inject() { cp "$1" "$t/bak"; }; restore() { cp "$t/bak" "$1"; }
expect baseline 0
inject "$t/fgd/b00.tsv"; sed -i '/^402000/d' "$t/fgd/b00.tsv"; expect inject:row 1 'unclassified 402000'; restore "$t/fgd/b00.tsv"
grep 402000 "$t/fgd/b00.tsv" > "$t/fgd/b01.tsv"; expect inject:dup 1 'dup 402000'; rm "$t/fgd/b01.tsv"
inject "$t/fg.tsv"; sed -i 's/\tD4\t/\tD2\t/' "$t/fg.tsv"; expect inject:depth 1 'final 401000'; restore "$t/fg.tsv"
inject "$t/covx.tsv"; : > "$t/covx.tsv"; expect inject:cov 1 'd4 401000 基本塊 401010'; restore "$t/covx.tsv"
inject "$t/funcs.tsv"; printf '404000\t20\t00\t40efa0\t\t\t404000\n' >> "$t/funcs.tsv"; expect inject:cand 1 'candidates'; restore "$t/funcs.tsv"
inject "$t/iface/401000.tsv"; sed -i '/esi+0x24/d' "$t/iface/401000.tsv"; expect fields:d1 1 'd1 401000 欄位 esi+0x24'; restore "$t/iface/401000.tsv"
inject "$t/asm"; sed -i '3d' "$t/asm"; expect inject:asm 1 'asm：'; restore "$t/asm"
inject "$t/asm"; sed -i '2s/^/x/' "$t/asm"; expect asm:path-line 0
restore "$t/asm"
inject "$t/fields.tsv"; rm "$t/fields.tsv"; expect fields:missing 1 'fields'; restore "$t/fields.tsv"
# 節點票的覆蓋登記寫在 cov-exempt.d/
inject "$t/covx.tsv"; mkdir -p "$t/covx.d"; cp "$t/covx.tsv" "$t/covx.d/401000.tsv"; : > "$t/covx.tsv"; expect cov:dir 0; rm -r "$t/covx.d"; restore "$t/covx.tsv"
# re/emu/run.sh 寫的證據檔名補零到 8 位，要認得
mv "$t/emu/401000.d3" "$t/emu/00401000.d3"; mv "$t/emu/401000.hit" "$t/emu/00401000.hit"; expect emu:padded 0
mv "$t/emu/00401000.d3" "$t/emu/401000.d3"; mv "$t/emu/00401000.hit" "$t/emu/401000.hit"
# 稽核 zoWN 發現 4：差分產物（jsonl）比 .d3 新＝.d3 不是這次的差分，不算 D3
touch -d '1 min ago' "$t/emu/401000.d3"; echo '{}' > "$t/emu/401000.jsonl"; expect emu:stale-d3 1 'd3 401000'
touch "$t/emu/401000.d3"; touch -d '1 min ago' "$t/emu/401000.jsonl"; expect emu:fresh-d3 0; rm "$t/emu/401000.jsonl"
# vtable 入口表：假 PE（ImageBase 0x400000，一節 .rdata VA 0x1000 檔案位移 0x200），槽 0x401008 放 0x405000
python3 -I -c '
import struct, sys
d = bytearray(0x400); d[0:2] = b"MZ"; struct.pack_into("<I", d, 0x3c, 0x80); d[0x80:0x84] = b"PE\0\0"
struct.pack_into("<H", d, 0x86, 1); struct.pack_into("<H", d, 0x94, 0xe0); struct.pack_into("<I", d, 0x80+0x34, 0x400000)
struct.pack_into("<IIII", d, 0x80+0x18+0xe0+8, 0x100, 0x1000, 0x100, 0x200); struct.pack_into("<I", d, 0x208, 0x405000)
open(sys.argv[1], "wb").write(d)' "$t/exe"
printf '405000\t401008\t夾具\n' > "$t/vte.tsv"; printf '405000\t排除\t-\tvt\t\t夾具 build/wiz8.asm:1\n' > "$t/fgd/b02.tsv"
run() { ASM=$t/asm ASMSHA=$t/asm.sha VTE=$t/vte.tsv EXE=$t/exe FIELDS=$t/fields.tsv FUNCS=$t/funcs.tsv FG=$t/fg.tsv FGD=$t/fgd IFACE=$t/iface CALC=$t/calc EMU=$t/emu COVX=$t/covx.tsv bash "$here/scripts/check-formulas.sh" >"$t/out" 2>&1; }
expect vtable:ok 0
inject "$t/vte.tsv"; sed -i 's/401008/40100c/' "$t/vte.tsv"; expect vtable:slot 1 'nofunc 405000'; restore "$t/vte.tsv"
inject "$t/fgd/b02.tsv"; sed -i 's/\t排除\t-\t/\t公式\tD0\t/' "$t/fgd/b02.tsv"; expect vtable:class 1 'vtable 405000'; restore "$t/fgd/b02.tsv"
mv "$t/exe" "$t/exe.x"; expect vtable:noexe 1 'nofunc 405000'; mv "$t/exe.x" "$t/exe"
expect restored 0
[ $bad = 0 ] && echo "PASS check-formulas" || { echo "FAIL check-formulas"; exit 1; }

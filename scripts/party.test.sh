#!/usr/bin/env bash
# party.test.sh：calc party 的驗收（goals/stage10.md B3「初始隊伍最佳化」）。
# 獨立對照：本檔的 Python 直接從 Wiz8.exe 讀種族初值（0x614cf0）與職業門檻（0x614e24），自己算創角可達的屬性，
# 與 calc party 的輸出比：單指標最大值、雙指標 Pareto 前緣（值的集合要完全相同）、輸出內互不支配。
# 創角規則（notes/re/growth.md:11–17、:21）：D=Σmax(0,M−R)，B=60−D<0 不可選；付門檻後 A=max(R,M)；自由點每項上限 min(B,max(3,⌈B/3⌉))，裸值 ≤100；
# 職業 2 要求性別 1。沒有 build/Wiz8.exe 印 SKIP。
set -uo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
fail() { echo "FAIL party $*"; exit 1; }
[ -f build/Wiz8.exe ] || { echo "SKIP party：沒有 build/Wiz8.exe"; exit 0; }
t=$(mktemp -d); trap 'rm -rf "$t"' EXIT
go test -count=1 -v -run 'TestParty' ./internal/calc ./tools/calc > "$t/go" 2>&1 || { grep -E 'FAIL|Error' "$t/go" | head; fail "go test TestParty 紅"; }
grep -q -- '--- PASS: TestParty' "$t/go" || fail "沒有 TestParty 測試"
for m in attr:0 attr:3 l100:6 attr:0,attr:5 attr:4,l100:1; do
  go run ./tools/calc party -exe build/Wiz8.exe -metric "$m" > "$t/out" 2> "$t/err" || fail "calc party -metric $m 非零退出：$(tail -3 "$t/err")"
  python3 -I - build/Wiz8.exe "$m" "$t/out" <<'EOF' || fail "-metric $m 與獨立計算不符"
import struct, sys, itertools
exe, metrics, out = sys.argv[1], sys.argv[2].split(","), sys.argv[3]
b = open(exe, "rb").read()
pe = struct.unpack_from("<I", b, 0x3c)[0]
nsec, optsz = struct.unpack_from("<H", b, pe+6)[0], struct.unpack_from("<H", b, pe+20)[0]
base = struct.unpack_from("<I", b, pe+24+28)[0]
secs = [struct.unpack_from("<8sIIII", b, pe+24+optsz+40*i) for i in range(nsec)]
def u32(va):
    rva = va - base
    for _, vs, va0, rs, raw in secs:
        if va0 <= rva < va0 + max(vs, rs):
            return struct.unpack_from("<I", b, raw + rva - va0)[0]
    raise SystemExit(f"va {va:#x} not in a section")
R = [[u32(0x614cf0+28*r+4*i) for i in range(7)] for r in range(11)]
M = [[u32(0x614e24+28*p+4*i) for i in range(7)] for p in range(15)]
def value(m, a):  # 越大越好：attr 取值；l100 取負的等級（越早越好），輸出欄位印正的等級
    kind, i = m.split(":"); i = int(i)
    return a[i] if kind == "attr" else -(1 + max(0, -(-(100 - a[i]) // 3)))
want = set()
for r, p in itertools.product(range(11), range(15)):
    B = 60 - sum(max(0, M[p][i] - R[r][i]) for i in range(7))
    if B < 0: continue
    A = [max(R[r][i], M[p][i]) for i in range(7)]
    K = min(B, max(3, -(-B // 3)))
    idx = sorted({int(m.split(":")[1]) for m in metrics})
    for add in itertools.product(range(K+1), repeat=len(idx)):
        if sum(add) > B: continue
        a = list(A)
        for i, x in zip(idx, add): a[i] = min(100, a[i] + x)
        want.add(tuple(value(m, a) for m in metrics))
front = {v for v in want if not any(w != v and all(x >= y for x, y in zip(w, v)) for w in want)}
rows = [l.rstrip("\n").split("\t") for l in open(out) if l.strip()]
hdr, body = rows[0], rows[1:]
cols = [hdr.index(m) for m in metrics]
got = {tuple((int(r[c]) if m.startswith("attr") else -int(r[c])) for c, m in zip(cols, metrics)) for r in body}
dom = [v for v in got if any(w != v and all(x >= y for x, y in zip(w, v)) for w in got)]
if got != front or dom:
    print(f"metric {metrics}: want front {sorted(front)[:8]}… ({len(front)}) got {sorted(got)[:8]}… ({len(got)}) dominated-in-output {dom[:4]}", file=sys.stderr); sys.exit(1)
EOF
done
# 不給 -metric（B3「不指定就輸出各指標的 Pareto 前緣」）＝14 個單一指標依序各一段、段間空行，每段等於 -metric 該指標的輸出
c=/tmp/claude-1000/calc-party-t; go build -o "$c" ./tools/calc || fail "build calc"
"$c" party -exe build/Wiz8.exe > "$t/all" 2> "$t/err" || fail "calc party 無 -metric 非零退出：$(tail -3 "$t/err")"
: > "$t/want"; first=1
for k in attr l100; do for i in 0 1 2 3 4 5 6; do
  [ $first = 1 ] || echo >> "$t/want"; first=0
  "$c" party -exe build/Wiz8.exe -metric "$k:$i" >> "$t/want" || fail "-metric $k:$i"
done; done
cmp -s "$t/all" "$t/want" || fail "無 -metric 的輸出不等於 14 段單指標輸出（$(grep -c '^race' "$t/all") 段）"
rm -f "$c"
echo "ok party"

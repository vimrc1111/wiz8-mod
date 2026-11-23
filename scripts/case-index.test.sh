#!/usr/bin/env bash
# case-index.test.sh [<addr>…]：票 Y4a–e 驗收（不給位址＝五個全驗）（grok 稽核 zoWN 發現 1–3：case 用 `also` 把同一值寫進固定幾筆記錄，「選了哪一筆」測不到）。
# 1. 稽核給的兩行會紅 case 在 boundary 裡（不准拿掉）；2. 五個節點的 case 不再用 `also`；
# 3. 物品 index 的 random 範圍上限 ≥100；4. 所驗節點 node-accept 全綠。index 尋址本身（selftest `index ok`）由 emu.test.sh 驗（票 Y4）。沒有 exe 印 SKIP。
set -uo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
fail() { echo "FAIL case-index $*"; exit 1; }
[ -f build/Wiz8.exe ] || { echo "SKIP case-index：沒有 build/Wiz8.exe"; exit 0; }
set -- ${@:-51cce0 51e920 55cc00 4ebc80 52b4f0}
python3 -I - "$@" <<'PY' || fail "case 檔結構"
import json, sys
bad = []
sel = {"00" + a for a in sys.argv[1:]}
need = {"0051cce0": [4, 11], "0051e920": [4, 10, 0, 5, 0, 0]}
for a, row in need.items():
    if a not in sel: continue
    d = json.load(open(f"re/emu/cases/{a}.json"))
    if row not in d.get("boundary", []): bad.append(f"{a} boundary 少了稽核行 {row}")
    names = [f["name"] for f in d["in"]]
    hi = d["random"]["ranges"][names.index("itemIndex")][1]
    if hi < 100: bad.append(f"{a} itemIndex random 上限 {hi} < 100")
for a in sorted(sel):
    d = json.load(open(f"re/emu/cases/{a}.json"))
    if any("also" in f for f in d["in"]): bad.append(f"{a} 還在用 also")
for b in bad: print(b)
sys.exit(1 if bad else 0)
PY
for a in "$@"; do
  bash scripts/node-accept.sh tickets/nodes/$a.list > /dev/null 2>&1 || fail "node-accept $a 紅：$(bash scripts/node-accept.sh tickets/nodes/$a.list 2>&1 | grep -m3 -E 'FAIL|diff=')"
done
echo "ok case-index"

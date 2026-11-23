#!/usr/bin/env bash
# node-accept.sh <位址清單檔>：節點票的驗收（goals/stage10.md B1）。清單一行 `位址<TAB>calc 函數名`（位址同 funcs.tsv，不補零；函數名＝nodes 登記的 F）。
# 對每個節點：跑模擬器 → TestEmu 寫 .d3 → 在登記簿的暫存副本把這些列改成 D4、calc 欄填清單的函數名 → check-formulas --progress。
# 登記簿本身不改（深度由 cc 收貨時回寫），所以平行票不會互相衝突。
# 每行 FAIL 都印成 `FAIL node-accept …`（impl-run 依 FAIL 後第一個字認測試名）。
set -uo pipefail
main() {
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
[ -s "${1:-}" ] || { echo "FAIL node-accept 用法：bash scripts/node-accept.sh <位址清單>"; return 1; }
t=$(mktemp -d); trap 'rm -rf "$t"' EXIT
bad=0
while read -r a _; do
  [ -n "$a" ] || continue
  a8=$(printf '%08x' "0x$a")
  if [ ! -f "re/emu/cases/$a8.json" ]; then echo "FAIL node $a 缺 re/emu/cases/$a8.json"; bad=1; continue; fi
  bash re/emu/run.sh "$a8" > "$t/emu-$a.log" 2>&1 || { echo "FAIL node $a 模擬器失敗：$(tail -3 "$t/emu-$a.log" | tr '\n' ' ')"; bad=1; }
done < "$1"
go test -count=1 -run 'TestEmu$' ./internal/calc > "$t/go.log" 2>&1 || { echo "FAIL node TestEmu 紅："; grep -E 'FAIL|diff|Error|error' "$t/go.log" | head -20; bad=1; }
mkdir -p "$t/fgd"; cp notes/re/formula-graph.tsv "$t/fg.tsv"; cp notes/re/formula-graph.d/*.tsv "$t/fgd/" 2>/dev/null
for f in "$t/fg.tsv" "$t/fgd/"*.tsv; do
  awk -F'\t' -v OFS='\t' -v L="$1" 'BEGIN { while ((getline l < L) > 0) if (split(l, p, "\t") >= 2) want[p[1]] = p[2] }
    ($1 in want) { if ($2 != "公式") { print "FAIL node " $1 " 不是公式類（" $2 "）" > "/dev/stderr" } $3 = "D4"; $5 = want[$1] } { print }' "$f" > "$f.new" 2>> "$t/cls" && mv "$f.new" "$f"
done
[ -s "$t/cls" ] && { cat "$t/cls"; bad=1; }
FG=$t/fg.tsv FGD=$t/fgd bash scripts/check-formulas.sh --progress > "$t/cf.log" 2>&1
# 沒有結論行＝沒跑完（例：awk 參數過長 exit 126），不能當成沒有 FAIL
tail -1 "$t/cf.log" | grep -qE '^(ok check-formulas|check-formulas: [0-9]+ 處不符)' || { echo "FAIL node check-formulas 沒跑完：$(tail -2 "$t/cf.log" | tr '\n' ' ')"; bad=1; }
# 只看清單裡節點的紅（其他節點的進度不歸本票）
grep -E '^FAIL' "$t/cf.log" | grep -v '^FAIL unclassified' | grep -Ff <(cut -f1 "$1" | sed 's/^/ /; s/$/ /') && bad=1
grep -E '^FAIL (candidates|fields|graph|funcs)' "$t/cf.log" && bad=1
[ $bad = 0 ] && echo "ok node-accept $(grep -c . "$1") 節點 D4" || { echo "FAIL node-accept"; return 1; }
}
main "$@" | sed -E 's/^FAIL (node-accept$)?/FAIL node-accept /; s/ $//'
exit "${PIPESTATUS[0]}"

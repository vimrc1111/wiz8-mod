#!/usr/bin/env bash
# fight.test.sh：calc fight 的驗收（goals/stage12.md A1「單角色近戰組合查詢」）。沒有 build/Wiz8.exe 印 SKIP。
# 1. 輸入登記完整：calc fight -nodes 列出的每個節點，`calc f <節點>` 印出的每個輸入名，在 notes/calc-fight-inputs.tsv 恰有一列；source=fixed 的列 reason 非空。
# 2. 已知答（notes/calc-qa.md Q4，cc 以 calc f 改欄求得）：L28 Samurai 村正對 Mad Marten，每下命中即死觸發 0.1900、即死 0.0741。
# 3. 已知答（Q3）：L20 Rogue 匕首（mode 1）背刺骰組數期望 2.120117（calc-qa Q3 第二區塊 mean gout 2.1201171875；R(100) 以 CRT rand 的 32768 值加權，非均勻故不是 2.12）。
# 4. L28 排名前十的情境（notes/calc-qa/fight-l28-*.json）全部 exit 0、輸出 round.damage（F 路徑不 panic）。
set -uo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
fail() { echo "FAIL fight $*"; exit 1; }
[ -f build/Wiz8.exe ] || { echo "SKIP fight：沒有 build/Wiz8.exe"; exit 0; }
DBS=$(dirname "$(find build/data-x -maxdepth 3 -name ITEMS.DBS 2>/dev/null | head -1)")
t=$(mktemp -d); trap 'rm -rf "$t"' EXIT
go build -o "$t/calc" ./tools/calc || fail "build calc"
F() { "$t/calc" fight -exe build/Wiz8.exe -dbs "$DBS" "$@"; }
F -nodes notes/calc-qa/fight-q4.json > "$t/nodes" 2> "$t/err" || fail "calc fight -nodes 非零退出：$(tail -3 "$t/err")"
[ -s "$t/nodes" ] || fail "calc fight -nodes 沒列出節點"
[ -f notes/calc-fight-inputs.tsv ] || fail "缺 notes/calc-fight-inputs.tsv"
while read -r a; do
  names=$("$t/calc" f "$a" 2>&1 | sed -n 's/^calc: [0-9a-f]* 輸入：//p')
  [ -n "$names" ] || fail "calc f $a 沒印輸入名"
  for n in $names; do
    c=$(awk -F'\t' -v a="$a" -v n="$n" '$1==a && $2==n' notes/calc-fight-inputs.tsv | wc -l)
    [ "$c" = 1 ] || fail "輸入登記 $a $n 出現 $c 次（要 1）"
  done
done < "$t/nodes"
awk -F'\t' 'NR>1 && $3=="fixed" && $5=="" {print; bad=1} END{exit bad}' notes/calc-fight-inputs.tsv || fail "fixed 列缺 reason"
val() { awk -F'\t' -v k="$1" '$1==k{print $2}' "$2"; }
near() { awk -v a="$1" -v b="$2" 'BEGIN{d=a-b; if(d<0)d=-d; exit !(a!="" && d<=0.00005)}'; }
F notes/calc-qa/fight-q4.json > "$t/q4" 2> "$t/err" || fail "q4 非零退出：$(tail -3 "$t/err")"
near "$(val main.instakill_trigger "$t/q4")" 0.19 || fail "q4 main.instakill_trigger=$(val main.instakill_trigger "$t/q4") 要 0.19"
near "$(val main.instakill "$t/q4")" 0.0741 || fail "q4 main.instakill=$(val main.instakill "$t/q4") 要 0.0741"
F notes/calc-qa/fight-q3.json > "$t/q3" 2> "$t/err" || fail "q3 非零退出：$(tail -3 "$t/err")"
near "$(val main.dice_groups_mean "$t/q3")" 2.1201171875 || fail "q3 main.dice_groups_mean=$(val main.dice_groups_mean "$t/q3") 要 2.120117"
n=0; for s in notes/calc-qa/fight-l28-*.json; do
  [ -f "$s" ] || continue; n=$((n+1))
  F "$s" > "$t/r" 2> "$t/err" || fail "$s 非零退出：$(tail -3 "$t/err")"
  [ -n "$(val round.damage "$t/r")" ] || fail "$s 沒有 round.damage"
done
[ $n -ge 10 ] || fail "fight-l28-*.json 只有 $n 個（要前十名各一）"
echo "ok fight"

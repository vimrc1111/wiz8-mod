#!/usr/bin/env bash
# calc-reach.sh：calc-qa 全部題目＋party.test 五個指標＋fight 每份 json，用 Go coverage 列出實際執行到的 internal/calc/n_*.go（init 不算）。
# 用途：goals/stage14.md F-only 範圍（MG 2026-10-11「D4 完再說」→ 只補 calc party／calc-qa 會走到的路徑）。沒有原版檔印 SKIP。
# 上限：檔案級；一個檔被算進來可能只因共用 helper 被呼叫（過度包含，不會漏）。
set -uo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
EXE=build/Wiz8.exe DBS=$(dirname "$(find build/data-x -maxdepth 3 -name ITEMS.DBS 2>/dev/null | head -1)")
[ -f "$EXE" ] && [ -f "$DBS/ITEMS.DBS" ] || { echo "SKIP calc-reach：沒有 build/Wiz8.exe 或 ITEMS.DBS"; exit 0; }
t=$(mktemp -d); trap 'rm -rf "$t"' EXIT; mkdir "$t/cov"; export GOCOVERDIR=$t/cov
go build -cover -o "$t/calc" ./tools/calc || { echo "FAIL calc-reach：建不起來"; exit 1; }
awk '/^```calc-qa$/{b=1;next} /^```$/{b=0} b && /^\$ calc /' notes/calc-qa.md | sed 's/^\$ calc //' |
while read -r a; do a=${a//\$EXE/$EXE}; a=${a//\$DBS/$DBS}
  # shellcheck disable=SC2086
  "$t/calc" $a > /dev/null 2>&1 || { echo "FAIL calc-reach：calc $(cut -c1-40 <<<"$a") 非零"; exit 1; }; done || exit 1
for m in attr:0 attr:3 l100:6 attr:0,attr:5 attr:4,l100:1; do "$t/calc" party -exe "$EXE" -metric "$m" > /dev/null || { echo "FAIL calc-reach：party $m"; exit 1; }; done
for j in notes/calc-qa/fight-*.json; do "$t/calc" fight -exe "$EXE" -dbs "$DBS" "$j" > /dev/null || { echo "FAIL calc-reach：fight $j"; exit 1; }; done
go tool covdata func -i="$t/cov" | awk '$NF!="0.0%" && $2!~/^init/' | grep -o 'n_[0-9a-f]*\.go' | sed 's/^n_//;s/\.go$//' | sort -u

#!/usr/bin/env bash
# calc-qa.test.sh [--progress]：重跑 notes/calc-qa.md 每一題（goals/stage10.md B3「不用再反組譯」）。
# 題目格式：每題一個 ```calc-qa 區塊；第一行 `$ calc <參數…>` 後面接預期 stdout（逐字比）；
#   答不出的題目改寫 `未取得: <addr> [<addr>…]`，這些節點在公式登記簿的深度必須是「未取得」；
#   `待節點: <addr>…` 只在 --progress 放行（節點還沒做完），預設（完成模式）即紅。
# 參數裡的 $EXE、$DBS 代換成 build/Wiz8.exe、build/data-x/DATA/DATABASES（玩家自有原版，不入 git）；沒有就 SKIP。
set -uo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
QA=${QA:-notes/calc-qa.md} mode=final; [ "${1:-}" = --progress ] && mode=progress
EXE=build/Wiz8.exe DBS=$(dirname "$(find build/data-x -maxdepth 3 -name ITEMS.DBS 2>/dev/null | head -1)")
[ -f "$EXE" ] && [ -f "$DBS/ITEMS.DBS" ] || { echo "SKIP calc-qa：沒有 build/Wiz8.exe 或 ITEMS.DBS"; exit 0; }
[ -f "$QA" ] || { echo "FAIL calc-qa：$QA 不存在"; exit 1; }
t=$(mktemp -d); trap 'rm -rf "$t"' EXIT
go build -o "$t/calc" ./tools/calc || { echo "FAIL calc-qa：calc 建不起來"; exit 1; }
depth() { cat notes/re/formula-graph.tsv notes/re/formula-graph.d/*.tsv | awk -F'\t' -v a="$1" '$1""==a"" {print $3}'; }
# 拆題：每個區塊寫成 $t/q<N>，第一行題名（區塊前最近的 ### 標題）
awk -v d="$t" '/^### /{h=$0} /^```calc-qa$/{n++; f=d"/q"n; print h > f; inb=1; next} /^```$/ && inb {inb=0; close(f); next} inb {print > f}' "$QA"
n=0 nb=0 pend=0
for q in "$t"/q*; do
  [ -f "$q" ] || continue; n=$((n+1)); name=$(head -1 "$q"); first=$(sed -n 2p "$q")
  case "$first" in
    '$ calc '*)
      args=${first#\$ calc }; args=${args//\$EXE/$EXE}; args=${args//\$DBS/$DBS}
      # shellcheck disable=SC2086  # 參數由題目檔逐字給，不含空白引號
      "$t/calc" $args > "$t/got" 2> "$t/err" || { echo "FAIL calc-qa $name：calc 非零退出：$(tail -2 "$t/err")"; nb=$((nb+1)); continue; }
      tail -n +3 "$q" > "$t/want"
      diff -u "$t/want" "$t/got" > "$t/d" || { echo "FAIL calc-qa $name：輸出不符"; head -20 "$t/d"; nb=$((nb+1)); } ;;
    未取得:*)
      for a in ${first#未取得:}; do
        [ "$(depth "$a")" = 未取得 ] || { echo "FAIL calc-qa $name：$a 深度是「$(depth "$a")」不是未取得"; nb=$((nb+1)); }
      done ;;
    待節點:*)
      pend=$((pend+1)); [ "$mode" = progress ] || { echo "FAIL calc-qa $name：還在等節點${first#待節點:}"; nb=$((nb+1)); } ;;
    *) echo "FAIL calc-qa $name：區塊第一行要是「\$ calc …」「未取得:」或「待節點:」"; nb=$((nb+1)) ;;
  esac
done
[ "$n" -gt 0 ] || { echo "FAIL calc-qa：$QA 沒有任何 calc-qa 區塊"; exit 1; }
[ "$nb" -eq 0 ] || exit 1
echo "ok calc-qa（$n 題，待節點 $pend）"

#!/usr/bin/env bash
# merge-node.sh <addr>：把節點票 worktree 的白名單檔搬進 main，登記簿標 D4＋calc 函數名
# 清單可多行（循環依賴的節點群一張票，票名取清單第一個位址）：每一行各搬各的檔、各標各的列。
set -euo pipefail; N=$(cd "$(dirname "$0")" && pwd); cd "$N/../.."
w=../wiz8-mod-N$1
while IFS=$'\t' read -r a fn; do
  [ -n "$a" ] || continue; a8=$(printf '%08x' "0x$a")
  for f in re/emu/cases/$a8.json notes/re/iface/$a.tsv internal/calc/n_$a.go notes/re/cov-exempt.d/$a.tsv; do [ -f $w/$f ] && { mkdir -p $(dirname $f); cp $w/$f $f; }; done
  for f in notes/re/formula-graph.tsv notes/re/formula-graph.d/*.tsv; do
    grep -q "^$a	" $f || continue
    awk -F'\t' -v OFS='\t' -v a=$a -v fn=$fn '$1""==a""{$3="D4";$5=fn}1' $f > $f.new && mv $f.new $f; done
done < tickets/nodes/$1.list

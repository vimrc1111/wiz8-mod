#!/usr/bin/env bash
# scc-ticket.sh <key> <成員…>：循環群（呼叫圖強連通分量）一張票：key 用 mkticket 產，其他成員併入清單、白名單與「循環群」段
set -euo pipefail; N=$(cd "$(dirname "$0")" && pwd); cd "$N/../.."
fnof(){ local nm; nm=$(cat notes/re/formula-graph.tsv notes/re/formula-graph.d/*.tsv | awk -F'\t' -v a=$1 '$1""==a""{print $4}')
  local fn; fn=$(awk -v s="$nm" 'BEGIN{n=split(s,p,"_"); o=p[1]; for(i=2;i<=n;i++) o=o toupper(substr(p[i],1,1)) substr(p[i],2); print o "Node"}')
  grep -qE "^func (\([^)]*\) )?$fn\(" internal/calc/*.go && fn=${fn}$1; echo $fn; }
k=$1; shift
NODE_EXEC=cc bash scripts/nodes/mkticket.sh $k $(fnof $k) "大（循環群 $k $*，cc）" >/dev/null
wl=; extra=
for a in "$@"; do a8=$(printf '%08x' "0x$a"); printf '%s\t%s\n' $a $(fnof $a) >> tickets/nodes/$k.list
  wl+=", re/emu/cases/$a8.json, notes/re/iface/$a.tsv, internal/calc/n_$a.go, notes/re/cov-exempt.d/$a.tsv"
  row=$(cat notes/re/formula-graph.tsv notes/re/formula-graph.d/*.tsv | awk -F'\t' -v a=$a '$1""==a""')
  extra+="- \`0x$a\` $(cut -f4 <<<"$row")（$(awk -F'\t' -v a=$a '$1""==a""{print $2" bytes，"split($7,x,",")" 塊；子呼叫 "$4}' build/re/funcs.tsv)）：$(cut -f6 <<<"$row")"$'\n'
done
sed -i "s|^\(白名單: .*\)$|\1$wl|" tickets/N-$k.md
cat >> tickets/N-$k.md <<T

## 循環群（一張票做完全部成員）
本票清單 \`tickets/nodes/$k.list\` 有 $(( $# + 1 )) 行：這些節點互相呼叫（呼叫圖的強連通分量），任何一個都不會單獨就緒，所以一起做。每個成員各自一套白名單檔（case、iface、n_<addr>.go、cov-exempt），各自登記 \`Nodes["<addr8>"]\`。calc 裡成員互相直接呼叫 Go 函數（相互遞迴）；模擬器照原樣跑整群。驗收一次跑整份清單，每個成員都要到 D4。
其他成員：
$extra
T
cat tickets/nodes/$k.list; grep ^白名單 tickets/N-$k.md

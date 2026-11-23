#!/usr/bin/env bash
# formula-candidates.sh [funcs.tsv]：從 Ghidra 函式表（re/Cands.java 產出）機械列出公式候選，印 `<入口hex>\t<前16bytes>`，依入口排序。
# 候選＝符合任一：
#   C1 直接呼叫亂數 0x40efa0，或呼叫「≤64 bytes 且直接呼叫 0x40efa0」的包裝函式；
#   C2 引用記錄基址全域（REC：角色 0x685174、角色副表 0x685178、物品 0x68516c、隊伍 0x6836a8）。
# 規則表與經參數傳入的記錄由 D1 閉合補上：公式節點的每個子呼叫都要登記成節點或「與規則無關」（check-formulas.sh）。
# 沒給 funcs.tsv 就用 Ghidra 重產 build/re/funcs.tsv（repo 外原版衍生物）。
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd)
REC=0x685174,0x685178,0x68516c,0x6836a8
f=${1:-}
if [ -z "$f" ]; then
  f=$here/build/re/funcs.tsv; mkdir -p "$here/build/re"
  bash "$here/re/run.sh" Cands.java "$REC" "$f" >/dev/null
fi
[ -s "$f" ] || { echo "formula-candidates: $f 空或不存在" >&2; exit 1; }
awk -F'\t' '
NR==FNR { if ((","$4",") ~ /,40efa0,/ && $2 <= 64) w[$1] = 1; next }
{ c = ((","$4",") ~ /,40efa0,/) || $5 != ""
  if (!c) { n = split($4, cs, ","); for (i = 1; i <= n; i++) if (cs[i] in w) { c = 1; break } }
  if (c) printf "%s\t%s\n", $1, $3 }' "$f" "$f" | sort

#!/usr/bin/env bash
# check-sigs.sh：notes/re/*.md 表格列「| 0x<VA> | <32 hex> |」逐條對 Wiz8.exe 比 bytes（stage3 §3.1）。
# 找不到遊戲 → 印 skip 留痕、exit 0。VA→檔案位移由 PE section 表算，不假設對齊。
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
G=${WIZ8_DIR:-"/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8"}; E=${WIZ8_EXE:-$G/Wiz8.exe}
[ -f "$E" ] || { echo "check-sigs: skip（找不到 $E）"; exit 0; }
u() { od -An -t u"$2" -j "$1" -N "$2" "$E" | tr -d ' '; }  # <位移> <寬度> → 無號小端整數
pe=$(u 60 4); ns=$(u $((pe + 6)) 2); opt=$(u $((pe + 20)) 2); ib=$(u $((pe + 52)) 4); sec=$((pe + 24 + opt))
off() { local i h vs va rp; for ((i = 0; i < ns; i++)); do h=$((sec + i * 40))
  vs=$(u $((h + 8)) 4); va=$(u $((h + 12)) 4); rp=$(u $((h + 20)) 4)
  (( $1 >= ib + va && $1 < ib + va + vs )) && { echo $(($1 - ib - va + rp)); return; }; done; echo -1; }
n=0 bad=0
while IFS='|' read -r _ a s _; do
  a=${a// /}; s=${s// /}; n=$((n + 1)); o=$(off $((a)))
  got=$([ "$o" -ge 0 ] && od -An -tx1 -j "$o" -N 16 "$E" | tr -d ' \n')
  [ "$got" = "$s" ] || { echo "FAIL $a 期望 $s 實得 ${got:-（不在任何 section）}"; bad=$((bad + 1)); }
done < <(grep -hE '^\| 0x[0-9a-f]{8} \| [0-9a-f]{32} \|' notes/re/*.md)
[ "$n" -gt 0 ] || { echo "check-sigs: 表格 0 列（解析壞了？）" >&2; exit 1; }
[ "$bad" = 0 ] || { echo "check-sigs: $bad／$n 條不符"; exit 1; }
echo "ok check-sigs：$n 條簽章相符（$(sha256sum < "$E" | cut -c1-12)）"

#!/usr/bin/env bash
# func-fields.sh <asm> <funcs.tsv>：單次掃描 objdump，輸出入口<TAB>排序去重的欄位運算元。
# funcs.tsv：入口 hex、大小（十進位 bytes）；範圍 [入口, 入口+大小)。
set -euo pipefail
[ "$#" = 2 ] || { echo 'usage: bash scripts/func-fields.sh <asm> <funcs.tsv>' >&2; exit 1; }
[ -r "$1" ] && [ -r "$2" ] || { echo 'func-fields: asm 或 funcs.tsv 無法讀取' >&2; exit 1; }
awk -v FUNCS="$2" '
function hex(s,   n, i, d) {
  sub(/^0[xX]/, "", s); s = tolower(s); n = 0
  for (i = 1; i <= length(s); i++) {
    d = index("0123456789abcdef", substr(s, i, 1)) - 1
    n = n * 16 + d
  }
  return n
}
BEGIN {
  # 分桶涵蓋所有範圍，也支援未排序、重疊的函式表與 asm。
  while ((getline l < FUNCS) > 0) {
    if (l == "" || l ~ /^#/) continue
    split(l, f, "\t"); a = f[1]; start[a] = hex(a); end[a] = start[a] + f[2]
    for (b = int(start[a] / 4096); b <= int((end[a] - 1) / 4096) && end[a] > start[a]; b++)
      bucket[b] = bucket[b] a ","
  }
  close(FUNCS)
}
/^[[:space:]]*[[:xdigit:]]+:/ {
  line = $0
  sub(/^[[:space:]]*/, "", line)
  addr = hex(substr(line, 1, index(line, ":") - 1))
  sub(/^[[:xdigit:]]+:/, "", line)
  # 夾具也可用字面的反斜線 t；真 objdump 可包含機器碼 bytes。
  gsub(/\\t/, "\t", line); sub(/^[[:space:]]*/, "", line)
  while (line ~ /^[[:xdigit:]][[:xdigit:]][[:space:]]/)
    sub(/^[[:xdigit:]][[:xdigit:]][[:space:]]+/, "", line)
  insn = line; gsub(/[[:space:]]/, "", insn)
  count = split(bucket[int(addr / 4096)], candidates, ",")
  for (j = 1; j <= count; j++) {
    a = candidates[j]
    if (a == "" || addr < start[a] || addr >= end[a]) continue
    if (addr == start[a]) prologue[a] = (insn == "pushebp" ? 1 : 0)
    else if (prologue[a] == 1) { frame[a] = (insn == "movebp,esp"); prologue[a] = 2 }
    if (insn ~ /^lea/) continue  # lea 是位址算術，不是記憶體讀寫
    rest = line
    while (match(rest, /\[[^]]*\]/)) {
      op = substr(rest, RSTART + 1, RLENGTH - 2)
      rest = substr(rest, RSTART + RLENGTH)
      gsub(/[[:space:]]/, "", op)
      if (op !~ /(^|[^[:alnum:]_])(eax|ebx|ecx|edx|esi|edi|esp|ebp|ax|bx|cx|dx|si|di|sp|bp)([^[:alnum:]_]|$)/) continue
      if (op ~ /^esp([+-]|$)/) continue
      if (op ~ /^(eax|ebx|ecx|edx|esi|edi|esp|ebp|ax|bx|cx|dx|si|di|sp|bp)$/) op = op "+0x0"
      fields[a, op] = 1
    }
  }
}
END {
  for (a in start) printf "%s\t\n", a
  for (key in fields) {
    split(key, parts, SUBSEP); a = parts[1]; op = parts[2]
    if (frame[a] && op ~ /^ebp([+-]|$)/) continue
    printf "%s\t%s\n", a, op
  }
}' "$1" | LC_ALL=C sort -u | awk -F'\t' '
function emit() { if (a != "") printf "%s\t%s\n", a, ops }
# 前綴強制字串比較，避免 4e1000 等 hex 被當成科學記號。
("x" $1) != ("x" a) { emit(); a = $1; ops = "" }
$2 != "" { ops = ops (ops == "" ? "" : ",") $2 }
END { emit() }'

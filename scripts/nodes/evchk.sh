#!/usr/bin/env bash
# evchk.sh <frag.tsv>：每列至少一條 wiz8.asm 證據行落在自己函式範圍內
N=$(cd "$(dirname "$0")" && pwd); cd "$N/../.."
f=$1; grep -v '^addr' $f | awk -F'\t' 'NR==FNR{sz[$1]=$2;next} {n=0; s=$6; while (match(s,/wiz8\.asm:[0-9]+/)) {print $1"\t"sz[$1]"\t"substr(s,RSTART+9,RLENGTH-9); s=substr(s,RSTART+RLENGTH); n++} if(!n) print $1"\t"sz[$1]"\t-"}' build/re/funcs.tsv - |
awk -F'\t' 'NR==FNR{l[FNR]=$0; next} {seen[$1]=1; if($3=="-")next; split(l[$3],p,":"); gsub(/ /,"",p[1]); a=strtonum("0x"p[1]); e=strtonum("0x"$1); if(a>=e&&a<e+$2) inr[$1]=1} END{n=0; for(k in seen){n++; if(!(k in inr)) {print "BAD",k; b++}} print "rows",n,"bad",b+0}' build/wiz8.asm -
cut -f2 $f | grep -v '^class' | sort | uniq -c | tr '\n' ' '; echo

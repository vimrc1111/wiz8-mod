#!/usr/bin/env bash
# ready.sh：印就緒節點（公式、D0、公式類依賴全是 D4，不含自己），一行 addr<TAB>塊數<TAB>名稱
# 依賴＝直接子呼叫，加上穿過至多 READY_HOPS（預設 2）層非公式函式可達的公式節點
# （4ef7e0 → 558640 → 557430 → 51c020）。ponytail: 層數上限是啟發式；不設限會穿過引擎派發把半個 exe 拉進來。
# scripts/nodes/ready-cut.tsv：「呼叫者<TAB>非公式跳點」不穿過（只影響派工順序；節點仍原生跑該跳點，理由見該檔）。
N=$(cd "$(dirname "$0")" && pwd); cd "$N/../.."
mkdir -p build/nodes-log
cat notes/re/formula-graph.tsv notes/re/formula-graph.d/*.tsv | awk -F'\t' '$2=="公式"' > build/nodes-log/formula.tsv
awk -F'\t' -v H="${READY_HOPS:-2}" -v CUT="$(grep -v '^#' scripts/nodes/ready-cut.tsv | cut -f1,2 | tr '\t\n' ' ,')" 'BEGIN{n2=split(CUT,cc,","); for(i=1;i<=n2;i++) if(cc[i]!="") cut[cc[i]]=1} NR==FNR{c[$1]=$4; nb[$1]=split($7,x,","); next} {d[$1]=$3; nm[$1]=$4; o[++n]=$1}
  END{for(i=1;i<=n;i++){a=o[i]; if(d[a]!="D0") continue
    delete seen; delete hop; sp=0; st[++sp]=a; seen[a]=1; hop[a]=0; ok=1
    while(sp>0 && ok){f=st[sp--]; k=split(c[f],cs,",")
      for(j=1;j<=k;j++){g=cs[j]; if(g in seen || !(g in c)) continue; seen[g]=1
        if(g in d){ if(d[g]!="D4") ok=0 } else if(hop[f]<H && !((f" "g) in cut)){ hop[g]=hop[f]+1; st[++sp]=g } }}
    if(ok) print a"\t"nb[a]"\t"nm[a]}}' build/re/funcs.tsv build/nodes-log/formula.tsv | sort -t$'\t' -k2n

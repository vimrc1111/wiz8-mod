#!/usr/bin/env bash
# harvest.sh：收已結束的節點票。cc 端在 worktree 重跑 node-accept；綠 → 搬進 main、移除 worktree；紅 → 列出留著
N=$(cd "$(dirname "$0")" && pwd); cd "$N/../.."
S=$(pwd)/build/nodes-log; mkdir -p $S
for w in ../wiz8-mod-N*; do
  a=${w##*-N}; grep -q '^結算' $S/impl-N$a.log 2>/dev/null || continue
  out=$(cd $w && bash scripts/node-accept.sh tickets/nodes/$a.list 2>&1 | tail -1)
  if [[ $out == ok* ]]; then $N/merge-node.sh $a && cp $w/build/emu/$(printf '%08x' 0x$a).* build/emu/ 2>/dev/null; git worktree remove --force $w && git branch -D -q wt-N$a; echo "OK $a"
  else echo "RED $a $(cd $w && bash scripts/node-accept.sh tickets/nodes/$a.list 2>&1 | grep -c FAIL) FAIL；$(grep -h '規格疑慮' $w/.claude/pm-diffs/N-$a.out | tail -1 | cut -c1-200)"; fi
done

#!/usr/bin/env bash
# wave.sh <K>：從就緒集取 K 個還沒開票的節點 → 產票、commit、開 worktree、派 codex
# WAVE_PRIO="a1 a2…"：就緒集裡這些先取（calc-qa 待節點優先）
# NODE_EXEC=cc：只產票、開 worktree，不派外部執行器（cc 子 agent 接手；外部額度用完時）
set -euo pipefail; N=$(cd "$(dirname "$0")" && pwd); cd "$N/../.."
S=$(pwd)/build/nodes-log; mkdir -p $S; D=~/.local/state/wiz8-mod-ghidra/decomp
picked=()
while IFS=$'\t' read -r a nb nm; do
  [ -f tickets/N-$a.md ] && continue
  fn=$(awk -v s="$nm" 'BEGIN{n=split(s,p,"_"); o=p[1]; for(i=2;i<=n;i++) o=o toupper(substr(p[i],1,1)) substr(p[i],2); print o "Node"}')
  grep -qE "^func (\([^)]*\) )?$fn\(" internal/calc/*.go && fn=${fn}$a
  if [ "$nb" -gt 60 ]; then m="大（codex gpt-6-astra medium）"; else m="中（codex gpt-6.1-sol medium）"; fi
  $N/mkticket.sh $a $fn "$m" >/dev/null; picked+=($a); [ ${#picked[@]} -ge $1 ] && break
done < <($N/ready.sh | awk -F'\t' -v p=" ${WAVE_PRIO:-} " 'index(p," "$1" "){print; next} {r[++n]=$0} END{for(i=1;i<=n;i++) print r[i]}')
[ ${#picked[@]} -gt 0 ] || { echo "沒有就緒節點"; exit 0; }
git add tickets/N-*.md tickets/nodes/ && git commit -qm "節點票規格：${picked[*]}

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019vf4NyYqvYzZybZ8Futjky"
for a in "${picked[@]}"; do
  b=$(grep -lx "$a" build/re/batch/b* | head -1) || true; w=../wiz8-mod-N$a
  git worktree add -q -b wt-N$a $w HEAD; mkdir -p $w/build/re $w/build/decomp
  cp build/wiz8.asm build/Wiz8.exe $w/build/; cp build/re/funcs.tsv build/re/fields.tsv $w/build/re/; ln -s ~/.local/state/wiz8-mod-emu/venv $w/build/venv; [ -z "$b" ] || cp $D/cand-$(basename "$b").c $w/build/decomp/
  [ "${NODE_EXEC:-}" = cc ] && continue
  model=; grep -q astra tickets/N-$a.md && model=gpt-6-astra
  (cd $w && CODEX_MODEL=${model:-} SANDBOX_EXTRA_RO=$HOME/.local/state/wiz8-mod-emu nohup bash "${CLAUDE_MEMORY:?需要 CLAUDE_MEMORY}"/scripts/impl-run.sh tickets/N-$a.md $PWD > $S/impl-N$a.log 2>&1 &)
done
echo "派出：${picked[*]}"

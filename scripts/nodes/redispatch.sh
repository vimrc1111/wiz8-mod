#!/usr/bin/env bash
# redispatch.sh <addr> <ticket>：既有節點 worktree 跟上 main、把上一輪半成品（case、證據）挪到 .claude/pm-diffs/prev/、重派 codex
# 挪半成品是因為 r0 不能有沒宣告的紅：殘留 case＋build/emu 證據會讓 TestEmu 紅、impl-run 拒派。票裡要寫「case 在 prev/，先搬回」。
set -euo pipefail; N=$(cd "$(dirname "$0")" && pwd); cd "$N/../.."
a=$1 t=$2 L=$(pwd)/build/nodes-log w=$(cd .. && pwd)/wiz8-mod-N$1
[ -d "$w" ] && [ -f "$t" ] || { echo "沒有 $w 或 $t" >&2; exit 1; }
(cd $w && git stash -q -u; git merge -q --ff-only main; git stash pop -q || true)
[ "$(git -C $w rev-parse HEAD)" = "$(git rev-parse HEAD)" ] || { echo "$w 跟不上 main" >&2; exit 1; }
mkdir -p $w/.claude/pm-diffs/prev
for f in $w/re/emu/cases/00$a.json $w/build/emu/00$a.*; do [ -e "$f" ] && mv "$f" $w/.claude/pm-diffs/prev/; done
(cd $w && go test -count=1 -run TestEmu ./internal/calc >/dev/null 2>&1) || { echo "$w TestEmu 挪完仍紅" >&2; exit 1; }
[ -f $L/impl-N$a.log ] && mv $L/impl-N$a.log $L/impl-N$a.$(date +%H%M%S).log
model=; grep -q astra "$t" && model=gpt-6-astra
(cd $w && CODEX_MODEL=${model:-} SANDBOX_EXTRA_RO=$HOME/.local/state/wiz8-mod-emu nohup bash "${CLAUDE_MEMORY:?需要 CLAUDE_MEMORY}"/scripts/impl-run.sh "$t" $w > $L/impl-N$a.log 2>&1 &)
echo "重派 $a：$t"

#!/usr/bin/env bash
# take.sh <addr> [備註]：收 cc 子 agent 做完的節點（沒有 impl-run log，harvest.sh 認不到）。
# worktree 跟上 main → worktree node-accept → merge-node → main node-accept＋vet＋calc 測試 → commit → 移除 worktree。任一步紅即停。
set -euo pipefail; N=$(cd "$(dirname "$0")" && pwd); cd "$N/../.."
a=$1 note=${2:-} w=../wiz8-mod-N$1
(cd $w && git stash -q -u && git merge -q --ff-only main && git stash pop -q)
[ "$(git -C $w rev-parse HEAD)" = "$(git rev-parse HEAD)" ] || { echo "RED $a worktree 跟不上 main"; exit 1; }
out=$(cd $w && bash scripts/node-accept.sh tickets/nodes/$a.list 2>&1) || { echo "RED $a worktree："; grep FAIL <<<"$out" | head; exit 1; }
$N/merge-node.sh $a
# merge-node 只搬白名單；worktree 其他改動會被下面的 remove --force 丟掉，先紅。
lost=$(git -C $w status --porcelain --untracked-files=all | cut -c4- | while read -r f; do cmp -s "$w/$f" "$f" || echo "$f"; done)
[ -z "$lost" ] || { echo "RED $a worktree 有白名單外改動未搬（worktree 保留）："; echo "$lost"; exit 1; }
for b in $(cut -f1 tickets/nodes/$a.list); do cp $w/build/emu/$(printf '%08x' 0x$b).* build/emu/ 2>/dev/null || true; done
out=$(bash scripts/node-accept.sh tickets/nodes/$a.list 2>&1) || { echo "RED $a main（已搬進工作區、未 commit）："; grep FAIL <<<"$out" | head; exit 1; }
[ -z "$(gofmt -l internal/calc)" ] && go vet ./... && go test -count=1 ./internal/calc >/dev/null || { echo "RED $a main gofmt／vet／test"; exit 1; }
git add -A internal/calc notes/re re/emu/cases
git commit -q -F - <<MSG
cc 子 agent 收貨 $(grep -c . tickets/nodes/$a.list) 節點：$a（worktree＋main node-accept ok）
${note:+
$note
}
Feedback-cc-subagent: [僅限本案] $a 綠
Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019vf4NyYqvYzZybZ8Futjky
MSG
git worktree remove --force $w && git branch -D -q wt-N$a
echo "OK $a $(git log --oneline -1 | cut -c1-7)"

#!/usr/bin/env bash
# 本機 CI 外殼：把某個 commit 抽出到暫存目錄跑一條檢查命令，留證據、回傳原樣退出碼。
# 正本在 claude-memory，各專案複製後只改常數（§6.9）。形狀來自 when 的 scripts/ci-local.sh。
# 要跑**整套 GitHub Actions workflow**（不是一條命令）：看 stripData 的 scripts/ci-local.sh（act＋dind，bd9853a）。
# 目前只有 stripData 用過；第二個專案要用時再收成本庫正本（§6.9）。
#
#   ci-run.sh <專案名> <命令> [引數...]
#
# 驗的是 commit，不是工作樹——工作樹的未 commit 改動不會被看到。
# 證據：${XDG_STATE_HOME:-$HOME/.local/state}/<專案名>-ci/<SHA>/
#   commit.txt worktree.txt cmd.txt source.sha256 check.log exit-code.txt
#
# cmd.txt 記**跑的是哪一條命令**。沒有它，證據只證明「這個 tip 上有某個命令回 0」，
# 不證明「檢查跑過」——`ci-run.sh <proj> true` 的 exit 0 會被閘門當成合格。
# 而且同 SHA 重跑時，**後跑的那條窄命令會變成現行證據**（舊的只是改名留著）。
# BenliX 2026-09-25 實測：先跑全套、再跑 10 支，閘門看到的是後者，分不出來。
# 閘門端的比對是 opt-in 的：`.ci-evidence` 第二行寫預期命令才會比。
# 環境變數：CI_RUN_SHA 指定要驗哪個 commit（預設 HEAD）。
#
# exit-code.txt 是**原樣傳出**的退出碼，判讀只准用「非零＝失敗」。
# 不要寫成 `== 1`：omiPlayer 實測 tsc 失敗是 2（2026-09-25）。
#
# 未入版的依賴（node_modules 之類）外殼刻意不管——那是專案常數，寫在你自己的
# 檢查腳本裡。**但借用之前要先確認鎖檔一致**，例如
#   cmp -s "$OLDPWD/bun.lock" ./bun.lock || { echo "鎖檔不同，拒絕借用" >&2; exit 1; }
# 少了這一步，驗到的是別版依賴裝出來的結果：綠了，但沒驗到那個 commit（§7.1 的形狀）。
# 來源：omiPlayer 接上時的做法，2026-09-25。
set -euo pipefail

if [ $# -lt 2 ]; then
  echo "usage: ci-run.sh <專案名> <命令> [引數...]" >&2
  exit 2
fi
project=$1; shift

cd "$(git rev-parse --show-toplevel)"
sha=$(git rev-parse --verify "${CI_RUN_SHA:-HEAD}")

state_root="${XDG_STATE_HOME:-$HOME/.local/state}/$project-ci"
evidence="$state_root/$sha"
mkdir -p "$state_root"

exec 9>"$state_root/ci-run.lock"
if ! flock -n 9; then
  echo "ci-run: 另一輪正在跑（$state_root/ci-run.lock）" >&2
  exit 1
fi

# 同 SHA 重跑：舊證據改名保留，不覆蓋（§8.3 的精神，證據是凍結類）
if [ -d "$evidence" ]; then
  mv -- "$evidence" "$evidence.$(date -u +%Y%m%dT%H%M%S.%NZ)"
fi
mkdir "$evidence"

printf '%s\n' "$sha" >"$evidence/commit.txt"
printf '%s\n' "$*" >"$evidence/cmd.txt"
git status --short >"$evidence/worktree.txt"

work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT

git archive "$sha" >"$work/source.tar"
(cd "$work" && sha256sum source.tar) >"$evidence/source.sha256"
mkdir "$work/src"
tar -xf "$work/source.tar" -C "$work/src"

# errexit 在 &&／|| 串列與子 shell 裡會被抑制，所以外面 +e、裡面 -e，
# 並用 PIPESTATUS[0] 取子 shell 的碼——不是 $?，那會讀到 tee 的 0。
# 用 PIPESTATUS 而不是靠 pipefail：pipefail 被誰拿掉都不影響這裡（lessons/2026-09-25-errexit-…）。
set +e
(
  set -e
  export GIT_CEILING_DIRECTORIES="$work"   # 不往上爬認野生 .git（如 /tmp/.git，mazewalker 2026-10-01）
  cd "$work/src"
  "$@"
) 2>&1 | tee "$evidence/check.log"
check_exit=${PIPESTATUS[0]}
set -e

printf '%s\n' "$check_exit" >"$evidence/exit-code.txt"
printf '%s\n' "$evidence"
exit "$check_exit"

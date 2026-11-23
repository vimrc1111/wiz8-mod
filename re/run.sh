#!/usr/bin/env bash
# re/run.sh <Script.java> [args…]：對已匯入的 wiz8 Ghidra 專案跑 re/ 下的腳本（-process，不重新分析）。
# Ghidra 拒絕以 . 開頭的路徑元素 → 專案經由 $GH（預設 $XDG_RUNTIME_DIR/wiz8-mod-gh symlink → ~/.local/state/wiz8-mod-ghidra）。
# 共用 /tmp 的固定名可被別的使用者先放 symlink 劫持 → 已存在時必須是自己的、且指向預期目標，否則拒絕。
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd)
T=$HOME/.local/state/wiz8-mod-ghidra; GH=${GH:-${XDG_RUNTIME_DIR:?需要 XDG_RUNTIME_DIR 或 GH}/wiz8-mod-gh}
[ -L "$GH" ] || ln -s "$T" "$GH"
[ -O "$GH" ] && [ "$(readlink "$GH")" = "$T" ] || { echo "re/run.sh: $GH 不是自己指向 $T 的 symlink，拒絕" >&2; exit 1; }
export JAVA_HOME=${JAVA_HOME:-/home/linuxbrew/.linuxbrew/opt/openjdk@21/libexec}; export PATH=$JAVA_HOME/bin:$PATH
s=$1; shift
"$(brew --prefix ghidra)/libexec/support/analyzeHeadless" "$GH" wiz8 -process Wiz8.exe -noanalysis -readOnly \
  -scriptPath "$here" -postScript "$s" "$@" 2>&1 | grep -vE '^\s*(INFO|WARN).*(HeadlessAnalyzer|Loading|Opening|REPORT)' 

#!/usr/bin/env bash
# gen-asm.sh [exe]：從使用者自有的 Wiz8.exe 重產 build/wiz8.asm（原版衍生，不入 git；約 0.5 秒）。
# 登記簿與 iface 的證據行號（build/wiz8.asm:<行>）依賴這份輸出逐行相同：參數只改這裡；產出雜湊（去掉含 exe 路徑的第 2 行）記在 notes/re/asm.sha256，check-formulas 比對。
# 之後：bash scripts/formula-candidates.sh（Ghidra 重產 funcs.tsv）、bash scripts/func-fields.sh build/wiz8.asm build/re/funcs.tsv > build/re/fields.tsv
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); exe=${1:-$here/build/Wiz8.exe}
mkdir -p "$here/build"
objdump -d -M intel --no-show-raw-insn "$exe" > "$here/build/wiz8.asm"
got=$(sed 2d "$here/build/wiz8.asm" | sha256sum | cut -d' ' -f1)
[ "$got" = "$(cut -f1 "$here/notes/re/asm.sha256")" ] && echo "ok gen-asm $got" || { echo "FAIL gen-asm：雜湊 $got 與 notes/re/asm.sha256 不符（exe 版本或 objdump 版本不同；證據行號可能錯位）" >&2; exit 1; }

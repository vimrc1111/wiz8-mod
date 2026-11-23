#!/usr/bin/env bash
# release.sh <輸出 zip>：發布包＝ddraw.dll＋zhpatch.exe（玩家端由原版產 PATCH.002，票 L）＋translations＋安裝／移除 bat＋說明＋字型授權。
# 不含任何原版檔案或原文衍生物（PATCH.002 不放）；包內清單機檢見結尾。
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
out=$(realpath -m "${1:?用法：release.sh <輸出 zip>}")
S=$(mktemp -d); trap 'rm -rf "$S"' EXIT
d=$S/wiz8-mod; mkdir -p "$d"
bash hook/ddraw/build.sh >/dev/null
cp hook/ddraw/out/ddraw.dll "$d/"
# uninstall.bat 用這個字串認「任何版本的 wiz8-mod dll」（findstr）；拿掉就會拒移自己的安裝
grep -qa 'wiz8-mod-ddraw.log' "$d/ddraw.dll" || { echo "FAIL release：ddraw.dll 缺 uninstall 識別字串 wiz8-mod-ddraw.log" >&2; exit 1; }
GOOS=windows GOARCH=amd64 go build -trimpath -o "$d/zhpatch.exe" ./tools/zhpatch
mkdir -p "$d/translations"; (cd translations && find . -name '*.json' -o -name '*.tsv' | cpio -pdm --quiet "$d/translations")
for f in install.bat uninstall.bat INSTALL.txt; do sed 's/\r*$/\r/' "release/$f" > "$d/$f"; done  # Windows 換行
mkdir -p "$d/LICENSES"; cp assets/OFL-*.txt "$d/LICENSES/"
# 包內只准這些種類（原文衍生物：.SLF、PATCH.*、.STI、.DBS、.NSF、.MSG、截圖 一律不准）
# 非一般檔（symlink 等）一律算違規：zip 會跟著連結把目標內容打包（稽核 Arel F1）
# ponytail: 硬連結／把原文改名成 .json 只能靠副檔名擋不住；要擋得改成只收 git 追蹤的檔
bad=$(cd "$d" && { find . ! -type f ! -type d; find . -type f | grep -vE '^\./(ddraw\.dll|zhpatch\.exe|install\.bat|uninstall\.bat|INSTALL\.txt|LICENSES/OFL-[A-Za-z]+\.txt|translations/.+\.(json|tsv))$'; } || true)
[ -z "$bad" ] || { echo "FAIL release：包內有不該有的檔：$bad" >&2; exit 1; }
rm -f "$out"; (cd "$S" && python3 -I -m zipfile -c "$out" wiz8-mod)
echo "release: $out（$(cd "$d" && find . -type f | wc -l) 檔，sha256 $(sha256sum "$out" | cut -c1-16)…）"

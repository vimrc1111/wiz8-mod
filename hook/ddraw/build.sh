#!/usr/bin/env bash
# 建 hook/ddraw/out/ddraw.dll。匯出清單從系統 ddraw 的匯出表算，不手寫。
#   SYS_DDRAW=<路徑>  系統 32 位元 ddraw（預設 /mnt/c/Windows/SysWOW64/ddraw.dll）
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd); out=$here/out; mkdir -p "$out"
sys=${SYS_DDRAW:-/mnt/c/Windows/SysWOW64/ddraw.dll}
# objdump 的 [Ordinal/Name Pointer] 表：`[   7] +base[   8]  0007 DirectDrawCreate`
i686-w64-mingw32-objdump -p "$sys" | awk '
  /\[Ordinal\/Name Pointer\] Table/ { on = 1; next }
  on && NF == 0 { on = 0 }
  on && /\+base/ { gsub(/[][]/, " "); print $(NF), $3 }' >"$out/exports.txt"
[ -s "$out/exports.txt" ] || { echo "build: 讀不到 $sys 的匯出表" >&2; exit 1; }
awk '{ printf "X(%s, %s)\n", $1, $2 }' "$out/exports.txt" >"$out/exports.h"
# detour 簽章：draw.md 的 H／F／G／Y，shot.md 的 P，wrap.md 的 W（含局部續行點），nsf.md 的 N，speed.md 的 T／D／A、party-move.md 的 Q（G、N、Y、T、D、A、M、Q 只在定義了它的區塊展開）。
for tag in H:A:0x00407650 H:B:0x004077d0 F:L:0x00406e00 G:G:0x004071f0 P:S:0x004229e0 \
  W:T:0x004f35b0 W:M:0x0058ac00 W:C:0x005d0050 W:D:0x005d0770 \
  W:P:0x005d2a50 W:MB:0x0058ae17 W:CS:0x005d00e9 \
  W:CA:0x005d03ad W:DS:0x005d0811 W:DA:0x005d0d24 W:EB:0x0058dcfc N:N:0x0055a23c Y:pitch:0x0058c73e Y:first:0x0058c62c Y:clip:0x0058c5c9 \
  T:mul1:0x004546fb T:mul2:0x00454720 T:copy:0x004554dd \
  D:end:0x004ea0be D:attack:0x0053eed3 D:result:0x00543734 \
  D:mode1:0x00587591 D:mode2:0x0058a4f0 A:anim:0x004c2b5d M:key:0x00401b30 M:frame:0x00426ceb \
  M:surface:0x00425590 M:rows:0x005ebdb4 M:map:0x0042a7b5 Q:party:0x0041f78f; do
  kind=${tag%%:*}; tag=${tag#*:}
  re=$here/../../notes/re/draw.md
  if [ "$kind" = P ]; then re=$here/../../notes/re/shot.md; fi
  if [ "$kind" = Q ]; then re=$here/../../notes/re/party-move.md; fi
  if [ "$kind" = M ]; then re=$here/../../notes/re/menu.md; fi
  if [ "$kind" = W ]; then re=$here/../../notes/re/wrap.md; fi
  if [ "$kind" = N ]; then re=$here/../../notes/re/nsf.md; fi
  if [ "$kind" = T ] || [ "$kind" = D ] || [ "$kind" = A ]; then re=$here/../../notes/re/speed.md; fi
  # 只認 check-sigs.sh 同一條列文法（稽核 F1：寬鬆解析會讓 CI 沒驗到的列進 hooks.h）
  sig=$(grep -E '^\| 0x[0-9a-f]{8} \| [0-9a-f]{32} \|' "$re" | awk -F'|' -v a="${tag#*:}" '$2 == " " a " " { print substr($3, 2, 32) }')
  [[ $sig =~ ^[0-9a-f]{32}$ ]] || { echo "build: $re 找不到 ${tag#*:} 的簽章" >&2; exit 1; }
  line="$kind(${tag%%:*}, ${tag#*:}, \"$sig\")"
  if [ "$kind" = G ] || [ "$kind" = N ] || [ "$kind" = Y ] || [ "$kind" = T ] || [ "$kind" = D ] || [ "$kind" = A ] || [ "$kind" = M ] || [ "$kind" = Q ]; then printf '#ifdef %s\n%s\n#endif\n' "$kind" "$line"; else echo "$line"; fi
done >"$out/hooks.h"
# CJK 圖集：tools/atlas 從 Fusion Pixel 12px zh-TW＋陰影（全部字高，不小於 12 級字，MG 2026-10-07）與 Noto Sans TC（點陣缺字時）光柵化，皆 OFL（assets/LICENSES）；字表＝常用字表＋全部譯文（translations/ 底下全部 .json，含子目錄：字串庫、msg、nsf）＋術語表用到的字（譯文存代號，術語字只在術語表）；覆蓋由 atlas-cover.sh 機檢
font=${WIZ8_FONT:-$HOME/.local/share/wiz8-mod/fonts/noto-sans-tc-2.004/NotoSansTC-Regular.otf}
pxfont=${WIZ8_PXFONT:-$HOME/.local/share/wiz8-mod/fonts/fusion-pixel-12px-2026.09.25/fusion-pixel-12px-proportional-zh_tw.ttf}
(cd "$here/../.." && go run ./tools/atlas -px "$pxfont" c "$font" "$out/atlas.h" assets/charset-common.txt $(find translations -name "*.json" | sort) translations/glossary.tsv)
{ echo "LIBRARY ddraw.dll"; echo EXPORTS; awk '{ printf "  %s=stub_%s @%s\n", $1, $1, $2 }' "$out/exports.txt"; } >"$out/ddraw.def"
i686-w64-mingw32-gcc -O2 -fno-tree-loop-distribute-patterns -Wall -Werror -shared -I"$out" -o "$out/ddraw.dll" "$here/proxy.c" "$out/ddraw.def" \
  -nostdlib -Wl,-e,_DllMain@12 -Wl,--kill-at -lkernel32 -luser32 -lgdi32
echo "build: $out/ddraw.dll（$(wc -l <"$out/exports.txt") 個匯出）"

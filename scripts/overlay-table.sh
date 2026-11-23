#!/usr/bin/env bash
# overlay-table.sh <雜訊 run-id,…> <情境 run-id>…：stage2 §2.2 判定，從截圖算，不靠目測。
# 區域：X′ 白塊 A、X″ 白塊 B、which 的 C／D（mk-overlay-patches.sh 的遊戲座標 ×SX／SY）。
# 規則（資料前寫定）：白塊佔滿整個區域，所以「出現」＝相對每個雜訊 run，區域內 ≥50% 像素改變；
#   雜訊底＝雜訊 run 兩兩之間同區域的改變比例最大值，必須 <5%，否則判定作廢（印 VOID）。倍數＝改變比例÷max(雜訊底, 1/區域像素)。
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
R=~/.local/state/wiz8-mod-runs; SX=${SX:-6}; SY=${SY:-4.5}  # 640×480 拉伸到 3840×2160（notes/zh-patch-log.md:23）
S=$(mktemp -d); trap 'rm -rf "$S"' EXIT; go build -o "$S/shot" ./tools/shot
rect() { awk -v sx="$SX" -v sy="$SY" -v x="$1" 'BEGIN{printf "%d,%d,%d,%d", x*sx, 40*sy, 80*sx, 40*sy}'; }
A=$(rect 40); B=$(rect 200); C=$(rect 360); D=$(rect 520)
IFS=, read -ra noise <<<"$1"; shift
frac() { "$S/shot" diff "$R/$1/menu.screen.png" "$R/$2/menu.screen.png" "$3" | jq '.changed/.pixels'; }
floor() { local m=0 i j; for ((i=0;i<${#noise[@]};i++)); do for ((j=i+1;j<${#noise[@]};j++)); do
  m=$(jq -n "[$m,$(frac "${noise[i]}" "${noise[j]}" "$1")]|max"); done; done; echo "$m"; }
fa=$(floor "$A"); fb=$(floor "$B"); fc=$(floor "$C"); fd=$(floor "$D")
echo "雜訊底（${#noise[@]} run 兩兩最大改變比例）：A[$A]=$fa B[$B]=$fb C[$C]=$fc D[$D]=$fd"
for f in $fa $fb $fc $fd; do jq -en "$f < 0.05" >/dev/null || { echo "VOID：雜訊底 ≥5%"; exit 1; }; done
echo "| run | A（×雜訊底） | B | C | D | 判定 |"; echo "|---|---|---|---|---|---|"
for rid in "$@"; do
  row="| $rid"; verdict=""
  for z in "A $A $fa" "B $B $fb" "C $C $fc" "D $D $fd"; do set -- $z
    m=$(for n in "${noise[@]}"; do frac "$n" "$rid" "$2"; done | jq -s min)
    px=$(awk -F, '{print $3*$4}' <<<"$2"); x=$(jq -n "$m / ([$3, 1/$px]|max) | .*10|round/10")
    row+=" | $m（×$x）"; jq -en "$m >= 0.5" >/dev/null && verdict+="$1"
  done
  echo "$row | ${verdict:-無標記} |"
done

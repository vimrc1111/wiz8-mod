#!/usr/bin/env bash
# 機檢：譯文（translations/ 底下全部 .json，含子目錄，與術語表）用到的 U+2000–U+FFFF 字都在 atlas.h 的字表裡；缺字遊戲裡會畫成「A」（6.4 實機，2026-10-07）。
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd); root=$here/../..
atlas=${1:-$here/out/atlas.h}
have=$(awk '/atlas_chars/ { on = 1; next } on && /};/ { exit } on' "$atlas" | grep -oE '0x[0-9a-f]{4}' | sort -u)
need=$(find "$root"/translations -name "*.json" -exec cat {} + | cat - "$root"/translations/glossary.tsv |
  LC_ALL=C.UTF-8 grep -oP '[\x{2000}-\x{FFFF}]' | sort -u | while IFS= read -r c; do printf '0x%04x\n' "'$c"; done | sort -u)
miss=$(comm -13 <(echo "$have") <(echo "$need"))
[ -z "$miss" ] || { echo "FAIL atlas 缺字 $(wc -l <<<"$miss") 個：$(head -20 <<<"$miss" | while read -r h; do printf "\\u${h#0x}"; done)"; exit 1; }
echo "PASS atlas-cover: 譯文 $(wc -l <<<"$need") 字全在圖集"

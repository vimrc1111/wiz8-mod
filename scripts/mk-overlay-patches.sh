#!/usr/bin/env bash
# mk-overlay-patches.sh：產生 stage2 §2.2 覆蓋規則實測用的標記 PATCH 到 build/overlay/（原文衍生物，不入版控）。
# X＝OPTIONS\$X.STI（預設 INTRO_BG；640×480 RGB565，64 byte 檔頭）。X′：遊戲座標 (40,40) 起 80×40 白塊；X″：(200,40) 起。
#   p2/PATCH.002＝X′  p23/PATCH.002＝X′、PATCH.003＝X″  lower/PATCH.002＝X′ 名稱小寫  slash/PATCH.002＝X′ 名稱用 /
#   probe/PATCH.002＝四張 INTRO_BG* 全標 X′（找出主選單實際用哪張，或證明 PATCH.002 沒被讀）
#   which/PATCH.002＝四張依序標在 x=40/200/360/520（區域 A/B/C/D）
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
G=${WIZ8_DIR:-"/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8"}
X=${X:-INTRO_BG}; O=build/overlay; rm -rf "$O"; mkdir -p "$O"
S=$(mktemp -d); trap 'rm -rf "$S"' EXIT; go build -o "$S/slf" ./tools/slf
"$S/slf" list "$G/Data/DATA.SLF" > "$S/list"
get() { # <名> <dst>：從 DATA.SLF 取 OPTIONS\<名>.STI
  local off size; read -r off size < <(awk -F'\t' -v n="OPTIONS\\\\$1.STI" '$3==n{print $1, $2}' "$S/list")
  [ "${size:-}" = 614464 ] || { echo "$1.STI 大小 ${size:-無}，非預期 614464" >&2; exit 1; }
  dd if="$G/Data/DATA.SLF" of="$2" bs=1M iflag=skip_bytes,count_bytes skip="$off" count="$size" status=none
  [ "$(head -c 4 "$2")" = STCI ] || { echo "$1.STI 不是 STCI" >&2; exit 1; }
}
mark() { # <src> <dst> <x0>：y 40..79、x x0..x0+79 填 0xFFFF
  cp "$1" "$2"; local y row; row=$(printf '\xff%.0s' $(seq 160))
  for y in $(seq 40 79); do printf '%s' "$row" | dd of="$2" bs=1 seek=$((64 + (y * 640 + $3) * 2)) conv=notrunc status=none; done
}
pack() { # <out> <PATCH 名> <entry 名>=<檔>…
  local o=$1 n=$2 d="$S/pk" e p off=532 ents='[]'; shift 2; rm -rf "$d"; mkdir -p "$d"
  for e in "$@"; do
    p=$(printf '%s' "${e%%=*}" | tr '\\' /); mkdir -p "$d/$(dirname "$p")"; cp "${e#*=}" "$d/$p"
    ents=$(jq --arg e "${e%%=*}" --argjson off "$off" --argjson sz "$(stat -c %s "${e#*=}")" \
      '.+[{name:$e,offset:$off,size:$sz,state:0,time:126525022660000000,extra:0}]' <<<"$ents")
    off=$((off + $(stat -c %s "${e#*=}")))
  done
  jq -n --arg n "$n" --argjson es "$ents" '{name:$n,dir:"Data\\",fields:[($es|length),33619967,1,0],entries:$es}' > "$d/.slf.json"
  mkdir -p "$O/$o"; "$S/slf" pack "$d" "$O/$o/$n"
}
get "$X" "$S/x.sti"; mark "$S/x.sti" "$S/x1.sti" 40; mark "$S/x.sti" "$S/x2.sti" 200
pack p2 PATCH.002 "OPTIONS\\$X.STI=$S/x1.sti"
pack p23 PATCH.002 "OPTIONS\\$X.STI=$S/x1.sti"; pack p23 PATCH.003 "OPTIONS\\$X.STI=$S/x2.sti"
pack lower PATCH.002 "$(tr A-Z a-z <<<"options\\$X.sti")=$S/x1.sti"
pack slash PATCH.002 "OPTIONS/$X.STI=$S/x1.sti"
pe=()
for b in INTRO_BG INTRO_BG_TEXT INTRO_BG_MENU INTRO_BG_CREDITS; do get $b "$S/$b.0"; mark "$S/$b.0" "$S/$b" 40; pe+=("OPTIONS\\$b.STI=$S/$b"); done
pack probe PATCH.002 "${pe[@]}"
pe=(); x0=40  # which：四張各標在不同 x（40/200/360/520），一次 run 認出主選單用哪張
for b in INTRO_BG INTRO_BG_CREDITS INTRO_BG_MENU INTRO_BG_TEXT; do mark "$S/$b.0" "$S/$b" $x0; pe+=("OPTIONS\\$b.STI=$S/$b"); x0=$((x0 + 160)); done
pack which PATCH.002 "${pe[@]}"  # entries 依 stricmp 排序（遊戲以 bsearch 查，notes/re/）
find "$O" -type f | sort | xargs sha256sum

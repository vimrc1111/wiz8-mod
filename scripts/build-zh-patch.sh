#!/usr/bin/env bash
# build-zh-patch.sh <輸出 PATCH 檔>：用 translations/stringdata.json（只有我們的譯文）改寫玩家自有 PATCH.001 的 STRINGS\STRINGDATA.DAT，
# 再依 translations/lib.tsv 改寫 DATA.SLF 內的索引字串庫（描述 DBS、MSG；stage6）與 NPC 對白 NSF（stage7），打成只含這些筆的 PATCH（建議放 Patches/PATCH.002；編號大者覆蓋，notes/zh-patch-log.md:127）。產物是原文衍生物，不入版控。
# 先過 check-translations（含原文 sha256 守衛：換了版本或筆號錯就失敗）；術語代號由 trcheck expand 展開。
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
G=${WIZ8_DIR:-"/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8"}
out=${1:?用法：build-zh-patch.sh <輸出 PATCH 檔>}; tr=${TRANSLATIONS:-translations/stringdata.json}
S=$(mktemp -d); trap 'rm -rf "$S"' EXIT
go build -o "$S/slf" ./tools/slf; go build -o "$S/strings" ./tools/strings; go build -o "$S/trcheck" ./tools/trcheck; go build -o "$S/strlib" ./tools/strlib; go build -o "$S/nsf" ./tools/nsf; go build -o "$S/fact" ./tools/fact; go build -o "$S/dbsname" ./tools/dbsname
"$S/slf" extract "$G/Patches/PATCH.001" "$S/p" >/dev/null
dat=$S/p/STRINGS/STRINGDATA.DAT; [ -f "$dat" ] || { echo "FAIL：PATCH.001 沒有 STRINGS\\STRINGDATA.DAT" >&2; exit 1; }
"$S/strings" decode "$dat" > "$S/src.json"
TR_DIR=$(dirname "$tr") bash scripts/check-translations.sh >&2   # 全表機檢（含 sha 守衛）不過不出包
TR_DIR=$(dirname "$tr") bash scripts/check-lib.sh >&2
"$S/trcheck" expand -tr "$tr" -glossary "$(dirname "$tr")/glossary.tsv" > "$S/tr.json"
jq --slurpfile t "$S/tr.json" 'reduce $t[0][] as $e (.; .[$e.id] = $e.zh)' "$S/src.json" > "$S/zh.json"
"$S/strings" encode "$S/zh.json" "$dat"
# 只留 STRINGDATA 一筆
jq '.entries |= map(select(.name == "STRINGS\\STRINGDATA.DAT"))' "$S/p/.slf.json" > "$S/m" && mv "$S/m" "$S/p/.slf.json"
[ "$(jq '.entries | length' "$S/p/.slf.json")" = 1 ] || { echo "FAIL：manifest 找不到 STRINGS\\STRINGDATA.DAT" >&2; exit 1; }
# 索引字串庫：從 DATA.SLF 取原檔、代入展開後的譯文、encode 回 $S/p/<路徑>，加進 manifest（時戳沿用 STRINGDATA）
"$S/slf" list "$G/Data/DATA.SLF" > "$S/list"
n_lib=0
while IFS=$'\t' read -r name trf; do
  [[ -z $name || $name == \#* ]] && continue
  read -r off size < <(N="$name" awk -F'\t' '$3==ENVIRON["N"] {print $1, $2}' "$S/list")
  dd if="$G/Data/DATA.SLF" of="$S/f" iflag=skip_bytes,count_bytes skip="$off" count="$size" status=none
  "$S/trcheck" expand -tr "$(dirname "$tr")/$trf" -glossary "$(dirname "$tr")/glossary.tsv" > "$S/ltr.json"
  dst="$S/p/${name//\\//}"; mkdir -p "$(dirname "$dst")"
  if [[ $name == *.NSF ]]; then
    # NSF：id＝攤平的 chunk 序號（check-lib.sh）；逐段累計序號後代入
    "$S/nsf" decode "$S/f" | jq --slurpfile t "$S/ltr.json" '($t[0] | map({(.id | tostring): .zh}) | add // {}) as $m
      | .segments |= ([foreach .[] as $s (0; . + ($s.text // [] | length); [$s, .])]
        | map(.[0] as $s | .[1] as $e | if $s.text then {text: [range($s.text | length) as $j
            | ($m[($e - ($s.text | length) + $j) | tostring] // $s.text[$j])]} else $s end))' > "$S/lzh.json"
    "$S/nsf" encode "$S/lzh.json" "$dst"
  elif [[ $name == *FACT.DBS ]]; then
    # FACT：id＝2×記錄＋欄位（check-lib.sh）
    "$S/fact" decode "$S/f" | jq --slurpfile t "$S/ltr.json" 'reduce $t[0][] as $e (.; .records[$e.id / 2 | floor].text[$e.id % 2] = $e.zh)' > "$S/lzh.json"
    "$S/fact" encode "$S/lzh.json" "$dst"
  elif [[ $name == *ITEMS.DBS ]]; then
    # ITEMS：id＝記錄序號
    "$S/dbsname" decode items "$S/f" | jq --slurpfile t "$S/ltr.json" 'reduce $t[0][] as $e (.; .records[$e.id].text[0] = $e.zh)' > "$S/lzh.json"
    "$S/dbsname" encode items "$S/lzh.json" "$dst"
  elif [[ $name == *SPELLTABLES.DBS ]]; then
    # SPELLTABLES：id＝記錄序號
    "$S/dbsname" decode spells "$S/f" | jq --slurpfile t "$S/ltr.json" 'reduce $t[0][] as $e (.; .records[$e.id].text[0] = $e.zh)' > "$S/lzh.json"
    "$S/dbsname" encode spells "$S/lzh.json" "$dst"
  elif [[ $name == *MONSTERS.DBS ]]; then
    # MONSTERS：id＝4×記錄＋欄位
    "$S/dbsname" decode monsters "$S/f" | jq --slurpfile t "$S/ltr.json" 'reduce $t[0][] as $e (.; .records[$e.id / 4 | floor].text[$e.id % 4] = $e.zh)' > "$S/lzh.json"
    "$S/dbsname" encode monsters "$S/lzh.json" "$dst"
  else
    "$S/strlib" decode "$S/f" > "$S/lib.json"
    jq --slurpfile t "$S/ltr.json" 'reduce $t[0][] as $e (.; .entries[$e.id].text = $e.zh)' "$S/lib.json" > "$S/lzh.json"
    "$S/strlib" encode "$S/lzh.json" "$dst"
  fi
  cp "$dst" "$S/want.$n_lib"; printf '%s\n' "${name//\\//}" >> "$S/wantnames"
  N="$name" jq '.entries[0] as $s | .entries += [$s + {name: env.N, offset: 0, size: 0}]' "$S/p/.slf.json" > "$S/m" && mv "$S/m" "$S/p/.slf.json"
  n_lib=$((n_lib+1))
done < "$(dirname "$tr")/lib.tsv"
# SLF 檔表依小寫檔名排序（DATA.SLF 實測如此；遊戲可能二分搜尋）
jq '.entries |= sort_by(.name | ascii_downcase)' "$S/p/.slf.json" > "$S/m" && mv "$S/m" "$S/p/.slf.json"
"$S/slf" pack "$S/p" "$out"
# 讀回：從產物解開每個字串庫／NSF，須與打包前 encode 出的檔逐 byte 一致
"$S/slf" extract "$out" "$S/rb" >/dev/null
i=0; while read -r rel; do
  cmp -s "$S/rb/$rel" "$S/want.$i" || { echo "FAIL：$out 讀回的 $rel 與打包前不一致" >&2; exit 1; }
  i=$((i+1))
done < <(cat "$S/wantnames" 2>/dev/null)
echo "build-zh-patch: $(jq length "$tr") 筆譯文＋$n_lib 個字串庫 → $out（sha256 $(sha256sum "$out" | cut -c1-16)…）"

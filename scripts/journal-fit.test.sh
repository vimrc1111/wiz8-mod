#!/usr/bin/env bash
# journal-fit.test.sh：日誌（FACT.DBS，translations/fact.json）譯文寬度檢查的校準（票 O）。
# 依據：MG 實機截圖（2026-10-08，發布版 2c2c229）。下列 9 條在日誌欄右緣被切掉，1232 完整顯示。
# 譯文釘在截圖當時的字串（之後縮短 fact.json 不影響本測試）；清空日誌溢出登記後跑 check-fit.sh。
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
S=$(mktemp -d); trap 'rm -rf "$S"' EXIT
clipped=(734 1224 1226 1230 1274 1285 1289 1382 1544)
pin='{
 "734":"找到 Jan-ette 時已太遲。Savant 把她們的船打下來了。",
 "1224":"第 7 枝……我們的命運是找到 Destinae Dominus",
 "1226":"第 7 枝……去 Marten'"'"'s Bluff。拿神像和話語。",
 "1230":"第 7 枝……去過 Bluff 後，到第 6 枝見 Shaman。",
 "1232":"Marten 的日記：「Eternal Trust Survives the Soul」",
 "1274":"見到 Dark Savant……就一下子。然後他消失了。",
 "1285":"Crock 的突變沼澤蛙死了。這裡的東西長真大。",
 "1289":"修好 Android 了！真想知道它知道些什麼……",
 "1382":"……入會：告訴他們是 Al-Adryian 派我們來的。",
 "1544":"……Cosmic Lords 的僕人。他說這裡還有別人："
}'
mk() { # <目錄> [<id> <zh>]：譯文副本，釘住截圖字串，清空日誌溢出登記（只留 # 註解行）
  cp -r translations "$1"
  jq --argjson p "$pin" --arg i "${2:-}" --arg z "${3:-}" \
    'map(.id as $id | if ($id|tostring)==$i then .zh=$z elif $p[$id|tostring] then .zh=$p[$id|tostring] else . end)' \
    translations/fact.json > "$1/fact.json"
  [ -f "$1/journal-overflow.tsv" ] && sed -i '/^[^#]/d' "$1/journal-overflow.tsv"
  return 0
}
fails=0
mk "$S/a"
out=$(TR_DIR=$S/a bash scripts/check-fit.sh 2>&1) && { echo "FAIL journal-fit：截圖被切的譯文、空登記仍綠"; exit 1; }
for id in "${clipped[@]}"; do
  grep -qE "^FAIL journal id=$id( |$)" <<<"$out" && echo "PASS clipped id=$id" || { echo "FAIL journal-fit：截圖被切的 id=$id 沒紅"; fails=$((fails+1)); }
done
grep -qE "^FAIL journal id=1232( |$)" <<<"$out" && { echo "FAIL journal-fit：截圖完整的 id=1232 被判超寬"; fails=$((fails+1)); } || echo "PASS fits id=1232"
# 同一條縮短後必須不紅（檢查是量寬，不是認 id）
mk "$S/b" 1289 "修好 Android 了！"
out=$(TR_DIR=$S/b bash scripts/check-fit.sh 2>&1) || true
grep -qE "^FAIL journal id=1289( |$)" <<<"$out" && { echo "FAIL journal-fit：縮短後的 id=1289 仍紅"; fails=$((fails+1)); } || echo "PASS short id=1289"
[ $fails = 0 ] && echo "ok journal-fit.test：截圖校準 11 項皆符" || exit 1

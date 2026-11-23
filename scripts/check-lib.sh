#!/usr/bin/env bash
# check-lib.sh：索引字串庫（描述 DBS、MSG；tools/strlib）譯文的 trcheck 機檢（stage6 6.3）。原文從玩家自有 DATA.SLF 現解，不入版控。
# 清單 translations/lib.tsv：SLF 內檔名<TAB>譯文檔（相對 translations/）。原文空的筆＝keep，其餘每筆都要有譯文（coverage）。
# .NSF（NPC 對白，stage7 7.2；tools/nsf）：id＝依檔內順序攤平的 chunk 序號；全大寫單字（EMPTY、UNKNOWN、BLANK、CLASSIFIED）是佔位詞＝keep。
#   關鍵字機檢（trcheck -keywords）：該檔的關鍵字定義，除了 YES／NO（是非題回應）、YOU（通用詞，譯文不留英文；MG 2026-10-07 裁定）與 NPC_／RPC_ 開頭（檔名標籤），原文以獨立詞出現就要在譯文保留英文、兩側不貼中文。
# FACT.DBS（日記事件，tools/fact）：id＝2×記錄序號＋欄位（0／1）。
# ITEMS.DBS／SPELLTABLES.DBS／MONSTERS.DBS（tools/dbsname）：id＝記錄序號（物品／法術）或 4×記錄＋欄位（怪物）；試做期只檢查已列出的譯文。
# 法術名（scripts/spell-names.py）：DBS 名稱檔以外，原文提到法術英文名，譯文要用 spells.json 的譯名；同形詞登記在 spell-names-exempt.tsv。
# TR_DIR 可指向別份譯文目錄（check-lib.test.sh 注入用）。
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
G=${WIZ8_DIR:-"/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8"}
T=${TR_DIR:-translations}
S=$(mktemp -d); trap 'rm -rf "$S"' EXIT
go build -o "$S/slf" ./tools/slf; go build -o "$S/strlib" ./tools/strlib; go build -o "$S/trcheck" ./tools/trcheck; go build -o "$S/nsf" ./tools/nsf; go build -o "$S/fact" ./tools/fact; go build -o "$S/dbsname" ./tools/dbsname
"$S/slf" list "$G/Data/DATA.SLF" > "$S/list"
: > "$S/none.tsv"
read -r off size < <(awk -F'\t' '$3=="DATABASES\\SPELLTABLES.DBS" {print $1, $2}' "$S/list")
dd if="$G/Data/DATA.SLF" of="$S/f" iflag=skip_bytes,count_bytes skip="$off" count="$size" status=none
"$S/dbsname" decode spells "$S/f" | jq '[.records[].text[]]' > "$S/spells-src.json"
rc=0
while IFS=$'\t' read -r name tr; do
  [[ -z $name || $name == \#* ]] && continue
  read -r off size < <(N="$name" awk -F'\t' '$3==ENVIRON["N"] {print $1, $2}' "$S/list")
  [ -n "${off:-}" ] || { echo "FAIL lib：DATA.SLF 沒有 $name"; rc=1; continue; }
  dd if="$G/Data/DATA.SLF" of="$S/f" iflag=skip_bytes,count_bytes skip="$off" count="$size" status=none
  kw=()
  if [[ $name == *.NSF ]]; then
    "$S/nsf" decode "$S/f" > "$S/nsf.json"
    jq '[.segments[].text // empty | .[]]' "$S/nsf.json" > "$S/src.json"
    jq -r 'to_entries[] | "\(.key)\t" + (if .value | test("^[A-Z]+$") then "keep\t佔位詞" else "ui\t" end)' "$S/src.json" > "$S/classes.tsv"
    jq -r '.segments[].kw // empty | select((ascii_upcase | IN("YES", "NO", "YOU")) or test("^(NPC|RPC)_") | not)' "$S/nsf.json" | sort -u > "$S/kw.txt"
    kw=(-keywords "$S/kw.txt")
  elif [[ $name == *FACT.DBS ]]; then
    "$S/fact" decode "$S/f" | jq '[.records[].text[]]' > "$S/src.json"
    jq -r 'to_entries[] | "\(.key)\t" + (if .value == "" then "keep\t原文空" else "ui\t" end)' "$S/src.json" > "$S/classes.tsv"
  elif [[ $name == *ITEMS.DBS || $name == *MONSTERS.DBS || $name == *SPELLTABLES.DBS ]]; then
    kind=items; [[ $name != *MONSTERS.DBS ]] || kind=monsters
    [[ $name != *SPELLTABLES.DBS ]] || kind=spells
    "$S/dbsname" decode "$kind" "$S/f" | jq '[.records[].text[]]' > "$S/src.json"
    jq -r --slurpfile t "$T/$tr" '($t[0] | map({(.id | tostring): true}) | add // {}) as $ids
      | to_entries[] | "\(.key)\t" + (if .value == "" then "keep\t原文空"
        elif $ids[.key | tostring] then "ui\t" else "keep\t名稱未譯（試做）" end)' "$S/src.json" > "$S/classes.tsv"
  else
    "$S/strlib" decode "$S/f" | jq '[.entries[].text]' > "$S/src.json"
    jq -r 'to_entries[] | "\(.key)\t" + (if .value == "" then "keep\t原文空" else "ui\t" end)' "$S/src.json" > "$S/classes.tsv"
  fi
  echo "== $name"
  "$S/trcheck" check -src "$S/src.json" -tr "$T/$tr" -classes "$S/classes.tsv" -glossary "$T/glossary.tsv" \
    -pronouns "$S/none.tsv" -pct "$S/none.tsv" -pronoun-forms=false "${kw[@]}" || rc=1
  [[ $name == *ITEMS.DBS || $name == *MONSTERS.DBS || $name == *SPELLTABLES.DBS ]] \
    || python3 -I scripts/spell-names.py "$S/spells-src.json" "$T/spells.json" "$T/glossary.tsv" "$T/spell-names-exempt.tsv" "$tr" "$S/src.json" "$T/$tr" || rc=1
  [[ $name != *MONSTERS.DBS ]] || python3 -I scripts/monster-names.py "$S/src.json" "$T/$tr" "$T/glossary.tsv" || rc=1
  off=
done < "$T/lib.tsv"
exit $rc

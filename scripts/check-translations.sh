#!/usr/bin/env bash
# check-translations.sh：trcheck 對 translations/ 全表機檢（stage5 5.2）。原文從玩家自有 PATCH.001 現解，不入版控。
# 法術名（scripts/spell-names.py，同 check-lib.sh）：原文提到法術英文名，譯文要用 spells.json 的譯名；SPELLTABLES.DBS 取自同一份 PATCH.001。
# TR_DIR 可指向別份譯文目錄（check-translations.test.sh 注入用）。
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
G=${WIZ8_DIR:-"/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8"}
T=${TR_DIR:-translations}
S=$(mktemp -d); trap 'rm -rf "$S"' EXIT
go build -o "$S/slf" ./tools/slf; go build -o "$S/strings" ./tools/strings; go build -o "$S/trcheck" ./tools/trcheck; go build -o "$S/dbsname" ./tools/dbsname
"$S/slf" extract "$G/Patches/PATCH.001" "$S/p" >/dev/null
"$S/strings" decode "$S/p/STRINGS/STRINGDATA.DAT" > "$S/src.json"
"$S/dbsname" decode spells "$S/p/DATABASES/SPELLTABLES.DBS" | jq '[.records[].text[]]' > "$S/spells-src.json"
rc=0
python3 -I scripts/spell-names.py "$S/spells-src.json" "$T/spells.json" "$T/glossary.tsv" "$T/spell-names-exempt.tsv" stringdata.json "$S/src.json" "$T/stringdata.json" || rc=1
"$S/trcheck" check -src "$S/src.json" -tr "$T/stringdata.json" -classes "$T/stringdata-classes.tsv" -glossary "$T/glossary.tsv" \
  -pronouns "$T/pronouns.tsv" -pct "$T/pct-literal.tsv" || rc=1
exit $rc

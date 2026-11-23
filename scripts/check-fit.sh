#!/usr/bin/env bash
# check-fit.sh：fitcheck 對展開後譯文量像素寬與明示換行數（stage5 5.3）。原文、字型、字表從玩家自有遊戲檔現取，不入版控。
# STRINGDATA 用 Wiz_Text_Font 當代表；日誌獨立用 Journal_Font 與 430px 文字區（notes/re/journal.md）。
# TR_DIR 可指向別份譯文目錄（check-fit.test.sh 注入用）。
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
G=${WIZ8_DIR:-"/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8"}
T=${TR_DIR:-translations}
S=$(mktemp -d); trap 'rm -rf "$S"' EXIT
for t in slf strings fact trcheck fitcheck; do go build -o "$S/$t" ./tools/$t; done
"$S/slf" extract "$G/Patches/PATCH.001" "$S/p" >/dev/null
"$S/slf" extract "$G/Data/DATA.SLF" "$S/d" >/dev/null
"$S/strings" decode "$S/p/STRINGS/STRINGDATA.DAT" > "$S/src.json"
"$S/trcheck" expand -tr "$T/stringdata.json" -glossary "$T/glossary.tsv" > "$S/tr.json"
rc=0
"$S/fitcheck" -exe "$G/Wiz8.exe" -font "$S/d/FONTS/WIZ_TEXT_FONT.STI" -src "$S/src.json" -tr "$S/tr.json" -over "$T/overflow.tsv" || rc=1
"$S/fact" decode "$S/d/DATABASES/FACT.DBS" | jq '[.records[].text[]]' > "$S/fact-src.json"
"$S/trcheck" expand -tr "$T/fact.json" -glossary "$T/glossary.tsv" > "$S/fact-tr.json"
"$S/fitcheck" -journal -exe "$G/Wiz8.exe" -font "$S/d/JOURNAL/JOURNAL_FONT.STI" -src "$S/fact-src.json" -tr "$S/fact-tr.json" -over "$T/journal-overflow.tsv" || rc=1
exit "$rc"

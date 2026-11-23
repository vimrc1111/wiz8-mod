#!/usr/bin/env bash
# check-translations.test.sh：對真譯文副本各注入一種違規，check-translations.sh 必須紅在該規則上；原副本必須綠。
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
S=$(mktemp -d); trap 'rm -rf "$S"' EXIT
fails=0
probe() { # <名> <規則> <id> <jq 改 stringdata.json 或 ->
  local d=$S/$1; cp -r translations "$d"
  [ "$4" = - ] || jq "$4" translations/stringdata.json > "$d/stringdata.json"
  local out; out=$(TR_DIR=$d bash scripts/check-translations.sh 2>&1) && { echo "FAIL inject:$1 仍綠"; fails=$((fails+1)); return; }
  grep -q "^FAIL $2 id=$3" <<<"$out" && echo "PASS inject:$1" || { echo "FAIL inject:$1 沒紅在 $2 id=$3：$out" | head -n 3; fails=$((fails+1)); }
}
TR_DIR=translations bash scripts/check-translations.sh >/dev/null && echo "PASS baseline" || { echo "FAIL baseline"; exit 1; }
# 取一條有 %d 的譯文、一條含術語代號的譯文
pid=$(jq -r '[.[]|select(.zh|test("%d"))][0].id' translations/stringdata.json)
cid=$(jq -r '[.[]|select(.zh|test("\\{\\{"))][0].id' translations/stringdata.json)
probe printf printf "$pid" "map(if .id==$pid then .zh|=sub(\"%d\";\"\") else . end)"
probe code code "$cid" "map(if .id==$cid then .zh|=sub(\"\\\\{\\\\{\";\"{{Zz\") else . end)"
probe coverage coverage "$pid" "map(select(.id!=$pid))"
probe pronoun pronoun 727 'map(if .id==727 then .zh="他之" else . end)'
probe pronoun-template pronoun 571 'map(if .id==571 then .zh="%s逃跑，只為保住%s的性命" else . end)'
probe spell spell 1995 'map(if .id==1995 then .zh="施放敲敲門" else . end)'
[ $fails = 0 ] && echo "ok check-translations.test：6 種注入皆紅" || exit 1

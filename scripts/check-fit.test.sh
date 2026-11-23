#!/usr/bin/env bash
# check-fit.test.sh：對真譯文副本各注入一種違規，check-fit.sh 必須紅在該規則上；原副本必須綠。
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
S=$(mktemp -d); trap 'rm -rf "$S"' EXIT
fails=0
probe() { # <名> <規則> <id> <改副本的指令>
  local d=$S/$1; cp -r translations "$d"
  (cd "$d" && eval "$4")
  local out; out=$(TR_DIR=$d bash scripts/check-fit.sh 2>&1) && { echo "FAIL inject:$1 仍綠"; fails=$((fails+1)); return; }
  grep -q "^FAIL $2 id=$3" <<<"$out" && echo "PASS inject:$1" || { echo "FAIL inject:$1 沒紅在 $2 id=$3：$out" | head -n 3; fails=$((fails+1)); }
}
TR_DIR=translations bash scripts/check-fit.sh >/dev/null && echo "PASS baseline" || { echo "FAIL baseline"; exit 1; }
oid=$(awk -F'\t' '!/^#/{print $1; exit}' translations/overflow.tsv)
gid=$(jq -r '[.[]|select(.zh|test("^[^\\n{]+$"))][0].id' translations/stringdata.json)  # 一條無換行、無代號、寬度內的譯文
grep -q "^$gid	" translations/overflow.tsv && { echo "FAIL fixture：$gid 已登記"; exit 1; }
probe over over "$oid" "sed -i '/^$oid	/d' overflow.tsv"
probe reason over-reason "$oid" "sed -i 's/^$oid	.*/$oid	/' overflow.tsv"
probe stale stale "$gid" "printf '$gid\t舊登記\n' >> overflow.tsv"
probe lines lines "$gid" "jq 'map(if .id==$gid then .zh=\"\n\"+.zh+\"\n\" else . end)' stringdata.json > x && mv x stringdata.json"
[ $fails = 0 ] && echo "ok check-fit.test：4 種注入皆紅" || exit 1

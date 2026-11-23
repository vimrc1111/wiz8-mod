#!/usr/bin/env bash
# check-lib.test.sh：對真譯文副本注入違規，check-lib.sh 必須紅在該規則上；原副本必須綠。
# 每次 check-lib.sh 約 25 s（逐檔起 jq／go／python），各注入互不相干 → 背景平行跑，結果寫 $S/res.*，最後彙總。
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
S=$(mktemp -d); trap 'rm -rf "$S"' EXIT
probe() { # <名> <規則> <id> <jq 改譯文檔> [譯文檔，預設 itemdesc.json]
  local d=$S/$1 f=${5:-itemdesc.json}; cp -r translations "$d"
  jq "$4" "translations/$f" > "$d/$f"
  local out; out=$(TR_DIR=$d bash scripts/check-lib.sh 2>&1) && { echo "FAIL inject:$1 仍綠"; return; }
  grep -q "^FAIL $2 id=$3" <<<"$out" && echo "PASS inject:$1" || { echo "FAIL inject:$1 沒紅在 $2 id=$3：$out" | head -n 3; }
}
{ TR_DIR=translations bash scripts/check-lib.sh >/dev/null && echo "PASS baseline" || echo "FAIL baseline"; } > "$S/res.0baseline" &
id=$(jq -r '.[0].id' translations/itemdesc.json)
probe sha sha "$id" "map(if .id==$id then .src_sha256=\"00\" else . end)" > "$S/res.sha" &
probe coverage coverage "$id" "map(select(.id!=$id))" > "$S/res.coverage" &
probe code code "$id" "map(if .id==$id then .zh+=\"{{Zz}}\" else . end)" > "$S/res.code" &
probe keep coverage 1 '. + [{"id":1,"src_sha256":"x","zh":"空"}]' > "$S/res.keep" &
# NSF 關鍵字：把譯文裡的 Arnika 換成中文音譯
probe keyword keyword 0 'map(if .id==0 then .zh|=gsub("Arnika";"阿尼卡") else . end)' nsf/NPC_MYLES.json > "$S/res.keyword" &
# 怪物名（monster-names.py）：火蟻改成不以種族「蟻」結尾；同名第二筆換譯
probe tail tail 1196 'map(if .id==1196 then .zh="火螞#"+(.zh|split("#")[1]) else . end)' monsters.json > "$S/res.tail" &
probe same same 2080 'map(if .id==2080 then .zh="炎蟻#"+(.zh|split("#")[1]) else . end)' monsters.json > "$S/res.same" &
# 法術名（spell-names.py）：法術書描述的「加速」改回英文 Haste；例外登記一筆不會命中的（過期）
probe spell spell 372 'map(if .id==372 then .zh|=gsub("加速";"Haste") else . end)' > "$S/res.spell" &
cp -r translations "$S/stale"; printf 'itemdesc.json\t16\tHaste\t注入\n' >> "$S/stale/spell-names-exempt.tsv"
{ out=$(TR_DIR=$S/stale bash scripts/check-lib.sh 2>&1) && echo "FAIL inject:stale 仍綠" \
  || { grep -q "^FAIL spell-exempt id=16" <<<"$out" && echo "PASS inject:stale" || echo "FAIL inject:stale 沒紅在 spell-exempt id=16"; }; } > "$S/res.stale" &
wait
cat "$S"/res.*
# 計數由彙總方從結果檔算：應有 baseline＋9 種注入共 10 行 PASS
[ "$(cat "$S"/res.* | grep -c "^PASS")" = 10 ] && ! grep -q "^FAIL" "$S"/res.* && echo "ok check-lib.test：9 種注入皆紅" || exit 1

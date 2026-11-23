#!/usr/bin/env bash
# text-match.sh <run-id>：stage3 §3.3 驗收。把該 run 的 wiz8-mod-text.log（hook/ddraw 寫的）逐條對 PATCH.001 的 STRINGDATA 解碼結果做完全相等比對，
# 結果（命中數、命中的筆號；不含原文）寫回 notes/runs/<run-id>.json 的 .stringdata_match。門檻：命中 ≥3 條不同記錄、行程 60 秒仍在、after_clean。
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
G=${WIZ8_DIR:-"/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8"}
rid=$1; j=notes/runs/$rid.json; log=~/.local/state/wiz8-mod-runs/$rid/wiz8-mod-text.log
[ -s "$log" ] || { echo "FAIL $rid：沒有 text log（$log）" >&2; exit 1; }
S=$(mktemp -d); trap 'rm -rf "$S"' EXIT
go run ./tools/slf extract "$G/Patches/PATCH.001" "$S/p1" >/dev/null
dat=$(find "$S/p1" -iname STRINGDATA.DAT | head -1); [ -n "$dat" ] || { echo "FAIL：PATCH.001 找不到 STRINGDATA.DAT" >&2; exit 1; }
go run ./tools/strings decode "$dat" > "$S/s.json"
# log 欄位：序號 tag x y fmt|raw 字串 原格式（proxy.c text_seen）
m=$(cut -f6 "$log" | tr -d '\r' | jq -R . | jq -s --slurpfile s "$S/s.json" '
  ($s[0] | to_entries | map({key:.value, value:.key}) | from_entries) as $ix
  | map(select(. != "" and $ix[.] != null) | $ix[.]) | unique | {records:., count:length}')
alive=$(jq '[.events[]|select(endswith(" alive"))]|length' "$j"); clean=$(jq .after_clean "$j")
jq --argjson m "$m" --argjson lines "$(wc -l < "$log")" '.stringdata_match = ($m + {log_lines:$lines})' "$j" > "$S/j" && mv "$S/j" "$j"
c=$(jq .count <<<"$m")
echo "$rid：log $(wc -l < "$log") 條，命中 STRINGDATA $c 筆不同記錄，alive=$alive after_clean=$clean"
[ "$c" -ge 3 ] && [ "$alive" -ge 1 ] && [ "$clean" = true ] || { echo "FAIL 3.3 驗收" >&2; exit 1; }
echo "ok 3.3 驗收"

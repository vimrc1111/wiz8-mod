#!/usr/bin/env bash
# cjk-check.sh <中文 run> <空字串 run> <無 patch run>：stage4 §4.3 驗收（三個 run 同一情境、只差 PATCH.002）。
# 中文 run 的 text log 裡每條含 CJK 的繪字：位置取 log 的 x,y，字高取 ddraw log 該字型槽的 gh；
# 用 tools/atlas cmp 比「中文幀 vs 空字串幀」與圖集期望遮罩，xor 上限 0（雜訊底 0，濾波暈已在遮罩規則內扣掉，tools/atlas/cmp.go）。
# 上限 0 的依據：本情境雜訊底 N0＝0（cjk-empty 兩次整張逐像素相同，notes/zh-patch-log.md:193 §1）。
# 字色（同檔 §2）：中文墨色與同 y、同字型槽的英文墨色（無 patch 幀，寬＝英文字數×gh）各通道差 ≤ T＝ceil(0.185×英文墨色與背景的最大通道差)。
# 負對照：無 patch 幀 vs 空字串幀在同一位置 xor 必須 > 0。三個 run 都要 60 秒仍在、after_clean。結果寫回 notes/runs/<中文 run>.json 的 .cjk_match。
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
font=${WIZ8_FONT:-$HOME/.local/share/wiz8-mod/fonts/noto-sans-tc-2.004/NotoSansTC-Regular.otf}
st=~/.local/state/wiz8-mod-runs; zh=$1 bg=$2 neg=$3
for r in "$zh" "$bg" "$neg"; do
  [ "$(jq '[.events[]|select(endswith(" alive"))]|length' "notes/runs/$r.json")" -ge 1 ] && [ "$(jq .after_clean "notes/runs/$r.json")" = true ] ||
    { echo "FAIL $r：沒到 60 秒或 after_clean≠true" >&2; exit 1; }
done
S=$(mktemp -d); trap 'rm -rf "$S"' EXIT
go build -o "$S/atlas" ./tools/atlas
# text log：序號 tag x y fmt|raw 字串 原格式 字型槽（proxy.c text_seen）；只取含 U+3000 以上字元的（中文）
tr -d '\r' < "$st/$zh/wiz8-mod-text.log" | awk -F'\t' '$6 ~ /[^\x01-\x7f]/ { print $3 "\t" $4 "\t" $8 "\t" $6 }' | sort -u > "$S/cjk.tsv"
[ -s "$S/cjk.tsv" ] || { echo "FAIL $zh：text log 沒有中文繪字" >&2; exit 1; }
fail=0 n=0; : > "$S/res.jsonl"
while IFS=$'\t' read -r x y slot s; do
  gh=$(grep -aoE "font slot=$slot n=[0-9]+ h=[0-9]+ gh=[0-9]+" "$st/$zh/wiz8-mod-ddraw.log" | tail -1 | sed 's/.*gh=//')
  [ -n "$gh" ] || { echo "FAIL：ddraw log 找不到字型槽 $slot 的 gh" >&2; exit 1; }
  zl=$("$S/atlas" cmp "$font" "$st/$zh/party.screen.png" "$st/$bg/party.screen.png" "$x" "$y" "$gh" "$s" 0 2>/dev/null | tail -1 || true)
  p=$(grep -oE 'xor=[0-9]+' <<<"$zl" || true); zi=$(sed -n 's/.* ink=\([0-9,-]*\).*/\1/p' <<<"$zl")
  # 英文：同 y、同字型槽；cmp 的期望字只拿來定框（字數×gh），只讀 ink／bg
  read -r ex en < <(tr -d '\r' < "$st/$neg/wiz8-mod-text.log" | awk -F'\t' -v y="$y" -v sl="$slot" '$4 == y && $8 == sl { print $3, length($6); exit }') || true
  [ -n "${ex:-}" ] || { echo "FAIL：無 patch run 找不到 y=$y 槽 $slot 的英文" >&2; exit 1; }
  el=$("$S/atlas" cmp "$font" "$st/$neg/party.screen.png" "$st/$bg/party.screen.png" "$ex" "$y" "$gh" "$(printf '一%.0s' $(seq "$en"))" 1000000 2>/dev/null | tail -1)
  ei=$(sed -n 's/.* ink=\([0-9,-]*\).*/\1/p' <<<"$el"); eb=$(sed -n 's/.* bg=\([0-9,-]*\).*/\1/p' <<<"$el")
  read -r cd T < <(awk -v z="$zi" -v e="$ei" -v b="$eb" 'BEGIN { split(z, Z, ","); split(e, E, ","); split(b, B, ","); m = 0; c = 0
    for (i = 1; i <= 3; i++) { d = E[i] - B[i]; if (d < 0) d = -d; if (d > c) c = d; d = Z[i] - E[i]; if (d < 0) d = -d; if (d > m) m = d }
    t = 0.185 * c; T = (t == int(t)) ? t : int(t) + 1; print m, T }')
  q=$("$S/atlas" cmp "$font" "$st/$neg/party.screen.png" "$st/$bg/party.screen.png" "$x" "$y" "$gh" "$s" 0 2>/dev/null | grep -oE 'xor=[0-9]+' || true)
  p=${p#xor=} q=${q#xor=}; [ -n "$p" ] && [ -n "$q" ] || { echo "FAIL：cmp 沒有輸出（$x,$y）" >&2; exit 1; }
  ok=$([ "$p" = 0 ] && [ "$q" -gt 0 ] && [ "$cd" -le "$T" ] && echo true || echo false); [ "$ok" = true ] || fail=$((fail + 1)); n=$((n + 1))
  echo "x=$x y=$y gh=$gh 字數=${#s} xor=$p 負對照 xor=$q 墨色 中=$zi 英=$ei 差=$cd T=$T $([ "$ok" = true ] && echo ok || echo FAIL)"
  jq -nc --argjson x "$x" --argjson y "$y" --argjson gh "$gh" --argjson n "${#s}" --argjson p "$p" --argjson q "$q" --argjson ok "$ok" \
    --arg zi "$zi" --arg ei "$ei" --argjson cd "$cd" --argjson T "$T" \
    '{x:$x,y:$y,gh:$gh,chars:$n,xor:$p,neg_xor:$q,ink_zh:$zi,ink_en:$ei,ink_diff:$cd,ink_T:$T,ok:$ok}' >> "$S/res.jsonl"
done < "$S/cjk.tsv"
jq --slurpfile r "$S/res.jsonl" --arg bg "$bg" --arg neg "$neg" '.cjk_match = {background_run:$bg, negative_run:$neg, xor_max:0, strings:$r}' \
  "notes/runs/$zh.json" > "$S/j" && mv "$S/j" "notes/runs/$zh.json"
[ "$fail" = 0 ] || { echo "FAIL 4.3：$fail／$n 條" >&2; exit 1; }
echo "ok 4.3：$n 條中文 xor=0、墨色差 ≤T、負對照全 >0"

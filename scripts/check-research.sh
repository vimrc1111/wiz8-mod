#!/usr/bin/env bash
# 研究報告機檢：「## 發現」段內每條 `- ` 結論須附來源（URL 或 path:line）與三態標記之一。
# 段落不存在或沒有任何結論 → 失敗（空的不算乾淨）。
set -euo pipefail
f=${1:?usage: check-research.sh <報告.md>}
[ -f "$f" ] || { echo "$f: 不存在"; echo "FAIL research:$(basename "$f" .md)"; exit 1; }
awk '
  /^## /        { in_sec = ($0 ~ /^## 發現/); if (in_sec) seen = 1; next }
  in_sec && /^- / {
    n++
    src = ($0 ~ /https?:\/\//) || ($0 ~ /[A-Za-z0-9_.\/-]+:[0-9]+/)
    tag = ($0 ~ /\[(已驗證|推測|未取得)/)
    if (!src) { printf "%s:%d: 缺來源\n", FILENAME, NR; bad++ }
    if (!tag) { printf "%s:%d: 缺三態標記\n", FILENAME, NR; bad++ }
  }
  END {
    if (!seen) { print FILENAME ": 沒有「## 發現」段"; bad = 1 }
    else if (!n) { print FILENAME ": 「## 發現」段沒有結論"; bad = 1 }
    else printf "%s: %d 條結論，%d 個問題\n", FILENAME, n, bad
    # impl-run 的 r0／預期紅只認得 `FAIL <名>` 形狀的失敗行
    if (bad) { print "FAIL research:" name; exit 1 }
  }' name="$(basename "$f" .md)" "$f"

#!/usr/bin/env bash
# nsf-pkg.sh <輸出目錄> <NPC 檔名主幹>…：從玩家自有 DATA.SLF 解出 NPC 對白，產生翻譯工作包（原文衍生，輸出目錄要在 repo 外、不入版控）。
# 每個 <N>：<N>.work.json（id、src_sha256、src、zh 空；佔位詞不列）、<N>.kw.txt（關鍵字，規則同 check-lib.sh）。
# 譯完寫 translations/nsf/<N>.json（拿掉 src），加進 translations/lib.tsv，再跑 check-lib.sh。
set -euo pipefail
[ $# -ge 2 ] || { echo "用法：nsf-pkg.sh <輸出目錄> <NPC_X>…" >&2; exit 2; }
out=$(realpath -m "$1"); shift
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
case $out/ in "$here"/*) echo "nsf-pkg：輸出目錄不可在 repo 內（原文衍生）" >&2; exit 2;; esac
G=${WIZ8_DIR:-"/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8"}
S=$(mktemp -d); trap 'rm -rf "$S"' EXIT
go build -o "$S/slf" ./tools/slf; go build -o "$S/nsf" ./tools/nsf
"$S/slf" list "$G/Data/DATA.SLF" > "$S/list"
mkdir -p "$out"
for n in "$@"; do
  read -r off size < <(N="NPC SCRIPTS\\$n.NSF" awk -F'\t' '$3==ENVIRON["N"] {print $1, $2}' "$S/list") || { echo "nsf-pkg：DATA.SLF 沒有 $n.NSF" >&2; exit 1; }
  dd if="$G/Data/DATA.SLF" of="$S/f" iflag=skip_bytes,count_bytes skip="$off" count="$size" status=none
  "$S/nsf" decode "$S/f" > "$S/nsf.json"
  jq '[.segments[].text // empty | .[]] | to_entries | map(select(.value | test("^[A-Z]+$") | not) | {id: .key, src: .value})' "$S/nsf.json" |
    python3 -I -c 'import json, hashlib, sys
json.dump([{"id": e["id"], "src_sha256": hashlib.sha256(e["src"].encode()).hexdigest(), "src": e["src"], "zh": ""} for e in json.load(sys.stdin)], sys.stdout, ensure_ascii=False, indent=1)' > "$out/$n.work.json"
  jq -r '.segments[].kw // empty | select((ascii_upcase | IN("YES", "NO", "YOU")) or test("^(NPC|RPC)_") | not)' "$S/nsf.json" | sort -u > "$out/$n.kw.txt"
  echo "$n $(jq length "$out/$n.work.json") 筆，關鍵字 $(wc -l < "$out/$n.kw.txt")"
done

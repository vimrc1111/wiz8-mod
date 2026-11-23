#!/usr/bin/env bash
# wiz8-run.sh <情境.json>：實機跑一次 Wiz8（stage2 §2.1）。
#   情境：{"id":"menu","files":[{"src":"<repo 或絕對路徑>","dst":"<遊戲目錄內相對路徑>"}],"duration":60,
#          "steps":[{"t":30,"shot":"menu"},{"t":8,"keys":"{ESC}"},{"t":9,"click":[1920,1080]}]}（每步恰一種；click＝螢幕像素左鍵）
#   wiz8-run.sh --check <情境.json>  只驗情境格式（離線，CI 用）
# 截圖存 ~/.local/state/wiz8-mod-runs/<run-id>/（含原作美術，不入版控）；量測 JSON 寫 notes/runs/<run-id>.json。
# 遊戲目錄前後 hashdir：執行前與 clean 只准雜訊，結束後與執行前只准雜訊；任何時候離開都殺行程、移除放入的檔案。
set -euo pipefail
here=$(cd "$(dirname "$0")/.." && pwd)
G=${WIZ8_DIR:-"/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8"}
CLEAN=${WIZ8_CLEAN:-$here/baseline/2026-10-07-c2-after.json}
NOISE='Saves/,*.CFG,steam_appid.txt'

check() {
  jq -e '
    (.id|type=="string" and test("^[A-Za-z0-9._-]+$")) and
    (.duration|type=="number" and .>0 and .<=600) and
    ((.args//[])|type=="array" and all(type=="string" and test("^/[A-Z]+$"))) and
    ((.files//[])|all(.src|type=="string") and all(.dst|type=="string" and (test("\\.\\.")|not) and (startswith("/")|not))) and
    (.steps|type=="array" and all(.t|type=="number" and .>=0) and all([has("shot"), has("keys"), has("click")]|map(select(.))|length==1) and
      all((.click//[0,0])|type=="array" and length==2 and all(type=="number" and .>=0 and .<8192 and .==floor)) and
      all((.shot//"x")|test("^[A-Za-z0-9_-]+$")))
  ' "$1" >/dev/null || { echo "FAIL scenario:$1" >&2; return 1; }
}
if [ "${1:-}" = --check ]; then check "$2"; echo "ok scenario:$2"; exit 0; fi
sc=$(realpath "$1"); check "$sc"
cd "$here"
S=$(mktemp -d); go build -o "$S/hashdir" ./tools/hashdir; go build -o "$S/shot" ./tools/shot
id=$(jq -r .id "$sc"); rid="$(date +%Y%m%d-%H%M%S)-$id"
state=~/.local/state/wiz8-mod-runs/$rid; mkdir -p "$state" notes/runs
wtmp=$(wslpath -w /mnt/c/Users/User/AppData/Local/Temp)'\wiz8-run\'$rid
ltmp=$(wslpath -u "$wtmp")
if tasklist.exe /FI "IMAGENAME eq Wiz8.exe" | grep -qi wiz8.exe; then echo "Wiz8.exe 已在跑，拒絕" >&2; exit 1; fi

"$S/hashdir" scan "$G" > "$state/before.json"
"$S/hashdir" diff -noise "$NOISE" "$CLEAN" "$state/before.json" > "$state/before.diff" || { cat "$state/before.diff"; echo "執行前遊戲目錄與 clean 有非雜訊差異，拒絕" >&2; exit 1; }

# ddraw 代理的 log（hook/ddraw/proxy.c）寫在 %TEMP%：先清掉上一次的，跑完收進 state（全文含原文字串，不入版控）
T=/mnt/c/Users/User/AppData/Local/Temp; logs=(wiz8-mod-ddraw.log wiz8-mod-text.log)
for f in "${logs[@]}"; do rm -f "$T/$f"; done
placed=()
cleanup() {
  taskkill.exe /IM Wiz8.exe /F >/dev/null 2>&1 || true
  # taskkill 回來時行程可能還握著 ddraw.dll（rm 得 Input/output error，run 035616）：等它消失、刪除重試
  for i in $(seq 20); do tasklist.exe /FI "IMAGENAME eq Wiz8.exe" | grep -qi wiz8.exe || break; sleep 0.5; done
  for f in "${placed[@]}"; do
    for i in $(seq 10); do rm -f "$G/$f" 2>/dev/null && break; sleep 0.5; done
    [ ! -e "$G/$f" ] || { echo "FAIL：刪不掉 $G/$f" >&2; return 1; }
  done
}
trap cleanup EXIT
# 產物比來源舊就不跑（run 085203：PATCH.002 早於 Kill→即死 譯文，MG 實機看到舊字）
# ddraw 的 CJK 圖集取自 translations：比譯文舊＝缺字（2026-10-08 怪物名缺字）
for pair in "build/zh/PATCH.002 translations" "hook/ddraw/out/ddraw.dll hook/ddraw" "hook/ddraw/out/ddraw.dll translations"; do
  set -- $pair
  [ ! -e "$here/$1" ] || [ -z "$(find "$here/$2" -newer "$here/$1" -type f ! -path "*/out/*" | head -1)" ] ||
    { echo "FAIL：$1 比 $2 舊，先重建" >&2; exit 1; }
done
while IFS=$'\t' read -r src dst; do
  [ -n "$src" ] || continue
  [ -e "$G/$dst" ] && { echo "$dst 已存在於遊戲目錄，拒絕覆蓋" >&2; exit 1; }
  case $src in /*) ;; *) src=$here/$src;; esac
  mkdir -p "$(dirname "$G/$dst")"; cp "$src" "$G/$dst"; placed+=("$dst")
  printf '%s\t%s\n' "$dst" "$(sha256sum < "$G/$dst" | cut -c1-64)" >> "$S/placed.tsv"
done < <(jq -r '(.files//[])[]|[.src,.dst]|@tsv' "$sc")

jq '.steps|sort_by(.t)' "$sc" > "$S/steps.json"
/mnt/c/Windows/SysWOW64/WindowsPowerShell/v1.0/powershell.exe -NoProfile -ExecutionPolicy Bypass \
  -File "$(wslpath -w "$here/scripts/wiz8-run.ps1")" -Dir "$(wslpath -w "$G")" -Steps "$(wslpath -w "$S/steps.json")" \
  -Duration "$(jq .duration "$sc")" -GameArgs "$(jq -r '(.args//[])|join(" ")' "$sc")" -Out "$wtmp" > "$state/ps.log" 2>&1 || echo "powershell rc=$?" >> "$state/ps.log"
cleanup; placed=()
mv "$ltmp"/* "$state"/ 2>/dev/null || true; rmdir "$ltmp" 2>/dev/null || true
for f in "${logs[@]}"; do [ -e "$T/$f" ] && mv "$T/$f" "$state/"; done

"$S/hashdir" scan "$G" > "$state/after.json"
after_rc=0; "$S/hashdir" diff -noise "$NOISE" "$state/before.json" "$state/after.json" > "$state/after.diff" || after_rc=$?

shots='{}'
for f in "$state"/*.png; do
  [ -e "$f" ] || continue
  shots=$(jq --arg k "$(basename "$f" .png)" --argjson v "$("$S/shot" stats "$f")" '.[$k]=$v' <<<"$shots")
done
jq -n --arg rid "$rid" --arg commit "$(git rev-parse HEAD)" --argjson sc "$(cat "$sc")" --argjson shots "$shots" \
  --arg before "$(cat "$state/before.diff")" --arg after "$(cat "$state/after.diff")" --argjson after_rc "$after_rc" \
  --arg events "$(tr -d '\r' < "$state/events.txt" 2>/dev/null)" --arg modules "$(tr -d '\r' < "$state/modules.txt" 2>/dev/null | tr '\n' ' ')" \
  --arg ps "$(tr -d '\r' < "$state/ps.log")" --arg placed "$(cat "$S/placed.tsv" 2>/dev/null)" \
  --arg ddraw "$(tr -d '\r' < "$state/wiz8-mod-ddraw.log" 2>/dev/null)" --argjson text_lines "$(cat "$state/wiz8-mod-text.log" 2>/dev/null | wc -l)" \
  '{run_id:$rid, commit:$commit, scenario:$sc, before_diff:$before, after_diff:$after, after_clean:($after_rc==0),
    events:($events|split("\n")|map(select(.!=""))), modules:($modules|split(" ")|map(select(.!=""))), powershell:$ps, placed:($placed|split("\n")|map(select(.!="")|split("\t")|{dst:.[0],sha256:.[1]})), shots:$shots,
    ddraw_log:($ddraw|split("\n")|map(select(.!=""))), text_log_lines:$text_lines}' \
  > "notes/runs/$rid.json"
echo "run $rid：after_clean=$([ $after_rc = 0 ] && echo true || echo false) 截圖 $(jq length <<<"$shots") 張 → notes/runs/$rid.json；圖在 $state"
[ "$after_rc" = 0 ] || { cat "$state/after.diff"; echo "執行後遊戲目錄有非雜訊差異" >&2; exit 1; }
# 情境排了 shot 卻沒圖＝沒跑到（PS 解析壞過一次，見 notes/runs/20261007-022609），不能讀成成功
want=$(jq '[.steps[]|select(has("shot"))]|length' "$sc"); got=$(find "$state" -maxdepth 1 -name "*.screen.png" | wc -l)
[ "$got" = "$want" ] || { echo "截圖 $got／$want 張，不符" >&2; exit 1; }
# 遊戲在時限前結束（崩潰、或從 Steam 另開一個把它關掉，run 041922／042256）＝沒跑完，不能讀成成功
tr -d '\r' < "$state/events.txt" 2>/dev/null | grep -q ' alive$' || { echo "Wiz8 在 $(jq .duration "$sc") 秒前結束：$(tr -d '\r' < "$state/events.txt" | grep exited)" >&2; exit 1; }

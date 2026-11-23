#!/usr/bin/env bash
# I9c (goals/stage16.md): Expert screening → pools → per-class rank → best parties. Output build/i9e/.
#   bash scripts/i9c-run.sh                 full run (screening is reused if build/i9e/screen-c*.tsv exist)
#   bash scripts/i9c-run.sh 8 10 3,1        pool-edge pass: widen pools and rerun only these classes
set -euo pipefail
cd "$(dirname "$0")/.."
exe=build/Wiz8.exe dbs=build/data-x/DATABASES out=build/i9e
top=${1:-6} top_off=${2:-8} cls=${3:-0,1,2,3,4,5,6,7,8,9}
mkdir -p "$out"
go build -o "$out/calc" ./tools/calc
py() { python3 -I scripts/i9-party.py "$@"; }
run() {
  local t0=$SECONDS rc=0
  "$out/calc" rank -exe "$exe" -dbs "$dbs" "$out/$1.json" > "$out/$1.tsv" || rc=$?
  echo "$1 rc=$rc $((SECONDS - t0)) s top=$top/$top_off" >> "$out/run.log"
}
echo "start $(date '+%F %T') top=$top/$top_off classes=$cls" >> "$out/run.log"
if ! ls "$out"/screen-c*.tsv >/dev/null 2>&1; then
  py screen "$dbs" "$out"
  for c in 0 1 2 3 4 5 6 7 8 9; do run "screen-c$c" & done
  wait
fi
py pool "$dbs" "$out" "$cls" "$top" "$top_off" | tee -a "$out/pool.log"
for c in ${cls//,/ }; do run "party-c$c" & done
wait
echo "end $(date '+%F %T')" >> "$out/run.log"
cat "$out/run.log"
if grep -q 'rc=[1-9]' "$out/run.log"; then echo "FAIL rc"; exit 1; fi
py party "$out" > "$out/party.md"  # check() asserts row counts per class first
cat "$out/party.md"

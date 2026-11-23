#!/usr/bin/env bash
# N3 (GOAL-wiz8calc.md): I9c frozen pool × SEN attrs × metric kill / kill-hi → build/n3/<metric>/rank6.tsv.
#   bash scripts/n3-run.sh          runs missing party-c*.tsv (8 jobs at once, nice 19), then rank6 twice and cmp
set -euo pipefail
cd "$(dirname "$0")/.."
exe=build/Wiz8.exe dbs=build/data-x/DATABASES out=build/n3
mkdir -p "$out"
go build -o "$out/calc" ./tools/calc
py() { python3 -I scripts/i9-party.py "$@"; }
for m in kill kill-hi; do
  mkdir -p "$out/$m"
  [ -f "$out/$m/party-c0.json" ] || py n3tmpl build/i9e "$out/$m" "$m"
done
jobs=()
for m in kill kill-hi; do for c in 0 1 2 3 4 5 6 7 8 9; do
  [ -s "$out/$m/party-c$c.tsv" ] || jobs+=("$m/party-c$c")
done; done
echo "start $(date '+%F %T') ${#jobs[@]} jobs; $(uptime)" >> "$out/run.log"
printf '%s\n' "${jobs[@]}" | xargs -r -P 8 -I{} bash -c '
  t0=$SECONDS rc=0
  nice -n 19 "$1/calc" rank -exe "$2" -dbs "$3" "$1/{}.json" > "$1/{}.tsv.part" || rc=$?
  [ $rc = 0 ] && mv "$1/{}.tsv.part" "$1/{}.tsv"
  echo "{} rc=$rc $((SECONDS - t0)) s" >> "$1/run.log"' _ "$out" "$exe" "$dbs"
echo "end $(date '+%F %T')" >> "$out/run.log"
for m in kill kill-hi; do
  py rank6 "$out/$m" > "$out/$m/rank6.tsv"
  py rank6 "$out/$m" | cmp - "$out/$m/rank6.tsv"  # each cell recomputes from the party TSVs to the same value
  [ "$(wc -l < "$out/$m/rank6.tsv")" = 211 ] || { echo "FAIL rank6 $m rows"; exit 1; }
done
echo "ok n3"

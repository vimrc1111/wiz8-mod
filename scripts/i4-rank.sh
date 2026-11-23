#!/usr/bin/env bash
# I4 rank v1 (goals/stage16.md): screen → main/reference → summary → party. Output build/i4/.
set -euo pipefail
cd "$(dirname "$0")/.."
exe=build/Wiz8.exe dbs=build/data-x/DATABASES out=build/i4
mkdir -p "$out"
go build -o "$out/calc" ./tools/calc
py() { python3 -I scripts/i4-rank.py "$@"; }
run() { "$out/calc" rank -exe "$exe" -dbs "$dbs" "$out/$1.json" > "$out/$1.tsv"; }
py screen "$dbs/ITEMS.DBS" "$out"
for m in 564 411; do run screen-$m; done
py main "$dbs/ITEMS.DBS" "$out"
for f in "$out"/main-*.json "$out"/ref*.json; do run "$(basename "$f" .json)"; done
py summary "$out" > "$out/summary.md"
# Secondary: MG party snapshots through calc fight (round.damage per scenario).
mkdir -p "$out/party"; py party "$out/party"
for f in "$out"/party/*.json; do
  printf '%s\t%s\n' "$(basename "$f" .json)" "$("$out/calc" fight -exe "$exe" -dbs "$dbs" "$f" | awk -F'\t' '$1=="round.damage"{print $2}')"
done > "$out/party.tsv"
echo "done: $out/summary.md"

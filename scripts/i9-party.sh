#!/usr/bin/env bash
# I9 six-member party (goals/stage16.md): per-class kill-rate rank over 56 L20+ monsters → best party. Output build/i9/.
# Needs build/i4/screen-*.tsv (scripts/i4-rank.sh) for the pools.
set -euo pipefail
cd "$(dirname "$0")/.."
exe=build/Wiz8.exe dbs=build/data-x/DATABASES out=build/i9
mkdir -p "$out"
go build -o "$out/calc" ./tools/calc
python3 -I scripts/i9-party.py templates "$dbs" "$out"
for c in 0 1 2 3 4 5 6 7 8 9; do "$out/calc" rank -exe "$exe" -dbs "$dbs" "$out/party-c$c.json" > "$out/party-c$c.tsv"; done
python3 -I scripts/i9-party.py party "$out" > "$out/party.md"
echo "done: $out/party.md"

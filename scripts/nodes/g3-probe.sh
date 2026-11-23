#!/usr/bin/env bash
# Five successful recasts; the optional depth also checks the calc arena limit.
set -euo pipefail
here=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
python3 - "$here/re/emu/cases/004fa4d0.json" "${1:-5}" <<'PY'
import json, sys
case = json.load(open(sys.argv[1]))
index = {f['name']: i for i, f in enumerate(case['in'])}
depth = int(sys.argv[2])
assert 1 <= depth <= 60
rows = [r for r in case['boundary'] if r[index['recastTargetMaxHP']] == 105]
assert len(rows) == 1
row = rows[0].copy()
row[index['recastTargetMaxHP']] = 100 + depth
for name, i in index.items():
    if name.startswith('r54f190_'):
        row[i] = int(int(name.rsplit('_', 1)[1]) >= depth - 1)
print(*row)
PY

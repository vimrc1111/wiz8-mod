#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ ! -d build/data-x/DATABASES ]]; then
    echo 'SKIP loot-where'
    exit 0
fi
scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT
python3 -I - "$scratch" <<'PY'
from pathlib import Path
import subprocess
import sys

root = Path('build/data-x/DATABASES')
scratch = Path(sys.argv[1])


def query(mode, value, directory=root):
    result = subprocess.run(['python3', '-I', 'scripts/loot-where.py', mode,
                             str(directory), str(value)], capture_output=True, text=True)
    assert result.returncode == 0, result.stderr
    return result.stdout


def rows(output, kind):
    return [line.split('\t') for line in output.splitlines() if line.startswith(kind + '\t')]


try:
    out74 = query('item', 74)
    tables = rows(out74, 'TABLE')
    assert len(tables) == 7, 'item 74 must have exactly seven table rows'
    assert {(int(r[1]), int(r[5])) for r in tables} == {
        (20, 2), (21, 1), (22, 2), (33, 3), (50, 1), (57, 1), (58, 7)
    }, 'item 74 table/weight mismatch'
    out34 = query('item', 34)
    assert {int(r[1]) for r in rows(out34, 'TABLE')} == {34, 50}, 'item 34 table mismatch'
    out86 = query('item', 86)
    assert any(r[1] == '283' and r[4:8] == ['1', '54', '10', '0.11']
               for r in rows(out86, 'MONSTER_SOURCE')), 'monster 283 must reference table 54 with c=10'
    monster = query('monster', 283)
    assert len(rows(monster, 'SLOT')) == 8, 'must show all eight monster slots'
    assert any(r[2] == '1' and r[4] == '54' for r in rows(monster, 'SLOT')), 'monster 283 table mismatch'
    assert any(r[1] == '54' and r[3] == '86' for r in rows(monster, 'TABLE_ITEM')), 'table 54 item 86 missing'
    assert any(r[3] == '58' for r in rows(out74, 'NPC_DEATH')), 'NPC death pool 58 missing'
    assert query('item', 'excaliber') == out34, 'name and ID queries differ'
    # Only a truncated input is written, outside the repository; output stays in memory.
    for filename in ('ITEMS.DBS', 'MONSTERS.DBS', 'NPC.DBS'):
        (scratch / filename).symlink_to((root / filename).resolve())
    (scratch / 'ITEMTABLES.DBS').write_bytes((root / 'ITEMTABLES.DBS').read_bytes()[:-1])
    bad = subprocess.run(['python3', '-I', 'scripts/loot-where.py', 'item', str(scratch), '74'],
                         capture_output=True, text=True)
    assert bad.returncode != 0 and 'ERROR loot-where:' in bad.stderr, 'truncated table must fail explicitly'
except (AssertionError, OSError) as exc:
    print(f'FAIL loot-where: {exc}', file=sys.stderr)
    sys.exit(1)
print('PASS loot-where')
PY

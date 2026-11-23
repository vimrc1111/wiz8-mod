#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../.."
# A normal-play count above eight is known; nine is the documented fallback.
# Both the native-stub list and the F selection world contain nine monsters.
python3 - <<'PYPROBE'
import json
from pathlib import Path
case = json.loads(Path("re/emu/cases/00547cb0.json").read_text())
names = [field["name"] for field in case["in"]]
row = dict(zip(names, case["boundary"][0]))
row.update(monCount=9, mode=0, levelB5=0, levelBD=0, sa_extraMonsterCount=8,
           sa_spellTargetType=7, sa_monsterSpecies=20)
print(" ".join(str(row[name]) for name in names))
PYPROBE

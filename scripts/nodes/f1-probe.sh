#!/usr/bin/env bash
# f1-probe.sh：印 544010 一組 spellIndex＝域上界（8）的合法輸入（單行空白分隔），取自 case 的 boundary 列（tickets/F1）。
set -euo pipefail
here=$(cd "$(dirname "$0")/../.." && pwd)
python3 -I -c '
import json, sys
c = json.load(open(sys.argv[1]))
r = next(r for r in c["boundary"] if r[84] == 8 and r[69] != 0)
print(*r)' "$here/re/emu/cases/00544010.json"

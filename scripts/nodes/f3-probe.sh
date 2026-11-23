#!/usr/bin/env bash
# f3-probe.sh：印 551ba0 一組怪物目標（種類 3）、目標 +4 角色索引＝−1 的合法輸入（單行空白分隔），取自 case 的 boundary 列（tickets/F3）。
set -euo pipefail
here=$(cd "$(dirname "$0")/../.." && pwd)
python3 -I -c '
import json, sys
c = json.load(open(sys.argv[1]))
r = next(r for r in c["boundary"] if r[10] == 3 and r[11] == -1)
print(*r)' "$here/re/emu/cases/00551ba0.json"

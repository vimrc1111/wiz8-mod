#!/usr/bin/env bash
# f4-probe.sh：印 4fa4d0 的 calc-qa Q8 情境（Mage L10、Fire 50、PL3 Fireball＝SPELLTABLES id 36：費用 6、等級 3、realm 0、人物施法者），單行空白分隔（tickets/F4）。
# 底列取 case 的 boundary 列中會走到 0x4fb4c0→0x4ff220 的一列（施法者 5、怪物目標），再改成上述情境：
# 戰鬥中、難度 2、進度 100；學派（Wizardry 等）全 0＝最壞情形；未沉默（c+0xa21＝0）；0x652da4＝0（否則 realm 0 必失敗）。
set -euo pipefail
here=$(cd "$(dirname "$0")/../.." && pwd)
python3 -I -c '
import json, sys
c = json.load(open(sys.argv[1]))
ix = {f["name"]: i for i, f in enumerate(c["in"])}
r = list(c["boundary"][198])
set_ = {"spell": 36, "PL": 3, "r49": 6, "r56": 3, "r133": 0, "c.school": 14, "c.fa21": 0, "c.f6cf": 0,
        "c.f5cb_0": 50, "c.fb45_0": 60, "g683f94": 1, "g6850d5": 2, "G.fb4": 100, "g652da4": 0}
for k in range(15): set_["c.skill%d" % k] = 10 if k == 14 else 0
for k in range(24, 28): set_["c.f1a3_%d" % k] = 0
for k in range(1, 6): set_["c.f5cb_%d" % k] = 0
for k, v in set_.items(): r[ix[k]] = v
print(*r)' "$here/re/emu/cases/004fa4d0.json"

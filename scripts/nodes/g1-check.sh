#!/usr/bin/env bash
# g1-check.sh：goals/stage14.md G1。550c10／54fa30／545090 的 case 要以位址樁換 4c6c30（已登記排除 c359052），
# 不再靠 setup 把 actor +0x1dc（offset 476）位元 0x400 固定設定讓它不動作（同 54e950 修法）。
cd "$(dirname "$0")/../.." || exit 1
python3 -I - <<'PY'
import json, sys
bad = 0
for a in ("00550c10", "0054fa30", "00545090"):
    d = json.load(open(f"re/emu/cases/{a}.json"))
    if "0x4c6c30" not in d.get("stubs", {}):
        print(f"FAIL g1-check {a}：stubs 沒有 0x4c6c30"); bad = 1
    for s in d.get("setup", []):
        v = s.get("value")
        if s.get("at", {}).get("offset") == 476 and isinstance(v, int) and v & 0x400:
            print(f"FAIL g1-check {a}：setup 仍固定 +0x1dc 位元 0x400（{s['at']}）"); bad = 1
sys.exit(bad)
PY
[ $? -eq 0 ] && echo "ok g1-check"

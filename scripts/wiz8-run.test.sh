#!/usr/bin/env bash
# wiz8-run.sh --check 的離線測試：出貨情境全過；壞情境各自 FAIL。
set -uo pipefail
cd "$(dirname "$0")/.."
fail=0; T=$(mktemp -d)
for f in scenarios/*.json; do bash scripts/wiz8-run.sh --check "$f" >/dev/null 2>&1 || { echo "FAIL 出貨情境被拒：$f"; fail=1; }; done
bad() { echo "$2" > "$T/$1.json"; if bash scripts/wiz8-run.sh --check "$T/$1.json" >/dev/null 2>&1; then echo "FAIL 壞情境被接受：$1"; fail=1; fi; }
bad no-id        '{"duration":10,"steps":[]}'
bad id-slash     '{"id":"a/b","duration":10,"steps":[]}'
bad dur-huge     '{"id":"a","duration":9999,"steps":[]}'
bad dst-dotdot   '{"id":"a","duration":10,"files":[{"src":"x","dst":"../x"}],"steps":[]}'
bad dst-abs      '{"id":"a","duration":10,"files":[{"src":"x","dst":"/x"}],"steps":[]}'
bad step-empty   '{"id":"a","duration":10,"steps":[{"t":1}]}'
bad shot-name    '{"id":"a","duration":10,"steps":[{"t":1,"shot":"../x"}]}'
bad two-kinds    '{"id":"a","duration":10,"steps":[{"t":1,"shot":"a","keys":"x"}]}'
bad click-1      '{"id":"a","duration":10,"steps":[{"t":1,"click":[5]}]}'
bad click-neg    '{"id":"a","duration":10,"steps":[{"t":1,"click":[-1,5]}]}'
bad click-frac   '{"id":"a","duration":10,"steps":[{"t":1,"click":[1.5,5]}]}'
bad args-str     '{"id":"a","duration":10,"args":"/LOAD","steps":[]}'
bad args-inject  '{"id":"a","duration":10,"args":["/LOAD x"],"steps":[]}'
bad args-lower   '{"id":"a","duration":10,"args":["/load"],"steps":[]}'
bad click-str    '{"id":"a","duration":10,"steps":[{"t":1,"click":"5,5"}]}'
[ $fail = 0 ] && echo "ok wiz8-run --check：$(ls scenarios/*.json | wc -l) 出貨情境＋15 壞情境"
exit $fail

#!/usr/bin/env bash
# regen.sh：重跑所有 case 的模擬器＋TestEmu，重生 build/emu 的 D3／D4 證據（check-formulas 完成模式讀這些）
set -uo pipefail; cd "$(dirname "$0")/../.."; bad=0
for c in re/emu/cases/*.json; do a=$(basename "$c" .json); bash re/emu/run.sh "$a" > /dev/null 2>&1 || { echo "FAIL regen 模擬器 $a"; bad=1; }; done
go test -count=1 -run 'TestEmu$' ./internal/calc > /dev/null || { echo "FAIL regen TestEmu"; bad=1; }
[ $bad = 0 ] && echo "ok regen $(ls build/emu/*.d3 | wc -l) 個 .d3"; exit $bad

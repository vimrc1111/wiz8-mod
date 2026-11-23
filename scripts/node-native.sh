#!/usr/bin/env bash
set -uo pipefail
cd "$(dirname "$0")/.." || exit 1
if (( $# == 0 )); then echo 'FAIL node-native usage: <addr>…'; exit 1; fi
export GOCACHE="${GOCACHE:-/tmp/wiz8-g6-go-cache}"
t=$(mktemp -d)
trap 'rm -rf "$t"' EXIT
bad=0
for addr in "$@"; do
  if [[ ! $addr =~ ^[[:xdigit:]]{1,8}$ ]]; then echo "FAIL node-native $addr invalid address"; bad=1; continue; fi
  a8=$(printf '%08x' "0x$addr")
  if ! bash re/emu/run.sh --native "$a8" > "$t/emu.log" 2>&1; then
    echo "FAIL node-native $addr emulator: $(tail -3 "$t/emu.log")"; bad=1; continue
  fi
  if [[ ! -s build/emu-f/$a8.jsonl ]]; then echo "FAIL node-native $addr empty jsonl"; bad=1; continue; fi
  if ! go test -count=1 -run "^TestFNative$/$a8$" ./internal/calc > "$t/go.log" 2>&1; then
    echo "FAIL node-native $addr $(cat "$t/go.log")"; bad=1; continue
  fi
  proof=$(cat "build/emu-f/$a8.d3f")
  if [[ $proof =~ ^cases=([1-9][0-9]*)\ diff=0$ ]]; then
    echo "ok node-native $addr cases=${BASH_REMATCH[1]}"
  else echo "FAIL node-native $addr invalid proof: $proof"; bad=1; fi
done
exit "$bad"

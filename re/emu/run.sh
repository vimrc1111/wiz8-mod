#!/usr/bin/env bash
set -euo pipefail
here=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
python="$here/build/venv/bin/python"
if [ ! -x "$python" ]; then
  python="$HOME/.local/state/wiz8-mod-emu/venv/bin/python"
fi
exec "$python" -I "$here/re/emu/emu.py" "$@"

#!/usr/bin/env bash
# 機檢：產出是 i386 PE DLL；匯出名與序號集合 == 系統 ddraw（由 objdump 重新算，不讀 build 的中間檔）。
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd)
dll=${1:-$here/out/ddraw.dll}; sys=${SYS_DDRAW:-/mnt/c/Windows/SysWOW64/ddraw.dll}
exports() { i686-w64-mingw32-objdump -p "$1" | awk '
  /\[Ordinal\/Name Pointer\] Table/ { on = 1; next }
  on && NF == 0 { on = 0 }
  on && /\+base/ { gsub(/[][]/, " "); print $3, $(NF) }' | sort; }
fmt=$(i686-w64-mingw32-objdump -f "$dll" | sed -n 's/.*file format //p')
[ "$fmt" = pei-i386 ] || { echo "FAIL format: $fmt"; exit 1; }
ch=$(i686-w64-mingw32-objdump -p "$dll" | awk '/^Characteristics/ { print $2 }')
(( ch & 0x2000 )) || { echo "FAIL not DLL (Characteristics $ch)"; exit 1; }
want=$(exports "$sys"); got=$(exports "$dll")
[ -n "$want" ] || { echo "FAIL 讀不到系統匯出表"; exit 1; }
if [ "$want" != "$got" ]; then echo "FAIL exports"; diff <(echo "$want") <(echo "$got"); exit 1; fi
# 匯入只准 KERNEL32／USER32／GDI32（不帶 CRT：遊戲目錄與舊系統未必有）
imp=$(i686-w64-mingw32-objdump -p "$dll" | sed -n 's/^\tDLL Name: //p' | sort -u | tr '\n' ' ')
for d in $imp; do case "${d,,}" in kernel32.dll|user32.dll|gdi32.dll) ;; *) echo "FAIL import $d"; exit 1 ;; esac; done
echo "PASS ddraw-proxy: $fmt, $(wc -l <<<"$want") 個匯出與系統一致, imports: $imp"

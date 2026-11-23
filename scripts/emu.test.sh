#!/usr/bin/env bash
# emu.test.sh：D3／D4 差分機制的驗收（goals/stage10.md B1／B2）。
# 模擬器跑原版函式 → build/emu/<addr>.jsonl＋.hit；Go 測試用 calc 節點重放同一串亂數 → build/emu/<addr>.d3。
# 沒有 build/Wiz8.exe 的環境印 SKIP、exit 0（CI 機器不一定有原版檔）。
set -uo pipefail
here=$(cd "$(dirname "$0")/.." && pwd); cd "$here"
fail() { echo "FAIL emu $*"; exit 1; }
[ -f build/Wiz8.exe ] || { echo "SKIP emu：沒有 build/Wiz8.exe"; exit 0; }
[ -x re/emu/run.sh ] || fail "re/emu/run.sh 不存在"
rm -f build/emu/00517970.* build/emu/00552250.*  # 只清自己的種子；其他節點的證據 check-formulas 要讀
for a in 00517970 00552250; do
  [ -f re/emu/cases/$a.json ] || fail "re/emu/cases/$a.json 不存在"
  bash re/emu/run.sh $a > /dev/null || fail "$a 模擬器跑不完"
  [ -s build/emu/$a.jsonl ] && [ -s build/emu/$a.hit ] || fail "$a 缺 build/emu/$a.jsonl 或 .hit"
done
go test -count=1 -run 'TestEmu' ./internal/calc > /dev/null || fail "go test TestEmu 紅"
for a in 00517970 00552250; do
  grep -qE '^cases=[1-9][0-9]* diff=0$' build/emu/$a.d3 2>/dev/null || fail "$a 的 .d3 不是 cases>0 diff=0：$(cat build/emu/$a.d3 2>/dev/null)"
done
# 反證：差分要抓得到錯的節點（測試專用的錯版骰子，結果加 1）
EMU_WRONG=1 go test -count=1 -run 'TestEmuWrong' ./internal/calc > /dev/null || fail "錯版節點沒被抓到（TestEmuWrong 紅）"
# 沒樁的 import 要紅並列位址
out=$(bash re/emu/run.sh --selftest-import 2>&1); rc=$?
[ $rc -ne 0 ] && printf '%s' "$out" | grep -qE 'unstubbed import 0x[0-9a-f]+' || fail "沒樁的 import 沒紅或沒列位址：$out"
# qsort 樁排序要對（亂數 40 個、7 個走 shortsort、1 個不呼叫比較函數）
bash re/emu/run.sh --selftest-qsort > /dev/null 2>&1 || fail "qsort 樁 selftest 紅：$(bash re/emu/run.sh --selftest-qsort 2>&1 | tail -1)"
# 第二批樁（票 Y2）：每行固定值由規格定，selftest 的合成呼叫端由實作寫
out=$(bash re/emu/run.sh --selftest-stubs 2>&1) || fail "--selftest-stubs 非零退出：$(printf '%s' "$out" | tail -3)"
for want in 'vswprintf "hp=-7 x=1f n=ab w=AB 0042%"' 'sprintf "hp=-7 w=AB n=ab 000"' 'vsprintf "42  |"' 'fopen NULL' 'case stub ok' 'heap ok' 'srHeap allocate ok, CreateFileA INVALID' 'GetLastError 2, srTexture this, srNode pop 4' '_CIacos 0.5, FormatMessageA 0, swprintf ok' 'floor -3, _CIasin 0.5, memmove overlap, _CIpow 27' 'wcscat abcd 4, atoi strchr stricmp strnicmp wcscmp strncpy' 'getNewFrameHandle 1 2, invalidateFrameHandle pop 4' 'getMsTime 777' 'GetTickCount 12345' 'fs0 ok' 'srAssertFail stop foo.cpp:77' 'unmapped eip=0x' \
    'wcscpy "hello" ok' '40efe0 pct ok' 'write global seq ok' 'srClass ok' 'st0 0x400c000000000000' 'index ok'; do  # wcscpy／srClass／st0：票 Y3；index：票 Y4
  printf '%s\n' "$out" | grep -qF "$want" || fail "--selftest-stubs 缺「$want」：$(printf '%s' "$out" | tail -5)"
done
# calc f 能查單一節點並印分布
f=$(go run ./tools/calc f 00517970 2 3 6) && grep -qE '^dist' <<<"$f" || fail "calc f 00517970 沒印 dist"
echo "ok emu"

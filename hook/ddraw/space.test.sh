#!/usr/bin/env bash
# space.h 中英間距（stage5，MG 2026-10-07 試玩：戰鬥 log 有的模板中英間有空格、有的沒有）：主機 gcc 逐條比對。
# 規則：漢字／假名／諺文 與 ASCII 英數字直接相鄰處補一個空白；全形標點、ASCII 標點、已有空白不補。
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd); t=$(mktemp -d); trap 'rm -rf "$t"' EXIT
cat >"$t/m.c" <<'C'
#include <stdio.h>
#include <string.h>
#include <wchar.h>
#include "space.h"
static const struct { const wchar_t *in, *want; const char *why; } T[] = {
    {L"Sprite施放Iceball", L"Sprite 施放 Iceball", "名字|中|法術"},
    {L"Armorplate已失效。", L"Armorplate 已失效。", "英|中、句號不動"},
    {L"施放Iceball，目標5 -- 44 點傷害", L"施放 Iceball，目標 5 -- 44 點傷害", "，前後不補、數字補"},
    {L"受到 5 點傷害", L"受到 5 點傷害", "已有空白不重複"},
    {L"箭(未鑑定)", L"箭(未鑑定)", "ASCII 括號不補"},
    {L"Curare Crab 3x", L"Curare Crab 3x", "純 ASCII 不動"},
    {L"「Kill」模組", L"「Kill」模組", "全形引號不補"},
    {L"中\nA", L"中\nA", "LF 不補"},
    {L"アa", L"ア a", "假名|英"},
    {L"한1", L"한 1", "諺文|數字"},
    {L"%s受到", L"%s 受到", "格式記號照字面（s 是英字母）"},
    {L"", L"", "空字串"},
};
int main(void) {
    int bad = 0;
    wchar_t out[64];
    for (unsigned i = 0; i < sizeof T / sizeof *T; i++) {
        unsigned short in[64], o[64];
        size_t n = wcslen(T[i].in);
        for (size_t k = 0; k <= n; k++) in[k] = (unsigned short)T[i].in[k];
        int r = space_cjk(o, 64, in);
        for (int k = 0; k < 64; k++) { out[k] = o[k]; if (!o[k]) break; }
        if (r != (int)wcslen(T[i].want) || wcscmp(out, T[i].want)) {
            printf("FAIL space-rule：%s 得 %d\n", T[i].why, r); bad = 1;
        }
    }
    // 容量：剛好放得下（含 NUL）回長度；少一格回 -1，且不寫超過 cap
    unsigned short in[] = {'a', 0x4e2d, 0}, o[8] = {0, 0, 0, 0, 0x7777, 0x7777, 0, 0};
    if (space_cjk(o, 4, in) != 3 || o[1] != ' ') { printf("FAIL space-cap：剛好\n"); bad = 1; }
    o[3] = 0x7777;
    if (space_cjk(o, 3, in) != -1 || o[3] != 0x7777) { printf("FAIL space-cap：不足\n"); bad = 1; }
    return bad;
}
C
gcc -Wall -Werror -I"$here" -o "$t/m" "$t/m.c" 2>"$t/err" || { cat "$t/err"; echo "FAIL space-rule（編譯）"; exit 1; }
"$t/m" || { echo "FAIL space-rule"; exit 1; }
echo "PASS space-rule"

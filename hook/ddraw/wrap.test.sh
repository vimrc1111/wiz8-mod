#!/usr/bin/env bash
# wrap.h 斷行規則（stage5 5.3）：主機 gcc 對一張 (前字, 後字) → 可否在兩字之間換行 的表逐條比對。
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd); t=$(mktemp -d); trap 'rm -rf "$t"' EXIT
cat >"$t/m.c" <<'C'
#include <stdio.h>
#include "wrap.h"
static const struct { unsigned short a, b; int want; const char *why; } T[] = {
    {'a', 'b', 0, "英文字中間"},        {'a', ' ', 1, "空白前"},          {' ', 'a', 1, "空白後"},
    {'%', 's', 0, "格式記號中間"},      {0x7acb, 0x4f0d, 1, "立|伍"},     {'a', 0x4f0d, 1, "英|中"},
    {0x4f0d, 'a', 1, "中|英"},          {0x4f0d, 0xff0c, 0, "中|，"},     {0x4f0d, 0x3002, 0, "中|。"},
    {0x4f0d, 0x3001, 0, "中|、"},       {0x4f0d, 0xff01, 0, "中|！"},     {0x4f0d, 0xff1f, 0, "中|？"},
    {0x4f0d, 0x300d, 0, "中|」"},       {0x4f0d, 0xff09, 0, "中|）"},     {0x300c, 0x4f0d, 0, "「|中"},
    {0xff08, 0x4f0d, 0, "（|中"},       {0xff0c, 0x4f0d, 1, "，|中"},     {0x3002, 'a', 1, "。|英"},
    {'1', 0x4f0d, 1, "數字|中"},        {0x4f0d, '1', 1, "中|數字"},      {0x30a2, 0x30a4, 1, "假名|假名"},
    {0xac00, 0xac01, 1, "諺文|諺文"},   {'a', 0xff0c, 0, "英|，"},
};
int main(void) {
    int bad = 0;
    for (unsigned i = 0; i < sizeof T / sizeof *T; i++) {
        int got = wrap_can_break(T[i].a, T[i].b) != 0;
        if (got != T[i].want) { printf("FAIL wrap-rule：%s U+%04X|U+%04X 得 %d 要 %d\n", T[i].why, T[i].a, T[i].b, got, T[i].want); bad = 1; }
    }
    return bad;
}
C
gcc -Wall -Werror -I"$here" -o "$t/m" "$t/m.c"
"$t/m" || { echo "FAIL wrap-rule"; exit 1; }
echo "PASS wrap-rule"

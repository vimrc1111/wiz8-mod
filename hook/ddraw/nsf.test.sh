#!/usr/bin/env bash
# nsf.h NSF 對白窄字往返（stage7，MG run 170652：Myles 回覆在第一個中文字截斷）：主機 gcc 逐條比對。
# 規則：≤0xFF 一 byte 與原版相同；>0xFF 往返不變；原版會出現的窄字串（ASCII、0x92 等單 byte）解回與一對一相同。
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd); t=$(mktemp -d); trap 'rm -rf "$t"' EXIT
cat >"$t/m.c" <<'C'
#include <stdio.h>
#include <string.h>
#include "nsf.h"
static int eq(const unsigned short *a, const unsigned short *b) { while (*a && *a == *b) a++, b++; return *a == *b; }
int main(void) {
    int bad = 0;
    static const unsigned short T[][16] = {
        {'H', 'i', 0x92, 's', 0},           // 原文：ASCII＋U+0092
        {0x4e2d, 0x6587, ' ', 'A', 0},       // 中文＋ASCII
        {0x0100, 0x07ff, 0x0800, 0xffff, 0}, // 2／3 byte 邊界
        {0x00ff, 0x00e9, 0x0092, 0},         // 單 byte 上界（Latin-1 後接 0x80–0xBF 會被讀成序列：nsf.h 上限）
        {0xd83d, 0xde00, 0},                 // 代理對逐 unit
        {0},
    };
    for (unsigned i = 0; i < sizeof T / sizeof *T; i++) {
        unsigned char n[64]; unsigned short w[64];
        nsf_narrow(n, T[i]);
        int len = 0; while (T[i][len]) len++;
        if ((int)strlen((char *)n) != nsf_narrow_len(T[i])) { printf("FAIL nsf-len #%u\n", i); bad = 1; }
        if (nsf_widen(w, n) != len || !eq(w, T[i])) { printf("FAIL nsf-roundtrip #%u\n", i); bad = 1; }
    }
    // 只有 ≤0xFF 時與 C locale 的 %S 相同：一 unit 一 byte
    unsigned char n[8];
    nsf_narrow(n, T[0]);
    if (memcmp(n, "Hi\x92s", 5)) { printf("FAIL nsf-ascii：窄字與原版不同\n"); bad = 1; }
    // 原版窄字串（含孤立高位 byte、看似但不合法的序列、過長編碼）解回一對一
    static const char *R[] = {"It\x92s", "\xe9t\xe9", "\xc3\xa9", "\xe0\x80\x80", "\xe4\xb8", "\xc4"};
    for (unsigned i = 0; i < sizeof R / sizeof *R; i++) {
        unsigned short w[16]; const unsigned char *s = (const unsigned char *)R[i];
        int k = nsf_widen(w, s), ok = k == (int)strlen(R[i]);
        for (int j = 0; ok && j < k; j++) ok = w[j] == s[j];
        if (!ok || nsf_has_seq(s)) { printf("FAIL nsf-legacy #%u\n", i); bad = 1; }
    }
    // 引號格式（0x0061c43c）：前後綴照抄、中間解回；L"%S" 等同 nsf_widen
    static const unsigned short Q[] = {' ', '"', '%', 'S', '"', 0}, P[] = {'%', 'S', 0}, QW[] = {' ', '"', 0x4e2d, 'A', '"', 0};
    unsigned short w[16];
    nsf_narrow(n, (const unsigned short[]){0x4e2d, 'A', 0});
    if (nsf_widen_fmt(w, Q, n) != 5 || !eq(w, QW)) { printf("FAIL nsf-fmt quoted\n"); bad = 1; }
    if (nsf_widen_fmt(w, P, n) != 2 || w[0] != 0x4e2d || w[1] != 'A' || w[2]) { printf("FAIL nsf-fmt plain\n"); bad = 1; }
    return bad;
}
C
gcc -Wall -Werror -I"$here" -o "$t/m" "$t/m.c" 2>"$t/err" || { cat "$t/err"; echo "FAIL nsf-rule（編譯）"; exit 1; }
"$t/m" || { echo "FAIL nsf-rule"; exit 1; }
echo "PASS nsf-rule"

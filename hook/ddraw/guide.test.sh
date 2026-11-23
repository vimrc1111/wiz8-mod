#!/usr/bin/env bash
# guide.h F7 攻略面板純函數（G3）：主機 gcc 逐條比對。內容 tsv 在 repo 外（攻略本衍生，不入 repo、不入發布包）。
# 規則：
#   guide_nav(&g, key, npages)：GuideState {layer, page}，回傳 1＝面板吃掉此鍵、0＝交給遊戲（G4，MG 2026-10-14 仿 F10：上下換層、左右翻頁）。
#     關著（layer 0）：GUIDE_TOGGLE → 第 1 層第 0 頁並吃鍵；其他鍵一律不吃、狀態不變。
#     開著：GUIDE_TOGGLE／GUIDE_ESC → 關（layer 0、page 0）；GUIDE_DOWN 層 +1（上限 3）、GUIDE_UP 層 −1（下限 1），換層 page 歸 0、到頂／底仍吃鍵；
#     GUIDE_RIGHT／GUIDE_LEFT page ±1，夾在 0..npages−1（npages<1 視為 1）；GUIDE_OTHER 不吃。
#   guide_paginate(step, n, avail, page_start[], max)：step[i] 是第 i 行佔的高度（>0），貪婪分頁：每頁 step 加總 <= avail，
#     單行比 avail 高仍自成一頁；page_start[k]＝第 k 頁第一行；最多 max 頁；回傳頁數；n==0 回 1 且 page_start[0]=0（空內容仍有一頁）。
#   guide_tag(layer[], i, first)：第 i 行要不要畫層標「(L/3)」（G5，MG 2026-10-14：分不出哪行是哪層）。layer[i]>0 且（i==first 頁首行，或 layer[i]!=layer[i-1]）回 layer[i]，否則 0；layer 0＝訊息行不標。
#   guide_select(tsv, len, map, upto, start[], len[], max)：tsv 每列「地圖索引\t層\t文字」，UTF-8、可 CRLF、末列可無換行；
#     # 開頭、空列、欄數不是 3、索引或層不是純十進位數字的列略過。挑 map 相符且 1<=層<=upto 的列，
#     先依層由小到大、同層依檔內順序，最多 max 列；start／len 指文字欄（不含換行與 CR）；回傳列數。
#   guide_wrap(s, n, adv, width, start[], max)：UTF-16 一段文字依 wrap.h 的 wrap_can_break 斷行，
#     每行寬（adv 加總）<= width；一個不可斷的片段比 width 還寬時硬切（至少一字）；行首空白不算進下一行；
#     start[i] 為第 i 行起點索引，最多 max 行，回傳行數；n==0 回 0。
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd); t=$(mktemp -d); trap 'rm -rf "$t"' EXIT
cat >"$t/m.c" <<'C'
#include <stdio.h>
#include <string.h>
#include "guide.h"
static int bad;
#define EXPECT(c, why) do { if (!(c)) { printf("FAIL guide-rule %s\n", why); bad = 1; } } while (0)
static int adv(unsigned short c) { return c < 128 ? 7 : 12; }
static int eq(const char *s, int n, const char *want) { return n == (int)strlen(want) && !memcmp(s, want, n); }
int main(void) {
    GuideState g = {0};
    EXPECT(guide_nav(&g, GUIDE_DOWN, 5) == 0 && g.layer == 0, "關著時方向鍵交給遊戲");
    EXPECT(guide_nav(&g, GUIDE_ESC, 5) == 0 && g.layer == 0, "關著時 Esc 交給遊戲");
    EXPECT(guide_nav(&g, GUIDE_TOGGLE, 5) == 1 && g.layer == 1 && g.page == 0, "F7 開在第 1 層第 0 頁");
    EXPECT(guide_nav(&g, GUIDE_UP, 5) == 1 && g.layer == 1, "第 1 層再按上仍吃鍵、停在第 1 層");
    EXPECT(guide_nav(&g, GUIDE_RIGHT, 3) == 1 && g.page == 1 && guide_nav(&g, GUIDE_RIGHT, 3) == 1 && g.page == 2, "右翻頁");
    EXPECT(guide_nav(&g, GUIDE_RIGHT, 3) == 1 && g.page == 2, "最後一頁再右不動");
    EXPECT(guide_nav(&g, GUIDE_DOWN, 3) == 1 && g.layer == 2 && g.page == 0, "下：層 +1、頁歸 0");
    EXPECT(guide_nav(&g, GUIDE_DOWN, 3) == 1 && g.layer == 3 && guide_nav(&g, GUIDE_DOWN, 3) == 1 && g.layer == 3, "下到第 3 層為止");
    EXPECT(guide_nav(&g, GUIDE_LEFT, 3) == 1 && g.page == 0, "第 0 頁再左不動");
    EXPECT(guide_nav(&g, GUIDE_UP, 3) == 1 && guide_nav(&g, GUIDE_UP, 3) == 1 && g.layer == 1, "上上回到剩一層");
    EXPECT(guide_nav(&g, GUIDE_RIGHT, 0) == 1 && g.page == 0, "npages 0 視為 1 頁");
    EXPECT(guide_nav(&g, GUIDE_OTHER, 3) == 0 && g.layer == 1, "開著時其他鍵交給遊戲");
    EXPECT(guide_nav(&g, GUIDE_ESC, 3) == 1 && g.layer == 0 && g.page == 0, "Esc 關");
    guide_nav(&g, GUIDE_TOGGLE, 3); guide_nav(&g, GUIDE_DOWN, 3); guide_nav(&g, GUIDE_RIGHT, 3);
    EXPECT(guide_nav(&g, GUIDE_TOGGLE, 3) == 1 && g.layer == 0 && g.page == 0, "開著再按 F7 關");
    EXPECT(guide_nav(&g, GUIDE_TOGGLE, 3) == 1 && g.layer == 1 && g.page == 0, "重開從第 1 層第 0 頁");

    int ps[8];
    static const int step[] = {16, 16, 24, 16, 16, 16};
    EXPECT(guide_paginate(step, 6, 48, ps, 8) == 3 && ps[0] == 0 && ps[1] == 2 && ps[2] == 4, "貪婪分頁、層間距算進高度");
    EXPECT(guide_paginate(step, 6, 1000, ps, 8) == 1 && ps[0] == 0, "放得下就一頁");
    static const int tall[] = {16, 100, 16};
    EXPECT(guide_paginate(tall, 3, 48, ps, 8) == 3 && ps[1] == 1 && ps[2] == 2, "單行比可用高還高自成一頁");
    EXPECT(guide_paginate(step, 6, 16, ps, 2) == 2, "max 截斷");
    EXPECT(guide_paginate(step, 0, 48, ps, 8) == 1 && ps[0] == 0, "空內容一頁");

    static const char tsv[] =
        "# 註解\t1\tx\r\n"
        "0\t2\tB1\r\n"
        "0\t1\tA1\r\n"
        "1\t1\tother\n"
        "\n"
        "0\t3\tC1\n"
        "0\tx\tbad-layer\n"
        "0\t1\n"
        "0\t1\tA2\textra\n"
        "-1\t1\tneg\n"
        "0\t1\tA3";
    const char *st[8]; int ln[8];
    int n = guide_select(tsv, sizeof tsv - 1, 0, 1, st, ln, 8);
    EXPECT(n == 2 && eq(st[0], ln[0], "A1") && eq(st[1], ln[1], "A3"), "第 1 層：只挑本圖、略過壞列、末列無換行、去 CR");
    n = guide_select(tsv, sizeof tsv - 1, 0, 3, st, ln, 8);
    EXPECT(n == 4 && eq(st[0], ln[0], "A1") && eq(st[1], ln[1], "A3") && eq(st[2], ln[2], "B1") && eq(st[3], ln[3], "C1"),
           "累加到第 3 層：層小的先、同層照檔內順序");
    EXPECT(guide_select(tsv, sizeof tsv - 1, 0, 3, st, ln, 3) == 3, "max 截斷");
    EXPECT(guide_select(tsv, sizeof tsv - 1, 5, 3, st, ln, 8) == 0, "沒有該圖回 0");
    EXPECT(guide_select(tsv, sizeof tsv - 1, 0, 0, st, ln, 8) == 0, "upto 0 回 0");
    EXPECT(guide_select("", 0, 0, 3, st, ln, 8) == 0, "空檔回 0");

    int b[16];
    static const unsigned short ab[] = {'a','b','c',' ','d','e','f'};
    n = guide_wrap(ab, 7, adv, 21, b, 16);
    EXPECT(n == 2 && b[0] == 0 && b[1] == 4, "英文在空白斷、下一行不帶空白");
    static const unsigned short cjk[] = {0x4e00,0x4e8c,0x4e09,0xff0c,0x56db};
    n = guide_wrap(cjk, 5, adv, 36, b, 16);
    EXPECT(n == 2 && b[1] == 2, "標點不放行首：三，一起換行");
    static const unsigned short word[] = {'a','b','c','d','e','f','g','h'};
    n = guide_wrap(word, 8, adv, 21, b, 16);
    EXPECT(n == 3 && b[1] == 3 && b[2] == 6, "超寬英文字硬切");
    n = guide_wrap(word, 8, adv, 3, b, 16);
    EXPECT(n == 8, "寬度小於一字仍每行至少一字");
    EXPECT(guide_wrap(word, 8, adv, 3, b, 2) == 2, "max 截斷");
    EXPECT(guide_wrap(word, 0, adv, 21, b, 16) == 0, "空字串 0 行");
    n = guide_wrap(cjk, 5, adv, 1000, b, 16);
    EXPECT(n == 1 && b[0] == 0, "放得下就一行");
    static const int lay[] = {1, 1, 2, 3, 3, 0};
    EXPECT(guide_tag(lay, 0, 0) == 1 && guide_tag(lay, 1, 0) == 0 && guide_tag(lay, 2, 0) == 2 && guide_tag(lay, 3, 0) == 3, "每層第一行標層");
    EXPECT(guide_tag(lay, 4, 4) == 3 && guide_tag(lay, 4, 0) == 0, "頁首行即使同層也標");
    EXPECT(guide_tag(lay, 5, 5) == 0, "訊息行不標");
    if (!bad) puts("PASS guide-rule");
    return bad;
}
C
gcc -std=c11 -Wall -Werror -I"$here" "$t/m.c" -o "$t/m" || { echo "FAIL guide-rule 編譯失敗（guide.h 缺或介面不符）"; exit 1; }
"$t/m"

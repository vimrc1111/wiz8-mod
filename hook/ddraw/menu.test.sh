#!/usr/bin/env bash
# menu.h 內掛選單狀態機（stage11 定義 1／stage12 B1）：主機 gcc 逐條比對。
# 規則：關著時只吃 MK_TOGGLE（開啟），其他鍵回 0 交給遊戲；開著時所有鍵都吃（回 1）。
# 上下移游標（不繞回）；左右 ±MENU_STEP，結果夾在 [MENU_MIN,MENU_MAX]；MK_RESET 本項回 MENU_ORIG、MK_RESET_ALL 全部回；MK_TOGGLE／MK_ESC 關閉。
# dirty：開啟時清 0，任一值實際改變就設 1（夾住沒變、本來就 100 再回原版都不算）。項目表順序與預設＝proxy.c 原 speed_init。
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd); t=$(mktemp -d); trap 'rm -rf "$t"' EXIT
cat >"$t/m.c" <<'C'
#include <stdio.h>
#include <string.h>
#include "menu.h"
static int bad;
#define EXPECT(c, why) do { if (!(c)) { printf("FAIL menu-rule %s\n", why); bad = 1; } } while (0)
int main(void) {
    EXPECT(MENU_N == 4 && MENU_MIN == 10 && MENU_MAX == 5000 && MENU_STEP == 25 && MENU_ORIG == 100, "常數");
    static const char *keys[] = {"monster_move_pct", "event_speed_pct", "anim_speed_pct", "party_move_pct"};
    static const int defs[] = {400, 300, 200, 100};
    for (int i = 0; i < 4 && MENU_N == 4; i++)
        EXPECT(!strcmp(menu_items[i].key, keys[i]) && menu_items[i].def == defs[i], "項目表 key／預設");
    MenuState s; memset(&s, 0, sizeof s);
    for (int i = 0; i < MENU_N; i++) s.val[i] = menu_items[i].def;
    EXPECT(menu_key(&s, MK_UP) == 0 && !s.open && s.sel == 0, "關著不吃方向鍵");
    EXPECT(menu_key(&s, MK_RIGHT) == 0 && s.val[0] == 400, "關著不改值");
    s.dirty = 1;
    EXPECT(menu_key(&s, MK_TOGGLE) == 1 && s.open && !s.dirty, "開啟、清 dirty");
    EXPECT(menu_key(&s, MK_OTHER) == 1 && s.open && !s.dirty, "開著吃其他鍵");
    EXPECT(menu_key(&s, MK_UP) == 1 && s.sel == 0, "上不繞回");
    const int L = MENU_N - 1; // 最後一列 party_move_pct，預設 100
    for (int i = 0; i < MENU_N; i++) menu_key(&s, MK_DOWN);
    EXPECT(s.sel == MENU_N - 1, "下不繞回");
    EXPECT(menu_key(&s, MK_RIGHT) == 1 && s.val[L] == 125 && s.dirty, "右 +25、設 dirty");
    menu_key(&s, MK_LEFT);
    EXPECT(s.val[L] == 100, "左 -25");
    s.val[L] = 4990; menu_key(&s, MK_RIGHT);
    EXPECT(s.val[L] == 5000, "上限 5000");
    s.dirty = 0; menu_key(&s, MK_RIGHT);
    EXPECT(s.val[L] == 5000 && !s.dirty, "夾住沒變不設 dirty");
    s.val[L] = 7; menu_key(&s, MK_LEFT);
    EXPECT(s.val[L] == 10, "ini 給的界外值左移夾到下限");
    s.val[L] = 7; menu_key(&s, MK_RIGHT);
    EXPECT(s.val[L] == 32, "界外值右移 7+25");
    menu_key(&s, MK_RESET);
    EXPECT(s.val[L] == 100 && s.val[0] == 400, "本項回原版、他項不動");
    s.dirty = 0; menu_key(&s, MK_RESET);
    EXPECT(!s.dirty, "已是 100 再回原版不設 dirty");
    menu_key(&s, MK_RESET_ALL);
    EXPECT(s.val[0] == 100 && s.val[1] == 100 && s.val[2] == 100 && s.val[3] == 100 && s.dirty, "全部回原版");
    EXPECT(menu_key(&s, MK_ESC) == 1 && !s.open, "Esc 關閉且吃掉");
    menu_key(&s, MK_TOGGLE);
    EXPECT(s.sel == MENU_N - 1, "重開保留游標");
    EXPECT(menu_key(&s, MK_TOGGLE) == 1 && !s.open, "熱鍵再按關閉");
    if (!bad) puts("PASS menu-rule");
    return bad;
}
C
gcc -std=c11 -Wall -Werror -I"$here" "$t/m.c" -o "$t/m" || { echo "FAIL menu-rule 編譯失敗（menu.h 缺或介面不符）"; exit 1; }
"$t/m"

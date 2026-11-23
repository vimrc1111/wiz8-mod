#ifndef WIZ8_MOD_MENU_H
#define WIZ8_MOD_MENU_H
#include <stddef.h>
enum { MENU_MIN = 10, MENU_MAX = 5000, MENU_STEP = 25, MENU_ORIG = 100 };
#ifndef MENU_APPLY
#define MENU_APPLY(name) NULL
#endif
typedef struct {
    const char *key; int def; const wchar_t *label;
    int min, max, step;
    int (*apply)(int);
} MenuItem;
static const MenuItem menu_items[] = {
    {"monster_move_pct", 400, L"怪物移動", MENU_MIN, MENU_MAX, MENU_STEP, MENU_APPLY(move)},
    {"event_speed_pct", 300, L"事件停頓", MENU_MIN, MENU_MAX, MENU_STEP, MENU_APPLY(event)},
    {"anim_speed_pct", 200, L"攻擊動畫", MENU_MIN, MENU_MAX, MENU_STEP, MENU_APPLY(anim)},
    {"party_move_pct", 100, L"隊伍移動", MENU_MIN, MENU_MAX, MENU_STEP, MENU_APPLY(party)}
};
#undef MENU_APPLY
enum { MENU_N = sizeof menu_items / sizeof *menu_items };
typedef struct { int open, sel, dirty; int val[MENU_N]; } MenuState;
enum { MK_OTHER, MK_TOGGLE, MK_UP, MK_DOWN, MK_LEFT, MK_RIGHT,
       MK_RESET, MK_RESET_ALL, MK_ESC };
static inline void menu_set(MenuState *s, int i, int v) {
    if (s->val[i] != v) { s->val[i] = v; s->dirty = 1; }
}
static inline int menu_key(MenuState *s, int key) {
    if (!s->open) {
        if (key != MK_TOGGLE) return 0;
        s->open = 1; s->dirty = 0; return 1;
    }
    if (key == MK_TOGGLE || key == MK_ESC) s->open = 0;
    else if (key == MK_UP && s->sel > 0) --s->sel;
    else if (key == MK_DOWN && s->sel < MENU_N - 1) ++s->sel;
    else if (key == MK_LEFT || key == MK_RIGHT) {
        /* Widen before adding: even a malformed ini INT_MAX must not overflow. */
        const MenuItem *item = &menu_items[s->sel];
        long long v = (long long)s->val[s->sel] + (key == MK_LEFT ? -item->step : item->step);
        menu_set(s, s->sel, v < item->min ? item->min : v > item->max ? item->max : (int)v);
    } else if (key == MK_RESET) menu_set(s, s->sel, MENU_ORIG);
    else if (key == MK_RESET_ALL)
        for (int i = 0; i < MENU_N; ++i) menu_set(s, i, MENU_ORIG);
    return 1;
}
#endif

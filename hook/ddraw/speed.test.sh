#!/usr/bin/env bash
# speed.h 戰鬥中怪物移動倍率（stage9 T）：主機 gcc 逐條比對。
# 規則：非戰鬥回原值；戰鬥中 base*clamp(pct,10,5000)/100；base 非有限或 ≤0 回原值。
# delay-rule（stage9 U）：delay_scaled(ms,pct)＝ms*100/clamp(pct,10,5000)，整數向下取整，ms≤600000 不溢位。
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd); t=$(mktemp -d); trap 'rm -rf "$t"' EXIT
cat >"$t/m.c" <<'C'
#include <stdio.h>
#include <math.h>
#include "speed.h"
static const struct { float base; int pct, combat; float want; const char *why; } T[] = {
    {2.5f, 400, 0, 2.5f, "探索不動"},
    {2.5f, 400, 1, 10.0f, "戰鬥 4x"},
    {2.5f, 100, 1, 2.5f, "100＝原版"},
    {2.5f, 1, 1, 0.25f, "下限 10%"},
    {2.5f, 99999, 1, 125.0f, "上限 5000%"},
    {2.5f, -5, 1, 0.25f, "負值當下限"},
    {0.0f, 400, 1, 0.0f, "0 不放大"},
    {-1.0f, 400, 1, -1.0f, "負 base 原樣"},
    {1.5f, 400, 255, 6.0f, "旗標任意非 0"},
};
int main(void) {
    int bad = 0;
    for (unsigned i = 0; i < sizeof T / sizeof *T; i++) {
        float got = speed_eff(T[i].base, T[i].pct, T[i].combat);
        if (fabsf(got - T[i].want) > 1e-4f) { printf("FAIL speed-rule %s: got %g want %g\n", T[i].why, got, T[i].want); bad = 1; }
    }
    float nan = NAN, inf = INFINITY;
    if (!isnan(speed_eff(nan, 400, 1))) { puts("FAIL speed-rule NaN 不放大"); bad = 1; }
    if (!isinf(speed_eff(inf, 400, 1))) { puts("FAIL speed-rule Inf 原樣"); bad = 1; }
    if (!bad) puts("PASS speed-rule");
    return bad;
}
C
gcc -std=c99 -Wall -Werror -I"$here" "$t/m.c" -o "$t/m" -lm || { echo "FAIL speed-rule 編譯失敗（speed.h 缺或介面不符）"; exit 1; }
"$t/m"
cat >"$t/d.c" <<'C'
#include <stdio.h>
#include "speed.h"
static const struct { unsigned ms; int pct; unsigned want; const char *why; } D[] = {
    {800, 100, 800, "100＝原版"},
    {800, 300, 266, "3x 向下取整"},
    {1500, 300, 500, "結束停頓 3x"},
    {800, 1, 8000, "下限 10%"},
    {800, -5, 8000, "負值當下限"},
    {800, 99999, 16, "上限 5000%"},
    {0, 300, 0, "0 維持 0"},
    {600000, 10, 6000000, "上限 ms 不溢位"},
};
int main(void) {
    int bad = 0;
    for (unsigned i = 0; i < sizeof D / sizeof *D; i++) {
        unsigned got = delay_scaled(D[i].ms, D[i].pct);
        if (got != D[i].want) { printf("FAIL delay-rule %s: got %u want %u\n", D[i].why, got, D[i].want); bad = 1; }
    }
    if (!bad) puts("PASS delay-rule");
    return bad;
}
C
gcc -std=c99 -Wall -Werror -I"$here" "$t/d.c" -o "$t/d" || { echo "FAIL delay-rule 編譯失敗（delay_scaled 缺或介面不符）"; exit 1; }
"$t/d"
cat >"$t/a.c" <<'C'
#include <stdio.h>
#include <math.h>
#include "speed.h"
// 序列集合依 notes/re/combat-anim.md：分類器 0x5430c0 的 6（9..0xc,0xe..0x10）、7（0xd,0x11）與施法 0x19；4 是移動（已有 monster_move_pct），不疊加。
static const struct { float m; int pct, combat, seq; float want; const char *why; } A[] = {
    {1.0f, 200, 1, 0x09, 2.0f, "近戰 9 加速"},
    {1.0f, 200, 1, 0x10, 2.0f, "近戰 0x10 加速"},
    {1.0f, 200, 1, 0x0d, 2.0f, "遠程 0xd 加速"},
    {1.0f, 200, 1, 0x11, 2.0f, "遠程 0x11 加速"},
    {1.0f, 200, 1, 0x19, 2.0f, "施法 0x19 加速"},
    {0.5f, 200, 1, 0x0b, 1.0f, "保留原 m（半速分支）"},
    {1.0f, 200, 0, 0x09, 1.0f, "探索不動"},
    {1.0f, 200, 1, 0x04, 1.0f, "移動序列不疊加"},
    {1.0f, 200, 1, 0x08, 1.0f, "8 不在集合"},
    {1.0f, 200, 1, 0x12, 1.0f, "0x12 不在集合"},
    {1.0f, 200, 1, 0x00, 1.0f, "0 不在集合"},
    {1.0f, 200, 1, -1, 1.0f, "負序列不在集合"},
    {1.0f, 100, 1, 0x09, 1.0f, "100＝原版"},
    {1.0f, 1, 1, 0x09, 0.1f, "下限 10%"},
    {1.0f, 99999, 1, 0x09, 50.0f, "上限 5000%"},
};
int main(void) {
    int bad = 0;
    for (unsigned i = 0; i < sizeof A / sizeof *A; i++) {
        float got = anim_m(A[i].m, A[i].pct, A[i].combat, A[i].seq);
        if (fabsf(got - A[i].want) > 1e-4f) { printf("FAIL anim-rule %s: got %g want %g\n", A[i].why, got, A[i].want); bad = 1; }
    }
    if (!isnan(anim_m(NAN, 200, 1, 9))) { puts("FAIL anim-rule NaN 不放大"); bad = 1; }
    if (anim_m(-1.0f, 200, 1, 9) != -1.0f) { puts("FAIL anim-rule 負 m 原樣"); bad = 1; }
    if (!bad) puts("PASS anim-rule");
    return bad;
}
C
gcc -std=c99 -Wall -Werror -I"$here" "$t/a.c" -o "$t/a" -lm || { echo "FAIL anim-rule 編譯失敗（anim_m 缺或介面不符）"; exit 1; }
"$t/a"
# party-rule（stage13 13.2）：party_dt(dt,pct,in_combat)＝探索中 dt*clamp(pct,10,5000)/100；戰鬥、dt 非有限或 ≤0 回原 dt。
cat >"$t/p.c" <<'C'
#include <stdio.h>
#include <math.h>
#include "speed.h"
static const struct { float dt; int pct, combat; float want; const char *why; } P[] = {
    {0.02f, 200, 0, 0.04f, "探索 2x"},
    {0.02f, 100, 0, 0.02f, "100＝原版"},
    {0.02f, 200, 1, 0.02f, "戰鬥不動"},
    {0.02f, 200, 255, 0.02f, "旗標任意非 0 不動"},
    {0.02f, 1, 0, 0.002f, "下限 10%"},
    {0.02f, 99999, 0, 1.0f, "上限 5000%"},
    {0.02f, -5, 0, 0.002f, "負值當下限"},
    {0.0f, 200, 0, 0.0f, "0 原樣"},
    {-1.0f, 200, 0, -1.0f, "負 dt 原樣"},
};
int main(void) {
    int bad = 0;
    for (unsigned i = 0; i < sizeof P / sizeof *P; i++) {
        float got = party_dt(P[i].dt, P[i].pct, P[i].combat);
        if (fabsf(got - P[i].want) > 1e-6f) { printf("FAIL party-rule %s: got %g want %g\n", P[i].why, got, P[i].want); bad = 1; }
    }
    if (!isnan(party_dt(NAN, 200, 0))) { puts("FAIL party-rule NaN 原樣"); bad = 1; }
    if (!isinf(party_dt(INFINITY, 200, 0))) { puts("FAIL party-rule Inf 原樣"); bad = 1; }
    if (!bad) puts("PASS party-rule");
    return bad;
}
C
gcc -std=c99 -Wall -Werror -I"$here" "$t/p.c" -o "$t/p" -lm || { echo "FAIL party-rule 編譯失敗（party_dt 缺或介面不符）"; exit 1; }
"$t/p"

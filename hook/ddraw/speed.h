#ifndef WIZ8_MOD_SPEED_H
#define WIZ8_MOD_SPEED_H

// ms <= 600000: the largest numerator is 60000000, within unsigned i386.
static inline unsigned delay_scaled(unsigned ms, int pct) {
    if (pct < 10) pct = 10;
    if (pct > 5000) pct = 5000;
    return ms * 100u / (unsigned)pct;
}

static inline float speed_eff(float base, int pct, int in_combat) {
    if (!in_combat || !__builtin_isfinite(base) || base <= 0.0f) return base;
    if (pct < 10) pct = 10;
    if (pct > 5000) pct = 5000;
    return base * pct / 100.0f;
}

static inline float anim_m(float m, int pct, int in_combat, int seq) {
    if ((seq >= 0x09 && seq <= 0x11) || seq == 0x19)
        return speed_eff(m, pct, in_combat);
    return m;
}

static inline float party_dt(float dt, int pct, int in_combat) {
    if (in_combat || !__builtin_isfinite(dt) || dt <= 0.0f) return dt;
    if (pct < 10) pct = 10;
    if (pct > 5000) pct = 5000;
    return dt * pct / 100.0f;
}

#endif

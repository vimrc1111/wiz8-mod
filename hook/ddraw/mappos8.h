// BitAtlas 位置協定（bitatlas/DESIGN.md §三.2）的 wiz8 寫入端：何時寫、寫什麼。純函式、不帶 CRT，測試 mappos8.test.sh。
// 取樣 ≤4 Hz；位置變動 >1 世界單位或換圖才寫，靜止 10 s 心跳；換圖要連兩次同 level（丟掉換圖途中的半套樣本）；寫失敗下次重試。
// theta：呼叫端傳 mp8_theta(yaw)，非有限＝不知道寫 null；原地轉向 >0.05 rad 也算變動。layer -1。
struct mp8 { int map, pend, chk, retry; unsigned seq, last_chk, last_wr; float x, y, z, t; };
#define MP8_NONE (-2) // 還沒寫過／沒有 pending
static void mp8_init(struct mp8 *s) { *s = (struct mp8){ MP8_NONE, MP8_NONE }; s->t = __builtin_nanf(""); }
static float mp8_abs(float v) { return v < 0 ? -v : v; }
static int mp8_ok(float v) { return __builtin_isfinite(v) && mp8_abs(v) < 1e9f; }
#define MP8_PI 3.14159265f
// 引擎 yaw（f32[[0x65a0f8]+4]，弧度、0x477cd9 環繞到 [0,2π)）→ BitAtlas theta（0＝+Z、π/2＝+X）。sign +1、零點 0：MG 2026-10-31 實玩 log 擬合（mappos8.test.sh「實機擬合」段）。
#ifndef MP8_YAW_SIGN
#define MP8_YAW_SIGN 1
#endif
#ifndef MP8_YAW_ZERO
#define MP8_YAW_ZERO 0.0f
#endif
static float mp8_wrap(float a) { while (a >= 2 * MP8_PI) a -= 2 * MP8_PI; while (a < 0) a += 2 * MP8_PI; return a; }
static float mp8_theta(float yaw) { return mp8_ok(yaw) ? mp8_wrap(MP8_YAW_SIGN * yaw + MP8_YAW_ZERO) : __builtin_nanf(""); }
static float mp8_tdiff(float a, float b) { float d = mp8_wrap(a - b); return d > MP8_PI ? 2 * MP8_PI - d : d; }

// level＝u32[0x686a70]（0xffffffff＝無圖）；valid＝指標可讀。回 1＝現在寫，s->map／s->seq 為要寫的值。
static int mp8_sample(struct mp8 *s, unsigned now, unsigned level, int valid, float x, float y, float z, float t) {
    if (!valid || !mp8_ok(x) || !mp8_ok(y) || !mp8_ok(z)) return 0;
    if (s->chk && now - s->last_chk < 250) return 0; // 無號相減：GetTickCount 回捲照算
    s->chk = 1, s->last_chk = now;
    int map = level == 0xffffffffu ? -1 : (int)level, changed = 0;
    if (map != s->map) {
        if (s->pend != map) { s->pend = map; return 0; }
        changed = 1;
    }
    s->pend = MP8_NONE;
    if (mp8_abs(x - s->x) > 1 || mp8_abs(y - s->y) > 1 || mp8_abs(z - s->z) > 1) changed = 1;
    if (__builtin_isfinite(t) != __builtin_isfinite(s->t) || (__builtin_isfinite(t) && mp8_tdiff(t, s->t) > 0.05f)) changed = 1;
    if (!changed && !s->retry && now - s->last_wr < 10000) return 0;
    s->map = map, s->x = x, s->y = y, s->z = z, s->t = t, s->last_wr = now, s->retry = 0, s->seq++;
    return 1;
}
static void mp8_wrote(struct mp8 *s, int ok) { if (!ok) s->retry = 1; }

// 有界附加；失敗回 -1（之後每步都傳遞 -1）
static int mp8_put(char *o, int cap, int n, const char *t) {
    if (n < 0) return -1;
    for (; *t; t++) { if (n >= cap - 1) return -1; o[n++] = *t; }
    o[n] = 0;
    return n;
}
static int mp8_int(char *o, int cap, int n, long v) {
    char b[24], *p = b + sizeof b - 1;
    unsigned long u = v < 0 ? 0ul - (unsigned long)v : (unsigned long)v;
    *p = 0;
    do *--p = (char)('0' + u % 10); while (u /= 10);
    if (v < 0) *--p = '-';
    return mp8_put(o, cap, n, p);
}
static long mp8_round(float v, long k) { double d = (double)v * k; return (long)(d < 0 ? d - 0.5 : d + 0.5); } // 遠離 0
static int mp8_fix2(char *o, int cap, int n, float v) { // 兩位小數；wsprintfA 不支援 %f
    long c = mp8_round(v, 100);
    if (c < 0) n = mp8_put(o, cap, n, "-"), c = -c;
    n = mp8_int(o, cap, n, c / 100);
    n = mp8_put(o, cap, n, c % 100 < 10 ? ".0" : ".");
    return mp8_int(o, cap, n, c % 100);
}

// MAPPOS.JS 內容：mapPos（整數 X/Z、dir -1）＋mapPose（同 seq）。回長度；非有限、|v|≥1e9、cap 不夠回 -1。
static int mp8_format(char *o, int cap, int map, float x, float y, float z, float t, unsigned seq) {
    if (cap < 1 || !mp8_ok(x) || !mp8_ok(y) || !mp8_ok(z)) return -1;
    int n = mp8_put(o, cap, 0, "mapPos(1,");
    n = mp8_int(o, cap, n, map); n = mp8_put(o, cap, n, ",");
    n = mp8_int(o, cap, n, mp8_round(x, 1)); n = mp8_put(o, cap, n, ",");
    n = mp8_int(o, cap, n, mp8_round(z, 1)); n = mp8_put(o, cap, n, ",-1,");
    n = mp8_int(o, cap, n, seq); n = mp8_put(o, cap, n, ");\n");
#ifndef INJ_NOPOSE
    n = mp8_put(o, cap, n, "mapPose(1,");
    n = mp8_int(o, cap, n, map); n = mp8_put(o, cap, n, ",");
    n = mp8_fix2(o, cap, n, x); n = mp8_put(o, cap, n, ",");
    n = mp8_fix2(o, cap, n, z); n = mp8_put(o, cap, n, ",");
    n = mp8_fix2(o, cap, n, y); n = mp8_put(o, cap, n, ",");
    n = __builtin_isfinite(t) ? mp8_fix2(o, cap, n, t) : mp8_put(o, cap, n, "null");
    n = mp8_put(o, cap, n, ",-1,");
    n = mp8_int(o, cap, n, seq); n = mp8_put(o, cap, n, ");\n");
#endif
    return n;
}

// W4 探索位元：引擎 bit 陣列（u32 低位先＝byte 低位先）取 ceil(nbits/8) byte 原樣 base64，附成 mapSeen(1,map,"…")
static int mp8_seen(char *o, int cap, int n, int map, const unsigned char *bits, unsigned nbits) {
    static const char b64[] = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    if (n < 0) return -1;
    if (map < 0 || !nbits) return n;
    unsigned len = (nbits + 7) / 8;
    n = mp8_put(o, cap, n, "mapSeen(1,");
    n = mp8_int(o, cap, n, map); n = mp8_put(o, cap, n, ",\"");
    for (unsigned i = 0; i < len && n >= 0; i += 3) {
        unsigned v = 0, k = len - i < 3 ? len - i : 3;
        for (unsigned j = 0; j < 3; j++) v = v << 8 | (j < k ? bits[i + j] : 0);
#ifdef INJ_SEENSHIFT
        v <<= 1;
#endif
        char q[5] = { b64[v >> 18 & 63], b64[v >> 12 & 63], k > 1 ? b64[v >> 6 & 63] : '=', k > 2 ? b64[v & 63] : '=', 0 };
        n = mp8_put(o, cap, n, q);
    }
    return mp8_put(o, cap, n, "\");\n");
}

// x-ray 傾印（X1 執行期驗證）：len bytes 寫成小寫 hex 附加在 o[n]；cap 不夠回 -1。
static int mp8_hex(char *o, int cap, int n, const unsigned char *p, unsigned len) {
    static const char h[] = "0123456789abcdef";
    if (n < 0 || (long)n + 2L * len >= cap) return -1;
    for (unsigned i = 0; i < len; i++) {
#ifdef INJ_HEXNIB
        o[n++] = h[p[i] & 15], o[n++] = h[p[i] >> 4];
#else
        o[n++] = h[p[i] >> 4], o[n++] = h[p[i] & 15];
#endif
    }
    o[n] = 0;
    return n;
}

// X2 x-ray：mapMobs(1,map,[[X,Z,tpl,flags],...])（bitatlas/DESIGN.md §三.2）。X／Z 量化到 0.5（站著時抖動不觸發重寫）；非有限或 |v|≥1e9 的項略過；map<0 不寫。
struct mp8_mob { float x, z; int tpl, fl; };
static int mp8_mobs(char *o, int cap, int n, int map, const struct mp8_mob *m, unsigned cnt) {
    if (n < 0) return -1;
    if (map < 0) return n;
    n = mp8_put(o, cap, n, "mapMobs(1,");
    n = mp8_int(o, cap, n, map); n = mp8_put(o, cap, n, ",[");
    int first = 1;
    for (unsigned i = 0; i < cnt && n >= 0; i++) {
        if (!mp8_ok(m[i].x) || !mp8_ok(m[i].z)) continue;
        n = mp8_put(o, cap, n, first ? "[" : ",["), first = 0;
#ifdef INJ_MOBSWAP
        n = mp8_fix2(o, cap, n, mp8_round(m[i].z, 2) * 0.5f); n = mp8_put(o, cap, n, ",");
        n = mp8_fix2(o, cap, n, mp8_round(m[i].x, 2) * 0.5f); n = mp8_put(o, cap, n, ",");
#else
        n = mp8_fix2(o, cap, n, mp8_round(m[i].x, 2) * 0.5f); n = mp8_put(o, cap, n, ",");
        n = mp8_fix2(o, cap, n, mp8_round(m[i].z, 2) * 0.5f); n = mp8_put(o, cap, n, ",");
#endif
        n = mp8_int(o, cap, n, m[i].tpl); n = mp8_put(o, cap, n, ",");
        n = mp8_int(o, cap, n, m[i].fl & 7); n = mp8_put(o, cap, n, "]");
    }
    return mp8_put(o, cap, n, "]);\n");
}
// 內容雜湊（FNV-1a）：呼叫端比對上次寫出的 mapMobs 行，變了就 mp8_wrote(…, 0) 強制下次取樣寫
static unsigned mp8_fnv(const char *p, int n) { unsigned h = 2166136261u; for (int i = 0; i < n; i++) h = (h ^ (unsigned char)p[i]) * 16777619u; return h; }

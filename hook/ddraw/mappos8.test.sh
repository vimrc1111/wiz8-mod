#!/usr/bin/env bash
# mappos8.h（GOAL-wiz8map W3）：ddraw hook 寫 MAPPOS.JS 的純函式，主機 gcc 逐條比對。
# mappos-rule：取樣節流 250 ms、靜止 10 s 心跳、換圖連兩次同 level 才寫、NaN／Inf／指標無效不寫、寫失敗下次重試；
#              輸出 mapPos(1,map,round X,round Z,-1,seq)＋mapPose(1,map,X,Z,Y,null,-1,seq) 同 seq，數字能被 bitatlas/mappos.ts 解析。
# seen-rule（W4）：mp8_seen 把引擎探索位元（B+0x18，u32 低位先＝byte 低位先）原樣 base64 成 mapSeen(1,map,"…")，bit 數取上整 byte；map<0 或 nbits 0 不寫。
# hex-rule（x-ray）：mp8_hex 照記憶體順序、高半位元在前。
# mobs-rule（X2）：mp8_mobs 寫 mapMobs(1,map,[[X,Z,tpl,flags]...])，X/Z 量化 0.5、flags 只留 bit0–2（戰鬥＋敵我）、非有限項略過、map<0 不寫；mp8_fnv 內容變才變。
# 注入：MP8_CFLAGS=-DINJ_NOPOSE（mappos8.h 不寫 mapPose）須紅；-DINJ_SEENSHIFT（位元錯一位）須紅；-DINJ_HEXNIB（半位元對調）須紅；-DINJ_MOBSWAP（X/Z 對調）須紅。
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd); t=$(mktemp -d); trap 'rm -rf "${t:?}"' EXIT
cat >"$t/m.c" <<'C'
#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include <math.h>
#include <regex.h>
#include "mappos8.h"
static int bad;
#define CHECK(c, why) do { if (!(c)) { printf("FAIL mappos-rule %s (line %d)\n", why, __LINE__); bad = 1; } } while (0)
static const char *POS = "^mapPos\\(1,(-?[0-9]+),(-?[0-9]+),(-?[0-9]+),(-?[0-9]+),([0-9]+)\\);$";
#define NUM "(-?[0-9]+(\\.[0-9]+)?(e[-+]?[0-9]+)?)"
static const char *POSE = "^mapPose\\(1,(-?[0-9]+)," NUM "," NUM "," NUM ",(null|" NUM "),(-?[0-9]+),([0-9]+)\\);$";
// 從 out 抓出第一行符合 re 的，抄進 line；找不到回 0
static int line_of(const char *out, const char *re, char *line) {
    regex_t r; regcomp(&r, re, REG_EXTENDED | REG_NOSUB);
    const char *p = out; int hit = 0;
    while (*p) {
        const char *e = strchr(p, '\n'); size_t n = e ? (size_t)(e - p) : strlen(p);
        char buf[256]; if (n >= sizeof buf) n = sizeof buf - 1;
        memcpy(buf, p, n); buf[n] = 0; if (n && buf[n - 1] == '\r') buf[n - 1] = 0;
        if (!regexec(&r, buf, 0, NULL, 0)) { strcpy(line, buf); hit = 1; break; }
        if (!e) break;
        p = e + 1;
    }
    regfree(&r); return hit;
}
static double field(const char *line, int k) { // k：括號內第 k 個逗號欄位（0 起，0＝版本）
    const char *p = strchr(line, '(') + 1;
    for (int i = 0; i < k; i++) p = strchr(p, ',') + 1;
    return strncmp(p, "null", 4) ? strtod(p, NULL) : NAN;
}
int main(void) {
    struct mp8 s; mp8_init(&s);
    // 換圖丟半套：第一個樣本只記 pending
    CHECK(mp8_sample(&s, 0, 3, 1, 10, 20, 30, NAN) == 0, "第一個樣本不寫（要連兩次同 level）");
    CHECK(mp8_sample(&s, 100, 3, 1, 10, 20, 30, NAN) == 0, "≤4 Hz：250 ms 內不取樣");
    CHECK(mp8_sample(&s, 300, 3, 1, 10, 20, 30, NAN) == 1, "連兩次同 level 才寫");
    CHECK(s.map == 3 && s.seq == 1, "寫入時 map＝level、seq 從 1 起");
    mp8_wrote(&s, 1);
    CHECK(mp8_sample(&s, 600, 3, 1, 10, 20, 30, NAN) == 0, "位置不變不寫");
    CHECK(mp8_sample(&s, 900, 3, 1, 60, 20, 30, NAN) == 1 && s.seq == 2, "移動就寫、seq+1");
    mp8_wrote(&s, 1);
    // 無效樣本
    CHECK(mp8_sample(&s, 1200, 3, 1, NAN, 20, 30, NAN) == 0, "NaN 不寫");
    CHECK(mp8_sample(&s, 1500, 3, 1, 60, INFINITY, 30, NAN) == 0, "Inf 不寫");
    CHECK(mp8_sample(&s, 1800, 3, 0, 99, 20, 30, NAN) == 0, "指標無效不寫");
    CHECK(s.seq == 2 && s.map == 3, "無效樣本不改狀態");
    // 換圖
    CHECK(mp8_sample(&s, 2100, 5, 1, 1, 2, 3, NAN) == 0, "換圖第一個樣本不寫");
    CHECK(mp8_sample(&s, 2400, 5, 1, 1, 2, 3, NAN) == 1 && s.map == 5 && s.seq == 3, "換圖第二個樣本寫");
    mp8_wrote(&s, 1);
    // 抖動：6、5、6、6 → 只有最後一次寫
    CHECK(mp8_sample(&s, 2700, 6, 1, 1, 2, 3, NAN) == 0, "抖動 6 pending");
    CHECK(mp8_sample(&s, 3000, 5, 1, 1, 2, 3, NAN) == 0, "回到 5 不寫且清 pending");
    CHECK(mp8_sample(&s, 3300, 6, 1, 1, 2, 3, NAN) == 0, "6 重新 pending");
    CHECK(mp8_sample(&s, 3600, 6, 1, 1, 2, 3, NAN) == 1 && s.map == 6, "6 連兩次才寫");
    mp8_wrote(&s, 1);
    // 心跳
    CHECK(mp8_sample(&s, 13300, 6, 1, 1, 2, 3, NAN) == 0, "靜止未滿 10 s 不寫");
    CHECK(mp8_sample(&s, 13600, 6, 1, 1, 2, 3, NAN) == 1, "靜止滿 10 s 心跳");
    unsigned hb = s.seq; mp8_wrote(&s, 0);
    // 寫失敗重試
    CHECK(mp8_sample(&s, 13900, 6, 1, 1, 2, 3, NAN) == 1 && s.seq > hb, "寫失敗下次取樣重試");
    mp8_wrote(&s, 1);
    CHECK(mp8_sample(&s, 14200, 6, 1, 1, 2, 3, NAN) == 0, "重試成功後恢復不寫");
    // 無圖
    CHECK(mp8_sample(&s, 14500, 0xffffffffu, 1, 1, 2, 3, NAN) == 0, "無圖第一個樣本 pending");
    CHECK(mp8_sample(&s, 14800, 0xffffffffu, 1, 1, 2, 3, NAN) == 1 && s.map == -1, "0xffffffff 寫 map -1");
    mp8_wrote(&s, 1);
    // 時鐘回捲（GetTickCount 49.7 天）
    struct mp8 w; mp8_init(&w);
    CHECK(mp8_sample(&w, 0xffffff00u, 2, 1, 0, 0, 0, NAN) == 0 && mp8_sample(&w, 0x00000040u, 2, 1, 0, 0, 0, NAN) == 1, "GetTickCount 回捲仍取樣");

    // 格式
    char out[256], ln[256];
    int n = mp8_format(out, sizeof out, 3, -1234.567f, 12.5f, 0.004f, NAN, 7);
    CHECK(n > 0 && n == (int)strlen(out), "format 回傳長度");
    CHECK(line_of(out, POS, ln) && field(ln, 1) == 3 && field(ln, 2) == -1235 && field(ln, 3) == 0 && field(ln, 4) == -1 && field(ln, 5) == 7,
          "mapPos(1,map,round X,round Z,-1,seq)");
    CHECK(line_of(out, POSE, ln), "mapPose 行符合 bitatlas/mappos.ts 文法");
    if (line_of(out, POSE, ln)) {
        CHECK(field(ln, 1) == 3 && fabs(field(ln, 2) + 1234.567) < 0.006 && fabs(field(ln, 3) - 0.004) < 0.006 && fabs(field(ln, 4) - 12.5) < 0.006,
              "mapPose X,Z,Y 數值（誤差 <0.006）");
        CHECK(isnan(field(ln, 5)) && field(ln, 6) == -1 && field(ln, 7) == 7, "mapPose theta null、layer -1、同 seq");
    }
    n = mp8_format(out, sizeof out, 0, 2.5f, -0.75f, -2.5f, NAN, 8);
    CHECK(line_of(out, POS, ln) && field(ln, 2) == 3 && field(ln, 3) == -3, "四捨五入遠離 0（2.5→3、-2.5→-3）");
    CHECK(line_of(out, POSE, ln) && fabs(field(ln, 4) + 0.75) < 0.006, "-0.75 帶負號");
    n = mp8_format(out, sizeof out, -1, 0, 0, 0, NAN, 9);
    CHECK(line_of(out, POS, ln) && field(ln, 1) == -1 && line_of(out, POSE, ln), "map -1 兩行都寫");
    CHECK(mp8_format(out, sizeof out, 3, NAN, 0, 0, NAN, 1) < 0, "format NaN 回 -1");
    CHECK(mp8_format(out, sizeof out, 3, 1e10f, 0, 0, NAN, 1) < 0, "format |v|≥1e9 回 -1");
    char small[24]; memset(small, 'Z', sizeof small);
    CHECK(mp8_format(small, 16, 3, 1, 2, 3, NAN, 1) < 0 && small[16] == 'Z', "cap 不夠回 -1 且不越界");
    // theta（朝向）：有值寫兩位小數；原地轉 >0.05 rad 要寫、0.03 不寫；跨 0/2π 算短弧
    n = mp8_format(out, sizeof out, 3, 1, 2, 3, 1.5f, 10);
    CHECK(line_of(out, POSE, ln) && fabs(field(ln, 5) - 1.5) < 0.006, "mapPose theta 有值");
    struct mp8 h; mp8_init(&h);
    mp8_sample(&h, 0, 2, 1, 0, 0, 0, 6.27f); mp8_sample(&h, 300, 2, 1, 0, 0, 0, 6.27f); mp8_wrote(&h, 1);
    CHECK(mp8_sample(&h, 600, 2, 1, 0, 0, 0, 0.01f) == 0, "跨 2π 轉 0.023 rad 不寫");
    CHECK(mp8_sample(&h, 900, 2, 1, 0, 0, 0, 6.17f) == 1 && h.t == 6.17f, "原地轉 0.1 rad 要寫");
    mp8_wrote(&h, 1);
    CHECK(mp8_sample(&h, 1200, 2, 1, 0, 0, 0, NAN) == 1, "theta 從有值變不知道要寫");
    // 實機擬合（MG 2026-10-31 log 37 段直走：sign +1、零點 -0.03≈0，R 0.999）：yaw → 移動方位 atan2(dX,dZ)
    { const float yaw[3] = { 1.130f, 3.630f, 4.580f }, dir[3] = { 1.004f, 3.609f, 4.585f };
      for (int i = 0; i < 3; i++) CHECK(fabsf(mp8_tdiff(mp8_theta(yaw[i]), dir[i])) < 0.15f, "實機 yaw→theta 對上移動方位"); }
    CHECK(isnan(mp8_theta(NAN)) && mp8_theta(-7) >= 0 && mp8_theta(100) < 6.2832f, "mp8_theta 非有限→NaN、輸出在 [0,2π)");
    // 探索位元：bit0、bit2、bit15 → bytes 05 80 → "BYA="；9 bit → 2 byte；24 bit → 無補位
    unsigned char bits[4] = { 0x05, 0x80, 0xff, 0x01 };
    n = mp8_seen(out, sizeof out, 0, 3, bits, 16);
    CHECK(n > 0 && !strcmp(out, "mapSeen(1,3,\"BYA=\");\n"), "mapSeen 16 bit");
    n = mp8_seen(out, sizeof out, 0, 12, bits, 9);
    CHECK(n > 0 && !strcmp(out, "mapSeen(1,12,\"BYA=\");\n"), "9 bit 取 2 byte");
    n = mp8_seen(out, sizeof out, 0, 0, bits, 24);
    CHECK(n > 0 && !strcmp(out, "mapSeen(1,0,\"BYD/\");\n"), "24 bit 無補位");
    n = mp8_seen(out, sizeof out, 5, 1, bits, 32);
    CHECK(n == 5 + (int)strlen("mapSeen(1,1,\"BYD/AQ==\");\n") && !strcmp(out + 5, "mapSeen(1,1,\"BYD/AQ==\");\n"), "接在既有內容後、32 bit");
    CHECK(mp8_seen(out, sizeof out, 0, -1, bits, 16) == 0 && mp8_seen(out, sizeof out, 0, 3, bits, 0) == 0, "map -1 或 0 bit 不寫（回原長）");
    memset(small, 'Z', sizeof small);
    CHECK(mp8_seen(small, 16, 0, 3, bits, 32) < 0 && small[16] == 'Z', "seen cap 不夠回 -1 且不越界");
    // x-ray hex：位元組順序照記憶體、高半位元在前；接在既有內容後；cap 不夠回 -1 不越界
    const unsigned char hx[3] = { 0x0f, 0xa0, 0x5c };
    n = mp8_hex(out, sizeof out, mp8_put(out, sizeof out, 0, "E="), hx, 3);
    CHECK(n == 8 && !strcmp(out, "E=0fa05c"), "mp8_hex 0f a0 5c");
    memset(small, 'Z', sizeof small);
    CHECK(mp8_hex(small, 6, 0, hx, 3) < 0 && small[6] == 'Z' && mp8_hex(small, 7, 0, hx, 3) == 6, "hex cap 不夠回 -1 且不越界");
    // x-ray 怪物：X/Z 對應、量化 0.5（12.3→12.5、-0.2→0）、flags 只留 bit0–2（戰鬥＋敵我）、NaN 項略過、空清單照寫、map<0 不寫、cap 不夠不越界
    { struct mp8_mob mb[3] = { { 12.3f, -7.74f, 501, 11 }, { NAN, 1, 9, 0 }, { -0.2f, 100, 77, 0 } };
      n = mp8_mobs(out, sizeof out, 0, 4, mb, 3);
      CHECK(n > 0 && !strcmp(out, "mapMobs(1,4,[[12.50,-7.50,501,3],[0.00,100.00,77,0]]);\n"), "mapMobs 格式");
      unsigned h1 = mp8_fnv(out, n); mb[0].x = 12.4f;
      CHECK(mp8_fnv(out, mp8_mobs(out, sizeof out, 0, 4, mb, 3)) == h1, "量化內抖動雜湊不變");
      mb[0].x = 13.0f;
      CHECK(mp8_fnv(out, mp8_mobs(out, sizeof out, 0, 4, mb, 3)) != h1, "移動 0.5 以上雜湊變");
      n = mp8_mobs(out, sizeof out, 0, 4, mb, 0);
      CHECK(n > 0 && !strcmp(out, "mapMobs(1,4,[]);\n"), "空清單照寫（整行取代＝該圖沒怪）");
      CHECK(mp8_mobs(out, sizeof out, 3, -1, mb, 3) == 3, "map<0 不寫（回原長）");
      memset(small, 'Z', sizeof small);
      CHECK(mp8_mobs(small, 16, 0, 4, mb, 3) < 0 && small[16] == 'Z', "mobs cap 不夠回 -1 且不越界"); }
    if (!bad) puts("PASS mappos-rule");
    return bad;
}
C
gcc -std=c99 -Wall -Werror -Wno-unused-function ${MP8_CFLAGS:-} -I"$here" "$t/m.c" -o "$t/m" -lm || { echo "FAIL mappos-rule 編譯失敗（mappos8.h 缺或介面不符）"; exit 1; }
"$t/m"
# 不帶 CRT：header 以 -ffreestanding -nostdlib 連得起來（DLL 只准 KERNEL32／USER32／GDI32）
cat >"$t/f.c" <<'C'
#include "mappos8.h"
int __attribute__((stdcall)) DllMainCRTStartup(void *a, unsigned b, void *c) { (void)a; (void)b; (void)c;
    static struct mp8 s; static char o[256]; mp8_init(&s);
    if (mp8_sample(&s, 0, 1, 1, 1, 2, 3, mp8_theta(1))) mp8_wrote(&s, 1);
    static struct mp8_mob mb[1]; mp8_fnv(o, mp8_mobs(o, sizeof o, 0, 1, mb, 1));
    return mp8_hex(o, sizeof o, mp8_seen(o, sizeof o, mp8_format(o, sizeof o, 1, 1, 2, 3, s.t, 1), 1, (const unsigned char *)o, 8), (const unsigned char *)o, 4); }
C
i686-w64-mingw32-gcc -std=c99 -O2 -Wall -Werror -Wno-unused-function -ffreestanding -fno-tree-loop-distribute-patterns -I"$here" -shared -nostdlib \
  -Wl,-e,_DllMainCRTStartup@12 "$t/f.c" -o "$t/f.dll" || { echo "FAIL mappos-rule mappos8.h 需要 CRT（不准）"; exit 1; }
echo "PASS mappos-freestanding"

// wrap.h：中文斷行規則（stage5 5.3），proxy.c 與 wrap.test.sh 共用；不准用 libc。規格見 tickets/F-wrap.md。
#pragma once
static inline int wrap_space(unsigned short c) {
    return c == 0x20 || (c >= 9 && c <= 13) || c == 0x85 || c == 0xa0 ||
        c == 0x1680 || (c >= 0x2000 && c <= 0x200a) || c == 0x2028 ||
        c == 0x2029 || c == 0x202f || c == 0x205f || c == 0x3000;
}
static inline int wrap_cjk(unsigned short c) {
    return (c >= 0x2e80 && c <= 0x9fff) || (c >= 0xac00 && c <= 0xd7af) ||
        (c >= 0xf900 && c <= 0xfaff) || (c >= 0xff00 && c <= 0xffef);
}
static inline int wrap_can_break(unsigned short prev, unsigned short next) {
    if (wrap_space(prev) || wrap_space(next)) return 1;
    if (!wrap_cjk(prev) && !wrap_cjk(next)) return 0;
    switch (next) {
    case 0x3001: case 0x3002: case 0xff0c: case 0xff0e:
    case 0xff01: case 0xff1f: case 0xff1a: case 0xff1b:
    case 0x3009: case 0x300b: case 0x300d: case 0x300f:
    case 0x3011: case 0x3015: case 0x3017: case 0x3019: case 0x301b:
    case 0x301e: case 0x301f: case 0xff09: case 0xff3d: case 0xff5d:
    case 0xff60: case 0xff61: case 0xff63: case 0xff64:
    case 0x2026: case 0x2025: case 0x2019: case 0x201d: return 0;
    }
    switch (prev) {
    case 0x3008: case 0x300a: case 0x300c: case 0x300e:
    case 0x3010: case 0x3014: case 0x3016: case 0x3018: case 0x301a:
    case 0x301d: case 0xff08: case 0xff3b: case 0xff5b:
    case 0xff5f: case 0xff62: case 0x2018: case 0x201c: return 0;
    }
    return 1;
}

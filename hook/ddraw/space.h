// UTF-16 中英間距；proxy.c 與 space.test.sh 共用，不使用 libc。
#pragma once

static inline int space_is_cjk(unsigned short c) {
    return (c >= 0x3040 && c <= 0x9fff) ||
        (c >= 0xac00 && c <= 0xd7af) || (c >= 0xf900 && c <= 0xfaff);
}

static inline int space_ascii_alnum(unsigned short c) {
    return (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') ||
        (c >= 'a' && c <= 'z');
}

static inline int space_cjk(unsigned short *dst, int cap, const unsigned short *src) {
    int n = 0;
    unsigned short prev = 0;
    if (cap <= 0) return -1;
    for (; *src; ++src) {
        unsigned short c = *src;
        if ((space_is_cjk(prev) && space_ascii_alnum(c)) ||
            (space_ascii_alnum(prev) && space_is_cjk(c))) {
            if (n >= cap - 1) return -1;
            dst[n++] = 0x20;
        }
        if (n >= cap - 1) return -1;
        dst[n++] = c;
        prev = c;
    }
    dst[n] = 0;
    return n;
}

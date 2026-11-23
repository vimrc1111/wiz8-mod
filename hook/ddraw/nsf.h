// NSF 對白的窄字往返；proxy.c 與 nsf.test.sh 共用，不使用 libc。
// 遊戲用 sprintf("%S") 把 NSF 的 UTF-16 轉成 char，再用 swprintf(L"%S") 轉回；C locale 遇 >0xFF 就停＝中文截斷（notes/re/nsf.md）。
// ≤0xFF 一 byte（同原版）；>0xFF 寫 UTF-8 2–3 byte。解碼只認「合法且值 >0xFF」的序列，其餘 byte 照原版一對一。
// ponytail: 原版窄字串若有 0xC4–0xEF 後接 0x80–0xBF 會被誤解；NSF 原文只有 0x09／0x0A／0x20–0x7E／0x92（R7），不會發生。
#pragma once

static inline int nsf_narrow_len(const unsigned short *s) {
    int n = 0;
    for (; *s; ++s) n += *s < 0x100 ? 1 : *s < 0x800 ? 2 : 3;
    return n;
}

static inline void nsf_narrow(unsigned char *d, const unsigned short *s) {
    for (; *s; ++s) {
        unsigned c = *s;
        if (c < 0x100) *d++ = c;
        else if (c < 0x800) { *d++ = 0xc0 | c >> 6; *d++ = 0x80 | (c & 0x3f); }
        else { *d++ = 0xe0 | c >> 12; *d++ = 0x80 | (c >> 6 & 0x3f); *d++ = 0x80 | (c & 0x3f); }
    }
    *d = 0;
}

// 在 s 起頭的多 byte 序列長度（2／3），不是就回 0；*v 得值
static inline int nsf_seq(const unsigned char *s, unsigned *v) {
    if (s[0] >= 0xc4 && s[0] <= 0xdf && (s[1] & 0xc0) == 0x80) {
        *v = (s[0] & 0x1f) << 6 | (s[1] & 0x3f);
        return 2;
    }
    if (s[0] >= 0xe0 && s[0] <= 0xef && (s[1] & 0xc0) == 0x80 && (s[2] & 0xc0) == 0x80) {
        *v = (s[0] & 0x0f) << 12 | (s[1] & 0x3f) << 6 | (s[2] & 0x3f);
        return *v >= 0x800 ? 3 : 0;
    }
    return 0;
}

static inline int nsf_has_seq(const unsigned char *s) {
    unsigned v;
    for (; *s; ++s) if (nsf_seq(s, &v)) return 1;
    return 0;
}

// 回寫出的 unit 數（不含 NUL）；不超過 byte 數＝原版 swprintf 會寫的長度
static inline int nsf_widen(unsigned short *d, const unsigned char *s) {
    int n = 0;
    while (*s) {
        unsigned v;
        int k = nsf_seq(s, &v);
        if (k) { d[n++] = v; s += k; }
        else d[n++] = *s++;
    }
    d[n] = 0;
    return n;
}

// 格式只有一個 %S、沒有其他轉換（L"%S"、L" \"%S\""）：前綴照抄、%S 用 nsf_widen、後綴照抄
static inline int nsf_widen_fmt(unsigned short *d, const unsigned short *f, const unsigned char *s) {
    int n = 0;
    for (; *f && !(f[0] == '%' && f[1] == 'S'); ++f) d[n++] = *f;
    if (*f) { n += nsf_widen(d + n, s); f += 2; }
    for (; *f; ++f) d[n++] = *f;
    d[n] = 0;
    return n;
}

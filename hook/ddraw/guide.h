// G4 guide rules: no OS or libc dependency.
#pragma once
#include "wrap.h"
typedef struct { int layer, page; } GuideState;
enum { GUIDE_OTHER, GUIDE_TOGGLE, GUIDE_ESC, GUIDE_UP, GUIDE_DOWN, GUIDE_LEFT, GUIDE_RIGHT };
static inline int guide_nav(GuideState *g, int key, int npages) {
    if (!g->layer) {
        if (key != GUIDE_TOGGLE) return 0;
        g->layer = 1; g->page = 0; return 1;
    }
    switch (key) {
    case GUIDE_TOGGLE: case GUIDE_ESC: g->layer = g->page = 0; return 1;
    case GUIDE_UP:
        if (g->layer > 1) { --g->layer; g->page = 0; }
        return 1;
    case GUIDE_DOWN:
        if (g->layer < 3) { ++g->layer; g->page = 0; }
        return 1;
    case GUIDE_LEFT: case GUIDE_RIGHT:
        if (npages < 1) npages = 1;
        g->page += key == GUIDE_RIGHT ? 1 : -1;
        if (g->page < 0) g->page = 0;
        if (g->page >= npages) g->page = npages - 1;
        return 1;
    default: return 0;
    }
}
static inline int guide_paginate(const int *step, int n, int avail, int *page_start, int max) {
    if (max < 1) return 0;
    int pages = 1, used = 0;
    page_start[0] = 0;
    for (int i = 0; i < n; ++i) {
        if (i > page_start[pages - 1] && (used > avail || step[i] > avail - used)) {
            if (pages == max) break;
            page_start[pages++] = i; used = 0;
        }
        used += step[i];
    }
    return pages;
}
static inline int guide_decimal(const char *s, int n) {
    int v = 0;
    if (!n) return -1;
    for (int i = 0; i < n; ++i) {
        if (s[i] < '0' || s[i] > '9' || v > (2147483647 - (s[i] - '0')) / 10) return -1;
        v = v * 10 + s[i] - '0';
    }
    return v;
}
static inline int guide_select(const char *tsv, int len, int map, int upto,
                              const char **start, int *length, int max) {
    int count = 0;
    for (int layer = 1; layer <= upto && layer <= 3; ++layer) {
        for (int pos = 0; pos < len && count < max;) {
            int begin = pos, tabs = 0, a = -1, b = -1;
            while (pos < len && tsv[pos] != '\n') {
                if (tsv[pos] == '\t') { if (!tabs) a = pos; else if (tabs == 1) b = pos; ++tabs; }
                ++pos;
            }
            int end = pos;
            if (pos < len) ++pos;
            if (end > begin && tsv[end - 1] == '\r') --end;
            if (end == begin || tsv[begin] == '#' || tabs != 2) continue;
            if (guide_decimal(tsv + begin, a - begin) != map || map < 0 ||
                guide_decimal(tsv + a + 1, b - a - 1) != layer) continue;
            if (start) start[count] = tsv + b + 1;
            if (length) length[count] = end - b - 1;
            ++count;
        }
    }
    return count;
}
static inline int guide_tag(const int *layer, int i, int first) {
    return layer[i] > 0 && (i == first || layer[i] != layer[i - 1]) ? layer[i] : 0;
}
// count == max means truncated: the end of start[max-1] is unknown, so the caller must not draw it.
static inline int guide_wrap(const unsigned short *s, int n, int (*adv)(unsigned short),
                            int width, int *start, int max) {
    int count = 0, pos = 0;
    while (pos < n && count < max) {
        while (pos < n && wrap_space(s[pos])) ++pos;
        if (pos == n) break;
        start[count++] = pos;
        int end = pos, pixels = 0, boundary = -1;
        while (end < n) {
            if (end > pos && wrap_can_break(s[end - 1], s[end])) boundary = end;
            int step = adv(s[end]);
            if (pixels + step > width) break;
            pixels += step;
            ++end;
        }
        if (end == n) break;
        pos = boundary > pos ? boundary : (end > pos ? end : pos + 1);
    }
    return count;
}

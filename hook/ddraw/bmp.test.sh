#!/usr/bin/env bash
# bmp.h 往返：主機 gcc 用 bmp.h 寫一張奇數寬（測補齊）的漸層 BMP，由 ffmpeg（獨立解碼器）解回 rgb24，逐 byte 比對期望。
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd); t=$(mktemp -d); trap 'rm -rf "$t"' EXIT
cat >"$t/m.c" <<'C'
#include <stdio.h>
#include "bmp.h"
int main(int c, char **v) {
    int w = 7, h = 3; unsigned char hd[54], row[64];
    unsigned size = bmp_header(hd, w, h), s = bmp_stride(w);
    if (size != 54 + s * h || s < 3u * w || s % 4) return 2;
    FILE *b = fopen(v[1], "wb"), *e = fopen(v[2], "wb");
    fwrite(hd, 1, 54, b);
    for (int y = h - 1; y >= 0; y--) {  // 檔案由下而上
        for (unsigned i = 0; i < s; i++) row[i] = 0;
        for (int x = 0; x < w; x++) { row[3 * x] = 200 - x; row[3 * x + 1] = 30 * y; row[3 * x + 2] = 10 * x + y; }
        fwrite(row, 1, s, b);
    }
    for (int y = 0; y < h; y++) for (int x = 0; x < w; x++) { unsigned char p[3] = {10 * x + y, 30 * y, 200 - x}; fwrite(p, 1, 3, e); }
    fclose(b); fclose(e); return c < 0;
}
C
gcc -Wall -Werror -I"$here" -o "$t/m" "$t/m.c"
"$t/m" "$t/a.bmp" "$t/want.rgb" || { echo "FAIL bmp-roundtrip：檔頭大小／stride 不對（rc=$?）"; exit 1; }
ffmpeg -v error -i "$t/a.bmp" -f rawvideo -pix_fmt rgb24 "$t/got.rgb" 2>"$t/err" || { echo "FAIL bmp-roundtrip：ffmpeg 解不開：$(head -1 "$t/err")"; exit 1; }
cmp -s "$t/want.rgb" "$t/got.rgb" || { echo "FAIL bmp-roundtrip：像素不符"; exit 1; }
echo "PASS bmp-roundtrip"

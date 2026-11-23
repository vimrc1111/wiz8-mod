// 24 位元 BMP 檔頭（截圖用，proxy.c 與 bmp.test.sh 共用同一份）。像素列由下而上、BGR、每列補到 4 的倍數。
#ifndef WIZ8_BMP_H
#define WIZ8_BMP_H
// 無效尺寸或 32 位元大小溢位回傳 0；不依賴結構 packing 或 libc。
static unsigned bmp_stride(int w) {
    if (w <= 0 || (unsigned)w > (0xffffffffu - 3u) / 3u) return 0;
    return ((unsigned)w * 3u + 3u) & ~3u;
}
static void bmp_u32(unsigned char *p, unsigned v) {
    for (unsigned i = 0; i < 4; i++) p[i] = (unsigned char)(v >> (8 * i));
}
// 寫 54 bytes 檔頭到 out，回傳整個檔案大小（檔頭＋bmp_stride(w)*h）
static unsigned bmp_header(unsigned char out[54], int w, int h) {
    unsigned stride = bmp_stride(w);
    if (!stride || h <= 0 || (unsigned)h > (0xffffffffu - 54u) / stride) return 0;
    unsigned pixels = stride * (unsigned)h;
    for (unsigned i = 0; i < 54; i++) out[i] = 0;
    out[0] = 'B'; out[1] = 'M';
    bmp_u32(out + 2, 54u + pixels);
    bmp_u32(out + 10, 54);
    bmp_u32(out + 14, 40);
    bmp_u32(out + 18, (unsigned)w);
    bmp_u32(out + 22, (unsigned)h);
    out[26] = 1; out[28] = 24;
    bmp_u32(out + 34, pixels);
    return 54u + pixels;
}
#endif

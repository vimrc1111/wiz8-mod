// ddraw.dll 代理：載入時記一行，把所有匯出轉給系統 ddraw；並對 Wiz8.exe 的繪字入口裝 inline detour 記字串（stage3 §3.3）。
// exports.h 由 build.sh 從系統 ddraw 的匯出表產生（X(名字, 序號)）。
// hooks.h 由 build.sh 從 notes/re/ 表格產生（含戰鬥移動 T、事件延遲 D、怪物動畫 A），簽章只有一份來源。
// atlas.h 由 tools/atlas 從 Fusion Pixel 12px zh-TW（全部字高，框 ≥ 12）與 Noto Sans TC（點陣缺字）產生（CJK 1-bit 字形；不含遊戲資料）。
#include <windows.h>
static int menu_apply_move(int pct), menu_apply_event(int pct), menu_apply_anim(int pct), menu_apply_party(int pct);
#define MENU_APPLY(name) menu_apply_##name
#include "menu.h"
#include "atlas.h"
#include "bmp.h"
#include "wrap.h"
#include "guide.h"
#include "space.h"
#include "nsf.h"
#include "speed.h"
#include "mappos8.h"

#define X(name, ord) void *real_##name;
#include "exports.h"
#undef X

// 裸跳板：不碰堆疊與暫存器，參數與呼叫慣例原封不動交給系統 ddraw
#define X(name, ord) \
    __attribute__((naked)) void stub_##name(void) { __asm__("jmp *%0" ::"m"(real_##name)); }
#include "exports.h"
#undef X

static void log_line(const char *msg) {
    char path[MAX_PATH + 32], exe[MAX_PATH], line[4 * MAX_PATH + 256];
    DWORD n = GetTempPathA(MAX_PATH, path);
    if (!n || n >= MAX_PATH) return;
    lstrcatA(path, "wiz8-mod-ddraw.log");
    exe[0] = 0;
    GetModuleFileNameA(NULL, exe, sizeof exe);
    exe[sizeof exe - 1] = 0;
    SYSTEMTIME t;
    GetLocalTime(&t);
    // wsprintfA 只處理前綴；訊息有 UTF-8 路徑，另以有界複製接上。
    int len = wsprintfA(line, "%04d-%02d-%02d %02d:%02d:%02d.%03d pid=%lu exe=%s ", t.wYear,
                        t.wMonth, t.wDay, t.wHour, t.wMinute, t.wSecond, t.wMilliseconds,
                        GetCurrentProcessId(), exe);
    lstrcpynA(line + len, msg, sizeof line - len - 2);
    len += lstrlenA(line + len);
    line[len++] = '\r'; line[len++] = '\n';
    HANDLE f = CreateFileA(path, FILE_APPEND_DATA, FILE_SHARE_READ | FILE_SHARE_WRITE, NULL, OPEN_ALWAYS,
                           FILE_ATTRIBUTE_NORMAL, NULL);
    if (f == INVALID_HANDLE_VALUE) return;
    DWORD w;
    WriteFile(f, line, len, &w, NULL);
    CloseHandle(f);
}

// ---- 繪字 detour ----
// 兩個入口（notes/re/draw.md）都是 cdecl (int x, int y, const wchar_t *fmt, ...)，開頭 6 bytes `sub esp,0x410` 無相對位址，
// 所以 detour＝5 byte jmp＋nop，跳板＝原 6 bytes＋jmp 回 +6。
#define TEXT_LOG_MAX 20000 // 只記前 N 條「不同」的字串（200 在 10 分鐘試玩裡就滿了，run 042619）
#define SEEN 32768         // 雜湊表（開放定址），要 > TEXT_LOG_MAX
static HANDLE text_log = INVALID_HANDLE_VALUE;
static volatile LONG text_n;
static DWORD seen[SEEN];

static DWORD fnv(const WCHAR *s, DWORD h) { for (; *s; s++) h = (h ^ *s) * 16777619u; return h; }

// 只有 wvsprintfW 與遊戲的 vswprintf 語意相同的格式才自己格式化（%s 都是寬字串）；其他（%f、*）只記原格式，避免讀錯參數。
static int simple_fmt(const WCHAR *f) {
    for (; *f; f++) {
        if (*f != '%') continue;
        f++;
        while (*f == '-' || *f == '0' || *f == '#' || *f == ' ' || *f == '+' || (*f >= '1' && *f <= '9') || *f == 'l') f++;
        if (!*f || !(*f == '%' || *f == 'd' || *f == 'i' || *f == 'u' || *f == 'x' || *f == 'X' || *f == 's' || *f == 'c')) return 0;
    }
    return 1;
}

static void put_utf8(char *dst, int cap, const WCHAR *w) {
    int n = WideCharToMultiByte(CP_UTF8, 0, w, -1, dst, cap, NULL, NULL);
    if (n <= 0) dst[0] = 0;
    for (char *p = dst; *p; p++) if (*p == '\r' || *p == '\n' || *p == '\t') *p = ' ';
}

static void text_seen_locked(int tag, int *args, const WCHAR *fmt, WCHAR *buf);
static void ext_chars(void);
__attribute__((used)) void __cdecl text_seen(int tag, int *args) {
    ext_chars();
    const WCHAR *fmt = (const WCHAR *)args[2];
    if (text_log == INVALID_HANDLE_VALUE || !fmt || text_n >= TEXT_LOG_MAX) return;
    // 緩衝區用 static（-nostdlib 下大於 4KB 的堆疊陣列要 __chkstk），以自旋鎖序列化；搶不到就不記
    static volatile LONG busy;
    if (InterlockedExchange(&busy, 1)) return;
    static WCHAR buf[1100];
    text_seen_locked(tag, args, fmt, buf);
    InterlockedExchange(&busy, 0);
}

static void text_seen_locked(int tag, int *args, const WCHAR *fmt, WCHAR *buf) {
    int ok = simple_fmt(fmt);
    if (ok) wvsprintfW(buf, fmt, (va_list)&args[3]);  // 上限 1024 字元
    else lstrcpynW(buf, fmt, 1024);
    DWORD h = fnv(buf, fnv(fmt, 2166136261u) + tag);
    if (!h) h = 1;
    for (DWORD i = h % SEEN;; i = (i + 1) % SEEN) {
        if (seen[i] == h) return;
        if (!seen[i]) { seen[i] = h; break; }
    }
    LONG k = InterlockedIncrement(&text_n);
    if (k > TEXT_LOG_MAX) return;
    static char line[3 * 1100 * 2 + 128];
    static char f8[3 * 1024 + 4], t8[3 * 1100 + 4];
    put_utf8(f8, sizeof f8, fmt);
    put_utf8(t8, sizeof t8, buf);
    // wsprintfA 上限 1024 字元，字串欄自己接
    int n = wsprintfA(line, "%ld\t%c\t%d\t%d\t%s\t", k, tag, args[0], args[1], ok ? "fmt" : "raw");
    lstrcpyA(line + n, t8); n += lstrlenA(t8);
    line[n++] = '\t'; lstrcpyA(line + n, f8); n += lstrlenA(f8);
    static DWORD t0;
    if (!t0) t0 = GetTickCount();
    n += wsprintfA(line + n, "\t%d\t%lu\t%08x", *(int *)0x005ff5f0, GetTickCount() - t0, args[-1]);  // 當前字型槽（draw.md DAT_005ff5f0）；第一筆起的毫秒（對 MG 截圖時間）；呼叫端返回位址
    line[n++] = '\r'; line[n++] = '\n';
    DWORD w;
    WriteFile(text_log, line, n, &w, NULL);
}

#define H(tag, addr, sig) static void *tramp_##tag;
#define F(tag, addr, sig) static void *tramp_##tag;
#define P(tag, addr, sig)
#define W(tag, addr, sig)
#include "hooks.h"
#undef W
#undef H
#undef F
#undef P
// pushal 後 [esp+32]＝返回位址，[esp+36] 起＝x, y, fmt, ...
#define H(tag, addr, sig) \
    __attribute__((naked)) static void hook_##tag(void) { \
        __asm__("pushal\n\tlea 36(%%esp), %%eax\n\tpush %%eax\n\tpush %0\n\tcall %P1\n\tadd $8, %%esp\n\tpopal\n\tjmp *%2" \
                ::"i"(#tag[0]), "i"(text_seen), "m"(tramp_##tag)); \
    }
#define F(tag, addr, sig)
#define P(tag, addr, sig)
#define W(tag, addr, sig)
#include "hooks.h"
#undef W
#undef H
#undef F
#undef P

static int hexval(char c) { return c <= '9' ? c - '0' : c - 'a' + 10; }

// 蓋掉開頭 len bytes（須是完整指令且無相對位址，draw.md 有逐一核對）：5 byte jmp＋nop；跳板＝原 len bytes＋jmp 回 at+len
static int install(char tag, BYTE *at, int len, const char *sig, void *hook, void **tramp) {
    char msg[96];
    for (int i = 0; i < 16; i++)
        if (at[i] != (BYTE)(hexval(sig[2 * i]) << 4 | hexval(sig[2 * i + 1]))) {
            wsprintfA(msg, "sig-mismatch %c 0x%08lx byte %d", tag, (DWORD)at, i);
            log_line(msg);
            return 0;
        }
    // 完全取代入口（tramp == NULL）不配置跳板，也不執行原函式。
    BYTE *t = tramp ? VirtualAlloc(NULL, 16, MEM_COMMIT | MEM_RESERVE, PAGE_EXECUTE_READWRITE) : NULL;
    DWORD old;
    if ((tramp && !t) || !VirtualProtect(at, len, PAGE_EXECUTE_READWRITE, &old)) {
        if (t) VirtualFree(t, 0, MEM_RELEASE);
        log_line("ERROR hook alloc/protect"); return 0;
    }
    if (tramp) {
        for (int i = 0; i < len; i++) t[i] = at[i];
        t[len] = 0xE9; *(DWORD *)(t + len + 1) = (DWORD)(at + len) - (DWORD)(t + len + 5);
        *tramp = t;
    }
    at[0] = 0xE9; *(DWORD *)(at + 1) = (DWORD)hook - (DWORD)(at + 5);
    for (int i = 5; i < len; i++) at[i] = 0x90;
    VirtualProtect(at, len, old, &old);
    FlushInstructionCache(GetCurrentProcess(), at, len);
    wsprintfA(msg, "hooked %c 0x%08lx", tag, (DWORD)at);
    log_line(msg);
    return 1;
}

// ---- 截圖：完全取代原 JPEG／surface lock 路徑 ----
// Win32 呼叫可能不設定 last-error；先清為 0，合成條件錯誤則給明確錯誤碼。
static void shot_capture(void) {
    HWND hwnd = NULL;
    HDC dc = NULL, source = NULL;
    HBITMAP bitmap = NULL;
    HGDIOBJ previous = NULL;
    HANDLE file = INVALID_HANDLE_VALUE;
    const char *step = NULL, *via = "screen";
    DWORD error = 0, pid = 0;
    int w = 0, h = 0, created = 0;
    WCHAR path[MAX_PATH], name[64];
    BYTE header[54];
    void *pixels = NULL;
    RECT client;
    POINT origin = {0, 0};
    SYSTEMTIME now;
    BITMAPINFO bi;
    char path8[3 * MAX_PATH], msg[3 * MAX_PATH + 96];
#define SHOT_CALL(label, expr) do { SetLastError(0); if (!(expr)) { \
        step = label; error = GetLastError(); goto done; } } while (0)
#define SHOT_ERROR(label, code) do { step = label; error = code; goto done; } while (0)
    SHOT_CALL("GetForegroundWindow", (hwnd = GetForegroundWindow()));
    SHOT_CALL("GetWindowThreadProcessId", GetWindowThreadProcessId(hwnd, &pid));
    if (pid != GetCurrentProcessId()) SHOT_ERROR("foreign-window", ERROR_ACCESS_DENIED);
    SHOT_CALL("GetClientRect", GetClientRect(hwnd, &client));
    w = client.right - client.left; h = client.bottom - client.top;
    unsigned size = bmp_header(header, w, h);
    if (!size) SHOT_ERROR("client-size", ERROR_INVALID_DATA);
    SHOT_CALL("ClientToScreen", ClientToScreen(hwnd, &origin));
    SHOT_CALL("CreateCompatibleDC", (dc = CreateCompatibleDC(NULL)));
    // 避免 -nostdlib 編譯器生成 memset。
    for (unsigned i = 0; i < sizeof bi; i++) ((BYTE *)&bi)[i] = 0;
    bi.bmiHeader.biSize = sizeof(BITMAPINFOHEADER);
    bi.bmiHeader.biWidth = w; bi.bmiHeader.biHeight = h;
    bi.bmiHeader.biPlanes = 1; bi.bmiHeader.biBitCount = 24;
    bi.bmiHeader.biCompression = BI_RGB; bi.bmiHeader.biSizeImage = size - 54;
    SHOT_CALL("CreateDIBSection", (bitmap = CreateDIBSection(dc, &bi, DIB_RGB_COLORS, &pixels, NULL, 0)));
    if (!pixels) SHOT_ERROR("DIB-pixels", ERROR_INVALID_DATA);
    // 同時清除列補齊與未被視窗繪製的區域；不依賴 DIB 初始內容。
    for (unsigned i = 0; i < size - 54; i++) ((BYTE *)pixels)[i] = 0;
    SetLastError(0);
    previous = SelectObject(dc, bitmap);
    if (!previous || previous == HGDI_ERROR) {
        previous = NULL; step = "SelectObject"; error = GetLastError(); goto done;
    }
    // 從螢幕 DC 抓 client 區：nGlide 全螢幕時 PrintWindow 回成功但給均勻暗圖（run 062022，3 張平均亮度 14.2、最大 15）。
    // 遊戲已確認在前景（上面核對 pid），螢幕上那塊就是遊戲畫面。
    SHOT_CALL("GetDC", (source = GetDC(NULL)));
    SHOT_CALL("BitBlt", BitBlt(dc, 0, 0, w, h, source, origin.x, origin.y, SRCCOPY));
    SHOT_CALL("GdiFlush", GdiFlush());
    SetLastError(0);
    DWORD n = GetEnvironmentVariableW(L"USERPROFILE", path, MAX_PATH);
    if (!n) { step = "USERPROFILE"; error = GetLastError(); goto done; }
    // 保留 Pictures/Wiz8 與時間戳檔名空間，且只接受絕對本機／UNC 路徑。
    if (n >= MAX_PATH - 80) SHOT_ERROR("profile-path-length", ERROR_FILENAME_EXCED_RANGE);
    if (!((n >= 3 && path[1] == L':' && path[2] == L'\\') ||
          (n >= 2 && path[0] == L'\\' && path[1] == L'\\')))
        SHOT_ERROR("profile-path", ERROR_BAD_PATHNAME);
    lstrcatW(path, L"\\Pictures");
    SetLastError(0);
    if (!CreateDirectoryW(path, NULL) && GetLastError() != ERROR_ALREADY_EXISTS) {
        step = "mkdir-Pictures"; error = GetLastError(); goto done;
    }
    lstrcatW(path, L"\\Wiz8");
    SetLastError(0);
    if (!CreateDirectoryW(path, NULL) && GetLastError() != ERROR_ALREADY_EXISTS) {
        step = "mkdir-Wiz8"; error = GetLastError(); goto done;
    }
    GetLocalTime(&now);
    wsprintfW(name, L"\\Wiz8-%04d%02d%02d-%02d%02d%02d-%03d.bmp", now.wYear, now.wMonth,
              now.wDay, now.wHour, now.wMinute, now.wSecond, now.wMilliseconds);
    lstrcatW(path, name);
    SHOT_CALL("path-UTF8", WideCharToMultiByte(CP_UTF8, 0, path, -1, path8, sizeof path8, NULL, NULL));
    SHOT_CALL("CreateFile", (file = CreateFileW(path, GENERIC_WRITE, 0, NULL, CREATE_NEW,
                                               FILE_ATTRIBUTE_NORMAL, NULL)) != INVALID_HANDLE_VALUE);
    created = 1;
    DWORD written;
    SHOT_CALL("WriteFile-header", WriteFile(file, header, sizeof header, &written, NULL));
    if (written != sizeof header) SHOT_ERROR("short-write-header", ERROR_WRITE_FAULT);
    SHOT_CALL("WriteFile-pixels", WriteFile(file, pixels, size - 54, &written, NULL));
    if (written != size - 54) SHOT_ERROR("short-write-pixels", ERROR_WRITE_FAULT);
done:
    // 清理也保留第一個錯誤；先退出 bitmap 的 DC 選取狀態再刪物件。
#define SHOT_CLEAN(label, expr) do { SetLastError(0); if (!(expr) && !step) { \
        step = label; error = GetLastError(); } } while (0)
    if (file != INVALID_HANDLE_VALUE) SHOT_CLEAN("CloseHandle", CloseHandle(file));
    if (previous) {
        SetLastError(0);
        HGDIOBJ restored = SelectObject(dc, previous);
        if ((!restored || restored == HGDI_ERROR) && !step) { step = "restore-bitmap"; error = GetLastError(); }
    }
    // DeleteDC 即使復原選取失敗也解除 bitmap 的選取。
    if (dc) SHOT_CLEAN("DeleteDC", DeleteDC(dc));
    if (bitmap) SHOT_CLEAN("DeleteObject", DeleteObject(bitmap));
    if (source) SHOT_CLEAN("ReleaseDC", ReleaseDC(NULL, source));
    if (step) {
        if (created) DeleteFileW(path); // 不留截斷的 BMP；保留原始錯誤碼。
        wsprintfA(msg, "shot ERROR %s %lu", step, error);
    } else {
        lstrcpyA(msg, "shot ok ");
        lstrcatA(msg, path8);
        wsprintfA(msg + lstrlenA(msg), " %dx%d via=%s", w, h, via);
    }
    log_line(msg);
#undef SHOT_CALL
#undef SHOT_ERROR
#undef SHOT_CLEAN
}

static void __cdecl shot(void) {
    static volatile LONG busy;
    if (!InterlockedExchange(&busy, 1)) {
        shot_capture();
        InterlockedExchange(&busy, 0);
    } else log_line("shot ERROR busy 170"); // ERROR_BUSY；防重入。
    volatile BYTE *b = (volatile BYTE *)0x00659728;
    *b = (*b - 1) & 1;
}

// ---- CJK：擴充遊戲自己的字表與字型（stage4 §4.2，路線見 notes/zh-patch-log.md:23）----
// 字表（0x00407ec0 建）：[0x006eb704]+4 → {u16 個數, wchar_t *字元}，繪字與量寬都線性查找，查不到用第 0 個。
// 字型物件（0x00406180 載入，0x00406320 釋放）：+0x04 像素資料大小、+0x14 ETRLE 像素資料、+0x18 子圖表、+0x1c 子圖數。
// 做法：每個字型載入後，子圖表擴成 252＋ATLAS_N 筆（原本不足 252 的空缺抄第 0 筆），尾端接上 CJK 字形；
// 字表在 252 個之後接上 atlas_chars。字元 252+j 就對到 CJK 字形 j，所有繪字、量寬、換色路徑照舊。
// 記憶體用 MSVCRT 的 malloc：0x00406320 用同一個 free 釋放 +0x14／+0x18（IAT 0x005eb224），換掉的舊緩衝當場 free。
#define BASE_CHARS 252
typedef struct { DWORD off, len; short ox, oy; WORD h, w; } Sub;
#define VO_SIZE(v) (*(DWORD *)((BYTE *)(v) + 0x04))
#define VO_PIX(v) (*(BYTE **)((BYTE *)(v) + 0x14))
#define VO_SUB(v) (*(Sub **)((BYTE *)(v) + 0x18))
#define VO_N(v) (*(WORD *)((BYTE *)(v) + 0xf6))
#define VO_PAL(v) (*(WORD **)((BYTE *)(v) + 0xe0)) // 16bpp 調色盤，blit 0x00411de0 以像素索引查表
static void *(__cdecl *crt_malloc)(size_t);
static void(__cdecl *crt_free)(void *);

static int crt_init(void) {
    if (crt_malloc) return 1;
    HMODULE m = GetModuleHandleA("msvcrt.dll");
    void *f = m ? (void *)GetProcAddress(m, "free") : NULL;
    if (!f || f != *(void **)0x005eb224) { log_line("ERROR font: msvcrt free 與遊戲 IAT 不同，不擴充"); return 0; }
    crt_free = f;
    crt_malloc = (void *)GetProcAddress(m, "malloc");
    return crt_malloc != NULL;
}

static void ext_chars(void) {
    BYTE *root = *(BYTE **)0x006eb704;
    BYTE *p = root ? *(BYTE **)(root + 4) : NULL;
    if (!p || *(WORD *)p != BASE_CHARS || !crt_init()) return;
    WCHAR *old = *(WCHAR **)(p + 4), *c = crt_malloc((BASE_CHARS + ATLAS_N) * 2);
    if (!c) return;
    for (int i = 0; i < BASE_CHARS; i++) c[i] = old[i];
    for (int j = 0; j < ATLAS_N; j++) c[BASE_CHARS + j] = atlas_chars[j];
    *(WCHAR **)(p + 4) = c;  // 舊陣列不釋放：0x00407ec0 的陣列遊戲本身也不 free
    *(WORD *)p = BASE_CHARS + ATLAS_N;
    char msg[64];
    wsprintfA(msg, "chars extended %d -> %d", BASE_CHARS, BASE_CHARS + ATLAS_N);
    log_line(msg);
}

static BYTE grid[64][64];

// 第 0 個字形（'A'）每列的字色索引：用到的索引 ≤4 種時，剔除亮度不到最亮者 1/4 的（陰影），剩下取最常見，全部列都用它；
// 否則（反鋸齒字型，5 色以上）全部列用像素數加權的亮度中位數那一色；亮字（日記以外）只算亮度不到最亮者一半以外的像素：
// dialog_font／Opt_title 的暗色雜邊佔多數，全體中位數取到近黑（dialog_font 66,66,57），離開遊戲確認框的字幾乎看不見（run 081820）。原本逐列取最常見：反鋸齒字每列亮度跳動
// （journal_font 'A' 各列 29～167），中文畫成明暗相間的條紋，淡色列在羊皮紙上消失＝點狀破碎（run 075346 截圖）。
// 依據 run 040543 的 font colors log：Wiz_Text_Font 是 fbda＋近黑 8001（陰影較多，取最常見畫成黑字，run 033741 FAIL）；
// 標題字型是 4 階漸層，英文墨色中位數＝最常見的 dab2（取最亮會偏亮，run 040151 FAIL）。
// 調色盤是 555（最高位另作他用）：dab2 以 555 解＝181,173,148，對上英文實測 181,175,148。
static int lum(WORD p) { return ((p >> 10) & 31) * 299 + ((p >> 5) & 31) * 587 + (p & 31) * 114; }
// *sh：被剔除的陰影索引裡最暗的（無則 0＝不畫 CJK 陰影；原字型沒陰影的，中文也不加）。
static void glyph_colors(const BYTE *pix, const Sub *a, const WORD *pal, BYTE *row, int rows, BYTE *sh, int dark_ink) {
    int h = a->h < 64 ? a->h : 64, w = a->w < 64 ? a->w : 64;
    const BYTE *q = pix + a->off;
    for (int y = 0; y < 64; y++) for (int x = 0; x < 64; x++) grid[y][x] = 0;
    for (int y = 0; y < a->h; y++) {
        int x = 0;
        for (BYTE c; (c = *q++);) {
            if (c & 0x80) { x += c & 0x7f; continue; }
            for (int k = 0; k < c; k++, x++, q++) if (y < h && x < w) grid[y][x] = *q;
        }
    }
    static WORD cnt[256];
    int distinct = 0, best = 0;
    for (int i = 0; i < 256; i++) cnt[i] = 0;
    for (int y = 0; y < h; y++) for (int x = 0; x < w; x++) if (grid[y][x] && !cnt[grid[y][x]]++) distinct++;
    for (int i = 1; i < 256; i++) if (cnt[i] > cnt[best]) best = i;
    if (pal) { // 字色判讀用：'A' 用到的 索引:像素數:調色盤值
        // log_line 的行緩衝 648 字元、前綴約 120（run 040406 列 65 色溢位 AV）：只列 ≤8 色
        char m[200]; int k = wsprintfA(m, "font colors distinct=%d", distinct);
        for (int i = 1; i < 256 && distinct <= 8; i++) if (cnt[i]) k += wsprintfA(m + k, " %d:%d:%04x", i, cnt[i], pal[i]);
        log_line(m);
    }
    if (distinct > 4 && pal) {
        int tot = 0, acc = 0, lo = -1, top = 0;
        for (int i = 1; i < 256; i++) if (cnt[i] && lum(pal[i]) > top) top = lum(pal[i]);
        if (!dark_ink) lo = top / 2 - 1;
        for (int i = 1; i < 256; i++) if (cnt[i] && lum(pal[i]) > lo) tot += cnt[i];
        for (;;) { // 依亮度由暗到亮累加，過半的那一色
            int m = 0, ml = 1 << 30;
            for (int i = 1; i < 256; i++) if (cnt[i] && lum(pal[i]) > lo && (lum(pal[i]) < ml)) ml = lum(pal[m = i]);
            if (!m) break;
            for (int i = 1; i < 256; i++) if (cnt[i] && lum(pal[i]) == ml) acc += cnt[i];
            best = m; lo = ml;
            if (2 * acc >= tot) break;
        }
    }
    if (distinct <= 4 && pal) {
        int top = 0;
        for (int i = 1; i < 256; i++) if (cnt[i] && lum(pal[i]) > top) top = lum(pal[i]);
        best = 0;
        for (int i = 1; i < 256; i++) if (cnt[i] && 4 * lum(pal[i]) >= top && cnt[i] > cnt[best]) best = i;
        for (int i = 1; i < 256; i++) if (cnt[i] && 4 * lum(pal[i]) < top && (!*sh || lum(pal[i]) < lum(pal[*sh]))) *sh = i;
    }
    for (int y = 0; y < rows; y++) row[y] = 0;
    for (int y = 0; y < rows && y < h; y++) {
        if (distinct <= 4 || pal) { row[y] = best; continue; }
        for (int i = 0; i < 256; i++) cnt[i] = 0;
        int rb = 0;
        for (int x = 0; x < w; x++) if (grid[y][x] && ++cnt[grid[y][x]] > cnt[rb]) rb = grid[y][x];
        row[y] = rb;
    }
    for (int y = 0; y < rows; y++) {
        if (row[y]) continue;
        for (int d = 1; d < rows; d++) {
            if (y - d >= 0 && row[y - d]) { row[y] = row[y - d]; break; }
            if (y + d < rows && row[y + d]) { row[y] = row[y + d]; break; }
        }
        if (!row[y]) row[y] = best ? best : 1;
    }
}

static BYTE *etrle_row(BYTE *o, const BYTE *bits, const BYTE *shb, int w, BYTE c, BYTE sh) {
    BYTE px[ATLAS_HMAX];
    for (int x = 0; x < w; x++) {
        BYTE m = 0x80 >> (x & 7);
        px[x] = bits[x >> 3] & m ? c : sh && shb[x >> 3] & m ? sh : 0;
    }
    for (int x = 0; x < w;) {
        int on = px[x] != 0, k = x;
        while (k < w && k - x < 127 && (px[k] != 0) == on) k++;
        if (on) { *o++ = k - x; for (; x < k; x++) *o++ = px[x]; }
        else { *o++ = 0x80 | (k - x); x = k; }
    }
    *o++ = 0;
    return o;
}

// 行距：0x004071f0 整個取代成 Sub[0].h＋oy＋加量（R8：88 處呼叫都經它取行高）。
// Wiz_Text_Font 系（訊息區、說明框）中文墨用滿 11 列，原行距 11 時首行或末行被框裁一列（6.4 實機）→ 加 1；其餘字型 0。
#define PITCH_EXTRA 1
static BYTE pitch_extra[25];
static int __cdecl line_height(int slot) {
    Sub *s = VO_SUB(((BYTE **)0x006eb6a0)[slot]);
    return (WORD)(s[0].h + s[0].oy + pitch_extra[slot]);
}

static int sig_ok(DWORD at, const char *sig) {
    for (int i = 0; i < 16; i++)
        if (((BYTE *)at)[i] != (BYTE)(hexval(sig[2 * i]) << 4 | hexval(sig[2 * i + 1]))) return 0;
    return 1;
}

// ---- 戰鬥中怪物移動倍率：只替換三個讀取運算元 ----
static volatile float speed_move __attribute__((aligned(4))) = 2.5f;
static volatile LONG speed_pct __attribute__((aligned(4)));
static volatile LONG event_speed_pct __attribute__((aligned(4)));
static volatile LONG anim_speed_pct __attribute__((aligned(4)));
static volatile float party_factor __attribute__((aligned(4))) = 1.0f;
static const DWORD party_return = 0x0041f794;
static int speed_prev_combat;
static MenuState menu;
static CRITICAL_SECTION menu_cs;
static char menu_ini[MAX_PATH];
static int menu_hotkey = VK_F10;
// Confine the cursor to the game window while it is foreground: a stray click on a second monitor
// minimizes the game, and focus loss sometimes crashes inside the system ddraw.dll.
static int window_clip = 1;
static volatile LONG menu_visible, guide_visible;
static GuideState guide;
static int xray_hotkey = VK_F11;
static volatile LONG xray_req;
static int guide_hotkey = VK_F7, guide_enabled, guide_size, guide_no_tsv, guide_truncated;
static char guide_path[MAX_PATH], *guide_tsv;
static WCHAR *guide_wide;
#define GUIDE_LIMIT (1024 * 1024)
static int guide_map(char name[51]) {
    name[0] = 0;
    if (!guide_enabled) return -1;
    DWORD map = *(volatile DWORD *)0x00686a70;
    if (map > 47) return -1;
    const char *source = (const char *)(0x00604478 + map * 107 + 50);
    int i = 0;
    while (i < 50 && source[i]) { name[i] = source[i]; ++i; }
    name[i] = 0;
    return (int)map;
}
static int guide_npages = 1;
static void guide_relayout(void);
static void guide_layout_reset(void);
static void guide_reload(void) {
    if (guide_tsv) HeapFree(GetProcessHeap(), 0, guide_tsv);
    if (guide_wide) HeapFree(GetProcessHeap(), 0, guide_wide);
    guide_tsv = NULL; guide_wide = NULL; guide_size = 0;
    guide_layout_reset();
    HANDLE file = guide_path[0] ? CreateFileA(guide_path, GENERIC_READ,
        FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE, NULL, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, NULL) : INVALID_HANDLE_VALUE;
    if (file != INVALID_HANDLE_VALUE) {
        DWORD high = 0, size = GetFileSize(file, &high), read = 0;
        if (!high && size <= GUIDE_LIMIT) {
            guide_tsv = HeapAlloc(GetProcessHeap(), 0, size + 1);
            guide_wide = HeapAlloc(GetProcessHeap(), 0, (size + 1) * sizeof(WCHAR));
            if (guide_tsv && guide_wide && ReadFile(file, guide_tsv, size, &read, NULL) && read == size) {
                guide_size = size; guide_tsv[size] = 0;
            } else {
                if (guide_tsv) HeapFree(GetProcessHeap(), 0, guide_tsv);
                if (guide_wide) HeapFree(GetProcessHeap(), 0, guide_wide);
                guide_tsv = NULL; guide_wide = NULL;
            }
        }
        CloseHandle(file);
    }
    if (!guide_tsv && !guide_no_tsv) {
        char msg[MAX_PATH + 32]; wsprintfA(msg, "guide no tsv %s", guide_path);
        log_line(msg); guide_no_tsv = 1;
    }
}
static void guide_log_key(void) {
    char name[51], msg[160];
    int map = guide_map(name);
    int lines = guide_select(guide_tsv, guide_size, map, guide.layer, NULL, NULL, GUIDE_LIMIT);
    wsprintfA(msg, "guide map=%d name=%s layer=%d page=%d/%d lines=%d", map, name, guide.layer, guide.page + 1, guide_npages, lines);
    log_line(msg);
}
static void guide_close(void) {
    guide.layer = guide.page = 0; guide_npages = 1; InterlockedExchange(&guide_visible, 0);
}

static void speed_init(void) {
    char path[MAX_PATH], msg[64];
    DWORD n = GetModuleFileNameA(NULL, path, sizeof path);
    if (n && n < sizeof path) {
        DWORD dir = n;
        while (dir && path[dir - 1] != '\\' && path[dir - 1] != '/') --dir;
        if (dir && dir + sizeof "wiz8mod.ini" <= sizeof path) {
            lstrcpyA(path + dir, "wiz8mod.ini");
            lstrcpyA(menu_ini, path);
            if (dir + sizeof "wiz8mod-guide.tsv" <= sizeof path) {
                lstrcpyA(path + dir, "wiz8mod-guide.tsv"); lstrcpyA(guide_path, path);
            }
        }
    }
    InitializeCriticalSection(&menu_cs);
    for (int i = 0; i < MENU_N; ++i)
        menu.val[i] = menu_ini[0] ? (int)GetPrivateProfileIntA("combat", menu_items[i].key,
                                      menu_items[i].def, menu_ini) : menu_items[i].def;
    for (int i = 0; i < MENU_N; ++i) menu_items[i].apply(menu.val[i]);
    if (menu_ini[0]) menu_hotkey = GetPrivateProfileIntA("menu", "hotkey_vk", VK_F10, menu_ini);
    if (menu_hotkey < 1 || menu_hotkey > 255) menu_hotkey = VK_F10;
    if (menu_ini[0]) guide_hotkey = GetPrivateProfileIntA("guide", "hotkey_vk", VK_F7, menu_ini);
    if (guide_hotkey < 1 || guide_hotkey > 255) guide_hotkey = VK_F7;
    wsprintfA(msg, "guide hotkey_vk=%d", guide_hotkey); log_line(msg);
    if (menu_ini[0]) xray_hotkey = GetPrivateProfileIntA("xray", "hotkey_vk", VK_F11, menu_ini);
    if (xray_hotkey < 1 || xray_hotkey > 255) xray_hotkey = VK_F11;
    wsprintfA(msg, "xray hotkey_vk=%d", xray_hotkey); log_line(msg);
    if (menu_ini[0]) window_clip = GetPrivateProfileIntA("window", "clip_cursor", 1, menu_ini) != 0;
    wsprintfA(msg, "window clip_cursor=%d", window_clip); log_line(msg);
    wsprintfA(msg, "menu hotkey_vk=%d", menu_hotkey);
    log_line(msg);
    wsprintfA(msg, "speed monster_move_pct=%d", speed_pct);
    log_line(msg);
    wsprintfA(msg, "speed event_speed_pct=%d", event_speed_pct);
    log_line(msg);
    wsprintfA(msg, "speed anim_speed_pct=%d", anim_speed_pct);
    log_line(msg);
    wsprintfA(msg, "speed party_move_pct=%d", menu.val[MENU_N - 1]);
    log_line(msg);
    speed_prev_combat = *(volatile BYTE *)0x00683f94 != 0;
    speed_move = speed_eff(*(volatile float *)0x006850ff, speed_pct, speed_prev_combat);
}

// 壞值沒有可表示的毫整數；log 用 0，超出 LONG 的有限值飽和，避免轉型未定義行為。
static LONG speed_milli(float value) {
    double scaled = (double)value * 1000.0;
    if (!__builtin_isfinite(value)) return 0;
    if (scaled >= 2147483647.0) return 2147483647L;
    if (scaled <= -2147483648.0) return (-2147483647L - 1);
    return (LONG)scaled;
}

static DWORD WINAPI speed_worker(LPVOID unused) {
    (void)unused;
    int logged = 0, prev = speed_prev_combat;
    for (;;) {
        float base = *(volatile float *)0x006850ff;
        int combat = *(volatile BYTE *)0x00683f94 != 0;
        float eff = speed_eff(base, speed_pct, combat);
        speed_move = eff;
        if (!logged && !prev && combat) {
            char msg[96];
            wsprintfA(msg, "speed combat base=%ld eff=%ld", speed_milli(base), speed_milli(eff));
            log_line(msg);
            logged = 1;
        }
        prev = combat;
        Sleep(50);
    }
    return 0;
}

static void install_speed(void) {
    static const struct { DWORD at; const char *sig; int offset; } sites[] = {
#define H(tag, addr, sig)
#define F(tag, addr, sig)
#define P(tag, addr, sig)
#define W(tag, addr, sig)
#define T(tag, addr, sig) {addr, sig, addr == 0x004554dd ? 1 : 2},
#include "hooks.h"
#undef T
#undef W
#undef P
#undef F
#undef H
    };
    _Static_assert(sizeof sites / sizeof *sites == 3, "speed group has three operands");
    DWORD old[3];
    int bad = 0;
    for (int i = 0; i < 3; ++i) {
        if (!sig_ok(sites[i].at, sites[i].sig)) {
            char msg[64];
            wsprintfA(msg, "sig-mismatch speed 0x%08lx", sites[i].at);
            log_line(msg);
            bad = 1;
        }
    }
    if (bad) return;
    // 全部取得寫入權限才動 bytes；同頁的權限須反序還原。
    int ready = 0;
    for (; ready < 3; ++ready) {
        if (!VirtualProtect((BYTE *)sites[ready].at + sites[ready].offset, 4,
                            PAGE_EXECUTE_READWRITE, &old[ready])) break;
    }
    if (ready == 3) {
        DWORD target = (DWORD)&speed_move;
        for (int i = 0; i < 3; ++i) {
            BYTE *operand = (BYTE *)sites[i].at + sites[i].offset;
            for (int j = 0; j < 4; ++j) operand[j] = (BYTE)(target >> (8 * j));
        }
    } else log_line("ERROR speed protect (group skipped)");
    while (ready) {
        --ready;
        DWORD ignored;
        if (!VirtualProtect((BYTE *)sites[ready].at + sites[ready].offset, 4, old[ready], &ignored))
            log_line("ERROR speed restore protect");
    }
    for (int i = 0; i < 3; ++i)
        FlushInstructionCache(GetCurrentProcess(), (BYTE *)sites[i].at + sites[i].offset, 4);
}

// ---- 戰鬥事件固定延遲：D 與 T 獨立，只改各指令 +3 的 imm32 ----
static const struct { DWORD at; const char *sig; } delay_sites[] = {
#define H(tag, addr, sig)
#define F(tag, addr, sig)
#define P(tag, addr, sig)
#define W(tag, addr, sig)
#define D(tag, addr, sig) {addr, sig},
#include "hooks.h"
#undef D
#undef W
#undef P
#undef F
#undef H
    };
static unsigned delay_original[5];
static int delay_ready;

static int delay_write(int pct, int live) {
    DWORD old[5];
    int ready = 0;
    for (; ready < 5; ++ready)
        if (!VirtualProtect((BYTE *)delay_sites[ready].at + 3, 4,
                            PAGE_EXECUTE_READWRITE, &old[ready])) break;
    int ok = ready == 5;
    if (ok) for (int i = 0; i < 5; ++i) {
        BYTE *operand = (BYTE *)delay_sites[i].at + 3;
        unsigned scaled = delay_scaled(delay_original[i], pct);
        /* All calls run on the game input/render thread; worker never writes code. */
        *(volatile DWORD *)operand = scaled;
        FlushInstructionCache(GetCurrentProcess(), operand, 4);
        if (!live) {
            char msg[80];
            wsprintfA(msg, "delay 0x%08lx %u->%u", delay_sites[i].at, delay_original[i], scaled);
            log_line(msg);
        }
    }
    else log_line("ERROR delay protect (group skipped)");
    while (ready) {
        --ready;
        DWORD ignored;
        if (!VirtualProtect((BYTE *)delay_sites[ready].at + 3, 4, old[ready], &ignored))
            log_line("ERROR delay restore protect");
    }
    if (ok && live) {
        char msg[64]; wsprintfA(msg, "delay live pct=%d", pct); log_line(msg);
    }
    return ok;
}
static void install_delay(void) {
    _Static_assert(sizeof delay_sites / sizeof *delay_sites == 5, "delay group");
    int bad = 0;
    for (int i = 0; i < 5; ++i) {
        if (!sig_ok(delay_sites[i].at, delay_sites[i].sig)) {
            char msg[64];
            wsprintfA(msg, "sig-mismatch delay 0x%08lx", delay_sites[i].at);
            log_line(msg); bad = 1;
        }
    }
    if (bad) return;
    for (int i = 0; i < 5; ++i)
        delay_original[i] = *(DWORD *)(delay_sites[i].at + 3);
    delay_ready = delay_write(event_speed_pct, 0);
}

static int menu_apply_move(int pct) { InterlockedExchange(&speed_pct, pct); return 1; }
static int menu_apply_anim(int pct) { InterlockedExchange(&anim_speed_pct, pct); return 1; }
static int menu_apply_party(int pct) { party_factor = party_dt(1.0f, pct, 0); return 1; }
static int menu_apply_event(int pct) {
    if (delay_ready && !delay_write(pct, 1)) return 0;
    InterlockedExchange(&event_speed_pct, pct); return 1;
}

// ---- Menu input: the game's WH_KEYBOARD runs before its WndProc. ----
static void *menu_key_tramp;
static void menu_install_overlay(DWORD at, const char *sig);
static WNDPROC menu_old_proc;
static HWND menu_hwnd;
static BYTE menu_eaten[256], menu_chars[512], menu_game_down[256], guide_eaten[256];
static LRESULT CALLBACK menu_keyboard(int code, WPARAM vk, LPARAM bits);

static void menu_save(void) {
    int ok = menu_ini[0] != 0;
    for (int i = 0; ok && i < MENU_N; ++i) {
        char value[24]; wsprintfA(value, "%d", menu.val[i]);
        ok = WritePrivateProfileStringA("combat", menu_items[i].key, value, menu_ini);
    }
    if (ok) { menu.dirty = 0; log_line("menu saved"); }
    else log_line("ERROR menu save");
}
static int menu_input(WPARAM vk, LPARAM bits, int up) {
    if (vk >= 256) return menu.open;
    unsigned scan = ((DWORD)bits >> 16) & 511;
    if (up) {
        int eat = menu_eaten[vk] || menu.open;
        menu_eaten[vk] = guide_eaten[vk] = 0;
        /* Keep character suppression until the next down for this scan code. */
        return eat;
    }
    menu_chars[scan] = 0;
    if (vk == (WPARAM)xray_hotkey) { // 傾印在 end-frame 做（遊戲執行緒），這裡只立旗標；按住重複不再觸發
        if (!(bits & (1L << 30))) InterlockedExchange(&xray_req, 1);
        menu_eaten[vk] = menu_chars[scan] = 1; return 1;
    }
    int direction = vk == VK_UP || vk == VK_DOWN || vk == VK_LEFT || vk == VK_RIGHT;
    if ((bits & (1L << 30)) && menu_eaten[vk] && !menu.open &&
        !(guide.layer && direction && vk != (WPARAM)menu_hotkey && vk != (WPARAM)guide_hotkey)) {
        menu_chars[scan] = 1;
        if (guide_eaten[vk]) guide_log_key();
        return 1;
    }
    if (!menu.open && vk != (WPARAM)menu_hotkey) {
        int key = vk == (WPARAM)guide_hotkey ? GUIDE_TOGGLE :
            vk == VK_ESCAPE ? GUIDE_ESC : vk == VK_UP ? GUIDE_UP :
            vk == VK_DOWN ? GUIDE_DOWN : vk == VK_LEFT ? GUIDE_LEFT :
            vk == VK_RIGHT ? GUIDE_RIGHT : GUIDE_OTHER;
        if (guide_enabled) {
            int was_open = guide.layer != 0;
            if (was_open) guide_relayout();
            GuideState next = guide;
            if (guide_nav(&next, key, guide_npages)) {
                if (!(bits & (1L << 30)) || (key != GUIDE_TOGGLE && key != GUIDE_ESC)) guide = next;
                guide_eaten[vk] = menu_eaten[vk] = menu_chars[scan] = 1;
                if (!was_open && guide.layer) {
                    guide_reload(); guide_truncated = 0;
                    /* Release movement already delivered before opening the panel. */
                    if (menu_key_tramp) for (int i = VK_LEFT; i <= VK_DOWN; ++i) if (menu_game_down[i]) {
                        ((HOOKPROC)menu_key_tramp)(HC_ACTION, i,
                            (LPARAM)(0xc0000001u | (MapVirtualKeyA(i, MAPVK_VK_TO_VSC) << 16)));
                        menu_game_down[i] = 0; menu_eaten[i] = 1;
                    }
                }
                if (guide.layer) guide_relayout();
                else guide_close();
                InterlockedExchange(&guide_visible, guide.layer && guide_tsv != NULL);
                guide_log_key();
                return 1;
            }
        }
    }
    int k = vk == (WPARAM)menu_hotkey ? MK_TOGGLE :
        vk == VK_UP ? MK_UP : vk == VK_DOWN ? MK_DOWN :
        vk == VK_LEFT ? MK_LEFT : vk == VK_RIGHT ? MK_RIGHT :
        vk == VK_HOME ? MK_RESET : vk == VK_END ? MK_RESET_ALL :
        vk == VK_ESCAPE ? MK_ESC : MK_OTHER;
    if (k == MK_TOGGLE && (bits & (1L << 30))) {
        menu_eaten[vk] = menu_chars[scan] = 1; return 1;
    }
    int was_open = menu.open;
    MenuState before = menu;
    int eat = menu_key(&menu, k);
    if (!eat) return 0;
    menu_eaten[vk] = menu_chars[scan] = 1;
    for (int i = 0; i < MENU_N; ++i)
        if (before.val[i] != menu.val[i] && !menu_items[i].apply(menu.val[i])) menu.val[i] = before.val[i];
    InterlockedExchange(&menu_visible, menu.open);
    if (!was_open && menu.open) {
        guide_close();
        static int opened;
        if (!opened) { opened = 1; log_line("menu open"); }
        /* Release keys delivered before opening, so movement/modifiers cannot stick. */
        if (menu_key_tramp) for (int i = 0; i < 256; ++i) if (menu_game_down[i]) {
            ((HOOKPROC)menu_key_tramp)(HC_ACTION, i,
                (LPARAM)(0xc0000001u | (MapVirtualKeyA(i, MAPVK_VK_TO_VSC) << 16)));
            menu_game_down[i] = 0; menu_eaten[i] = 1;
        }
    }
    if (was_open && !menu.open && menu.dirty) menu_save();
    return 1;
}
static LRESULT CALLBACK menu_keyboard(int code, WPARAM vk, LPARAM bits) {
    /* HC_NOREMOVE is a peek, not a key event; never apply twice. */
    if (code == HC_ACTION) {
        EnterCriticalSection(&menu_cs);
        int up = ((DWORD)bits >> 31) != 0;
        int eat = menu_input(vk, bits, up);
        if (!eat && vk < 256) menu_game_down[vk] = !up;
        LeaveCriticalSection(&menu_cs);
        if (eat) return 1;
    } else if (code == HC_NOREMOVE) return CallNextHookEx(NULL, code, vk, bits);
    return ((HOOKPROC)menu_key_tramp)(code, vk, bits);
}
static LRESULT CALLBACK menu_wndproc(HWND hwnd, UINT msg, WPARAM wp, LPARAM lp) {
    EnterCriticalSection(&menu_cs);
    int eat = 0;
    if (msg == WM_KEYDOWN || msg == WM_SYSKEYDOWN || msg == WM_KEYUP || msg == WM_SYSKEYUP)
        eat = menu_input(wp, lp, msg == WM_KEYUP || msg == WM_SYSKEYUP);
    if (msg == WM_CHAR || msg == WM_SYSCHAR || msg == WM_DEADCHAR || msg == WM_SYSDEADCHAR)
        eat = menu.open || menu_chars[((DWORD)lp >> 16) & 511];
    WNDPROC prev = menu_old_proc;
    if (msg == WM_NCDESTROY) menu_hwnd = NULL;
    if (msg == WM_CLOSE || msg == WM_DESTROY || msg == WM_NCDESTROY || (msg == WM_SHOWWINDOW && !wp)) {
        char m[48]; wsprintfA(m, "diag exit msg=0x%x", msg); log_line(m);
    }
    if (msg == WM_ACTIVATEAPP) {
        /* Alt+Tab 的 Alt 放開送到別的程式，遊戲的鍵盤 hook 以為 Alt 還按著（MG 10-31：切回後方向鍵失效）：
           回到遊戲時補送放開。 */
        char m[64]; int n = 0, alt = menu_game_down[VK_MENU];
        if (menu_key_tramp) for (int i = 0; i < 256; ++i) if (menu_game_down[i]) {
            ((HOOKPROC)menu_key_tramp)(HC_ACTION, i,
                (LPARAM)(0xc0000001u | (MapVirtualKeyA(i, MAPVK_VK_TO_VSC) << 16)));
            menu_game_down[i] = 0; menu_eaten[i] = 1; n++;
        }
        wsprintfA(m, "diag activateapp=%d released=%d alt=%d", (int)wp, n, alt); log_line(m);
    }
    if (window_clip && (msg == WM_NCDESTROY || (msg == WM_ACTIVATEAPP && !wp))) ClipCursor(NULL);
    LeaveCriticalSection(&menu_cs);
    return eat ? 0 : CallWindowProcA(prev, hwnd, msg, wp, lp);
}
static BOOL CALLBACK menu_find_window(HWND hwnd, LPARAM unused) {
    (void)unused;
    DWORD pid;
    GetWindowThreadProcessId(hwnd, &pid);
    if (pid != GetCurrentProcessId() || GetWindow(hwnd, GW_OWNER) ||
        (DWORD)GetClassLongA(hwnd, GCL_WNDPROC) != 0x004011e0) return TRUE;
    EnterCriticalSection(&menu_cs);
    SetLastError(0);
    LONG prev = SetWindowLongA(hwnd, GWL_WNDPROC, (LONG)menu_wndproc);
    DWORD error = GetLastError();
    if (prev) { menu_old_proc = (WNDPROC)prev; menu_hwnd = hwnd; log_line("hooked menu wndproc"); }
    else { char msg[80]; wsprintfA(msg, "ERROR menu wndproc %lu", error); log_line(msg); }
    LeaveCriticalSection(&menu_cs);
    return FALSE;
}
static DWORD WINAPI menu_worker(LPVOID unused) {
    (void)unused;
    for (;;) {
        EnterCriticalSection(&menu_cs);
        int hooked = menu_hwnd != NULL;
        LeaveCriticalSection(&menu_cs);
        if (!hooked) EnumWindows(menu_find_window, 0);
        Sleep(100);
    }
    return 0;
}
/* MG 11-01：關遊戲後視窗不見、Wiz8.exe 留著 Not Responding，Steam 一直顯示執行中（pid 20528，根因未取得）。
   遊戲視窗被掛過之後，若「視窗沒了」或「隱藏且 IsHungAppWindow」連續 15 s，記 log 後結束行程。
   不經 menu_cs：卡在 menu_cs 本身也要看得到（記 OwningThread）。 */
static DWORD WINAPI exit_watch(LPVOID unused) {
    (void)unused;
    int ever = 0, last = -1, dead_ms = 0;
    for (;;) {
        Sleep(500);
        HWND hwnd = *(HWND volatile *)&menu_hwnd;
        if (hwnd) ever = 1;
        if (!ever) continue;
        int vis = hwnd && IsWindowVisible(hwnd), hung = hwnd && IsHungAppWindow(hwnd);
        int state = (hwnd != NULL) | vis << 1 | hung << 2;
        if (state != last) {
            char m[96];
            wsprintfA(m, "diag exit-watch hwnd=%d visible=%d hung=%d cs_owner=%lu", hwnd != NULL, vis, hung,
                      (DWORD)(DWORD_PTR)menu_cs.OwningThread);
            log_line(m);
            last = state;
        }
        dead_ms = (!hwnd || (!vis && hung)) ? dead_ms + 500 : 0;
        if (dead_ms >= 15000) {
            log_line("exit-watch window gone/hung 15s: TerminateProcess");
            TerminateProcess(GetCurrentProcess(), 0);
        }
    }
    return 0;
}
static void install_menu_input(void) {
#define H(tag, addr, sig)
#define F(tag, addr, sig)
#define P(tag, addr, sig)
#define W(tag, addr, sig)
    int overlay_ok = 1;
#define M(tag, addr, sig) \
    if (addr == 0x0042a7b5) { guide_enabled = sig_ok(addr, sig); \
        if (!guide_enabled) log_line("sig-mismatch guide 0x0042a7b5"); } \
    else if (addr != 0x00401b30 && !sig_ok(addr, sig)) { log_line("sig-mismatch menu " #addr); overlay_ok = 0; }
#include "hooks.h"
#undef M
#define M(tag, addr, sig) \
    if (addr == 0x00401b30) { \
        if (!sig_ok(addr, sig)) log_line("sig-mismatch menu " #addr); \
        else install('M', (BYTE *)addr, 6, sig, menu_keyboard, &menu_key_tramp); \
    } else if (addr == 0x00426ceb && overlay_ok) menu_install_overlay(addr, sig);
#include "hooks.h"
#undef M
#undef W
#undef P
#undef F
#undef H
    HANDLE thread = CreateThread(NULL, 0, menu_worker, NULL, 0, NULL);
    if (thread) CloseHandle(thread);
    else { char msg[80]; wsprintfA(msg, "ERROR menu wndproc %lu", GetLastError()); log_line(msg); }
    HANDLE watch = CreateThread(NULL, 0, exit_watch, NULL, 0, NULL);
    if (watch) CloseHandle(watch);
    else log_line("ERROR exit-watch thread");
}

// ---- srGERD endFrame: flush queued geometry, overlay, then present. ----
static void (__attribute__((thiscall)) *menu_end_frame_real)(void *);
typedef struct { BYTE *p; LONG pitch; int w, h, bytes; DWORD white; } MenuCanvas;
static void menu_pixel(MenuCanvas *c, int x, int y, DWORD color) {
    if (x < 0 || y < 0 || x >= c->w || y >= c->h) return;
    BYTE *p = c->p + y * c->pitch + x * c->bytes;
    if (c->bytes == 2) *(WORD *)p = color; else *(DWORD *)p = color;
}
static int menu_advance(unsigned short ch) {
    return ch >= 0x20 && ch <= 0x7e ? ascii_adv[ch - 0x20] : atlas_cell[12 - ATLAS_HMIN];
}
static int menu_text(MenuCanvas *c, int x, int y, const WCHAR *s, DWORD color) {
    int cell = atlas_cell[12 - ATLAS_HMIN], stride = (cell + 7) / 8;
    for (; *s; ++s) {
        if (x + menu_advance(*s) > c->w) break;
        if (*s >= 0x20 && *s <= 0x7e) {
            const BYTE *bits = ascii_bits + (*s - 0x20) * 24;
            for (int py = 0; py < 12; ++py) for (int px = 0; px < 12; ++px)
                if (bits[py * 2 + px / 8] & (0x80 >> (px & 7))) menu_pixel(c, x + px, y + py, color);
            x += menu_advance(*s);
        } else {
            int lo = 0, hi = ATLAS_N;
            while (lo < hi) { int mid = (lo + hi) / 2; if (atlas_chars[mid] < *s) lo = mid + 1; else hi = mid; }
            if (lo < ATLAS_N && atlas_chars[lo] == *s) {
                const BYTE *bits = atlas_bits + atlas_off[12 - ATLAS_HMIN] + lo * cell * stride;
                for (int py = 0; py < cell; ++py) for (int px = 0; px < cell; ++px)
                    if (bits[py * stride + px / 8] & (0x80 >> (px & 7))) menu_pixel(c, x + px, y + py, color);
            }
            x += cell;
        }
    }
    return x;
}
// Surface virtual row accessors convert using its PixelFormat and pitch, including 565/555.
// SR.DLL confirms slots 41/42: (ARGB buffer, y, x0, x1), with x1 exclusive.
typedef void (__attribute__((thiscall)) *MenuGetRow)(void *, DWORD *, LONG y, LONG x0, LONG x1);
typedef void (__attribute__((thiscall)) *MenuSetRow)(void *, const DWORD *, LONG y, LONG x0, LONG x1);
// Layout is cached until map, layer, page or surface dimensions change; protected by menu_cs.
typedef struct { WCHAR *text; int n, y; } GuideLine;
static GuideLine guide_lines[1026];
static int guide_starts[1027], guide_steps[1026], guide_pages[1026], guide_line_layer[1026];
#define GUIDE_TAG_W 48 // right-hand column reserved for the "(L/3)" layer tag
static WCHAR guide_title[160];
static int guide_line_count, guide_panel_height, guide_cut;
static int guide_layout_map = -2, guide_layout_layer, guide_layout_page;
static int guide_layout_width, guide_layout_height;
static int guide_view_width = 600, guide_view_height = 448, guide_surface_seen;
static void guide_layout_reset(void) { guide_layout_map = -2; }
static void guide_layout(int w, int height) {
    char name[51]; int map = guide_map(name);
    if (map == guide_layout_map && guide.layer == guide_layout_layer && guide.page == guide_layout_page &&
        w == guide_layout_width && height == guide_layout_height) return;
    guide_layout_map = map; guide_layout_layer = guide.layer;
    guide_layout_width = w; guide_layout_height = height;
    guide_line_count = 0; guide_cut = 0;
    int used = 0, previous_layer = 0, failed = 0;
    int total = guide_select(guide_tsv, guide_size, map, guide.layer, NULL, NULL, GUIDE_LIMIT);
    if (!total && map >= 0 && !guide_select(guide_tsv, guide_size, map, 3, NULL, NULL, GUIDE_LIMIT)) {
        guide_steps[0] = 16; guide_line_layer[0] = 0;
        guide_lines[guide_line_count++] = (GuideLine){L"這張圖沒有攻略", -1, 0};
    }
    const char **rows = total ? HeapAlloc(GetProcessHeap(), 0, total * sizeof(*rows)) : NULL;
    int *lengths = total ? HeapAlloc(GetProcessHeap(), 0, total * sizeof(*lengths)) : NULL;
    if (total && (!rows || !lengths)) { failed = 1; log_line("ERROR guide layout allocate"); }
    if (rows && lengths) {
        int before = 0;
        guide_select(guide_tsv, guide_size, map, guide.layer, rows, lengths, total);
        for (int layer = 1; layer <= guide.layer && !guide_cut && !failed; ++layer) {
            int upto = guide_select(guide_tsv, guide_size, map, layer, NULL, NULL, GUIDE_LIMIT);
            int gap = upto > before && previous_layer ? 8 : 0;
            if (upto > before) previous_layer = layer;
            for (int i = before; i < upto && !guide_cut && !failed; ++i) {
                int n = lengths[i] ? MultiByteToWideChar(CP_UTF8, 0, rows[i], lengths[i], guide_wide + used, guide_size - used) : 0;
                if (lengths[i] && !n) { failed = 1; log_line("ERROR guide layout UTF-8"); break; }
                WCHAR *text = guide_wide + used;
                used += n + 1; text[n] = 0;
                // max 1027 > the 1026-line cap, so the cut fires before a truncated last start is drawn.
                int count = guide_wrap(text, n, menu_advance, w - 24 - GUIDE_TAG_W, guide_starts, 1027);
                if (!n) { guide_starts[0] = 0; count = 1; }
                for (int j = 0; j < count; ++j) {
                    if (guide_line_count == 1026) { guide_cut = 1; break; }
                    int end = j + 1 < count ? guide_starts[j + 1] : n;
                    while (end > guide_starts[j] && wrap_space(text[end - 1])) --end;
                    guide_steps[guide_line_count] = 16 + gap; gap = 0; guide_line_layer[guide_line_count] = layer;
                    guide_lines[guide_line_count++] = (GuideLine){text + guide_starts[j], end - guide_starts[j], 0};
                }
            }
            before = upto;
        }
    }
    if (rows) HeapFree(GetProcessHeap(), 0, rows);
    if (lengths) HeapFree(GetProcessHeap(), 0, lengths);
    if (guide_cut) {
        guide_lines[1025] = (GuideLine){L"…", -1, 0}; guide_line_layer[1025] = 0;
        if (!guide_truncated) { log_line("guide truncated"); guide_truncated = 1; }
    }
    guide_npages = guide_paginate(guide_steps, guide_line_count, height - 36, guide_pages, 1026);
    if (guide.page >= guide_npages) guide.page = guide_npages - 1;
    guide_layout_page = guide.page;
    int first = guide_pages[guide.page];
    int last = guide.page + 1 < guide_npages ? guide_pages[guide.page + 1] : guide_line_count;
    int y = 28;
    for (int i = first; i < last; ++i) {
        // Count the gap for pagination, but omit it at the top of a page.
        if (i != first) y += guide_steps[i] - 16;
        guide_lines[i].y = y; y += 16;
    }
    guide_panel_height = y + 8 < height ? y + 8 : height;
    WCHAR wide_name[51], hotkey[20];
    int nn = MultiByteToWideChar(CP_ACP, 0, name, -1, wide_name, 51);
    if (!nn) wide_name[0] = 0;
    if (map < 0) lstrcpyW(wide_name, L"不在地圖上");
    if (guide_hotkey >= VK_F1 && guide_hotkey <= VK_F24) wsprintfW(hotkey, L"F%d", guide_hotkey - VK_F1 + 1);
    else wsprintfW(hotkey, L"VK%d", guide_hotkey);
    if (map < 0) wsprintfW(guide_title, L"%s  第 %d/3 層  第 %d/%d 頁  上下：層  左右：翻頁  %s 關",
        wide_name, guide.layer, guide.page + 1, guide_npages, hotkey);
    else wsprintfW(guide_title, L"%s #%d  第 %d/3 層  第 %d/%d 頁  上下：層  左右：翻頁  %s 關",
        wide_name, map, guide.layer, guide.page + 1, guide_npages, hotkey);
}
static void guide_relayout(void) {
    if (!guide_surface_seen && menu_hwnd) {
        RECT client;
        if (GetClientRect(menu_hwnd, &client) && client.right > 32 && client.bottom > 32) {
            guide_view_width = client.right - 32;
            if (guide_view_width > 600) guide_view_width = 600;
            guide_view_height = client.bottom - 32;
        }
    }
    guide_layout(guide_view_width, guide_view_height);
}
static void guide_draw(MenuCanvas *c) {
    menu_text(c, 12, 8, guide_title, c->white);
    int first = guide_pages[guide.page];
    int last = guide.page + 1 < guide_npages ? guide_pages[guide.page + 1] : guide_line_count;
    for (int i = first; i < last; ++i) {
        GuideLine *line = &guide_lines[i];
        // Static messages are terminated; temporarily terminate cached text at its wrap boundary.
        if (line->n < 0) menu_text(c, 12, line->y, line->text, c->white);
        else {
            WCHAR saved = line->text[line->n]; line->text[line->n] = 0;
            menu_text(c, 12, line->y, line->text, c->white);
            line->text[line->n] = saved;
        }
        int tag = guide_tag(guide_line_layer, i, first);
        if (tag) {
            WCHAR label[8]; wsprintfW(label, L"(%d/3)", tag);
            menu_text(c, c->w - 12 - GUIDE_TAG_W + 8, line->y, label, 0xffffd040u);
        }
    }
}
static void menu_render(void *gerd) {
    MenuState snap;
    EnterCriticalSection(&menu_cs);
    snap = menu;
    int show_guide = !snap.open && guide_visible && guide_enabled;
    if (!snap.open && !show_guide) { LeaveCriticalSection(&menu_cs); return; }
    ((void (__attribute__((thiscall)) *)(void *))*(void **)0x005eb654)(gerd); // flushRenderers
    BYTE *surface = ((void *(__attribute__((thiscall)) *)(void *))*(void **)0x005eba58)(gerd);
    if (!surface) {
        static LONG failed;
        if (!InterlockedExchange(&failed, 1)) log_line("ERROR menu overlay lockBuffer");
        LeaveCriticalSection(&menu_cs); return;
    }
    LONG width = *(LONG *)(surface + 0x1c), height = *(LONG *)(surface + 0x20);
    if (width > 32 && width <= 16384 && height > 32 && height <= 16384) {
        int w = width - 32, h = height - 32;
        guide_view_width = w > 600 ? 600 : w; guide_view_height = h; guide_surface_seen = 1;
        if (show_guide) {
            if (w > 600) w = 600;
            guide_layout(w, h); h = guide_panel_height;
        } else {
            if (w > 560) w = 560;
            if (h > (MENU_N + 2) * 24) h = (MENU_N + 2) * 24;
        }
        DWORD *pixels = HeapAlloc(GetProcessHeap(), 0, w * h * sizeof(DWORD));
        if (pixels) {
            void **vt = *(void ***)surface;
            MenuGetRow get_row = (MenuGetRow)vt[41];
            MenuSetRow set_row = (MenuSetRow)vt[42];
            for (int y = 0; y < h; ++y) get_row(surface, pixels + y * w, 16 + y, 16, 16 + w);
            MenuCanvas c = {(BYTE *)pixels, w * 4, w, h, 4, 0xffffffffu};
            for (int y = 0; y < h; ++y) for (int x = 0; x < w; ++x) {
                DWORD old = pixels[y * w + x];
                int highlight = !show_guide && y >= 8 + snap.sel * 24 && y < 30 + snap.sel * 24;
                pixels[y * w + x] = highlight ? c.white : (old & 0xff000000u) | ((old >> 1) & 0x007f7f7fu);
            }
            if (show_guide) guide_draw(&c);
            else {
                for (int i = 0; i < MENU_N; ++i) {
                    DWORD color = i == snap.sel ? 0xff000000u : c.white;
                    menu_text(&c, 12, 12 + i * 24, menu_items[i].label, color);
                    WCHAR value[24]; wsprintfW(value, L"%d%%", snap.val[i]);
                    menu_text(&c, 128, 12 + i * 24, value, color);
                }
                menu_text(&c, 12, 20 + MENU_N * 24, L"上下選項 左右調整 HOME原版 END全部 ESC關閉", c.white);
            }
            for (int y = 0; y < h; ++y) set_row(surface, pixels + y * w, 16 + y, 16, 16 + w);
            HeapFree(GetProcessHeap(), 0, pixels);
            static LONG drawn;
            if (!InterlockedExchange(&drawn, 1)) log_line("menu overlay drawn srGERD");
        } else log_line("ERROR menu overlay allocate");
    } else log_line("ERROR menu overlay surface dimensions");
    ((void (__attribute__((thiscall)) *)(void *))*(void **)0x005eba60)(gerd);
    LeaveCriticalSection(&menu_cs);
}
// ---- BitAtlas 即時位置（GOAL-wiz8map W3）：每幀問 mp8_sample，要寫才讀指標、寫 exe 目錄的 MAPPOS.JS（暫存檔＋換名） ----
// 讀值（R17:13、:25、:32）：level＝u32[0x686a70]；隊伍世界物件 W＝[0x659c0c]，W+0x100/+0x104/+0x108＝X/Y/Z float；W+0xd4 是泛用物件朝向 getter（0x453970）讀的欄，實玩 log 對不上隊伍走向，不用。
// 朝向：轉向函式 0x41f58b 以 ecx＝D＝[0x65a0f8] 呼叫 0x477b90，0x477cd9 做 D+4 += dt×角速度並環繞 [0,2π)＝隊伍 yaw（弧度）。
static int mappos_readable(const void *p, DWORD n) {
    MEMORY_BASIC_INFORMATION m;
    if (!p || !VirtualQuery(p, &m, sizeof m) || m.State != MEM_COMMIT || (m.Protect & (PAGE_NOACCESS | PAGE_GUARD))) return 0;
    return (const char *)p + n <= (const char *)m.BaseAddress + m.RegionSize;
}
static void mappos_tick(void) {
    static struct mp8 s;
    static char path[MAX_PATH + 16], tmp[MAX_PATH + 16];
    static LONG logged, failed;
    if (!path[0]) {
        DWORD n = GetModuleFileNameA(NULL, path, MAX_PATH);
        if (!n || n >= MAX_PATH) { path[0] = '?'; log_line("ERROR mappos8 v1 exe path"); return; }
        while (n && path[n - 1] != '\\' && path[n - 1] != '/') n--;
        path[n] = 0;
        lstrcatA(path, "MAPPOS.JS");
        lstrcpyA(tmp, path); lstrcatA(tmp, ".tmp");
        mp8_init(&s);
        char msg[MAX_PATH + 40]; wsprintfA(msg, "mappos8 v1 path=%s", path); log_line(msg);
    }
    if (path[0] == '?') return;
    unsigned level = *(volatile unsigned *)0x00686a70;
    const char *w = *(char *volatile *)0x00659c0c;
    int ok = mappos_readable(w + 0xd4, 0x10c - 0xd4);
    float x = ok ? *(volatile float *)(w + 0x100) : 0, y = ok ? *(volatile float *)(w + 0x104) : 0, z = ok ? *(volatile float *)(w + 0x108) : 0;
    const char *d = *(char *volatile *)0x0065a0f8;
    float yaw = mappos_readable(d + 4, 4) ? *(volatile float *)(d + 4) : __builtin_nanf("");
    int prev = s.map;
    // X2 x-ray：個體向量 A／B（notes/re/monsters-live.md）每 250 ms 收一次；內容雜湊變了就強制寫。存活旗標未取得，先全送＋診斷 log
    static struct mp8_mob mobs[256];
    static unsigned nmob, mob_at, mob_h;
    static char mobl[256 * 40 + 32];
    static int nmobl;
    unsigned now = GetTickCount();
    if (now - mob_at >= 250) {
        mob_at = now, nmob = 0;
        static LONG diag;
        static char dl[256 * 64 + 16]; int dk = mp8_put(dl, sizeof dl, 0, "diag mobs");
        for (int vi = 0; vi < 2; vi++) {
            const char *v = *(char *volatile *)(vi ? 0x00683fb9 : 0x00683fad);
            if (!mappos_readable(v, 12)) continue;
            unsigned cnt = *(volatile unsigned *)(v + 8);
            const DWORD *data = *(const DWORD *volatile *)v;
            if (cnt > 1000 || !mappos_readable(data, cnt * 4)) continue;
            for (unsigned i = 0; i < cnt && nmob < 256; i++) {
                const char *e = (const char *)data[i];
                if (!mappos_readable(e, 0x28b)) continue;
                const char *m = *(char *const *)(e + 0xc);
                if (!mappos_readable(m + 0x118, 0x100)) continue;
                struct mp8_mob *o = &mobs[nmob++];
                o->x = *(volatile float *)(m + 0x118), o->z = *(volatile float *)(m + 0x120);
                o->tpl = *(volatile int *)(e + 8), o->fl = (*(volatile BYTE *)(e + 0x15) == 1) | (*(volatile BYTE *)(e + 0x16) & 3) << 1;  // bit0 戰鬥、bit1–2 敵我（E+0x16：0 中立 1 敵對 2 友善，原版自動地圖 0x582d9a 的分派）
                char t[64]; wsprintfA(t, " %d:%d:%lx:%x:%x:%x:%x:%x", *(int *)e, o->tpl, *(DWORD *)(e + 0x10), *(BYTE *)(e + 0x14), *(BYTE *)(e + 0x15), *(BYTE *)(e + 0x16), *(BYTE *)(e + 0x28a), *(BYTE *)(m + 0x217));
                dk = mp8_put(dl, sizeof dl, dk, t);
            }
        }
        nmobl = mp8_mobs(mobl, sizeof mobl, 0, level == 0xffffffffu ? -1 : (int)level, mobs, nmob);
        unsigned h = nmobl > 0 ? mp8_fnv(mobl, nmobl) : 0;
        if (h != mob_h) {
            mob_h = h, mp8_wrote(&s, 0);
            // 判讀存活旗標用（id:tpl:E+0x10:E+0x14:E+0x15:E+0x16:E+0x28a:M+0x217；後兩欄＝原版自動地圖顯示條件）：MG 自然打完戰鬥後讀 log 看死屍是否殘留、哪欄變
            // log_line 一行約 1.3 KB：分段寫，每段 ≤1000 字
            if (InterlockedIncrement(&diag) <= 3000 && dk > 0)
                for (int at = 0; at < dk; at += 1000) {
                    char c = at + 1000 < dk ? dl[at + 1000] : 0;
                    if (c) dl[at + 1000] = 0;
                    log_line(dl + at);
                    if (c) dl[at + 1000] = c;
                }
        }
    }
    if (!mp8_sample(&s, now, level, ok, x, y, z, mp8_theta(yaw))) return;
    static char js[16384];
    int n0 = mp8_format(js, sizeof js, s.map, x, y, z, s.t, s.seq), n = n0;
    // 探索位元（W4，引擎碼 0x46ce30 鄰近）：B＝[0x68f288]，[B+0x18]＝bit 陣列、[B+0x1c]＝bit 數＝本層節點表筆數；讀不到或超量就只寫位置
    const char *b = *(char *volatile *)0x0068f288;
    unsigned nbits = 0;
    if (n0 > 0 && mappos_readable(b + 0x18, 8)) {
        const unsigned char *bits = *(const unsigned char *volatile *)(b + 0x18);
        nbits = *(volatile unsigned *)(b + 0x1c);
        if (nbits > 40000 || !mappos_readable(bits, (nbits + 7) / 8)) nbits = 0;
        int m = mp8_seen(js, sizeof js, n0, s.map, bits, nbits);
        if (m > 0) n = m;
    }
    // mapMobs 只在本圖（s.map）與收樣同圖時附上；換圖途中不送舊圖的怪
    if (n > 0 && nmobl > 0 && s.map >= 0 && (unsigned)s.map == level) { int m = mp8_put(js, sizeof js, n, mobl); if (m > 0) n = m; }
    HANDLE f = n > 0 ? CreateFileA(tmp, GENERIC_WRITE, FILE_SHARE_READ, NULL, CREATE_ALWAYS, FILE_ATTRIBUTE_NORMAL, NULL) : INVALID_HANDLE_VALUE;
    DWORD wr = 0;
    int good = f != INVALID_HANDLE_VALUE && WriteFile(f, js, n, &wr, NULL) && wr == (DWORD)n;
    if (f != INVALID_HANDLE_VALUE) CloseHandle(f);
    good = good && MoveFileExA(tmp, path, MOVEFILE_REPLACE_EXISTING);
    mp8_wrote(&s, good);
    if (!good) { if (InterlockedIncrement(&failed) <= 20) log_line("ERROR mappos8 write"); return; }
    // log 與 JS 同一行（換行改空白）＋raw yaw（擬合零點／正負號用）；前 2000 筆全記，之後只記換圖（每秒 4 筆會灌爆 log）
    if (InterlockedIncrement(&logged) <= 2000 || prev != s.map) {
        char line[320];
        js[n0] = 0; // log 不帶 mapSeen 本體，只記 bit 數
        int k = mp8_put(line, sizeof line, 0, "mappos ");
        k = mp8_put(line, sizeof line, k, js);
        k = mp8_put(line, sizeof line, k, " seen=");
        k = mp8_int(line, sizeof line, k, nbits);
        k = mp8_put(line, sizeof line, k, " yaw=");
        k = mp8_ok(yaw) ? mp8_fix2(line, sizeof line, k, yaw) : mp8_put(line, sizeof line, k, "nan");
        if (k > 0) {
            for (char *c = line; *c; c++) if (*c == '\n') *c = ' ';
            log_line(line);
        }
    }
}
// ---- x-ray 傾印（X1 執行期驗證，notes/re/monsters-live.md「執行期傾印驗證清單」）：熱鍵按一次寫 %TEMP%\wiz8-mod-xray-N.txt ----
// 每行一欄：level、隊伍 W、四個向量（+0 資料指標、+8 筆數；元素＝指標陣列）、個體 E 全 0x425 B、M 前 0x348 B＋座標、備援場景清單篩 vtable 0x5ed22c。
static HANDLE xray_f;
static int xray_ok;
static void xray_w(const char *t, int n) { DWORD wr; if (n < 0 || !WriteFile(xray_f, t, n, &wr, NULL) || wr != (DWORD)n) xray_ok = 0; }
static void xray_mem(const char *tag, const void *p, unsigned len) { // tag＋位址＋hex；讀不到寫 unreadable
    static char line[2 * 0x425 + 64];
    int k = wsprintfA(line, "%s@0x%lx=", tag, (DWORD)p);
    k = mappos_readable(p, len) ? mp8_hex(line, sizeof line, k, p, len) : mp8_put(line, sizeof line, k, "unreadable");
    k = mp8_put(line, sizeof line, k, "\r\n");
    xray_w(line, k);
}
static void xray_vec(const char *tag, DWORD at) { // 向量：E 指標陣列；個體向量順帶傾印 E 與 M
    char hdr[96];
    const char *v = *(char *volatile *)at;
    unsigned cnt = mappos_readable(v, 12) ? *(volatile unsigned *)(v + 8) : 0;
    const DWORD *data = mappos_readable(v, 12) ? *(const DWORD *volatile *)v : NULL;
    xray_w(hdr, wsprintfA(hdr, "vec %s at=0x%lx v=0x%lx data=0x%lx count=%u\r\n", tag, at, (DWORD)v, (DWORD)data, cnt));
    if (cnt > 1000 || !mappos_readable(data, cnt * 4)) { xray_w("vec skipped\r\n", 13); return; }
    int indiv = tag[0] == 'E';
    for (unsigned i = 0; i < cnt; i++) {
        const char *e = (const char *)data[i];
        xray_w(hdr, wsprintfA(hdr, "%s[%u]\r\n", tag, i));
        xray_mem(indiv ? "E" : "G", e, indiv ? 0x425 : 0x12b);
        if (!indiv || !mappos_readable(e, 0x10)) continue;
        const char *m = *(char *const *)(e + 0xc);
        if (!m) continue;
        xray_mem("M", m, 0x348);
        if (mappos_readable(m + 0xec, 0x120 + 4 - 0xec)) {
            char ln[128];
            int k = mp8_put(ln, sizeof ln, 0, "Mpos ");
            k = mp8_fix2(ln, sizeof ln, k, *(volatile float *)(m + 0x118)); k = mp8_put(ln, sizeof ln, k, " ");
            k = mp8_fix2(ln, sizeof ln, k, *(volatile float *)(m + 0x11c)); k = mp8_put(ln, sizeof ln, k, " ");
            k = mp8_fix2(ln, sizeof ln, k, *(volatile float *)(m + 0x120)); k = mp8_put(ln, sizeof ln, k, " yaw ");
            k = mp8_fix2(ln, sizeof ln, k, *(volatile float *)(m + 0xec)); k = mp8_put(ln, sizeof ln, k, "\r\n");
            xray_w(ln, k);
        }
    }
}
static void xray_dump(void) {
    static LONG seq;
    char path[MAX_PATH + 40], msg[MAX_PATH + 80];
    DWORD n = GetTempPathA(MAX_PATH, path);
    LONG no = InterlockedIncrement(&seq);
    if (!n || n >= MAX_PATH) { log_line("xray dump fail temp path"); MessageBeep(MB_ICONHAND); return; }
    wsprintfA(path + n, "wiz8-mod-xray-%lu-%ld.txt", GetCurrentProcessId(), no);
    xray_f = CreateFileA(path, GENERIC_WRITE, FILE_SHARE_READ, NULL, CREATE_ALWAYS, FILE_ATTRIBUTE_NORMAL, NULL);
    if (xray_f == INVALID_HANDLE_VALUE) { wsprintfA(msg, "xray dump %ld fail open %s", no, path); log_line(msg); MessageBeep(MB_ICONHAND); return; }
    xray_ok = 1;
    SYSTEMTIME t; GetLocalTime(&t);
    char ln[160];
    xray_w(ln, wsprintfA(ln, "xray %ld tick=%lu time=%02d:%02d:%02d.%03d level=%u combat=%u\r\n", no, GetTickCount(),
        t.wHour, t.wMinute, t.wSecond, t.wMilliseconds, *(volatile unsigned *)0x00686a70, *(volatile BYTE *)0x00683f94));
    const char *w = *(char *volatile *)0x00659c0c;
    if (mappos_readable(w + 0x100, 12)) {
        int k = mp8_put(ln, sizeof ln, 0, "party ");
        k = mp8_fix2(ln, sizeof ln, k, *(volatile float *)(w + 0x100)); k = mp8_put(ln, sizeof ln, k, " ");
        k = mp8_fix2(ln, sizeof ln, k, *(volatile float *)(w + 0x104)); k = mp8_put(ln, sizeof ln, k, " ");
        k = mp8_fix2(ln, sizeof ln, k, *(volatile float *)(w + 0x108)); k = mp8_put(ln, sizeof ln, k, "\r\n");
        xray_w(ln, k);
    } else xray_w("party unreadable\r\n", 18);
    xray_vec("E-A", 0x00683fad); xray_vec("E-B", 0x00683fb9);
    xray_vec("G-A", 0x00683fb1); xray_vec("G-B", 0x00683fbd);
    // 備援：場景清單 [[0x659ab4]+0xb8]，count +4、資料指標 +0xc；只傾印 vtable 0x5ed22c（M 類）
    const char *sc = *(char *volatile *)0x00659ab4;
    const char *lst = mappos_readable(sc + 0xb8, 4) ? *(char *volatile *)(sc + 0xb8) : NULL;
    unsigned cnt = mappos_readable(lst, 0x10) ? *(volatile unsigned *)(lst + 4) : 0;
    const DWORD *data = mappos_readable(lst, 0x10) ? *(const DWORD *volatile *)(lst + 0xc) : NULL;
    xray_w(ln, wsprintfA(ln, "scene list=0x%lx data=0x%lx count=%u\r\n", (DWORD)lst, (DWORD)data, cnt));
    unsigned hits = 0;
    if (cnt <= 20000 && mappos_readable(data, cnt * 4))
        for (unsigned i = 0; i < cnt; i++) {
            const DWORD *o = (const DWORD *)data[i];
            if (!mappos_readable(o, 4) || *o != 0x005ed22c) continue;
            hits++;
            xray_w(ln, wsprintfA(ln, "scene[%u]\r\n", i));
            xray_mem("S", o, 0x348);
        }
    else xray_w("scene skipped\r\n", 15);
    xray_w(ln, wsprintfA(ln, "end scene_m=%u\r\n", hits));
    CloseHandle(xray_f);
    wsprintfA(msg, "xray dump %ld %s %s scene_m=%u", no, xray_ok ? "ok" : "fail", path, hits); log_line(msg);
    MessageBeep(xray_ok ? MB_OK : MB_ICONHAND);
}
static void __attribute__((thiscall)) menu_end_frame(void *gerd) {
    // Preserve all non-primary/offscreen renderers and avoid recursive engine flushes.
    static volatile LONG busy;
    if ((menu_visible || guide_visible) && gerd && gerd == *(void **)0x00659634 && !InterlockedExchange(&busy, 1)) {
        menu_render(gerd);
        InterlockedExchange(&busy, 0);
    }
    HWND hwnd = menu_hwnd;
    RECT r;
    if (window_clip && hwnd && GetForegroundWindow() == hwnd && !IsIconic(hwnd) && GetClientRect(hwnd, &r)) {
        MapWindowPoints(hwnd, NULL, (POINT *)&r, 2);
        static RECT last;
        if (!EqualRect(&r, &last)) {
            char m[80]; wsprintfA(m, "diag clip %ld,%ld-%ld,%ld", r.left, r.top, r.right, r.bottom); log_line(m);
            last = r;
        }
        ClipCursor(&r);
    }
    if (gerd && gerd == *(void **)0x00659634) {
        mappos_tick();
        if (InterlockedExchange(&xray_req, 0)) xray_dump();
    }
    menu_end_frame_real(gerd);
}
static void menu_install_overlay(DWORD at, const char *sig) {
    if (!sig_ok(at, sig)) { char msg[80]; wsprintfA(msg, "sig-mismatch menu 0x%08lx", at); log_line(msg); return; }
    HMODULE sr = GetModuleHandleA("SR.DLL");
    void *expected = sr ? (void *)GetProcAddress(sr, "?endFrame@srGERD@@QAEXXZ") : NULL;
    void **slot = (void **)0x005eb628;
    if (!expected || *slot != expected) { log_line("sig-mismatch menu 0x005eb628"); return; }
    DWORD old;
    if (!VirtualProtect(slot, sizeof *slot, PAGE_READWRITE, &old)) { log_line("ERROR menu overlay protect"); return; }
    menu_end_frame_real = (void (__attribute__((thiscall)) *)(void *))expected;
    InterlockedExchangePointer(slot, menu_end_frame);
    DWORD ignored;
    if (!VirtualProtect(slot, sizeof *slot, old, &ignored)) log_line("ERROR menu overlay restore protect");
    log_line("hooked menu overlay srGERD endFrame/lockBuffer");
}

// ---- 怪物攻擊／施法動畫：只改局部 call 的 rel32 ----
// fastcall 的 ecx＝B、edx 不用，唯一堆疊參數 m；callee 清 4 bytes，匹配原 thiscall。
static void __attribute__((fastcall)) anim_update(BYTE *b, void *unused_edx, float m) {
    (void)unused_edx;
    BYTE *a = *(BYTE **)(b + 0x1d8);
    if (a && anim_speed_pct != 100)
        m = anim_m(m, anim_speed_pct, *(volatile BYTE *)0x00683f94,
                   *(signed char *)(a + 0xa4));
    ((void (__attribute__((thiscall)) *)(BYTE *, float))0x004a6e20)(b, m);
}

// Consume the original ST(0) first; scaling uses one balanced temporary x87 slot.
// EAX is scratch here and is restored to the original overwritten result EBP.
__attribute__((naked)) static void party_stub(void) {
    __asm__ volatile (
        "fstps 0x14(%%esi)\n\t"
        "cmpb $0, 0x00683f94\n\t"
        "jne 1f\n\t"
        "movl 0x14(%%esi), %%eax\n\t"
        "testl %%eax, %%eax\n\t"
        "jle 1f\n\t"
        "cmpl $0x7f800000, %%eax\n\t"
        "jae 1f\n\t"
        "flds 0x14(%%esi)\n\t"
        "fmuls %0\n\t"
        "fstps 0x14(%%esi)\n\t"
        "1: movl %%ebp, %%eax\n\t"
        "jmp *%1\n\t"
        :: "m"(party_factor), "m"(party_return));
}

static void install_party(void) {
    static const struct { DWORD at; const char *sig; } sites[] = {
#define H(tag, addr, sig)
#define F(tag, addr, sig)
#define P(tag, addr, sig)
#define W(tag, addr, sig)
#define Q(tag, addr, sig) {addr, sig},
#include "hooks.h"
#undef Q
#undef W
#undef P
#undef F
#undef H
    };
    _Static_assert(sizeof sites / sizeof *sites == 1, "party has one detour");
    DWORD at = sites[0].at;
    char msg[64];
    if (!sig_ok(at, sites[0].sig)) {
        wsprintfA(msg, "sig-mismatch party 0x%08lx", at);
        log_line(msg);
        return;
    }
    BYTE *code = (BYTE *)at;
    DWORD old;
    if (!VirtualProtect(code, 5, PAGE_EXECUTE_READWRITE, &old)) {
        log_line("ERROR party protect (hook skipped)");
        return;
    }
    DWORD rel = (DWORD)party_stub - (at + 5);
    code[0] = 0xE9;
    for (int j = 0; j < 4; ++j) code[j + 1] = (BYTE)(rel >> (8 * j));
    FlushInstructionCache(GetCurrentProcess(), code, 5);
    DWORD ignored;
    if (!VirtualProtect(code, 5, old, &ignored)) log_line("ERROR party restore protect");
    wsprintfA(msg, "hooked party 0x%08lx", at);
    log_line(msg);
}

static void install_anim(void) {
    static const struct { DWORD at; const char *sig; } sites[] = {
#define H(tag, addr, sig)
#define F(tag, addr, sig)
#define P(tag, addr, sig)
#define W(tag, addr, sig)
#define A(tag, addr, sig) {addr, sig},
#include "hooks.h"
#undef A
#undef W
#undef P
#undef F
#undef H
    };
    _Static_assert(sizeof sites / sizeof *sites == 1, "anim has one call operand");
    DWORD at = sites[0].at;
    char msg[64];
    if (!sig_ok(at, sites[0].sig)) {
        wsprintfA(msg, "sig-mismatch anim 0x%08lx", at);
        log_line(msg);
        return;
    }
    BYTE *operand = (BYTE *)at + 1;
    DWORD old;
    if (!VirtualProtect(operand, 4, PAGE_EXECUTE_READWRITE, &old)) {
        log_line("ERROR anim protect (hook skipped)");
        return;
    }
    DWORD rel = (DWORD)anim_update - (at + 5);
    for (int j = 0; j < 4; ++j) operand[j] = (BYTE)(rel >> (8 * j));
    FlushInstructionCache(GetCurrentProcess(), operand, 4);
    DWORD ignored;
    if (!VirtualProtect(operand, 4, old, &ignored)) log_line("ERROR anim restore protect");
    wsprintfA(msg, "hooked anim 0x%08lx", at);
    log_line(msg);
}

// 把 at 起 n bytes 換成 call fn＋nop；先比對 16 bytes 簽章
static void hook_call(DWORD at, const char *sig, void (*fn)(void), int n, const char *name) {
    char msg[96];
    if (!sig_ok(at, sig)) { wsprintfA(msg, "sig-mismatch %s", name); log_line(msg); return; }
    BYTE *c = (BYTE *)at;
    DWORD old;
    if (!VirtualProtect(c, n, PAGE_EXECUTE_READWRITE, &old)) { wsprintfA(msg, "ERROR %s protect", name); log_line(msg); return; }
    c[0] = 0xE8; *(DWORD *)(c + 1) = (DWORD)fn - (DWORD)(c + 5);
    for (int i = 5; i < n; i++) c[i] = 0x90;
    VirtualProtect(c, n, old, &old);
    FlushInstructionCache(GetCurrentProcess(), c, n);
    wsprintfA(msg, "hooked %s 0x%08lx", name, at);
    log_line(msg);
}

// 訊息區逐行迴圈的 y 位移是寫死的 imm8 11（draw.md 0x0058c73e），不經 line_height → 同樣加 PITCH_EXTRA。
// 同一迴圈也畫選道具清單（窗格 2），清單的點擊判定（0x005902fd／0x0059044c）寫死除以 11 → 只對窗格 0（訊息紀錄）加，
// 否則游標與高亮錯位（2026-10-08 MG 實機，run play-pane：訊息紀錄 pane=0、選道具 pane=2）。
#define STR_(x) #x
#define STR(x) STR_(x)
__attribute__((naked)) static void msg_pitch(void) {
    // 取代 0x0058c73e 起 12 bytes：mov edx,[esp+0x14]; mov eax,[esp+0x10]; inc ebx; add edx,0xb（呼叫端 esp＝本函式 +4）；之後 cmp ebx,eax 自設旗標
    __asm__("movl 0x18(%esp), %edx\n\tmovl 0x14(%esp), %eax\n\tincl %ebx\n\taddl $0xb, %edx\n\t"
            "cmpw $0, 0x00686905\n\tjne 1f\n\taddl $" STR(PITCH_EXTRA) ", %edx\n1:\tret");
}
static void install_msg_pitch(DWORD at, const char *sig) { hook_call(at, sig, msg_pitch, 12, "msg pitch"); }

// 行距 12 時框只容 6 行（run 180720：第 7 行被框底裁）。滿 7 行時迴圈改從第 1 行起畫 6 行＝最新 6 行仍可見，不動捲動邏輯（7 散在約 45 處）。
// 只對窗格 0（訊息紀錄）：清單套用會藏掉第一項且點擊差一列（同上 msg_pitch）。
// ponytail: 捲到頂時最舊那一行看不到；要完整就得改所有捲動處的 7。
__attribute__((naked)) static void msg_first(void) {
    // 取代 0x0058c62c 的 mov ecx,[0x68edcc]; mov [esp+0x14],ebx（ebx＝0）；呼叫端 [esp+0x10] 行數、+0x14 y、+0x18 行號＝本函式 +4
    __asm__("movl 0x0068edcc, %ecx\n\tmovl $0, 0x18(%esp)\n\tcmpl $7, 0x14(%esp)\n\tjne 1f\n\t"
            "cmpw $0, 0x00686905\n\tjne 1f\n\tmovl $1, %ebx\n\tmovl %ebx, 0x1c(%esp)\n1:\tret");
}
static void install_msg_first(DWORD at, const char *sig) { hook_call(at, sig, msg_first, 10, "msg first"); }

// 戰鬥時訊息與選道具清單同為窗格 2，行距維持 11（點擊判定），但中文墨用滿 11 列，第 7 行底被文字區下緣裁約 1 列（2026-10-08 MG 實機截圖）。
// 訊息迴圈設文字區（0x0058c5e9 呼叫 0x00407220，JA2 SetFontDestBuffer 形：x1,y1,x2,y2）時 y2＝[ctx+0x2d8] → 加 2；下方到狀態列尚有餘裕，不動點擊判定。
__attribute__((naked)) static void msg_clip(void) {
    // 取代 0x0058c5c9 的 mov ecx,[eax+0x2d8]（6 bytes）
    __asm__("movl 0x2d8(%eax), %ecx\n\taddl $2, %ecx\n\tret");
}
static void install_msg_clip(DWORD at, const char *sig) { hook_call(at, sig, msg_clip, 6, "msg clip"); }

static void ext_font(int slot, const char *path) {
    char msg[MAX_PATH + 160];
    BYTE *v = ((BYTE **)0x006eb6a0)[slot];
    if (!v || !crt_init()) return;
    WORD n = VO_N(v);
    Sub *sub = VO_SUB(v);
    BYTE *pix = VO_PIX(v);
    if (!n || n > BASE_CHARS || !sub || !pix) {
        wsprintfA(msg, "font slot=%d n=%d 不擴充 %s", slot, n, path);
        log_line(msg);
        return;
    }
    DWORD used = 0;
    for (int i = 0; i < n; i++) if (sub[i].off + sub[i].len > used) used = sub[i].off + sub[i].len;
    int gh = sub[0].h < ATLAS_HMIN ? ATLAS_HMIN : sub[0].h > ATLAS_HMAX ? ATLAS_HMAX : sub[0].h;
    // 字框 c ≥ 字高（不小於 12 級字，MG 2026-10-07）：多出的上下平分，oy 為負＝超出原行高
    int c = atlas_cell[gh - ATLAS_HMIN], stride = (c + 7) / 8, oy = (gh - c) / 2;
    BYTE row[ATLAS_HMAX], crow[ATLAS_HMAX], sh = 0;
    // 字框比字高大時最後一列只剩陰影（墨 11 列從框頂起）：不畫，免得壓到下一行的頂列（run 075346 戰鬥 log 疊字）
    int hc = c > gh ? c - 1 : c;
    glyph_colors(pix, &sub[0], VO_PAL(v), row, gh, &sh, slot == 1); // slot 1＝journal_font：羊皮紙上的暗墨
    for (int y = 0; y < c; y++) crow[y] = row[y + oy < 0 ? 0 : y + oy >= gh ? gh - 1 : y + oy];
    DWORD bound = (DWORD)ATLAS_N * c * (c + 1) * 2;
    BYTE *np = crt_malloc(used + bound);
    Sub *ns = crt_malloc((BASE_CHARS + ATLAS_N) * sizeof(Sub));
    if (!np || !ns) { if (np) crt_free(np); if (ns) crt_free(ns); log_line("ERROR font: malloc"); return; }
    for (DWORD i = 0; i < used; i++) np[i] = pix[i];
    for (int i = 0; i < BASE_CHARS; i++) ns[i] = sub[i < n ? i : 0];
    BYTE *o = np + used;
    const BYTE *bits = atlas_bits + atlas_off[gh - ATLAS_HMIN], *shb = atlas_shadow + atlas_off[gh - ATLAS_HMIN];
    for (int j = 0; j < ATLAS_N; j++) {
        Sub *s = &ns[BASE_CHARS + j];
        s->off = o - np; s->ox = 0; s->oy = oy; s->h = hc; s->w = c;
        for (int y = 0; y < hc; y++) o = etrle_row(o, bits + (j * c + y) * stride, shb + (j * c + y) * stride, c, crow[y], sh);
        s->len = (o - np) - s->off;
    }
    wsprintfA(msg, "font slot=%d n=%d h=%d gh=%d cell=%d oy=%d size=%lu used=%lu color0=%d rgb=%04x shadow=%d +%lu bytes %s", slot, n, sub[0].h, gh, c, oy,
              VO_SIZE(v), used, row[0], VO_PAL(v) ? VO_PAL(v)[row[0]] : 0, sh, (DWORD)(o - np) - used, path);
    log_line(msg);
    const char *base = path;
    for (const char *q = path; *q; q++) if (*q == '\\') base = q + 1;
    char pre[14];
    lstrcpynA(pre, base, sizeof pre);
    if (slot < 25 && !lstrcmpiA(pre, "wiz_text_font")) {
        pitch_extra[slot] = PITCH_EXTRA;
        wsprintfA(msg, "pitch slot=%d +%d", slot, PITCH_EXTRA);
        log_line(msg);
    }
    VO_PIX(v) = np; VO_SUB(v) = ns; VO_N(v) = BASE_CHARS + ATLAS_N; VO_SIZE(v) = o - np;
    crt_free(pix); crt_free(sub);
}

static int __cdecl font_load(char *path) {
    int slot = ((int(__cdecl *)(char *))tramp_L)(path);
    if (slot >= 0) { ext_font(slot, path); ext_chars(); }
    return slot;
}

// ---- UTF-16 版面斷行（入口／局部續行的 ABI 見 notes/re/wrap.md）----
#define H(tag, addr, sig)
#define F(tag, addr, sig)
#define P(tag, addr, sig)
#define W(tag, addr, sig) static void *wrap_tramp_##tag __attribute__((unused));
#include "hooks.h"
#undef H
#undef F
#undef P
#undef W

static void wrap_hit(volatile LONG *seen_hit, DWORD address) {
    if (!InterlockedExchange(seen_hit, 1)) {
        char msg[40];
        wsprintfA(msg, "wrap hit 0x%08lx", address);
        log_line(msg);
    }
    ext_chars(); // 量寬可能先於第一次繪字。
}
static int wrap_has_cjk(const WCHAR *s) {
    if (s) for (; *s; ++s) if (wrap_cjk(*s)) return 1;
    return 0;
}
static int wrap_char_width(WCHAR c, int font) {
    WCHAR one[2] = {c, 0};
    return (unsigned short)((short(__cdecl *)(const WCHAR *, int))0x00407010)(one, font);
}
// 回傳非空行的結束位置；不可分的超寬詞整詞保留（不拆 ASCII 詞或禁則組）。
// 不改來源、沒有固定字數上限；呼叫端決定是否消耗明示換行。
static const WCHAR *wrap_line_end(const WCHAR *s, int font, int limit, int *width) {
    const WCHAR *p = s, *last = NULL;
    int x = 0, last_width = 0;
    while (*p && *p != '\n') {
        x += wrap_char_width(*p++, font);
        if (!*p || *p == '\n' || wrap_can_break(p[-1], *p)) {
            if (x > limit) {
                if (last) { *width = last_width; return last; }
                *width = x; return p;
            }
            last = p; last_width = x;
        }
    }
    *width = x;
    return p;
}

// thiscall：ecx＝文字物件，無堆疊參數。ASCII／不分行旗標完整走原版。
static void __attribute__((thiscall)) wrap_object(BYTE *obj) {
    static volatile LONG hit;
    wrap_hit(&hit, 0x004f35b0);
    WCHAR *old = *(WCHAR **)(obj + 0x34);
    if (!wrap_has_cjk(old) || (obj[0x38] & 0x40) || !crt_init()) {
        ((void(__attribute__((thiscall)) *)(BYTE *))wrap_tramp_T)(obj);
        return;
    }
    unsigned n = lstrlenW(old);
    if (n > 0x3ffffffe) return;
    WCHAR *out = crt_malloc((2 * n + 1) * sizeof(WCHAR));
    if (!out) return; // 原字串仍歸遊戲所有。
    const WCHAR *p = old;
    WCHAR *q = out;
    int lines = 1, widest = 0;
    int limit = *(int *)(obj + 0x0c) - *(int *)(obj + 0x04);
    int font = *(int *)(obj + 0x28);
    while (*p) {
        int width;
        const WCHAR *end = wrap_line_end(p, font, limit > 0 ? limit - 1 : 0, &width);
        if (width > widest) widest = width;
        while (p < end) *q++ = *p++;
        if (*p) {
            *q++ = '\n'; ++lines;
            if (*p == '\n') ++p;
        }
    }
    *q = 0;
    *(WCHAR **)(obj + 0x34) = out;
    *(int *)(obj + 0x2c) = lines;
    *(int *)(obj + 0x3c) = widest;
    crt_free(old);
}

// thiscall：ecx＝彈出框；唯一堆疊參數＝文字；ret 4，eax＝行數。
static int __attribute__((thiscall)) wrap_popup(BYTE *obj, const WCHAR *s) {
    static volatile LONG hit;
    wrap_hit(&hit, 0x005d2a50);
    if (!wrap_has_cjk(s) || !crt_init())
        return ((int(__attribute__((thiscall)) *)(BYTE *, const WCHAR *))wrap_tramp_P)(obj, s);
    int font = *(int *)0x0064fde8, limit = *(int *)(obj + 0x30);
    unsigned count = 0;
    const WCHAR *p = s;
    do {
        int width;
        p = wrap_line_end(p, font, limit, &width);
        ++count;
        if (*p == '\n') ++p;
    } while (*p);
    if (count > 0x3fffffff) return 0;
    WCHAR **lines = crt_malloc(count * sizeof(*lines));
    if (!lines) return 0;
    p = s;
    for (unsigned i = 0; i < count; ++i) {
        int width;
        const WCHAR *end = wrap_line_end(p, font, limit, &width);
        unsigned n = end - p;
        lines[i] = crt_malloc((n + 1) * sizeof(WCHAR));
        if (!lines[i]) {
            while (i) crt_free(lines[--i]);
            crt_free(lines); return 0;
        }
        for (unsigned j = 0; j < n; ++j) lines[i][j] = *p++;
        lines[i][n] = 0;
        if (*p == '\n') ++p;
    }
    *(WCHAR ***)(obj + 0x8c) = lines;
    return count; // 呼叫端 0x005d2906 寫 +0x90；原有釋放路徑照舊。
}

// push continuation, pushfl, pushal 的佈局；ESP 不以 C 的呼叫框架猜測。
typedef struct {
    DWORD di, si, bp, saved_sp, bx, dx, cx, ax, flags, ip;
} WrapFrame;
_Static_assert(sizeof(WrapFrame) == 40, "i386 wrap frame");
#define WRAP_THUNK(tag, continuation, helper) \
    __attribute__((naked)) static void wrap_hook_##tag(void) { \
        __asm__("push %0\n\tpushfl\n\tpushal\n\tpush %%esp\n\tcall %P1\n\tadd $4, %%esp\n\tpopal\n\tpopfl\n\tret" \
                :: "i"(continuation), "i"(helper)); \
    }

// 入口記錄後走跳板；不碰 __chkstk 的相對 call，僅複製其前的 mov eax。
#define WRAP_ENTRY(tag, address) \
    static void wrap_enter_##tag(WrapFrame *r) { \
        static volatile LONG hit; wrap_hit(&hit, address); r->ip = (DWORD)wrap_tramp_##tag; \
    } \
    WRAP_THUNK(tag, 0, wrap_enter_##tag)
WRAP_ENTRY(C, 0x005d0050)
WRAP_ENTRY(D, 0x005d0770)

// 鎖持有到訊息分行／複製完成；重入時直接走原字串，不等待。
static WCHAR space_buf[4096];
static volatile LONG space_busy;
static DWORD space_return;
static void wrap_leave_M(WrapFrame *r) {
    r->ip = space_return; // 先取回返回位址，再讓下一個呼叫重用緩衝。
    InterlockedExchange(&space_busy, 0);
}
WRAP_THUNK(MR, 0, wrap_leave_M)

static void wrap_enter_M(WrapFrame *r) {
    static volatile LONG hit, space_hit;
    wrap_hit(&hit, 0x0058ac00);
    r->ip = (DWORD)wrap_tramp_M;
    // frame 之後是入口原堆疊：返回位址、字型索引、UTF-16 字串、其餘三參數。
    DWORD *sp = (DWORD *)(r + 1);
    const WCHAR *s = (const WCHAR *)sp[2];
    if (!wrap_has_cjk(s) || InterlockedExchange(&space_busy, 1)) return;
    int n = space_cjk(space_buf, 4096, s);
    if (n < 0 || n == lstrlenW(s)) {
        InterlockedExchange(&space_busy, 0);
        return;
    }
    space_return = sp[0];
    sp[0] = (DWORD)wrap_hook_MR;
    sp[2] = (DWORD)space_buf;
    if (!InterlockedExchange(&space_hit, 1)) log_line("space hit 0x0058ac00");
}
WRAP_THUNK(M, 0, wrap_enter_M)

// 訊息分行器已複製一字並量寬；edi 現在指下一字，esi 指下一輸出位置。
// 走原有「記下詞界」分支即可；它會在溢寬時回到這個字，不會丟字。
static void wrap_message_boundary(WrapFrame *r) {
    const WCHAR *p = (const WCHAR *)r->di;
    if (*p && p[-1] && (wrap_cjk(p[-1]) || wrap_cjk(*p)) && wrap_can_break(p[-1], *p))
        r->ip = 0x0058ae39;
    else r->ip = ((WORD)r->ax == 0x20) ? 0x0058ae39 : 0x0058ae1d;
}
WRAP_THUNK(MB, 0, wrap_message_boundary)

// 富文字兩個 pass 共用同一個邊界條件；控制詞 U+00B2..B5／LF 留給原版。
static int wrap_virtual_boundary(const WCHAR *p, unsigned word_len, const WCHAR *word) {
    return word_len && *p > 0x20 && word[0] != '\n' &&
        !(word[0] >= 0xb2 && word[0] <= 0xb5) &&
        (wrap_cjk(p[-1]) || wrap_cjk(*p)) && wrap_can_break(p[-1], *p);
}
static void wrap_rich_scan(WrapFrame *r, int draw) {
    BYTE *sp = (BYTE *)(r + 1);
    const WCHAR *p = (const WCHAR *)r->ax;
    unsigned len = (WORD)(draw ? r->bx : r->si);
    WCHAR *word = (WCHAR *)(sp + (draw ? 0x30 : 0x2c));
    int split = wrap_virtual_boundary(p, len, word);
    WORD c = *p;
    r->ax = (r->ax & 0xffff0000) | c; // 被覆蓋的 mov ax,[eax]
    if (split) {
        // 原 pass 最後會遞增索引；虛擬詞界不消耗來源字元。
        --*(DWORD *)(sp + (draw ? 0x28 : 0x24));
        r->ip = draw ? 0x005d0832 : 0x005d019c;
    } else if (c == 0x20 || !c) r->ip = draw ? 0x005d0832 : 0x005d019c;
    else if (!draw && c == '\n') r->ip = 0x005d0109;
    else r->ip = draw ? 0x005d081f : 0x005d0189;
}
static void wrap_count_scan(WrapFrame *r) { wrap_rich_scan(r, 0); }
static void wrap_draw_scan(WrapFrame *r) { wrap_rich_scan(r, 1); }
WRAP_THUNK(CS, 0, wrap_count_scan)
WRAP_THUNK(DS, 0, wrap_draw_scan)

static void wrap_rich_space(WrapFrame *r, int draw) {
    BYTE *sp = (BYTE *)(r + 1);
    const WCHAR *p = *(const WCHAR **)(sp + (draw ? 0x2c : 0x28));
    // 原輸入只有空白或 NUL 能到此。額外的 CJK 詞界不加入可見空白，
    // 不改上一次 cmp 的 EFLAGS（draw 的下一條 jg 需要它）。
    *(WCHAR *)r->di = (*p && *p != 0x20) ? 0 : 0x20;
}
static void wrap_count_space(WrapFrame *r) { wrap_rich_space(r, 0); }
static void wrap_draw_space(WrapFrame *r) { wrap_rich_space(r, 1); }
WRAP_THUNK(CA, 0x005d03b2, wrap_count_space)
WRAP_THUNK(DA, 0x005d0d29, wrap_draw_space)

// 輸入框重排：保留原本的索引陣列擴充、游標／捲動更新，僅替換找下一行起點。
static void wrap_edit_boundary(WrapFrame *r) {
    BYTE *root = *(BYTE **)0x0068edcc;
    BYTE *obj = *(BYTE **)(root + 0x1fc);
    const WCHAR *s = *(const WCHAR **)obj;
    const WCHAR *p = s + r->di;
    if (!wrap_has_cjk(p)) { r->ip = (DWORD)wrap_tramp_EB; return; }
    int width;
    const WCHAR *end = wrap_line_end(p, *(int *)(root + 0x2e8),
                                    *(int *)(obj + 0x24) - (int)r->bx, &width);
    if (*end == '\n') ++end;
    r->di = end - s;
    r->ax = (DWORD)obj;
    r->cx = (DWORD)root;
    r->ip = *end ? 0x0058dda9 : 0x0058df54;
}
WRAP_THUNK(EB, 0, wrap_edit_boundary)
#undef WRAP_ENTRY
#undef WRAP_THUNK

static void install_wrap_hooks(void) {
    // 所有入口與局部續行先一起驗；任何一個不符就整組不裝。
    int bad = 0;
#define H(tag, addr, sig)
#define F(tag, addr, sig)
#define P(tag, addr, sig)
#define W(tag, addr, sig) do { \
    for (int i = 0; i < 16; ++i) if (((BYTE *)addr)[i] != \
        (BYTE)(hexval(sig[2*i]) * 16 + hexval(sig[2*i+1]))) { \
        char msg[80]; wsprintfA(msg, "sig-mismatch wrap 0x%08lx byte %d", (DWORD)addr, i); \
        log_line(msg); bad = 1; break; \
    } \
} while (0);
#include "hooks.h"
#undef H
#undef F
#undef P
#undef W
    if (bad) return;
    // 富文字先裝不添加空白的續行點，成功才裝會製造虛擬詞界的掃描點。
    // 入口跳板皆止於第一條相對 call／branch 之前。
#define WRAP_INSTALL(tag, len, hook, trampoline) \
    install('W', (BYTE *)wrap_addr_##tag, len, wrap_sig_##tag, hook, trampoline)
#define H(tag, addr, sig)
#define F(tag, addr, sig)
#define P(tag, addr, sig)
#define W(tag, addr, sig) const DWORD wrap_addr_##tag = addr; const char *wrap_sig_##tag = sig;
#include "hooks.h"
#undef H
#undef F
#undef P
#undef W
    WRAP_INSTALL(T, 5, wrap_object, &wrap_tramp_T);
    WRAP_INSTALL(P, 5, wrap_popup, &wrap_tramp_P);
    if (WRAP_INSTALL(M, 5, wrap_hook_M, &wrap_tramp_M))
        WRAP_INSTALL(MB, 6, wrap_hook_MB, NULL);
    if (WRAP_INSTALL(C, 6, wrap_hook_C, &wrap_tramp_C) &&
        WRAP_INSTALL(CA, 5, wrap_hook_CA, NULL))
        WRAP_INSTALL(CS, 7, wrap_hook_CS, NULL);
    if (WRAP_INSTALL(D, 6, wrap_hook_D, &wrap_tramp_D) &&
        WRAP_INSTALL(DA, 5, wrap_hook_DA, NULL))
        WRAP_INSTALL(DS, 7, wrap_hook_DS, NULL);
    // 0x0058dca0 不裝入口：同函式 0x0058df21 跳回 0x0058dca4，落在入口 10 bytes 內。
    WRAP_INSTALL(EB, 7, wrap_hook_EB, &wrap_tramp_EB);
#undef WRAP_INSTALL
}

static void install_text_hooks(void) {
    char path[MAX_PATH + 32];
    DWORD n = GetTempPathA(MAX_PATH, path);
    // 文字 log 開不了只關掉記錄，繪字與字型 hook 照裝（稽核 F3：否則中文全畫成 A）
    if (n && n < MAX_PATH) {
        lstrcatA(path, "wiz8-mod-text.log");
        text_log = CreateFileA(path, GENERIC_WRITE, FILE_SHARE_READ, NULL, CREATE_ALWAYS, FILE_ATTRIBUTE_NORMAL, NULL);
    }
    if (text_log == INVALID_HANDLE_VALUE) log_line("ERROR open text log");
#define H(tag, addr, sig) install(#tag[0], (BYTE *)addr, 6, sig, hook_##tag, &tramp_##tag);
#define F(tag, addr, sig) install(#tag[0], (BYTE *)addr, 8, sig, font_load, &tramp_##tag);
#define P(tag, addr, sig)
#define W(tag, addr, sig)
#include "hooks.h"
#undef W
#undef H
#undef F
#undef P
}

static void install_shot_hook(void) {
#define H(tag, addr, sig)
#define F(tag, addr, sig)
    // 開頭第一條 mov ecx,[0x005eb70c] 為 6 bytes；整條覆蓋，不需跳板。
#define P(tag, addr, sig) install(#tag[0], (BYTE *)addr, 6, sig, shot, NULL);
#define W(tag, addr, sig)
#include "hooks.h"
#undef W
#undef H
#undef F
#undef P
}

// exe 內的 UTF-16 字面字串（不在 STRINGDATA）：.data 可寫，原字串逐字比對相符才換成不長於原文的譯文（MG 2026-10-07：對話框「隊伍金幣：546685g」）。
static void patch_wstrs(void) {
    static const struct { DWORD at; const WCHAR *en, *zh; } T[] = {
        {0x0064fc84, L"%s %dg", L"%s %d\x91d1"},
        {0x0064a0a0, L"%dg", L"%d\x91d1"},
        // 戰鬥結算「獲得 %d %s」（STRINGDATA 557）的 %s：0x004ea008 依數量選 kill／kills（run 075346「獲得 4 kills」）
        {0x006174b4, L"kills", L"\x6b21\x64ca\x6bba"},
        {0x006174c0, L"kill", L"\x6b21\x64ca\x6bba"},
    };
    for (unsigned i = 0; i < sizeof T / sizeof *T; i++) {
        WCHAR *w = (WCHAR *)T[i].at;
        char msg[64];
        if (lstrcmpW(w, T[i].en)) { wsprintfA(msg, "sig-mismatch wstr 0x%08lx", T[i].at); log_line(msg); continue; }
        lstrcpyW(w, T[i].zh);
        wsprintfA(msg, "wstr patched 0x%08lx", T[i].at);
        log_line(msg);
    }
}

// NSF 對白（notes/re/nsf.md）：0x0055a242 的 sprintf(dst, "%S", src) 改成 nsf_store（>0xFF 寫 UTF-8，緩衝不夠就 realloc），
// swprintf／vswprintf 的 IAT 對 L"%S"（0x0061c4b4）、L" \"%S\""（0x0061c43c，訊息區回顯 NPC 台詞）且字串含 UTF-8 序列時解回；其餘呼叫原樣轉給 msvcrt。
static int(__cdecl *nsf_vswprintf_real)(WCHAR *, const WCHAR *, __builtin_va_list);
static volatile LONG nsf_hits;
static int __cdecl nsf_vswprintf(WCHAR *d, const WCHAR *f, __builtin_va_list ap) {
    if (f == (const WCHAR *)0x0061c4b4 || f == (const WCHAR *)0x0061c43c) {
        __builtin_va_list c;
        __builtin_va_copy(c, ap);
        const BYTE *s = __builtin_va_arg(c, const BYTE *);
        __builtin_va_end(c);
        if (s && nsf_has_seq(s)) {
            if (InterlockedIncrement(&nsf_hits) == 1) log_line("nsf widen utf8");
            return nsf_widen_fmt(d, f, s);
        }
    }
    return nsf_vswprintf_real(d, f, ap);
}
static int __cdecl nsf_swprintf(WCHAR *d, const WCHAR *f, ...) {
    __builtin_va_list ap;
    __builtin_va_start(ap, f);
    int r = nsf_vswprintf(d, f, ap);
    __builtin_va_end(ap);
    return r;
}
static void __cdecl nsf_store(char **slot, const WCHAR *src) {
    int n = nsf_narrow_len(src);
    if (n > lstrlenW(src)) {
        char *p = ((void *(__cdecl *)(void *, size_t)) * (void **)0x005eb1f8)(*slot, n + 1);  // IAT realloc，與遊戲 malloc／free 同一個 msvcrt
        if (p) { *slot = p; nsf_narrow((BYTE *)p, src); return; }
    }
    ((int(__cdecl *)(char *, const char *, ...)) * (void **)0x005eb21c)(*slot, "%S", src);  // 全 ≤0xFF：原版 sprintf
}
// 呼叫點：[esp+4]=dst、[esp+12]=src；eax＝指標陣列 [ebx+1]、esi＝索引（dst＝[eax+esi*4]）
__attribute__((naked)) static void nsf_store_thunk(void) {
    __asm__("push 12(%%esp)\n\tlea (%%eax,%%esi,4), %%eax\n\tpush %%eax\n\tcall %P0\n\tadd $8, %%esp\n\tret" ::"i"(nsf_store));
}
static int nsf_iat(DWORD at, const char *name, void *hook, void **real) {
    HMODULE m = GetModuleHandleA("msvcrt.dll");
    void *f = m ? (void *)GetProcAddress(m, name) : NULL;
    DWORD old;
    if (!f || *(void **)at != f || !VirtualProtect((void *)at, 4, PAGE_READWRITE, &old)) return 0;
    if (real) *real = f;
    *(void **)at = hook;
    VirtualProtect((void *)at, 4, old, &old);
    return 1;
}
static void install_nsf(DWORD at, const char *sig) {
    for (int i = 0; i < 16; i++)
        if (((BYTE *)at)[i] != (BYTE)(hexval(sig[2 * i]) << 4 | hexval(sig[2 * i + 1]))) { log_line("sig-mismatch nsf"); return; }
    // nsf_vswprintf 只照「一個 %S、無其他轉換」解讀這兩個格式
    if (lstrcmpW((const WCHAR *)0x0061c4b4, L"%S") || lstrcmpW((const WCHAR *)0x0061c43c, L" \"%S\"")) { log_line("sig-mismatch nsf fmt"); return; }
    if (!nsf_iat(0x005eb1f0, "vswprintf", nsf_vswprintf, (void **)&nsf_vswprintf_real)) { log_line("ERROR nsf: vswprintf IAT"); return; }
    // swprintf 失敗時 vswprintf 已換也無害：沒有 UTF-8 序列的呼叫原樣轉給 msvcrt；不改呼叫點就不會產生序列
    if (!nsf_iat(0x005eb188, "swprintf", nsf_swprintf, NULL)) { log_line("ERROR nsf: swprintf IAT"); return; }
    BYTE *c = (BYTE *)at + 6;  // push 0x61c4b0; push ecx; ↓ call [0x005eb21c]（6 bytes）
    DWORD old;
    if (!VirtualProtect(c, 6, PAGE_EXECUTE_READWRITE, &old)) { log_line("ERROR nsf protect"); return; }
    c[0] = 0xE8; *(DWORD *)(c + 1) = (DWORD)nsf_store_thunk - (DWORD)(c + 5); c[5] = 0x90;
    VirtualProtect(c, 6, old, &old);
    FlushInstructionCache(GetCurrentProcess(), c, 6);
    log_line("hooked nsf 0x0055a242 + swprintf/vswprintf IAT");
}

// 閃退診斷：c000041d 只說「回呼途中出錯」，原始例外的位址在這裡記（前 32 筆，略過除錯器通知類）。
static LONG CALLBACK diag_veh(EXCEPTION_POINTERS *e) {
    static volatile LONG n;
    EXCEPTION_RECORD *r = e->ExceptionRecord;
    DWORD c = r->ExceptionCode;
    if (c == 0x406D1388 || c == 0x40010006 || c == 0x4001000A || InterlockedIncrement(&n) > 32)
        return EXCEPTION_CONTINUE_SEARCH;
    HMODULE m = NULL; char mod[MAX_PATH] = "?", msg[MAX_PATH + 160];
    if (GetModuleHandleExA(GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS | GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT,
                           (LPCSTR)r->ExceptionAddress, &m) && m) GetModuleFileNameA(m, mod, sizeof mod);
    mod[sizeof mod - 1] = 0;
    wsprintfA(msg, "diag exception code=%08lx addr=%p off=%08lx info=%08lx,%08lx tid=%lu combat=%d mod=%s", c,
              r->ExceptionAddress, (DWORD)((char *)r->ExceptionAddress - (char *)m),
              r->NumberParameters > 0 ? (DWORD)r->ExceptionInformation[0] : 0,
              r->NumberParameters > 1 ? (DWORD)r->ExceptionInformation[1] : 0, GetCurrentThreadId(),
              *(volatile BYTE *)0x00683f94, mod);
    log_line(msg);
    return EXCEPTION_CONTINUE_SEARCH;
}

BOOL WINAPI DllMain(HINSTANCE inst, DWORD reason, LPVOID reserved) {
    if (reason == DLL_PROCESS_DETACH) log_line(reserved ? "diag detach process-exit" : "diag detach unload");
    if (reason != DLL_PROCESS_ATTACH) return TRUE;
    DisableThreadLibraryCalls(inst);
    char sys[MAX_PATH];
    // 32 位元行程在 64 位元 Windows 上，GetSystemWow64Directory 給 SysWOW64；32 位元 Windows 則回 0，退回 System32
    UINT n = GetSystemWow64DirectoryA(sys, sizeof sys);
    if (!n) n = GetSystemDirectoryA(sys, sizeof sys);
    if (!n || n + 11 >= sizeof sys) { log_line("ERROR system dir"); return FALSE; }
    lstrcatA(sys, "\\ddraw.dll");
    HMODULE real = LoadLibraryA(sys);
    if (!real) { log_line("ERROR LoadLibrary system ddraw"); return FALSE; }
    int missing = 0;
#define X(name, ord) if (!(real_##name = (void *)GetProcAddress(real, #name))) missing++;
#include "exports.h"
#undef X
    char msg[MAX_PATH + 64];
    wsprintfA(msg, "attach real=%s missing=%d", sys, missing);
    log_line(msg);
    AddVectoredExceptionHandler(1, diag_veh);
    speed_init(); // 先填有效值；新執行緒在 DllMain 返回後才開始跑，不等待。
    install_delay();
    install_anim();
    install_party();
    install_menu_input();
    HANDLE speed_thread = CreateThread(NULL, 0, speed_worker, NULL, 0, NULL);
    if (speed_thread) {
        CloseHandle(speed_thread);
        install_speed();
    } else log_line("ERROR speed thread (group skipped)");
    install_text_hooks();
    install_shot_hook();
    // 行距：前兩條指令 4＋7 bytes 整段覆蓋（draw.md），不需跳板
#define H(tag, addr, sig)
#define F(tag, addr, sig)
#define P(tag, addr, sig)
#define W(tag, addr, sig)
#define G(tag, addr, sig) install('G', (BYTE *)addr, 11, sig, line_height, NULL);
#include "hooks.h"
#undef G
#undef W
#undef H
#undef F
#undef P
    install_wrap_hooks();
    patch_wstrs();
#define H(tag, addr, sig)
#define F(tag, addr, sig)
#define P(tag, addr, sig)
#define W(tag, addr, sig)
#define N(tag, addr, sig) install_nsf(addr, sig);
#include "hooks.h"
#undef N
    // 訊息區三個 Y 互相依賴（行距加大要配裁切與起始行），先全部驗簽章，任一不符就整組不裝（稽核 F2）
    int y_bad = 0;
#define Y(tag, addr, sig) if (!sig_ok(addr, sig)) { log_line("sig-mismatch msg " #tag " (Y group skipped)"); y_bad = 1; }
#include "hooks.h"
#undef Y
#define Y(tag, addr, sig) install_msg_##tag(addr, sig);
    if (!y_bad) {
#include "hooks.h"
    }
#undef Y
#undef W
#undef H
#undef F
#undef P
    return TRUE;
}

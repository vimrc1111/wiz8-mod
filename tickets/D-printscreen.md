# D 實作：遊戲內 PrintScreen 改由代理自己存圖（不再崩）

驗收: bash scripts/ci-check.sh
規模: 中（C，ddraw 代理加一個 detour＋BMP 輸出）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 hook/ddraw/bmp.test.sh、scripts/ci-check.sh、notes/re/*.md 與白名單外任何檔；不寫入遊戲目錄；不執行遊戲；不加 CRT 或第三方依賴；不放寬或跳過測試
白名單: hook/ddraw/proxy.c、hook/ddraw/bmp.h、hook/ddraw/build.sh、hook/ddraw/check.sh
預期紅: bmp-roundtrip
執行器: codex

## 背景
Wizardry 8（Wiz8.exe，i386）在現代 Windows 按 PrintScreen 會先寫出 `Wiz8NNNNN.JPG` 再崩潰（崩在系統 ddraw.dll，0xC000041D；不載我們的代理也一樣崩＝原版相容問題）。
我們的代理 `hook/ddraw/proxy.c` 已被遊戲載入，並以 detour 改寫遊戲函式（`install()`，簽章從 `notes/re/*.md` 表格經 build.sh 產生 `hooks.h`）。
遊戲截圖函式的事實見 `notes/re/shot.md`（0x004229e0，`void f(void)`，前 16 bytes 已列，check-sigs.sh 每次 CI 對真 exe 驗）。

## 要做
1. `hook/ddraw/bmp.h`：實作 `bmp_stride`、`bmp_header`（24 位元、BITMAPINFOHEADER、由下而上、正的高度）。`bash hook/ddraw/bmp.test.sh` 要綠。
   proxy.c 與測試共用這一份（proxy 是 -nostdlib，bmp.h 不准用 libc）。
2. build.sh：把 `notes/re/shot.md` 表格中 0x004229e0 的簽章也產進 hooks.h（新種類，例如 `P(S, 0x004229e0, "…")`），讀法照現有 draw.md 那段，找不到簽章就 build 失敗。
3. proxy.c：install 時把 0x004229e0 開頭改成 `jmp` 到我們的函式，**完全取代**原函式（不呼叫原函式、不需要跳板）；簽章不符就照現有慣例 log `sig-mismatch` 並不裝。
   我們的函式：
   - 找本行程的遊戲視窗（例：`GetForegroundWindow` 後以 `GetWindowThreadProcessId` 核對 pid；不是本行程就 log 並跳過存圖）。
   - 擷取 client 區：先 `PrintWindow(hwnd, dc, 2)`（PW_RENDERFULLCONTENT，硬體繪製的視窗也拿得到）；失敗才退回 `BitBlt` 視窗 DC。用 `CreateDIBSection` 取 24 位元像素。
   - 寫 BMP 到 `%USERPROFILE%\Pictures\Wiz8\Wiz8-YYYYMMDD-HHMMSS-mmm.bmp`（`GetLocalTime`；目錄不存在就建）。**不寫遊戲目錄**（實機驗收要遊戲目錄 diff 乾淨）。
   - 結束前照原函式行為把 0x00659728 那個 byte 的最低位元翻轉（`b = (b - 1) & 1`）。
   - 每次 log 一行：`shot ok <路徑> <w>x<h> via=<printwindow|bitblt>` 或 `shot ERROR <哪一步> <GetLastError>`。任何失敗都不得讓遊戲崩（釋放 DC／物件／handle）。
4. check.sh：匯入白名單加 `gdi32.dll`（系統 DLL；仍不准 CRT）。

## 驗收（CI 驗不到的部分）
執行期行為（按 3 次 PrintScreen、遊戲活到 60 秒、圖檔存在且不是全黑、遊戲目錄 diff 乾淨）由 cc 實機驗，不在本單；交付時說明你怎麼確保失敗路徑不崩即可。

## 交付回覆
附「規格疑慮」段：這張單哪裡寫得不清楚、哪個作法你判斷有風險（例：PrintWindow 對 nGlide 視窗的行為）；沒有就寫「無」。

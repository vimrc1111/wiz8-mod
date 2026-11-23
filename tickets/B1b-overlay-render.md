# B1b 修：F10 疊加框畫壞（只出一條細橫線＋一小段白）

驗收: bash hook/ddraw/menu.test.sh && bash hook/ddraw/speed.test.sh && bash scripts/check-sigs.sh && bash hook/ddraw/build.sh && bash hook/ddraw/check.sh
規模: 小（查 ABI＋改繪製）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改驗收腳本與白名單外任何檔；不寫入遊戲目錄；不執行遊戲；不改既有 T／U／W／M patch 的位址與簽章；不加 CRT 或第三方依賴；不放寬或跳過測試；筆記不貼反組譯碼或反編譯碼
白名單: hook/ddraw/proxy.c、hook/ddraw/menu.h、notes/re/menu.md
執行器: codex

## 現象（MG 2026-10-12 實機，B1 5a46f6d）
- log 有 `menu open`、`menu overlay drawn srGERD`（疊加路徑有跑到）。
- 畫面：左上只有一條橫跨約 560 遊戲像素的 1 px 細線，線上一小段白；沒有半透明底框、沒有字、沒有反白列。遊戲 1280×1024、16-bit。
- 本票無機械紅測（現象只在實機）；靜態驗收只防回歸。實機由 cc 跑 wiz8-run 截圖判讀，判不過再開票。

## 線索（未驗證，自己查清楚再改）
- `hook/ddraw/proxy.c:810` `menu_render`：讀 `vt[41]`／`vt[42]` 當 `getPixelRow`／`setPixelRow`，呼叫 `(buf, 16, 16+y, w)`。
- SR.DLL 匯出 `?getPixelRow@srColorSurface@@UAEXPAKJJJ@Z`、`?setPixelRow@srColorSurface@@UAEXPBKJJJ@Z`：三個 long 的意義（x,y,count？y,x,count？還是 x0,x1,y？）未證實。`notes/re/menu.md:35` 只由一個呼叫點推定。
- 材料：`build/sr.dll`（遊戲原檔副本，唯讀，勿入版控）、`build/Wiz8.exe`、`build/wiz8.asm`。
- 沙箱內讀不到 /mnt/c：材料皆為 worktree 內實體檔。自跑驗收時用 `SYS_DDRAW=build/sysddraw.dll WIZ8_EXE=build/Wiz8.exe`（系統 ddraw 副本、遊戲 exe 副本）。
- 其他可能：槽位號錯（vtable 起點／偏移）、surface 寬高欄位錯、在錯的時機（lock 前／後）讀寫、ARGB 格式假設。逐一以 SR.DLL 本體確認。

## 要做
1. 查 SR.DLL 的 getPixelRow／setPixelRow 本體與 srColorSurface vtable，確認參數語意與槽位；依據（位址＋用途，不貼碼）更新 `notes/re/menu.md`，舊推定改寫時標明更正。
2. 修 `menu_render` 使底框、三列字、反白、提示列都畫到 surface 左上 (16,16) 起的區域。
3. 交付回覆列：根因一句、依據位址、改了哪幾行。

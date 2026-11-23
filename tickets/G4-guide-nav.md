# G4 實作：攻略面板改 F10 式導覽（上下換層、左右翻頁）

驗收: bash hook/ddraw/guide.test.sh && bash hook/ddraw/menu.test.sh && bash hook/ddraw/wrap.test.sh && bash hook/ddraw/atlas-cover.sh && bash scripts/check-sigs.sh && bash hook/ddraw/build.sh && bash hook/ddraw/check.sh
規模: 小（guide.h 兩個純函數＋proxy.c 按鍵與排版）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 hook/ddraw/guide.test.sh 與其他既有測試、scripts/；不改白名單外任何檔；不寫入遊戲目錄；不執行遊戲；不下載任何東西；不改既有 hook 的位址與簽章；不加 CRT 或第三方依賴；不放寬或跳過測試；不動 tools/atlas 與 out/atlas.h 的既有陣列；筆記不貼反組譯碼或反編譯碼；repo 不放攻略內容、原版 bytes 或從 exe 讀出的表
白名單: hook/ddraw/guide.h、hook/ddraw/proxy.c、notes/re/menu.md
預期紅: guide-rule
執行器: codex

## 背景
- G3（0a748bf）的 F7 是「按一次多一層、第四次關」，第 3 層內容超過畫面就截斷（實機 Arnika 33 列，log `guide truncated`）。
- MG 2026-10-14：「F10 的設計比較優雅。上下就是項目，左右就是調整。所以其實三層可以是下下兩次開到底，左右就是翻頁了」「開出攻略視窗後，上上就是剩下一層」。
- 跑驗收前 `export WIZ8_EXE=$PWD/build/Wiz8.exe SYS_DDRAW=$PWD/build/sys-ddraw.dll WIZ8_FONT=$PWD/build/fonts/NotoSansTC-Regular.otf WIZ8_PXFONT=$PWD/build/fonts/fusion-pixel-12px-proportional-zh_tw.ttf`（build/ 已備好副本）。
- 現行程式：`proxy.c` `menu_input`（guide 分支）、`guide_layout`、`guide_draw`、`guide_log_key`、`guide_close`；`guide.h` `guide_key`。

## 要做
1. `guide.h`：以 `guide_nav`／`guide_paginate`／`GUIDE_*` 鍵碼取代 `guide_key`，`GuideState` 加 `page`；規則以 `guide.test.sh` 檔頭與斷言為準。`guide_select`、`guide_wrap` 不變。
2. `proxy.c` 按鍵（`menu_input` 的 guide 分支）：
   - F10 選單沒開時：熱鍵→`GUIDE_TOGGLE`；面板開著時 Esc→`GUIDE_ESC`、↑↓←→→對應鍵、其他鍵 `GUIDE_OTHER`。`guide_nav` 回 1 才吃鍵（沿用 `menu_eaten`／`menu_chars` 與 auto-repeat 處理：方向鍵按住重複要能連續翻頁／換層，熱鍵與 Esc 的 repeat 照舊吃掉不作用）。
   - 面板開著時方向鍵被面板吃掉（不能用方向鍵走路），其他鍵照常交給遊戲。關著時一切照舊交給遊戲。
   - 開啟（關→開）時重讀 tsv，同 G3。F10 開選單時關攻略面板，同 G3。
   - 換層或翻頁後重排版；`npages` 取目前層的分頁結果。
3. 排版：內容＝第 1 層到目前層累加（同 G3，層間空半行＝下一層第一行的 step 多 8）。所有行先算 step（一般 16，換層首行 24），用 `guide_paginate` 依可用高度（面板內標題下方到底邊）分頁，只畫 `page` 那一頁，頁首行從標題下方開始（頁首的層間距不另加空白也可，交付寫明）。只有超過 1026 行上限才 log `guide truncated`。
4. 標題：`<地圖表名> #<索引>  第 L/3 層  第 P/N 頁  上下：層  左右：翻頁  <熱鍵> 關`（P 從 1 起算；字都在 atlas）。索引 −1、「這張圖沒有攻略」行為同 G3。
5. log：每次被面板吃掉的鍵一行 `guide map=<索引> name=<名> layer=<層> page=<P>/<N> lines=<列數>`；不吃的鍵不 log。
6. `notes/re/menu.md`「G3 攻略面板」節更新按鍵與 log 行；寫「G4」小節。

## 驗收（CI 驗不到的部分）
cc 裝進遊戲資料夾，MG 實機：F7 開、下下到第 3 層、左右翻完 Arnika、上上回第 1 層、Esc／F7 關；面板開著時非方向鍵仍能操作遊戲。

## 交付回覆
改了哪些函式；頁首層間距的處理；規格疑慮段，沒有就寫「無」。

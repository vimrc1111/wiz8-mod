# G3 實作：F7 攻略面板（自用；內容在 repo 外）

驗收: bash hook/ddraw/guide.test.sh && bash hook/ddraw/menu.test.sh && bash hook/ddraw/wrap.test.sh && bash hook/ddraw/atlas-cover.sh && bash scripts/check-sigs.sh && bash hook/ddraw/build.sh && bash hook/ddraw/check.sh
規模: 中（純函數＋既有 F10 疊加路徑複用＋ASCII 字形）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 hook/ddraw/guide.test.sh 與其他既有測試、scripts/；不改白名單外任何檔；不寫入遊戲目錄；不執行遊戲；不下載任何東西；不改既有 hook 的位址與簽章；不加 CRT 或第三方依賴；不放寬或跳過測試；筆記不貼反組譯碼或反編譯碼；repo 不放攻略內容、原版 bytes 或從 exe 讀出的表
白名單: hook/ddraw/guide.h、hook/ddraw/proxy.c、hook/ddraw/build.sh、tools/atlas/atlas.go、tools/atlas/main.go、notes/re/menu.md
預期紅: guide-rule
執行器: codex

## 背景
- MG 2026-10-14 要內建攻略：在遊戲裡按 F7 顯示目前地圖的攻略，按一次多一層（1 區域與主線 → 2 NPC／商店／關鍵物品 → 3 秘密與解法），第四次關閉。
- 內容來自官方攻略本，**不入 repo、不入發布包**：MG 自己放在遊戲資料夾 `wiz8mod-guide.tsv`（與 `wiz8mod.ini` 同目錄，同 `speed_init` 的取法）。沒有檔案＝按 F7 什麼都不畫，只記 log。
- 目前地圖索引（`notes/re/misc.md:65`）：`u32 [0x00686a70]`，無地圖 `0xffffffff`；寫入點 `0x0042a7b5` 開頭是 `mov ds:0x686a70,esi`（`89 35 70 6a 68 00`）。地圖表 `0x00604478`，每筆 107 bytes，+50 是 ASCII 地圖名（NUL 結尾，最長 50），索引上限 47。
- 疊加沿用 F10 選單（`notes/re/menu.md`「呈現路徑」，`proxy.c` `menu_end_frame`／`menu_render`／`menu_text`），按鍵沿用 `menu_keyboard`＋`menu_wndproc`。
- 跑驗收前 `export WIZ8_EXE=$PWD/build/Wiz8.exe SYS_DDRAW=$PWD/build/sys-ddraw.dll WIZ8_FONT=$PWD/build/fonts/NotoSansTC-Regular.otf WIZ8_PXFONT=$PWD/build/fonts/fusion-pixel-12px-proportional-zh_tw.ttf`（build/ 已備好副本）。材料：`build/wiz8.asm`、`build/Wiz8.exe`（檔內偏移＝VA−0x400000）、`build/sr.dll`。

## 要做
1. `guide.h`（純函數，主機 gcc 可編，不 include windows.h、不用 libc）：介面與規則以 `guide.test.sh` 檔頭與斷言為準；斷行用 `wrap.h` 的 `wrap_can_break`／`wrap_space`。
2. `proxy.c`：
   - 熱鍵 `[guide] hotkey_vk`（同 `wiz8mod.ini`），缺則 0x76（F7），1–255 外退回 F7；啟動 log `guide hotkey_vk=<值>`。
   - 先查 F7 遊戲有沒有用：`build/wiz8.asm` 有 `cmp …,0x76`（0x4f9b1c、0x4fb6bc、0x4fc05c、0x528362、0x532883 等），判斷是不是按鍵處理；結論寫進 `notes/re/menu.md`（位址＋用途，不貼碼）。遊戲有用就照樣預設 F7，交付回覆寫明衝突。
   - 按鍵：熱鍵與 Esc 由攻略面板吃掉（Esc 只在面板開著時吃）；面板開著時其他鍵照常交給遊戲（可以邊走邊看）。F10 選單開著時 F7 不作用；按 F10 開選單時先關掉攻略面板。兩者共用 `menu_cs`。
   - 簽章：`0x0042a7b5` 的 16 bytes 照既有慣例登記在 `notes/re/menu.md` 表格，build.sh 加 `M:map:0x0042a7b5`（產進 hooks.h；`check-sigs.sh` 要綠），對不上 log `sig-mismatch guide 0x0042a7b5`，面板停用（不讀 0x686a70）。只讀不寫遊戲記憶體。
   - 開啟（0→1）時重讀 tsv（MG 改檔不用重開遊戲）：`CreateFileA`＋`ReadFile`，上限 1 MiB，超過或讀不到當作沒檔；UTF-8 轉 UTF-16 用 `MultiByteToWideChar`。之後 2、3 層用同一份快取。
   - 每次按鍵 log `guide map=<索引> name=<地圖表名> layer=<層> lines=<列數>`（MG 實機對照地圖索引用；每次按鍵一行，不是每幀）；沒檔 log `guide no tsv <路徑>` 一次。
   - 畫面：沿用 `menu_render` 的 lock／row／unlock 流程（同一個 endFrame 包裝，不另掛 hook）。框在 (16,16)，寬 min(surface 寬−32, 600)，高依行數、上限 surface 高−32；行高 16。
     第一行標題：`<地圖表名> #<索引>  第 N/3 層  F7 下一層`；索引 −1 時標題寫「不在地圖上」，tsv 沒有該圖寫「這張圖沒有攻略」。
     內容：`guide_select`（upto＝目前層）→ 每列 `guide_wrap`（advance 與實際繪製一致）→ 依序畫；放不下時最後一行畫「…」並 log 一次 `guide truncated`。層與層之間空半行。
   - ASCII 字形：現在 `menu_ascii` 只有 20 個字，攻略有大量英文（人名、地名、物品）。改成 0x20–0x7e 全部可畫：`tools/atlas` 從同一個 Fusion Pixel 字型另外輸出 ASCII 字形陣列（新名稱，例如 `ascii_bits`／`ascii_adv`）到 `out/atlas.h`。**既有 `atlas_chars`／`atlas_bits`／`atlas_off`／`atlas_cell` 的內容一個 byte 都不准變**（遊戲內文字 hook 用的是它們）：交付前把改前、改後 `out/atlas.h` 裡這四個陣列抽出來比對，結果寫進交付回覆。`menu_text` 改用新 ASCII 字形（F10 選單一起受益），移除 `menu_ascii`。
3. `notes/re/menu.md` 新增「G3 攻略面板」一節：熱鍵、F7 衝突結論、tsv 路徑與格式、簽章表、log 行、已知限制。

## 已知限制（不用處理）
- 一張圖一份攻略；Arnika 城內不同建築、同圖多區不細分。
- 內容 tsv 由 cc 另做（攻略本 OCR＋翻譯），本票只用 `guide.test.sh` 的形狀。

## 驗收（CI 驗不到的部分）
cc 把 `~/.local/state/wiz8-mod-ref/guide/guide-zh.tsv` 放進遊戲資料夾，實機 run：log 有 `guide hotkey_vk=`、無 sig-mismatch；按 F7 與畫面由 MG 試玩。

## 交付回覆
F7 衝突結論；四個 atlas 陣列比對結果；改了哪些函式；規格疑慮段，沒有就寫「無」。

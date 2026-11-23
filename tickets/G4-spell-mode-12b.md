# G4 驗 4fa5cc 豁免前提：法術記錄 +0x12b 模式只有 0..3（goals/stage14.md，後續票 4b）

驗收: { WIZ8_DBS=build/data-x/DATABASES go test -count=1 -v -run 'TestSpellMode12bDomain$' ./internal/calc 2>&1 | grep -q -- '--- PASS: TestSpellMode12bDomain' || { echo 'FAIL TestSpellMode12bDomain'; false; }; }
規模: 小
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔（含 re/emu/emu.py、scripts/node-accept.sh、scripts/nodes/ 既有檔，除非白名單列出）；不刪 boundary 既有列；不縮 random 範圍或輸入域閃差異；不放寬任何驗證；不寫入遊戲目錄；不執行遊戲或 exe；筆記與程式註解不貼組語或反編譯碼；不把原版 bytes、從 exe／DBS 讀出的表寫進 repo
白名單: internal/calc/spell12b_test.go, notes/re/misc.md:77
預期紅: TestSpellMode12bDomain
執行器: muse

## 背景
- `notes/re/cov-exempt.d/4fa4d0.tsv:1` 豁免 0x4fa5cc：理由是「法術資料的模式（記憶體記錄 +0x12b）只有 0..3」，但沒用實際資料驗過。
- 記憶體中的法術表：0x65be1c 指向的陣列，stride 0x1bf，由 0x4acc.. 一帶的載入函式從 SPELLTABLES.DBS 建出（檔頭 150／1）。DBS 不是記憶體記錄的連續拷貝（`internal/calc/load.go:38` LoadSpells 的 DBS 記錄長 704 bytes，`round_test.go:257`）。

## 要做的
1. 在 `build/wiz8.asm` 找出載入函式把 DBS 哪個欄位寫進記憶體 +0x12b（附行號），寫進 `notes/re/misc.md:77`（`## 發現` 段每條附 `build/wiz8.asm:<行>`，只寫位址與自己的話，不貼組語）。
2. `internal/calc/spell12b_test.go`：`TestSpellMode12bDomain`，WIZ8_DBS 未設就 t.Skip；設了就讀 SPELLTABLES.DBS，依步驟 1 的對應取出每筆法術的模式值，印出直方圖（t.Logf），任何一筆不在 0..3 就 t.Errorf 指名法術索引與值。可重用 LoadSpells 的讀檔方式，不改 load.go。
3. 結果不論 0..3 是否成立都照實寫進 spell-12b.md；不成立時測試保持紅，不要改成綠（由 cc 開下一張票修豁免）。
4. 四段注入：把允許集合改成 0..2 → 若資料有 3 則 FAIL → 副本還原 → PASS；資料沒有 3 就改用其他會紅的注入並說明。

## 規格疑慮
寫在最終回覆；沒有就寫「無」。

# M 實作：tools/dbsname（物品名 ITEMS.DBS／怪物名 MONSTERS.DBS 的 decode／encode，接進兩條建置路）

驗收: bash scripts/dbsname.test.sh && bash scripts/zhpatch.test.sh && go test -count=1 ./...
規模: 中（codex gpt-6.1-sol；Go 標準庫；照 internal/fact 的形狀做一份定長記錄名稱欄 codec）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 scripts/dbsname.test.sh、scripts/zhpatch.test.sh、任何 *_test.go 與白名單外任何檔；不加第三方依賴；不寫入遊戲目錄；不執行遊戲；不放寬或跳過測試；不貼遊戲原文
白名單: internal/dbsname/dbsname.go、tools/dbsname/main.go、tools/zhpatch/zhpatch.go、scripts/build-zh-patch.sh、scripts/check-lib.sh
預期紅: dbsname
執行器: codex

## 目標
MG 要試做物品名、怪物名中文化（先 5＋5 筆實機看列表擠不擠）。兩個 DBS 是定長記錄、名稱欄是定寬 UTF-16LE（`notes/research/2026-10-07-R6-dbs-msg-format.md` 附錄 A）：
- `DATABASES\ITEMS.DBS`：u32 筆數＋筆數×`0x10d`；欄位位移 `0`，寬 32 units。kind＝`items`。
- `DATABASES\MONSTERS.DBS`：u32 筆數＋筆數×`0x297`；欄位位移 `0x0/0x30/0x60/0x90`，各寬 24 units（單數、複數、未辨識單數、未辨識複數）。kind＝`monsters`。

## 做法
- `internal/dbsname`：`DecodeFile(kind, in string, out io.Writer)`、`EncodeFile(kind, in, out string)`。JSON 形狀照 `internal/fact`：`{"records":[{"raw":"<整筆 hex>","text":["…"]}]}`，text 每筆 1（items）或 4（monsters）欄。
  - text＝欄內首個 NUL 之前的 units（surrogate 檢查同 fact）；欄內沒有 NUL 就報錯。
  - encode：text 與 raw 解出的原文相同 → raw 原樣（NUL 後的殘留位元組不動）；不同 → 該欄清零後寫入新 units。新字串須 ≤ 寬−1 units（留結尾 NUL），超長、內含 NUL、筆數或長度不符就報錯且**不建立／不截斷輸出檔**。
  - 檔長要等於 `4＋筆數×stride`。
- `tools/dbsname/main.go`：`dbsname decode <items|monsters> <in>`（stdout JSON）、`dbsname encode <items|monsters> <in.json> <out>`；CLI 形狀照 `tools/fact/main.go`。
- 譯文 id：items＝記錄序號；monsters＝4×記錄序號＋欄位（0–3）。`scripts/build-zh-patch.sh`（新增 `*ITEMS.DBS`／`*MONSTERS.DBS` 分支，照 FACT 分支）與 `tools/zhpatch/zhpatch.go`（`translate` 的 switch 加同樣兩種）語義一致，sha 守衛同既有。
- `scripts/check-lib.sh`：同兩個分支，原文抽成攤平陣列（順序＝id）。**試做期允許部分翻譯**：譯文檔有列的 id → `ui`；沒列的 → `keep\t名稱未譯（試做）`。原文空字串 → `keep\t原文空`。
- 現有 `translations/lib.tsv` 不動（試做的譯文檔由 cc 另行加入）；驗收用 fixture 譯文目錄。

## 參考
`internal/fact/fact.go`（codec 形狀範本）、`scripts/build-zh-patch.sh` FACT 分支、`scripts/check-lib.sh` FACT 分支、`tools/zhpatch/zhpatch.go:65` `translate`。

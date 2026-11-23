# N 實作：tools/dbsname 加 kind `spells`（法術名 SPELLTABLES.DBS）

驗收: bash scripts/dbsname.test.sh && bash scripts/zhpatch.test.sh && go test -count=1 ./...
規模: 小（codex gpt-6.1-sol；Go 標準庫；在票 M 的 internal/dbsname 上加一種版面）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 scripts/dbsname.test.sh、scripts/zhpatch.test.sh、任何 *_test.go 與白名單外任何檔；不加第三方依賴；不寫入遊戲目錄；不執行遊戲；不放寬或跳過測試；不貼遊戲原文
白名單: internal/dbsname/dbsname.go、tools/dbsname/main.go、tools/zhpatch/zhpatch.go、scripts/build-zh-patch.sh、scripts/check-lib.sh
預期紅: dbsname
執行器: codex

## 目標
MG 要先試做法術名中文化（法術清單行距 13，開遊戲即可看）。`DATABASES\SPELLTABLES.DBS`（`notes/research/2026-10-07-R6-dbs-msg-format.md` 第 16–17、55 行）：
- 檔頭 8 bytes：u32 筆數（150）＋u32 版本（1）。之後筆數×`0x2c0`。
- 名稱欄位移 `0x19c`，寬 64 units UTF-16LE。kind＝`spells`，每筆 1 欄。

## 做法
- `internal/dbsname`：版面加檔頭長度。items／monsters 檔頭＝4（只有筆數），spells＝8。
  - 筆數後的檔頭 bytes（spells 的版本欄）要原樣往返：JSON 頂層加 `"head":"<hex>"`（筆數之後、第一筆之前的 bytes）；items／monsters 的 head 為空，JSON 可省略該欄（`omitempty`），既有 items／monsters 輸出不變。
  - encode：head 長度不符版面 → 報錯且不寫出。檔長檢查改為 `檔頭＋筆數×stride`。
  - 其餘規則（NUL、超長、raw 保留、錯誤不寫出）同票 M。
- CLI usage 字串加 `spells`。
- 譯文 id：spells＝記錄序號。`scripts/build-zh-patch.sh` 加 `*SPELLTABLES.DBS` 分支、`tools/zhpatch/zhpatch.go` `translate` 加同一種，語義同 items。
- `scripts/check-lib.sh`：`*SPELLTABLES.DBS` 同 items 分支（試做期部分翻譯：有列 → `ui`，沒列 → `keep\t名稱未譯（試做）`，原文空 → `keep\t原文空`）。
- `translations/lib.tsv` 不動（cc 另加）。

## 參考
票 M 實作（commit 054e22c）：`internal/dbsname/dbsname.go` 的 `kindLayout`、`scripts/build-zh-patch.sh` ITEMS 分支、`scripts/check-lib.sh` ITEMS／MONSTERS 分支、`tools/zhpatch/zhpatch.go` ITEMS 分支。

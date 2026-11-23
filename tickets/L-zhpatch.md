# L 實作：tools/zhpatch（玩家端安裝：原版＋譯文 → PATCH.002，單一執行檔）

驗收: bash scripts/zhpatch.test.sh && go test -count=1 ./...
規模: 中（codex gpt-6-astra medium；Go 標準庫；把既有工具的邏輯抽成可共用的 internal 套件，再組成一支）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 scripts/zhpatch.test.sh、任何 *_test.go、scripts/*.sh 與白名單外任何檔；不加第三方依賴（只用 Go 標準庫）；不寫入遊戲目錄；不執行遊戲；不放寬或跳過測試；不貼遊戲原文
白名單: tools/zhpatch/main.go、tools/zhpatch/zhpatch.go、internal/slf/slf.go、internal/stringdata/stringdata.go、internal/strlib/strlib.go、internal/nsf/nsf.go、internal/fact/fact.go、internal/glossary/glossary.go、tools/fact/fact.go、tools/fact/main.go、tools/nsf/main.go、tools/nsf/nsf.go、tools/slf/main.go、tools/slf/slf.go、tools/strings/main.go、tools/strings/strings.go、tools/strlib/main.go、tools/strlib/strlib.go、tools/trcheck/main.go、tools/trcheck/trcheck.go
預期紅: zhpatch
執行器: codex

## 目標
發布形態是「玩家自備原版＋套用我們的 ddraw.dll 與譯文」：PATCH.002 是原文衍生物，不能隨發布包散布，要在玩家機器上用他自己的原版產生。現在的產生器 `scripts/build-zh-patch.sh` 要 bash、jq、Go，玩家（Windows）沒有。
做 `tools/zhpatch`：`zhpatch <遊戲目錄> <translations 目錄> <輸出 PATCH 檔>`，產物與 `scripts/build-zh-patch.sh` **逐 byte 相同**（不跑 check-translations／check-lib 的譯文機檢；sha 守衛要有，見下）。要能 `GOOS=windows go build`。

## 做法
- 照 `scripts/build-zh-patch.sh` 的每一步實作（讀 PATCH.001 的 STRINGS\STRINGDATA.DAT、套 translations/stringdata.json；依 translations/lib.tsv 逐筆從 DATA.SLF 取原檔、套譯文、encode；manifest 時戳沿用 STRINGDATA、檔表依小寫檔名排序；pack）。術語代號 `{{Term}}` 展開＝`trcheck expand` 的語義（translations/glossary.tsv）。
- 既有工具都是 `package main`，不能互相 import：把需要的 decode／encode／pack／expand 邏輯搬到白名單裡的 internal 套件（STRINGDATA 的套件叫 `internal/stringdata`，避開標準庫 strings；術語展開叫 `internal/glossary`）；既有 `*_test.go` 呼叫的函式名在原工具保留（可改成薄包裝），原工具的 main 改成呼叫 internal（各工具的 CLI 行為與既有 `*_test.go` 不變，`go test ./...` 全綠）。不要複製第二份邏輯。
- 各工具既有的 JSON 中介形狀（`strings`、`strlib`、`nsf`、`fact` 的 decode 輸出）只是 shell 管線用的；zhpatch 內部可直接操作 Go 結構，不必經 JSON，只要產物逐 byte 相同。

## sha 守衛（拒絕＝exit 2、stderr 說明、**不寫出輸出檔**）
每筆譯文的 `src_sha256` 要等於原版對應原文（UTF-8）的 sha256；任何一筆不符就整個拒絕（玩家原版版本不同或筆號錯）。範圍：stringdata.json、lib.tsv 列出的每一檔（字串庫、MSG、NSF、FACT）。id 規則：strlib＝entries 序號；NSF＝依檔內順序攤平的 text chunk 序號（見 build-zh-patch.sh 的 NSF 分支）；FACT＝2×記錄序號＋欄位。
輸出先寫暫存檔，全部成功才改名到輸出路徑。

## 參考
`scripts/build-zh-patch.sh`（每一步的權威）、`scripts/check-lib.sh`（id 規則與原文抽取）、`notes/zh-patch-log.md:127`（PATCH 編號覆蓋）。

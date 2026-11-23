# K 實作：tools/fact（事件資料庫 FACT.DBS decode／encode；日記事件文字）

驗收: go test ./tools/fact
規模: 小（Go 標準庫，單一工具；定長記錄）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 tools/fact/fact_test.go 與白名單外任何檔；不加第三方依賴（只用 Go 標準庫）；不寫入遊戲目錄；不執行遊戲；不放寬或跳過測試；不貼遊戲原文（單字級可寫）
白名單: tools/fact/main.go、tools/fact/fact.go
預期紅: TestFixtureDecode、TestFixtureRoundTrip、TestEncodeChanges、TestRealFact
執行器: codex

## 目標
把 `tools/fact/main.go` 的 `decodeFile`／`encodeFile`（現在回「未實作」）做完，讓 `tools/fact/fact_test.go`（驗收測試，cc 寫，不准改）全綠。程式可拆到 `tools/fact/fact.go`。風格照 `tools/strlib/main.go`。

## 格式（cc 從原檔解出；807 筆全數驗過）
- 檔頭 u32 n；之後 n 筆定長 472 bytes 記錄；檔長＝4＋n×472。
- 記錄：+0 u32 序號、+4 名稱（ASCII，52 bytes）、**+56 文字欄 0**（200 bytes）、**+256 文字欄 1**（200 bytes）、+456 12 bytes 0、+468 u32。
- 文字欄：UTF-16LE，到第一個 u16 NUL 為止；NUL 之後可能有殘渣（舊緩衝區），名稱欄同。工具只解讀兩個文字欄，其餘位元組不解讀。

## 輸出形狀
`fact decode <in>`：stdout 輸出 JSON `{"records":[{"raw":"<472 bytes 小寫 hex>","text":["欄 0","欄 1"]}, ...]}`。
`fact encode <in.json> <out>`：寫 u32 記錄數＋各筆：以 raw 為底；某欄的 text 與 raw 該欄解出的字**相同**→該欄原樣（含殘渣）；**不同**→該欄寫新字的 UTF-16LE、其餘補 0 到 200 bytes。

## decode 拒絕（exit 2＋error，stdout 不輸出任何東西——先全部解完再寫）
不到 4 bytes；檔長≠4＋n×472；文字欄 200 bytes 內沒有 NUL；NUL 前有落單的代理對。

## encode 拒絕（exit 2，且不寫出檔案）
不是物件、沒有記錄；raw 不是 hex 或不是 472 bytes；text 不是恰好 2 段；某段含 U+0000；某段 UTF-16 超過 99 units（要留 NUL）。

## 計數來源
cc 用一支 Python 掃描（不在 repo）照上面格式解：807 筆、欄 0 非空 264、欄 1 非空 73、最長 57 units、全部 1614 段以 `\n` 接起的 sha256 寫在測試裡。
另核對：MG 實機 text log 裡日記字型（slot 1）畫出的 148 句英文全在這兩欄內（`local_data_check`）。

## 環境
- 遊戲目錄唯讀掛在 `/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8`；TestRealFact 從 `Data/DATA.SLF` 讀，會跑。
- 不用網路、不用 go get。

## 交付回覆
最後回覆附「規格疑慮」段：規格哪裡不清楚、測試哪裡你認為前提錯了；沒有就寫「無」。

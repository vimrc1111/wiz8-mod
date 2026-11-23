# J 實作：tools/nsf（NPC 對白腳本 NSF decode／encode）

驗收: bash scripts/ci-check.sh
規模: 中（Go 標準庫，單一工具；格式有未解欄位，靠文法辨識文本）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 tools/nsf/nsf_test.go 與白名單外任何檔；不加第三方依賴（只用 Go 標準庫）；不寫入遊戲目錄；不執行遊戲；不放寬或跳過測試；不貼反編譯碼或遊戲原文（單字級關鍵字可寫）
白名單: tools/nsf/main.go、tools/nsf/nsf.go
預期紅: TestFixtureDecode、TestFixtureRoundTrip、TestEncodeResizes、TestRealAll
執行器: codex

## 目標
把 `tools/nsf/main.go` 的 `decodeFile`／`encodeFile`（現在回「未實作」）做完，讓 `tools/nsf/nsf_test.go`（驗收測試，cc 寫，不准改）全綠。程式可拆到 `tools/nsf/nsf.go`。風格照 `tools/strlib/main.go`。

## 輸出形狀（工具只抽「玩家看得到的字」，其餘原樣保留）
`nsf decode <in>`：stdout 輸出 JSON `{"segments":[...]}`，每段恰好一種：
- `{"raw":"<小寫 hex>"}`：原樣位元組（未解欄位、前導、記錄全在這裡）。相鄰 raw 必須合併成一段。
- `{"text":["chunk", ...]}`：文本記錄＝u8 chunk 數 m（1–4）＋m 個 chunk（u16 unit 數＋UTF-16LE units，無 NUL）。段包含那個 m 位元組。
- `{"kw":"MARTEN"}`：關鍵字定義的長度＋字串部分＝u16 位元組數＋可列印 ASCII。前面 13 位元組前導（K、quad、u32 0、quad）留在前一個 raw。
全部段依序串接＝原檔。

`nsf encode <in.json> <out>`：照段寫出，text 依 chunk 數寫 m、依 UTF-16 長度寫每個 u16；kw 寫 u16 長度。無位移表要重算（R7）。

## 格式依據
`notes/research/2026-10-07-R7-nsf-format.md` 附錄 A（muse 研究，cc 抽查：165 檔檔頭、位移 26 首文本、附錄 B 三檔錨點吻合）。重點：
- 檔頭 u32 0xFFFFFFFF＋u32 count；位移 26 恆為第一段文本（之前 26 bytes 為 raw）。
- 文本前通常是 12 位元組節點前導，第 1 位元組 MM＝文本的 m。
- 檔內無對齊；unit 只見 0x09、0x0A、0x20–0x7E、0x0092。
- 關鍵字定義：`u8 K(1–3), u32 A, u32 0, u32 B, u16 n, u8[n]`。

## 計數來源（cc 獨立掃描，與 R7 的差異）
cc 用一支 Python 掃描（不在 repo）：從位移 26 起，遇「b[p]∈1–4 且 b[p+12]＝b[p]、p+12 起是合法文本」收文本；遇「b[p]∈1–3、b[p+5:p+9] 全 0、n≥2 的可列印 ASCII」收關鍵字；否則前進 1 byte。結果：
- 文本 10320（R7 寫 10306＋14 例前導尾變體，吻合）。
- 關鍵字 4590。R7 寫 4594；差的 4 筆是 n＝1 的誤判（quad 的低位元組恰好可列印：NPC_CONSTABLE 8432、NPC_MADRAS 24090、NPC_SCREGHOLO 9551、RPC_DRAZIC 19604），其中 RPC_DRAZIC 那筆會吞掉 0x4cad 的文本前導。
- 全部 6 unit 以上的 UTF-16 可列印連續段都落在文本內（R7 的「31 段無前導第二文本」在此掃描下都被前導規則收到，R7 的說法未證實）。
不必照這個掃描法寫，測試只驗結果。

## decode 拒絕（exit 2＋error，stdout 不輸出任何東西——先全部解完再寫）
太短（不到 26 bytes）、魔數不是 0xFFFFFFFF、位移 26 不是完整合法文本（含 chunk 越界）。

## encode 拒絕（exit 2，且不寫出檔案）
不是物件；某段不是恰好一種欄位；raw 不是 hex；text 不是 1–4 個 chunk、chunk 空字串、chunk 編碼後超過 65535 units；kw 空或含非 ASCII 可列印字元、超過 65535 bytes。

## 環境
- 遊戲目錄唯讀掛在 `/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8`；真實檔測試從 `Data/DATA.SLF` 讀 165 個 NSF，會跑。
- 不用網路、不用 go get。

## 交付回覆
最後回覆附「規格疑慮」段：規格哪裡不清楚、測試哪裡你認為前提錯了（例：計數 10320／4590 若你的解析有不同而有理由，列出差異的檔名＋位移，不要改程式湊數字）；沒有就寫「無」。

# I 實作：tools/strlib（索引字串庫 decode／encode：物品／法術描述 DBS＋全部 MSG）

驗收: bash scripts/ci-check.sh
規模: 小（Go 標準庫，單一工具）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 tools/strlib/strlib_test.go 與白名單外任何檔；不加第三方依賴（只用 Go 標準庫）；不寫入遊戲目錄；不執行遊戲；不放寬或跳過測試；不貼反編譯碼或遊戲原文
白名單: tools/strlib/main.go、tools/strlib/strlib.go
預期紅: TestFixtureDecode、TestFixtureRoundTrip、TestEncodeRecomputesOffsets、TestRejectsMalformed、TestRealDesc、TestRealMsg
執行器: codex

## 目標
把 `tools/strlib/main.go` 的 `run` 骨架（現在回「未實作」）做完，讓 `tools/strlib/strlib_test.go`（驗收測試，cc 寫，不准改）全綠。程式可拆到 `tools/strlib/strlib.go`。風格照 `tools/strings/main.go`（`run(args, out) (int, error)`、exit 0／2、錯誤訊息帶檔名）。

## 格式（cc 以 Wiz8.exe 讀取器 0x0052ff80 與 54 個真實檔核對；取代 R6 報告「MSG 段後為腳本尾」的說法——那是位移表）
小端序。
- 檔頭 5 bytes：u32 版本（實檔皆 1，照抄不檢）＋u8 旗標。旗標 0＝明文 UTF-16LE；1＝每個 unit 存 `0x9696 − c`（mod 2^16）；其他值拒絕。
- 接著 N 筆，緊密相接：u32 a、u32 b（元資料，照抄）、u32 unit 數 n、n 個 u16。n＞0 時最後一個 unit（解碼後）必為 NUL，其餘不得為 NUL。n＝0 與 n＝1（只有 NUL）是兩種不同的「空」，往返要保留。n＞2000 拒絕（遊戲讀取器上限）。
- 尾段兩種形狀（`tail`）：
  - `dbs`：N 個 u32 絕對位移（第 i 個＝第 i 筆 a 的檔內位移）、u32 N、u32 0。
  - `msg`：u32 0xffffffff、N 個 u32 絕對位移、u32 N、u32 E（E＝最後一筆結束處＝那個 0xffffffff 的位移）。
- 檔案剛好用完。decode 時位移鏈必須逐筆吻合（第 i 個位移＝實際位置，最後一筆結束處＝尾段起點），不吻合即拒絕。

## 行為規格
- `strlib decode <in>`：stdout 輸出 JSON `{"header":"<5 bytes 小寫 hex>","tail":"dbs"|"msg","entries":[{"a":u32,"b":u32,"text":"<去掉結尾 NUL 的字串>","nul":bool}]}`。`nul`＝該筆有結尾 NUL。
- decode 拒絕（exit 2＋error，stdout 不輸出任何東西——先全部驗完再寫）：太短；旗標非 0／1；筆數、位移、n 越界（int64 算）；位移鏈不吻合；尾段不是上面兩種形狀之一；字串內部 NUL；n＞0 而末 unit 非 NUL；孤立 surrogate；n＞2000。
- `strlib encode <in.json> <out>`：照 header 旗標決定是否 0x9696 變換，依 `tail` 重算全部位移（msg 另重算 E）。拒絕：不是物件；header 不是 5 bytes hex 或旗標非 0／1；tail 不是 dbs／msg；a／b 超出 u32；text 含 NUL；text 非空但 nul＝false；編碼後 n＞2000。
- 真實檔（`$WIZ8_DIR/Data/DATA.SLF`）：`DATABASES\ITEMDESC.DBS`、`SPELLDESC.DBS`、`SPELLEFFECT.DBS`（tail dbs，旗標 0），51 個 `*.MSG`（tail msg，旗標 1），`encode(decode(x))` 都必須與 x byte 相同。

## 環境
- 遊戲目錄唯讀掛在 `/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8`；真實檔測試會跑。
- 不用網路、不用 go get。

## 交付回覆
最後回覆附「規格疑慮」段：規格哪裡不清楚、測試哪裡你認為前提錯了（要回報，不要遷就改程式湊測試）；沒有就寫「無」。

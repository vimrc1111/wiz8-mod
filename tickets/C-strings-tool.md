# C 實作：tools/strings（STRINGDATA.DAT 的 decode／encode）

驗收: bash scripts/ci-check.sh
規模: 小（Go 標準庫，單一工具）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 tools/strings/strings_test.go 與白名單外任何檔；不加第三方依賴（只用 Go 標準庫）；不寫入遊戲目錄；不放寬或跳過測試
白名單: tools/strings/main.go、tools/strings/strings.go
預期紅: TestFixtureDecode、TestFixtureRoundTrip、TestEncodeFromJSON、TestRealRoundTrip、TestRealKnownStrings
註：TestRejectsMalformed 在骨架上是假綠（骨架什麼都拒絕）；實作後仍須綠。
執行器: codex

## 目標
把 `tools/strings/main.go` 的 `run` 骨架（現在回「未實作」）做完，讓 `tools/strings/strings_test.go`（驗收測試，cc 寫，不准改）全綠。程式可拆到 `tools/strings/strings.go`。風格照 `tools/slf/main.go`（`run(args, out) (int, error)`、exit 0／1／2、錯誤訊息帶檔名）。

## 格式（cc 已在 Wiz8.exe 0x00518360／0x005300e0 靜態確認，notes/re/misc.md:34）
- 小端序。u32 筆數 N；接著 N 筆，每筆 u32 位元組長度 L＋L bytes（L/2 個 u16）。
- 每個 u16 存的是 `0x9696 − c`（mod 2^16），c 為 UTF-16 code unit。每筆最後一個 unit 解碼後必為 0（NUL）。
- 檔案剛好用完，無尾隨資料。

## 行為規格
- `strings decode <in.dat>`：stdout 輸出 JSON 字串陣列，每筆一個字串（去掉結尾 NUL，UTF-16 → UTF-8）。
- decode 拒絕（exit 2＋error、stdout 不輸出任何東西——先全部驗完再寫）：檔太短；筆數超過實際內容；L 越界（用 int64 算）；L 為奇數；L＝0（沒有 NUL）；尾隨資料；最後一個 unit 不是 NUL；字串內部有 NUL；孤立 surrogate。
- `strings encode <in.json> <out.dat>`：讀 JSON 字串陣列，照上面格式寫出（每筆補結尾 NUL）。拒絕：不是陣列、元素不是字串、字串含 NUL、非法 UTF-8。
- 真實檔：`$WIZ8_DIR/Patches/PATCH.001` 內 `STRINGS\STRINGDATA.DAT`（2412 筆，全 ASCII），`encode(decode(x))` 必須與 x byte 相同。

## 環境
- 遊戲目錄唯讀掛在 `/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8`；真實檔測試會跑。
- 不用網路、不用 go get。

## 交付回覆
最後回覆附「規格疑慮」段：規格哪裡不清楚、測試哪裡你認為前提錯了（要回報，不要遷就改程式湊測試）；沒有就寫「無」。

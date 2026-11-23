# B 實作：tools/slf（SLF／PATCH.00N 的 list／extract／pack）

驗收: bash scripts/ci-check.sh
規模: 中（Go 標準庫，單一工具）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 tools/slf/slf_test.go 與白名單外任何檔；不加第三方依賴（只用 Go 標準庫）；不寫入遊戲目錄；不放寬或跳過測試
白名單: tools/slf/main.go、tools/slf/slf.go
預期紅: TestFixtureListExtract、TestFixturePackRoundTrip、TestRealArchives、TestRealPatchPackRoundTrip
執行器: codex

## 目標
把 `tools/slf/main.go` 的 `run` 骨架（現在回「未實作」）做完，讓 `tools/slf/slf_test.go`（驗收測試，cc 寫，不准改）全綠。程式可以拆到 `tools/slf/slf.go`。風格照 `tools/hashdir/main.go`（`run(args, out) (int, error)`、exit 0／1／2、錯誤訊息帶檔名）。

## 格式（依 notes/research/2026-10-07-A1-slf-patch-format.md，已由 cc 抽查）
- 小端序。檔頭固定 0x214 bytes：+0x000 檔名（0x100，NUL 結尾零填充）、+0x100 目錄名（0x100，同）、+0x200 u32 entry 數、+0x204..+0x213 四個 u32（原樣保留）。
- 目錄在檔尾：起點 = 檔大小 − entry 數×0x118。每筆 0x118：+0x000 檔名 0x100（NUL 結尾、`\` 分隔）、+0x100 u32 絕對偏移、+0x104 u32 大小、+0x108 u32 狀態、+0x10C u64 FILETIME、+0x114 u32（原樣保留）。
- 無壓縮、無加密；資料區在 [0x214, 目錄起點)。

## 行為規格
- `slf list <檔>`：依目錄順序每筆一行 `offset<TAB>size<TAB>name`（十進位，name 原樣含 `\`）。
- 驗證（list／extract 都做，任何一項不過 → exit 2＋error，不輸出半套）：檔案 ≥ 0x214；entry 數×0x118 放得下（目錄起點 ≥ 0x214）；每筆 offset ≥ 0x214 且 offset+size ≤ 目錄起點（用 int64 算，防溢位）；彼此不重疊（排序後檢查）；檔名有 NUL 結尾。
- `slf extract <檔> <輸出目錄>`：`\` 轉成路徑分隔，寫到輸出目錄底下。**檔名是不受信任輸入**：拒絕空名、絕對路徑、`X:` 磁碟代號、任何 `..` 段、清理後不在輸出目錄內的路徑；有一筆壞就整個拒絕且不寫任何檔（先全部驗完再寫）。另寫 `<輸出目錄>/.slf.json`：檔頭名、目錄名、+0x204..+0x213 四欄、每筆（依目錄順序）的 name、state、time、+0x114 欄、原 offset。
- `slf pack <extract 出的目錄> <輸出檔>`：讀 `.slf.json`；資料依原 offset 由小到大的順序從 0x214 連續寫入（offset 重算，檔案大小以磁碟上的檔為準——改過的檔大小可變）；目錄依 `.slf.json` 的順序寫；其餘欄位原樣。未改動時 `pack(extract(x))` 必須與 x byte 相同（測試有驗 PATCH.001）。

## 環境
- 遊戲目錄唯讀掛在原路徑 `/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8` 與 `…/Wizardry8_Uninstall`；真實檔測試會跑（DATA.SLF 143 MB、SOUND.SLF 等）。extract 用串流（io.SectionReader＋io.Copy），不要整檔讀進記憶體。
- 不用網路、不用 go get。

## 交付回覆
最後回覆附「規格疑慮」段：規格哪裡不清楚、測試哪裡你認為前提錯了（要回報，不要遷就改程式湊測試）；沒有就寫「無」。

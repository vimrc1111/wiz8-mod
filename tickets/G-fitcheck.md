# G 實作：tools/fitcheck（譯文溢出離線機檢，stage5 5.3）

驗收: go vet ./tools/fitcheck && go test -count=1 ./tools/fitcheck
規模: 小（Go 標準庫，單一工具）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 tools/fitcheck/fitcheck_test.go 與白名單外任何檔；不加第三方依賴（只用 Go 標準庫）；不讀寫遊戲目錄；不放寬或跳過測試
白名單: tools/fitcheck/main.go、tools/fitcheck/fitcheck.go
預期紅: TestGood、TestOver、TestOffXCounts、TestRegistered、TestRegisteredNeedsReason、TestStale、TestUnknownChars、TestMixedASCII、TestEmptySourceOver、TestBadSTI、TestNoTable、TestTableLongerThanFont、TestExplicitLines
執行器: codex

## 背景
Wizardry 8 字串庫中文化。遊戲以 STI 點陣字型畫字，按遊戲字表把字元對應到字形；中文字由我們的 hook 畫，前進寬＝字高。
本工具在 CI 比較每條譯文與原文的像素寬，譯文較寬就可能溢出固定寬的按鈕／標籤；確認過可接受的登記在溢出表。
測試 `tools/fitcheck/fitcheck_test.go` 是規格的一部分（fixture 構造即輸入格式範例）；與本單衝突時以測試為準並在回覆列出。

## 介面
`run(args []string, out io.Writer) int`（main 呼叫它；測試直接呼叫）。

`fitcheck -exe <遊戲執行檔> -font <字型.sti> -src <原文.json> -tr <展開後譯文.json> -over <溢出表.tsv>`
- 每個問題印一行 `FAIL <規則> id=<n>: <說明>`，全部檢完才結束；有問題 rc=1。
- 全過印一行 `ok fitcheck: <譯文條數> 條；寬度內 <a>、登記溢出 <b>`，rc=0。
- 輸入檔讀不到、格式壞、STI 檔頭不是 `STCI`、執行檔找不到字表 → rc=2。

## 輸入格式
- 原文：JSON 字串陣列，索引＝id。
- 譯文：`[{"id":n,"zh":"…"}]`（已展開，無術語代號）。只檢譯文表裡有的 id。
- 溢出表 TSV：`id<TAB>理由`；`#` 開頭為註解。
- STI：64 位元組檔頭（0x00 `STCI`；0x18 u32 色數；0x1c u16 子圖數）；色數為 256 時接 768 位元組調色盤；之後每子圖 16 位元組 `offset u32, size u32, offX i16, offY i16, height u16, width u16`（little-endian）。
- 字表：執行檔裡第一段「`66 c7 00 <imm16>` 後接 `83 c0 02`」的連續序列；序列中允許夾 `83 c4 08`（略過）；遇到其他位元組就結束。第 i 個 imm16＝字形 i 對應的字元。不完整的 `66 c7 00` 開頭（後面沒接 `83 c0 02`）不算。

## 量寬
- 字在字表第 i 格：前進寬＝字形 i 的 offX＋width；i 超出子圖數 → 0。
- `\n`：0（不查表）。
- 不在字表、碼點 ≥ U+2000：中文前進寬＝全部子圖最大 height。
- 不在字表、碼點 < U+2000：照遊戲行為退回字形 0。
- 字串寬＝逐字前進寬總和（printf 記號照字面量，不代入）。

## 規則（名稱即 FAIL 行的 <規則>）
- over：譯文寬 > 原文寬，且 id 不在溢出表。說明 `zh <w>px > en <w>px`。
- lines：譯文 `\n` 數 > 原文 `\n` 數（溢出表不豁免）。說明 `zh <n> LF > en <n> LF`。
- over-reason：溢出表該列理由空白。
- stale：溢出表的 id 實際不溢出（或不在譯文表）。

## 交付回覆
附「規格疑慮」段；沒有就寫「無」。

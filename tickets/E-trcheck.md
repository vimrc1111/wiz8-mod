# E 實作：tools/trcheck（譯文表機檢＋術語代號展開，stage5 5.2）

驗收: bash scripts/ci-check.sh
規模: 小（Go 標準庫，單一工具）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 tools/trcheck/trcheck_test.go 與白名單外任何檔；不加第三方依賴（只用 Go 標準庫）；不讀寫遊戲目錄；不放寬或跳過測試
白名單: tools/trcheck/main.go、tools/trcheck/trcheck.go
預期紅: TestGood、TestClassesMissing、TestClassesBadClass、TestKeepNeedsReason、TestUntranslated、TestKeepTranslated、TestShaMismatch、TestPrintfOrder、TestPrintfCount、TestPrintfType、TestPrintfPercentEscape、TestPctLiteralNeedsRegistry、TestPctLiteralNoNewSpec、TestUnknownCode、TestBrokenCode、TestRawTerm、TestPronounForms、TestPronounTemplateDoubled、TestPronounTemplateUntranslated、TestExpand
執行器: codex

## 背景
Wizardry 8 字串庫（STRINGDATA）翻成中文。我方譯文表是 JSON，原文由 `tools/strings decode` 在本機解出（不入版控）。本工具在 CI 對全表機檢，並在建置時把術語代號展開。
測試 `tools/trcheck/trcheck_test.go` 是規格的一部分（fixture 構造即輸入格式範例）；與本單衝突時以測試為準並在回覆列出。

## 介面
`run(args []string, out io.Writer) int`（main 呼叫它；測試直接呼叫）。

`trcheck check -src <原文.json> -tr <譯文.json> -classes <分類.tsv> -glossary <術語.tsv> -pronouns <代名詞.tsv> -pct <百分號字面.tsv>`
- 每個問題印一行 `FAIL <規則> id=<n>: <說明>`，全部檢完才結束（不要第一個就停）；有問題 rc=1。
- 全過印一行 `ok trcheck: <總條數> 條；譯 <a>、不翻 <b>`，rc=0。輸入檔讀不到或格式壞 rc=2。

`trcheck expand -tr <譯文.json> -glossary <術語.tsv>` → stdout JSON 陣列 `[{"id":n,"zh":"展開後"}]`（順序同譯文表）；代號錯 rc=1。

## 輸入格式
- 原文：JSON 字串陣列，索引＝id。
- 譯文：`[{"id":n,"src_sha256":"<原文 UTF-8 的 sha256 hex>","zh":"…"}]`。zh 可含術語代號 `{{<術語英文>}}`。
- 分類 TSV：`id<TAB>class<TAB>reason`；`#` 開頭為註解。class ∈ ui｜combat｜system｜keep。
- 術語 TSV：`英文<TAB>中文<TAB>類別<TAB>索引`；`#` 開頭為註解。只用前兩欄。
- 代名詞 TSV：`模板id<TAB>第幾個 %s（1 起）<TAB>組`，組 ∈ obj｜poss｜refl；`#` 開頭為註解。
- 百分號字面 TSV：`id<TAB>理由`；`#` 開頭為註解。

## 規則（名稱即 FAIL 行的 <規則>）
- classes：0..len(原文)-1 每個 id 恰好一行；class 合法；keep 必有非空理由；id 超出範圍或重複也算。
- coverage：class≠keep 的必須在譯文表；keep 的不得在譯文表。
- sha：譯文的 src_sha256 ≠ 原文的 sha256。
- printf：格式記號＝`%[-+ #0]*[0-9]*(\.[0-9]+)?(h|l)?[diouxXcs%]`。原文與**展開後**譯文的記號序列（字面字串）必須完全相同（數量、順序、型別、`%%`）。
  例外：id 在百分號字面表 → 原文的 % 是字面，譯文的記號序列必須為空。
- code：`{{…}}` 內的詞不在術語表第一欄，或有不成對的 `{{`／`}}`。
- raw-term：把代號拿掉後的譯文裡，出現任一「中文欄≠英文欄」術語的中文欄字串（術語要走代號）。只查中文欄 ≥2 字的術語；單字術語（力、火…）常見於一般詞，不查（TestRawTermSingleCharExempt）。
- pronoun：
  - 形式：對 k=0,1,2，譯文[727+k] 必須＝譯文[724+k]＋「的」；譯文[730+k] 必須＝譯文[724+k]＋「自己」。違反的報 727+k 或 730+k；724..732 缺譯文報缺的那個。
  - 模板：代名詞表每列，模板必須有譯文（沒有報模板 id）；展開後至少有「第幾個」個 `%s`；把該組三個形式（obj＝724–726、poss＝727–729、refl＝730–732 的譯文）各代入那個 `%s`，結果含「的的」或「自己自己」就報模板 id。

## 交付回覆
附「規格疑慮」段；沒有就寫「無」。

# R9 研究：translated_Keywords.txt 機制與點擊取詞（7.3）

驗收: bash scripts/check-research.sh notes/research/2026-10-07-R9-translated-keywords.md
規模: 中（研究，一份報告）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單以外的檔；不寫入遊戲目錄；不執行遊戲或任何下載物；報告不貼反編譯碼（位址＋行號＋一句話描述）；不送 email 或真實姓名
白名單: notes/research/2026-10-07-R9-translated-keywords.md
預期紅: research:2026-10-07-R9-translated-keywords
執行器: muse

## 背景
Wizardry 8（Wiz8.exe v1.2.4，i386）中文化。NPC 對話靠關鍵字：玩家打字或在對白裡點一個詞，遊戲拿該詞去比對 NPC 的關鍵字定義（NSF 檔，`notes/re/nsf.md`）。我們的譯文把關鍵字寫成「銀行（ bank ）」（英文留在全形括號內、兩側空格）。
- EXE 內有 `Data\Strings\translated_Keywords.txt` 與 `Data\Strings\English_Keywords.txt` 路徑字串（`notes/research/2026-10-07-A3-text-font-format.md:20`），本機兩檔都不存在（`notes/research/2026-10-07-R7-nsf-format.md:40`）。推測是外語版用來把譯詞對回英文關鍵字的表。
- 反組譯全文 `build/wiz8.asm`（objdump intel）。已知：繪字 `notes/re/draw.md`、分行 `notes/re/wrap.md`、NSF 存字 `notes/re/nsf.md`。
- 目標（7.3 可選升級）：玩家在對白裡點中文詞（或括號內英文）時，送出的是英文關鍵字；或用 translated_Keywords.txt 讓中文詞直接對到英文關鍵字。

## 問題（每條結論附來源：`build/wiz8.asm:<行>`、`notes/...:<行>`，與三態標記）
1. 誰讀這兩個檔（引用路徑字串的函式位址）；檔不存在時的行為（跳過、報錯、預設）。
2. 檔案格式：從解析碼推（分隔字元、每行欄位、編碼 ANSI／UTF-16、大小寫、上限筆數）。給一個最小範例內容（自己編的詞，不抄原作）。
3. 讀進來的表在哪裡被查：打字輸入關鍵字時？點擊對白取詞時？比對 NSF 關鍵字前做 translated→English 替換？列呼叫鏈位址。
4. 點擊取詞：對白文字被點時，如何從點擊座標取出「一個詞」（以空白切？用字寬表量？），位址。切詞對 UTF-16 中文（無空白）會怎樣；我們的「 bank 」兩側空格寫法點括號內英文時是否會取到 `bank`。
5. 結論：(a) 只靠放一份 translated_Keywords.txt（不改 exe）能否讓中文詞生效；(b) 若不能，最小 hook 點（位址＋前 16 bytes，供簽章表用）。
6. 開放題：會讓這條路失敗的地方（關鍵字長度上限、輸入框只收 ASCII、大小寫轉換）。

## 報告格式
`notes/research/2026-10-07-R9-translated-keywords.md`，含 `## 發現` 段，每條 `- ` 結論附來源與 `[已驗證]`／`[推測]`／`[未取得]`。另含 `## 規格疑慮` 段（沒有就寫「無」）。

# R7 研究：NSF（NPC 對白腳本）格式（stage7 7.1）

驗收: bash scripts/check-research.sh notes/research/2026-10-07-R7-nsf-format.md
規模: 中（研究，一份報告）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單以外的檔；不寫入遊戲目錄；不執行遊戲或任何下載物；報告不貼反編譯碼、不貼遊戲原文（以檔名＋位移、記錄號指稱；單字級關鍵字可寫）；不送 email 或真實姓名
白名單: notes/research/2026-10-07-R7-nsf-format.md
預期紅: research:2026-10-07-R7-nsf-format
執行器: muse

## 背景
Wizardry 8（Wiz8.exe v1.2.4，i386）中文化。階段 7 要翻 NPC 對白。下一步 codex 寫 NSF decode／encode 工具，驗收是 165 個原檔往返 byte 一致、改長度後內部偏移重算，所以格式要精確到位元組。
- 資料：`build/data-x/NPC SCRIPTS/*.NSF`（165 個，DATA.SLF 全抽出）；`build/data-x/DATABASES/NPC.DBS`。
- 反組譯全文 `build/wiz8.asm`（objdump intel）。載入點字串 `Data\NPC Scripts\%s.nsf`（`notes/research/2026-10-07-A3-text-font-format.md:18`）。
- 已知（`notes/research/2026-10-07-A3-text-font-format.md:19`、`notes/research/2026-10-07-R5-pronouns-keywords.md:27`）：對白是 u16 長度前綴 UTF-16LE，關鍵字是 u16 長度前綴單位元組串；同一關鍵字有多個別名拼法。**以上要重新驗，不當前提。**
- 參考已解的同廠格式：`tools/strlib/strlib.go`（索引字串庫，0x9696−c 編碼旗標）。
- 德文翻譯整包換 NSF（`notes/research/2026-10-07-fan-translation-de.md`）、法文換 3 個 NSF（`notes/research/2026-10-07-fan-translation-fr.md`）：只作「文字可整段換、長度可變」的旁證。
- 玩家實機觀察（MG 2026-10-07）：對話視窗外，NPC 在畫面上自己講的話（訊息區）也是英文；這些文字在哪個檔未知。

## 問題（每條結論附來源：`build/...` 檔名＋十六進位位移、`build/wiz8.asm:<行>`、`tools/...:<行>` 或 URL，與三態標記）
1. NSF 檔頭與整體結構：區段種類、順序、計數欄、偏移／長度欄（絕對或相對）。每種記錄的位元組版式表：欄位｜寬度｜意義｜證據。至少用 3 個大小差很多的檔對照。
2. 文字欄：哪些記錄有玩家看得到的文字（對白、選項、交易、問候、告別、自言自語），各自編碼（UTF-16LE／單位元組／0x9696−c／其他），有無結尾 NUL，有無格式記號（`%`、換行、色碼、名字代入）。
3. 關鍵字：存在哪裡、怎麼和對白連結（對白裡的關鍵字是否另有標記）、別名怎麼存；`Data\Strings\translated_Keywords.txt`／`English_Keywords.txt` 在讀檔路徑中的角色（`build/wiz8.asm` 載入函式；本機有沒有這兩個檔）。
4. 遊戲讀 NSF 的函式（位址）：一次讀整檔還是逐段；有無固定大小緩衝或長度上限（譯文變長、UTF-16 字數變多會不會溢位）。
5. 對話視窗外的 NPC 台詞（訊息區）出自哪個檔：NSF 某區段、NPC.DBS、還是別處。
6. 165 個檔是否全是同一版式；有沒有例外檔（大小 0、不同版本號）。
7. 開放題：會讓「decode→encode 往返 byte 一致」做不到的地方（壓縮、校驗碼、填充、未使用的殘留位元組）。

## 報告格式
`notes/research/2026-10-07-R7-nsf-format.md`，含 `## 發現` 段，每條 `- ` 結論附來源與 `[已驗證]`／`[推測]`／`[未取得]`。另含 `## 規格疑慮` 段（沒有就寫「無」）。

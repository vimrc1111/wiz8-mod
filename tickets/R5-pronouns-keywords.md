# R5 研究：代名詞模板與對白關鍵字點擊（stage5 5.2／5.5）

驗收: bash scripts/check-research.sh notes/research/2026-10-07-R5-pronouns-keywords.md
規模: 中（研究，一份報告）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單以外的檔；不寫入遊戲目錄；不執行遊戲或任何下載物；報告不貼反編譯碼、不貼遊戲原文（字串一律以 STRINGDATA 索引或檔名＋位移指稱；單字級的關鍵字可寫）；不送 email 或真實姓名
白名單: notes/research/2026-10-07-R5-pronouns-keywords.md
預期紅: research:2026-10-07-R5-pronouns-keywords
執行器: muse

## 背景
Wizardry 8（Wiz8.exe v1.2.4，i386）中文化。字串庫 STRINGDATA 共 2412 條，已解碼成 `build/stringdata.src.json`（JSON 陣列，索引＝id）。
執行期字串表是指標陣列 0x0068c09c（`mov reg, ds:0x68c09c` 後以 `[reg+id*4]` 取用；見 `notes/re/misc.md:34`）。反組譯全文 `build/wiz8.asm`（objdump intel）。遊戲資料檔全部抽出在 `build/data-x/`（SLF 內的目錄結構）。
已知：MESSAGES\*.MSG 與 STRINGDATA 同編碼（u16＝0x9696−c），但字串可能從奇數位移開始。0x0058fdd0 是對白框取詞函式（略過空白、取到下一個空白為止的一個詞，回傳起訖索引與 x 範圍）。
id 721–732：Male、Female、It、him、her、it、his、her、its、himself、herself、itself。直接以常數位移引用它們的程式碼**沒有找到**（`[reg+0xb44..0xb70]` 查無），推測是以「基底＋性別」計算索引。

## 問題（每條結論附來源：`build/wiz8.asm:<行>`、`notes/…:<行>` 或 URL，與三態標記）
1. 代名詞 724–732 在哪裡被取用（函式位址、索引怎麼算）？哪些 STRINGDATA 模板（id）的哪一個 `%s`（第幾個）會被代入哪一組（受格 him/her/it、所有格 his/her/its、反身 himself/herself/itself）？請列成表：模板 id｜第幾個 %s｜組｜證據。721–723 同樣處理。
   也請列出：會把**角色名、怪物名或物品名**代入、且同一句也用到代名詞的模板，及代入順序。
2. NPC 對白文字存在哪些檔（`build/data-x/` 下的路徑）？編碼、每筆的結構（只寫格式，不貼內容）。
3. 「關鍵字」的來源：哪些詞會被當成可點的關鍵字（資料檔裡的清單？對白裡的標記？還是任何詞都可點、加進清單後由對話系統比對？）。點擊對白中的詞 → 加入關鍵字清單的呼叫鏈（位址）。
4. 若對白句子是中文、只有關鍵字保留英文且前後加空白，取詞（0x0058fdd0）與加入清單是否仍能取到那個英文詞？中文字（無空白）連成的一段被點到會發生什麼？列出依據。
5. 開放題：與上述相關、但我沒問到的風險。

## 報告格式
`notes/research/2026-10-07-R5-pronouns-keywords.md`，含 `## 發現` 段，每條 `- ` 結論附來源與 `[已驗證]`／`[推測]`／`[未取得]`。

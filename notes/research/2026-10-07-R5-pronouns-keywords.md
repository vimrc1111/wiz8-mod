# R5 代名詞模板與對白關鍵字點擊

## 發現

### Q1 代名詞 724–732／721–723 的取用與模板對應

- id 721–732 的 id↔詞對應以解碼表為準（索引＝id），組定義 obj＝724–726、poss＝727–729、refl＝730–732 [已驗證] (build/stringdata.src.json:1)：組定義另見 (tickets/E-trcheck.md:29)。
- 執行期字串表是指標陣列，呼叫端以 `[reg+id*4]` 直接索引、無共用 getter [已驗證] (notes/re/strings.md:8)：解碼規則見 (notes/re/strings.md:6)。
- 直接以常數位移引用 721–732（`[reg+0xb44..0xb70]` 形）的程式碼全檔案零命中 [已驗證] (build/wiz8.asm:1)：1146 處表載入點逐一核對，鄰接 12 行內無上述位移。
- `push 0x2d1..0x2dc` 形立即數僅出現在行號式記錄呼叫（同形皆伴隨虛擬檔名字串位址與間接呼叫），不是字串 id [已驗證] (build/wiz8.asm:152138)：同形另見 (build/wiz8.asm:290363)。
- 基底＋性別算式（`add`／`lea`／`mov` 帶 `0x2d1..0x2dc` 作字串索引）全檔案無此用途，命中者皆為結構位移或區域變數 [已驗證] (build/wiz8.asm:1)。
- 代名詞／名稱字串實際走 WORD id 查表：`0x61e434`（位元組索引×8 取 WORD 作 STRINGDATA id 再查表）[已驗證] (build/wiz8.asm:315490)：同表另見 (build/wiz8.asm:387101)。
- 另一路 id 查表 `0x61ec14`（位元組索引×4 取 dword id）[已驗證] (build/wiz8.asm:316101)：其輸出餵向 `.data` 內 C 格式字串而非 STRINGDATA 模板。
- 其餘 id 查表（`0x61e658`／`0x61e810`／`0x61e988`／`0x61e3a4`／純量 `0x61e602`）同為「先查 id 再查表」兩段式 [已驗證] (build/wiz8.asm:314627)：另見 (build/wiz8.asm:316118)。
- 模板 id 571｜第 2 個 %s｜poss｜證據：先取表驅動字串、再以 571 格式化，槽位在英語語義上為所有格 [推測] (build/wiz8.asm:315500)：表載入見 (build/wiz8.asm:315486)。
- 模板 id 571｜第 1 個 %s｜人物／怪物名（`0x4e5150` 回傳，跨戰鬥站點一致用法）｜代入順序：第 1 槽名、第 2 槽代名詞 [推測] (build/wiz8.asm:315486)。
- 模板 id 495／496 與表驅動字串同句出現，但各 %s 槽位對應（哪槽是代名詞、名或物）[未取得] (build/wiz8.asm:387112)：格式化器非常駐 sprintf，參數規約未解。
- id 721–723 在 STRINGDATA 模板中的取用 [未取得] (build/wiz8.asm:316101)：已見的性別位元組→字串路徑餵向 C 格式字串，STRINGDATA 模板側無對應站點。
- 含 2 個以上 %s 的模板全列（代名詞候選母集合）[已驗證] (build/stringdata.src.json:1)：218、219、337、387、395、396、398、399、400、470、495、496、500、571、580、583、598、1869。
- 上列中 580／583／1869 等模板的代名詞槽身份（或是否含代名詞槽）[未取得] (build/stringdata.src.json:1)：其格式化站點尚未逐一定位。

### Q2 NPC 對白文字所在檔案與結構

- NPC 對白文字檔：`build/data-x/MESSAGES/` 下 15 個 MSG、`build/data-x/QUOTES/PCS/` 下 36 個 MSG、`build/data-x/NPC SCRIPTS/` 下 165 個 NSF [已驗證] (build/data-x/MESSAGES/ARNIKA.MSG:0)：載入點分別見 (notes/research/2026-10-07-A3-text-font-format.md:10) 與 (notes/research/2026-10-07-A3-text-font-format.md:18)。
- MSG 編碼與 STRINGDATA 同（u16＝0x9696−c）；首筆偶位移解出可列印字元，奇數位移起筆的背景說法本輪未另驗 [已驗證] (build/data-x/MESSAGES/ARNIKA.MSG:13)：檔頭 u32 筆數見 (build/data-x/MESSAGES/ARNIKA.MSG:0)。
- MSG 每筆結構（只寫格式）：u32 單位數＋N 個 u16 編碼單位，筆間以 ff 哨兵分隔 [已驗證] (build/data-x/MESSAGES/ARNIKA.MSG:13)：QUOTES 同容器見 (build/data-x/QUOTES/PCS/F_AGGR10.MSG:0)。
- NSF 混合雙格式：對白為 u16 長度前綴 UTF-16LE，關鍵字為 u16 長度前綴單位元組串 [已驗證] (notes/research/2026-10-07-A3-text-font-format.md:19)：本機以對白樣本與關鍵字樣本複核 [已驗證] (build/data-x/NPC SCRIPTS/NPC_MARTEN.NSF:21)。
- 同一 NSF 內同一關鍵字並存多個別名拼法（單字級：MARTEN／MARTAN／MARTIN 等，另有 ASCENSION 系多個別名），為容錯比對的旁證 [已驗證] (build/data-x/NPC SCRIPTS/NPC_MARTEN.NSF:123)。
- EXE 內建 `Data\\Strings\\translated_Keywords.txt` 與 `Data\\Strings\\English_Keywords.txt` 兩份關鍵字清單 [已驗證] (notes/research/2026-10-07-A3-text-font-format.md:20)。
- `STRINGS/MGSKEYBOARD.INI` 是鍵盤初始化設定（UTF-16），不是對白關鍵字 [已驗證] (build/data-x/STRINGS/MGSKEYBOARD.INI:1)。

### Q3 關鍵字來源與點擊→入列呼叫鏈

- `0x0058fdd0` 本體見 asm，全文僅兩個呼叫點：`0x58b1ae` 與 `0x58ff4a` [已驗證] (build/wiz8.asm:540017)：呼叫點見 (build/wiz8.asm:533811) 與 (build/wiz8.asm:540152)。
- 建表鏈：`0x58af60`（調用見 `0x58ad43`／`0x58ae76`）與 `0x58fee0`（調用見 `0x55f79b`）以 `0x58fdd0` 切全詞、0xa 位元組節點經 `0x5e2480` 鏈入 `0x68f2f0` 陣列 [已驗證] (build/wiz8.asm:533638)：批量版見 (build/wiz8.asm:540100)，每緩衝 `0x15e` 上限見 (build/wiz8.asm:533679)。
- 點擊→入列鏈：對白框處理經 `0x58ffc0`（調用見 `0x58d739`／`0x58d74c`）做逐詞繪製／命中，再以詞索引呼叫 `0x58b5f0` 附加到關鍵字清單 [推測] (build/wiz8.asm:536811)：`0x58b5f0` 本體見 (build/wiz8.asm:534155)，其調用見 (build/wiz8.asm:482458)。
- 滑鼠座標到詞索引的比對函式本體 [未取得] (build/wiz8.asm:536811)：`0x58ffc0` 內逐詞 x 範圍已具，命中判定入口未定位。
- 關鍵字來源三者並存：NSF 內清單（含別名）[已驗證] (build/data-x/NPC SCRIPTS/NPC_MARTEN.NSF:123)；EXE 內建清單 [已驗證] (notes/research/2026-10-07-A3-text-font-format.md:20)；建表迴圈切的是全部詞、關鍵字身份由後續對話系統比對決定 [推測] (build/wiz8.asm:533811)。
- 點詞可免打字輸入（解析器輸入的替代路徑）[已驗證] (notes/research/2026-10-07-fan-translation-de.md:70)：關鍵字清單寫入存檔、舊檔混用英文見 (notes/research/2026-10-07-fan-translation-de.md:63)。

### Q4 中文對白＋英文關鍵字的取詞行為

- 取詞分隔符僅 U+0020 空白與 NUL：前導跳空白、逢空白或 NUL 收尾 [已驗證] (build/wiz8.asm:540026)：另見 (build/wiz8.asm:540055) 與 (build/wiz8.asm:540059)。
- 中文句中前後加空白的英文詞仍被切成獨立 token，判據只看 0x20、與前後是否為 CJK 無關 [已驗證] (build/wiz8.asm:540055)。
- 無空白的 CJK 連段整段成為一個「詞」（起訖索引橫跨整段）[已驗證] (build/wiz8.asm:540069)：逐詞 x 範圍以量寬函式計算見 (build/wiz8.asm:540051)。
- 被點到的 CJK 段同樣走入列（`0x58b5f0` 不檢查詞表），但對話系統無此關鍵字故無回應；英文關鍵字比對具別名容錯 [推測] (build/wiz8.asm:534155)：別名旁證見 (build/data-x/NPC SCRIPTS/NPC_MARTEN.NSF:123)。
- CJK 字寬與換行行為（量寬 `0x406ea0`／`0x407010` 對 CJK 的回傳）[未取得] (build/wiz8.asm:540083)：溢框風險屬 5.3 範疇見 (goals/stage5.md:12)。

### Q5 其餘風險

- printf 記號序列中譯必須與原文完全一致（數量、順序、型別），中文語序不能調槽 [已驗證] (tickets/E-trcheck.md:36)：poss 加「的」／refl 加「自己」後的「的的」「自己自己」另有專檢 [已驗證] (tickets/E-trcheck.md:42)。
- 代名詞形式約束（727+k＝724+k＋「的」、730+k＝724+k＋「自己」）是譯文硬約束，模板機檢依賴它 [已驗證] (tickets/E-trcheck.md:41)。
- `0x61ec14` 路徑的 C 格式字串在 `.data`（非 STRINGDATA），覆蓋 STRINGDATA 改不到 721–723 相關顯示、將殘留英文 [推測] (build/wiz8.asm:316101)。
- NSF 關鍵字是單位元組串而對白是 UTF-16LE，中譯後解析器比對的正規化（大小寫、寬窄）[未取得] (notes/research/2026-10-07-A3-text-font-format.md:19)。
- 每緩衝 350 詞上限；CJK 無空白整句一詞不佔量，但英文關鍵字一詞一槽，超長對白有截斷風險 [已驗證] (build/wiz8.asm:533679)：入列側同上限見 (build/wiz8.asm:534161)。
- 物品名／敵人名有固定長度上限，中文名在 571／495 系同句模板需同步量框 [已驗證] (notes/research/2026-10-07-fan-translation-de.md:71)。
- 驗證新關鍵字流程須開新遊戲（舊存檔殘留英文清單）[已驗證] (notes/research/2026-10-07-fan-translation-de.md:63)：兩步走第 1 步（保留英文、前後加空白）見 (goals/stage5.md:26)。

## 方法與命令紀錄

唯讀檢視，未執行遊戲或任何下載物，未寫入遊戲目錄。asm 全文檢索（`68c09c` 1146 處、`0xb44..0xb70`、`push 0x2d1..0x2dc`、`add/lea 0x2d1..0x2dc`、`0x61e434`／`0x61ec14` 等表）以行號定位後逐段判讀；`build/data-x/` 下 MSG／NSF／INI 以 `xxd` 與位元組解析抽查結構（解碼僅在本機驗可列印性，報告內不貼原文）；跨檔來源以既有研究（A3、德文翻譯、E 規格、stage5 目標）互證。未取得 EXE 本體，故 0x61 系表內容（各性別值對應到 724–732 哪一 id）無法在本機驗證。

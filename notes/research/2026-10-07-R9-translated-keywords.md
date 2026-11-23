# R9：translated_Keywords.txt 機制與點擊取詞（7.3）

範圍：唯讀研究，未執行遊戲或任何下載物，未寫入遊戲目錄。反組譯全文 `build/wiz8.asm` 在本工作區為懸空連結（`build` 指向不存在的路徑），asm 行號引用承襲 R5／R7 的已驗證結論，未在本環境重驗（見 `## 規格疑慮`）。報告不貼反編譯碼與遊戲原文，只記位址、前 16 bytes（有者）、行為（一句話）；單字級關鍵字可寫；自編範例已註明。

## 發現

- EXE 內建 `Data\Strings\translated_Keywords.txt` 與 `Data\Strings\English_Keywords.txt` 兩路徑字串，位移見 A3 命令紀錄（2400332／2400372） [已驗證] (notes/research/2026-10-07-A3-text-font-format.md:20)
- 上述兩檔在本機英文安裝不存在（`STRINGS` 目錄僅兩檔），英文版無此二檔即可運行 [已驗證] (notes/research/2026-10-07-R7-nsf-format.md:40)
- 引用該兩路徑字串的函式位址未取得：僅含 `.text` 段的反組譯無字串交叉引用可錨定，`OpenVirtualFile（0x00404c80）` 直接呼叫點 62 處皆不在對話區段 [未取得] (notes/research/2026-10-07-R7-nsf-format.md:43)
- 兩檔是否為 NSF 外的第二關鍵字源、載入順序與覆蓋規則未取得，是否被讀取本身亦未定位 [未取得] (notes/research/2026-10-07-R7-nsf-format.md:41)
- 檔不存在時的行為未取得；旁證是德文翻譯包（PATCH.002，資源表僅 DBS／STRINGDATA／MSG／NSF／STI，無此二檔）遊戲可正常運行，故缺檔應為容忍（跳過或走預設），而非報錯 [推測] (notes/research/2026-10-07-fan-translation-de.md:52)
- 讀檔統一經虛擬檔層（開／讀／定位／關），NSF 應同路；若引擎真讀此二檔，PATCH 疊層覆蓋同樣適用 [已驗證] (notes/research/2026-10-07-R7-nsf-format.md:44)
- 檔案分隔字元從解析碼推：解析碼未定位，無從推起 [未取得] (notes/research/2026-10-07-R7-nsf-format.md:41)
- 每行欄位結構（幾欄、哪欄是譯詞、哪欄是英文鍵）未取得 [未取得] (notes/research/2026-10-07-R7-nsf-format.md:41)
- 檔案編碼（ANSI／UTF-16）未取得；僅知 NSF 側關鍵字為單位元組可列印 ASCII 串 [未取得] (notes/research/2026-10-07-R7-nsf-format.md:28)
- 大小寫規則未取得：NSF 關鍵字大小寫混排，比對是否大小寫無關屬引擎行為、未驗 [未取得] (notes/research/2026-10-07-R7-nsf-format.md:28)
- 上限筆數未取得；已知相鄰約束僅 NSF 單關鍵字實測最長 42 位元組與對白建表每緩衝 350 詞上限 [未取得] (notes/research/2026-10-07-R7-nsf-format.md:25)
- 最小範例內容（自編假設，非原作；方向假設為譯詞對英文鍵，如 `river=河川` 形，實際分隔符與欄位順序未確認） [推測] (tickets/R9-translated-keywords.md:15)
- 對白切詞函式為 `0x58fdd0`，全文僅兩個呼叫點（`0x58b1ae` 與 `0x58ff4a`） [已驗證] (notes/research/2026-10-07-R5-pronouns-keywords.md:34)
- 建表鏈為 `0x58af60` 與 `0x58fee0` 以 `0x58fdd0` 切全詞、`0xa` 位元組節點經 `0x5e2480` 鏈入 `0x68f2f0` 陣列，每緩衝 `0x15e`（350）詞上限 [已驗證] (notes/research/2026-10-07-R5-pronouns-keywords.md:35)
- 點擊後入列鏈為對白框經 `0x58ffc0` 做逐詞繪製與命中、再以詞索引呼叫 `0x58b5f0` 附加到關鍵字清單 [推測] (notes/research/2026-10-07-R5-pronouns-keywords.md:36)
- 滑鼠座標到詞索引的命中判定入口未定位（`0x58ffc0` 內逐詞 x 範圍已有，判定入口未得） [未取得] (notes/research/2026-10-07-R5-pronouns-keywords.md:37)
- 譯詞轉英文替換點（打字前、點擊後、或比對 NSF 關鍵字前）未取得；NSF 關鍵字為單位元組串而對白為 UTF-16LE，比對正規化（大小寫、寬窄）亦未驗 [未取得] (notes/research/2026-10-07-R5-pronouns-keywords.md:54)
- 點詞是打字輸入的替代路徑（德文版以點詞繞過解析器不收變音字元） [已驗證] (notes/research/2026-10-07-R5-pronouns-keywords.md:39)
- 取詞分隔符僅 U+0020 空白與 NUL：前導跳空白、逢空白或 NUL 收尾 [已驗證] (notes/research/2026-10-07-R5-pronouns-keywords.md:43)
- 前後加空白的英文詞仍被切成獨立 token，判據只看 0x20、與前後是否為 CJK 無關 [已驗證] (notes/research/2026-10-07-R5-pronouns-keywords.md:44)
- 無空白的 CJK 連段整段成為一個詞（起訖索引橫跨整段） [已驗證] (notes/research/2026-10-07-R5-pronouns-keywords.md:45)
- 逐詞 x 範圍以量寬函式計算，點擊命中即落到起訖索引對應的詞 [已驗證] (notes/research/2026-10-07-R5-pronouns-keywords.md:45)
- 量寬函式對 CJK 的回傳值未取得，超寬 CJK 行的命中精度屬分行器範疇 [未取得] (notes/research/2026-10-07-R5-pronouns-keywords.md:47)
- 「 bank 」兩側空格寫法下，點括號內英文會取到 `bank`：空白切分使 token 恰為 `bank`，全形括號不是分隔符故不沾黏；點中文則取到含全形括號的整段 [推測] (notes/research/2026-10-07-R5-pronouns-keywords.md:44)
- 被點到的 CJK 段同樣走入列（`0x58b5f0` 不檢查詞表），但對話系統無此關鍵字故無回應 [推測] (notes/research/2026-10-07-R5-pronouns-keywords.md:46)
- 對白輸入區重排（`0x58dca0` 一帶）沒有碰 `0x58fdd0`，點擊取詞路徑獨立於輸入區 [已驗證] (notes/re/wrap.md:26)
- 7.2 第一步要求關鍵字在譯文中保留英文、前後加空格（與「 bank 」寫法一致） [已驗證] (goals/stage7.md:15)
- 結論(a)：只靠放一份 translated_Keywords.txt（不改 exe）不能讓中文詞生效：讀取點與查表點皆未定位，且英文版缺檔可跑、德文包無此檔，引擎讀它的證據為零 [推測] (notes/research/2026-10-07-R7-nsf-format.md:41)
- 結論(b)：最小 hook 點為入列函式 `0x58b5f0`（備選切詞 `0x58fdd0`、逐詞繪製 `0x58ffc0`）；三者前 16 bytes 本環境無 exe 未取得，以 `re/Xrefs.java`（印函式入口加前 16 bytes）在有 exe 環境提取後再入簽章表 [未取得] (re/Xrefs.java:1)
- 關鍵字長度上限：NSF 實測最長 42 位元組，引擎是否截斷或拒收更長譯詞未驗 [未取得] (notes/research/2026-10-07-R7-nsf-format.md:25)
- 德文版解析器輸入不接受變音字元，繞法是點對白中的詞 [已驗證] (notes/research/2026-10-07-fan-translation-de.md:70)
- 中文手打關鍵字同理大概率被輸入框拒收，中文詞只能走點擊路徑 [推測] (notes/research/2026-10-07-fan-translation-de.md:70)
- 關鍵字清單寫入存檔，舊存檔混用英文關鍵字，驗證新關鍵字須開新遊戲 [已驗證] (notes/research/2026-10-07-fan-translation-de.md:63)
- 部分關鍵字嵌在 packed EXE 字串內改不到（如 formation／地名類），此路徑的中文化上限與 txt 無關 [已驗證] (notes/research/2026-10-07-fan-translation-de.md:62)
- 物品名與敵人名有固定長度上限（德文版被迫縮寫），同類定寬約束可能適用於關鍵字欄 [已驗證] (notes/research/2026-10-07-fan-translation-de.md:71)
- 每緩衝 350 詞上限兩側皆在，超長對白的英文關鍵字有截斷風險 [已驗證] (notes/research/2026-10-07-R5-pronouns-keywords.md:55)
- 即使點擊取到英文鍵，NPC 回覆經 `swprintf` 轉回寬字路徑顯示中文仍依賴 nsf hook（訊息區另一格式 `L" %S"` 須一併處理，否則亂碼） [已驗證] (notes/re/nsf.md:11)
- `MGSKEYBOARD.INI` 是鍵盤初始化設定，不是對白關鍵字，排查時勿混淆 [已驗證] (notes/research/2026-10-07-R5-pronouns-keywords.md:30)

## 規格疑慮

- 本工作區 `build/wiz8.asm` 為懸空連結，asm 行號引用承襲 R5／R7，未在本環境重驗；要求「每條附 `build/wiz8.asm:<行>`」在本環境無法新增。
- Q2 要求「從解析碼推」檔案格式，但引用路徑的函式與解析碼皆未定位，格式全段只能記 `[未取得]`，範例為自編假設。
- Q5b 要求 hook 點「前 16 bytes」，候選位址已給（`0x58b5f0`／`0x58fdd0`／`0x58ffc0`），位元組須在有 exe 環境以 `re/Xrefs.java` 提取。

## 方法與命令紀錄

唯讀：`ls` 確認工作區結構與懸空連結；`grep -rn` 掃描 `translated_Keywords`、`58fdd0`、`keyword` 在本工作區的全部命中；`read_file` 逐份核對 A3／R5／R7／德文翻譯／draw／wrap／nsf／stage7。未執行遊戲或任何下載物，未寫入遊戲目錄，未 commit。

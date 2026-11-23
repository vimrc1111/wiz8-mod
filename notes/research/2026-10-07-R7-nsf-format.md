# R7：NSF（NPC 對白腳本）格式（stage7 7.1）

範圍：`build/data-x/NPC SCRIPTS/` 下 165 個 NSF（`NPC_` 96、`RPC_` 13、`VOC_` 56）全量唯讀解析；
對照 `build/data-x/DATABASES/NPC.DBS`、`build/data-x/MESSAGES/`＋`QUOTES/PCS/` 共 51 MSG、
`build/wiz8.asm`（僅 `.text` 段反組譯）、`tools/strlib/strlib.go`。
報告不貼反組譯碼與遊戲原文，以檔名＋十進位位移、記錄位移指稱；單字級關鍵字可寫。
A3／R5 的舊結論（UTF-16LE 對白、單位元組關鍵字、別名拼法）全部重新驗算，不當前提。

## 發現

- 165 檔全是同一檔頭：`u32 0xFFFFFFFF`＋`u32 count`，count 值域 1–237（214 佔 53 檔最多）；無大小 0 檔 [已驗證] (build/data-x/NPC SCRIPTS/NPC_ZANT.NSF:0)
- 檔頭之後是 6 位元組檔前導 `00 00 XXXX TAG`，首文本恆在位移 26（165/165，含下述例外檔） [已驗證] (build/data-x/NPC SCRIPTS/VOC_DSBOMB.NSF:8)
- TAG 164 檔為 `2f 00`，僅 1 檔例外（見 Q6）；XXXX 全集 22 種值（9000 佔 60 檔、8984 佔 57 檔為主），同值橫跨 NPC／RPC／VOC 前綴、非 NPC 身份 [已驗證] (build/data-x/NPC SCRIPTS/NPC_BELA_AP.NSF:10)
- 每段文本前是 12 位元組節點前導 `MM quadA quadB XX 00 00`（1＋4＋4＋3）；10306 例為此形，另 14 例尾部變體（見下） [已驗證] (build/data-x/NPC SCRIPTS/NPC_MARTEN.NSF:199)
- 節點前導 MM 恆等於其文本標記（10306/10306）；quadA／B 屬 `0x002F／0x004B／0x004C／0x0000` 等家族，大量跨檔重複出現、不是檔內位移 [已驗證] (build/data-x/NPC SCRIPTS/NPC_BELA_AP.NSF:259)
- 全集未發現任何指向文本／記錄起點的絕對或相對位移欄；改長度後須重算的是長度前綴（文本 unit 數、關鍵字位元組數、chunk 數），不是位移表 [已驗證] (build/data-x/NPC SCRIPTS/NPC_MARTEN.NSF:123)
- 文本記錄＝標記 u8（m）＋ m 個 chunk，m∈{1,2,3,4}；chunk＝u16 unit 數＋UTF-16LE units，首 chunk 帶標記、後續 chunk 無標記裸接 [已驗證] (build/data-x/NPC SCRIPTS/VOC_TRANG_INTRO.NSF:26)
- m＝chunk 數 100％成立（10306/10306）：m1 10091 筆皆單 chunk、m2 168 筆皆 2 chunk、m3 31 筆皆 3 chunk、m4 16 筆皆 4 chunk；m1 文本後從不接裸 chunk（緊接下段皆為節點前導或定長記錄） [已驗證] (build/data-x/NPC SCRIPTS/NPC_YAMIR.NSF:27927)
- 檔內位元組無對齊要求：文本負載可起於奇數位移，解碼器須非對齊讀 u16（首例文本起於位移 29） [已驗證] (build/data-x/NPC SCRIPTS/VOC_VI_INTRO.NSF:26)
- 文本無結尾 NUL：全集末 unit 無一為 0，長度恰為 unit 數；單位元組關鍵字定義亦無結尾 NUL [已驗證] (build/data-x/NPC SCRIPTS/VOC_DSBOMB.NSF:101)
- 文本編碼是明文 UTF-16LE，不是 `0x9696` 變換（全集 86 種相異 units：`0x09／0x0A／0x20–0x7E／0x92`，高位元組恆 0） [已驗證] (build/data-x/NPC SCRIPTS/NPC_ZANT.NSF:31660)
- 唯一非 ASCII unit 是 U+0092（20 次、3 檔；cp1252 的右單引號位置 widen 而來），無其他 CJK／變音／全形 [已驗證] (build/data-x/NPC SCRIPTS/VOC_VI_INTRO.NSF:26)
- `%` 全集僅 4 次，皆字面百分比（數字＋`%`＋換行），無 `%s／%d` 等格式記號；無 CR、無色碼、無名字代入記號 [已驗證] (build/data-x/NPC SCRIPTS/VOC_VI_INTRO.NSF:26)
- 換行只有 LF（92 段文本、13 檔；72 段內嵌、8 段尾隨、0 段前導）；TAB 屬合法 unit [已驗證] (build/data-x/NPC SCRIPTS/VOC_TRANG_INTRO.NSF:26)
- 最長單文本 505 units（例外檔內，位移 3768）；最長 04 文本總計 974 units；最長關鍵字 42 位元組；英文最長行對譯文緩衝的直接上限未在 NSF 側發現 [已驗證] (build/data-x/NPC SCRIPTS/NPC_LORD_BRAFFIT.NSF:3768)
- 佔位文本為字面 `EMPTY`（1841）、`UNKNOWN`（1563）、`BLANK`（52），皆長度前綴字串、不可正規化改寫 [已驗證] (build/data-x/NPC SCRIPTS/NPC_MYLES.NSF:1147)
- 關鍵字定義 4594 筆，形為 13 位元組前導 `K quadA u32(0) quadB`＋`u16 位元組數`＋單位元組串；K∈{1,2,3}（2089／1087／1418） [已驗證] (build/data-x/NPC SCRIPTS/NPC_MARTEN.NSF:123)
- 關鍵字字元集為可列印 ASCII（0x20–0x7E，含空格與 `'`），大小寫混排；比對是否大小寫無關屬引擎行為，未驗 [已驗證] (build/data-x/NPC SCRIPTS/NPC_MARTEN.NSF:2506)
- 別名機制重新驗證成立：同一話題多個別名拼法並列存放（MARTEN／MARTAN／MARTIN；PEAK 系 5 種拼法；BLUFF 系 4 種），為容錯比對的旁證 [已驗證] (build/data-x/NPC SCRIPTS/NPC_MARTEN.NSF:123)
- 關鍵字 A 欄可為 quad 常數（多屬 `0x004C` 家族）或 3 字母 ASCII 標籤（如話題縮寫）；語義未解 [已驗證] (build/data-x/NPC SCRIPTS/NPC_AL-ADRYIAN.NSF:4526)
- 同一數值在回應定義與連結記錄兩處相等（id 137 見兩種記錄），即關鍵字→回應有顯式 id 連結；id 語義（全域或檔內）未定 [已驗證] (build/data-x/NPC SCRIPTS/NPC_MARTEN.NSF:163)
- `07` 首 18 位元組記錄 210 筆（`07 u16`＋零負載，末例第 6 位元組為 01）；操作數如 765／766／767／416，疑為 fact／flag id（NPC.DBS 規則側用 u32 fact id），未證 [已驗證] (build/data-x/NPC SCRIPTS/NPC_BELA_AP.NSF:187)
- `08／11／18` 首 18 位元組零負載記錄各 333／60／36 筆（嚴格全零樣式普查）；`03／06` 首 13 位元組記錄（`op＋u32＋u32(1)＋u32`）各 983／234 筆 [已驗證] (build/data-x/NPC SCRIPTS/VOC_DSBOMB.NSF:174)
- `02` 首連結記錄含 quad＋小整數目標（如 368／369 遞增），與 12 位元組節點前導不同形（無零 u32 夾層）；目標語義未解 [已驗證] (build/data-x/NPC SCRIPTS/NPC_MARTEN.NSF:255)
- `01／02／04` 首 13 位元組記錄（`op＋u32＋u32(1)＋u32`）散見於文本尾（如尾序列 `01 02 00 00 00…`），與 983 筆 `03` 記錄同科；全集計數未做 [已驗證] (build/data-x/NPC SCRIPTS/VOC_BELA_CC.NSF:1084)
- 31 段文本無 12 位元組前導，以 1 位元組分隔直銜前段之後（分隔值域散佈 `0x00–0x4B`），標記皆 01、長度皆 65–89 units；疑為同節點第二文本，語義未解 [已驗證] (build/data-x/NPC SCRIPTS/NPC_MYLES.NSF:500)
- 14 段文本的節點前導尾部為變體（`XX YY 01` 或 `00 01 00`，非常見 `XX 00 00`），chunk 規則仍成立；尾位元組語義未解 [已驗證] (build/data-x/NPC SCRIPTS/VOC_SHAMAN_DYING.NSF:26)
- 5 處疑似短文本經逐一核對為 op-record 誤判（`04／01／02 01 00 00 00…` 13 位元組形的長度欄toz），非真實文本；普查已剔除 [已驗證] (build/data-x/NPC SCRIPTS/NPC_HELI.NSF:18471)
- 同一句子跨檔原文重用存在（65-unit 句在 4 檔逐位元組相同，hash 一致），中譯可共用譯文但須逐檔改寫 [已驗證] (build/data-x/NPC SCRIPTS/NPC_EMMONS.NSF:7424)
- `translated_Keywords.txt／English_Keywords.txt` 本機不存在（`STRINGS` 目錄僅兩檔）；EXE 內路徑見 A3 [已驗證] (build/data-x/STRINGS/MGSKEYBOARD.INI:1)
- 上述兩檔在讀檔路徑中的角色：EXE 內建路徑（位移見 A3）；是否為 NSF 外的第二關鍵字源、載入順序與覆蓋規則，反組譯僅 `.text` 段、無字串交叉引用可錨定，未取得 [未取得] (notes/research/2026-10-07-A3-text-font-format.md:20)
- 舊存檔殘留英文關鍵字、須開新檔驗證，德文翻譯 readme 有明文記載；與 NSF 內關鍵字清單並存 [已驗證] (notes/research/2026-10-07-fan-translation-de.md:63)
- NSF 解析器本體位址未取得：`OpenVirtualFile（0x00404c80）` 直接呼叫點 62 處皆不在對話區段（`build/wiz8.asm:530000–545000` 行區間零命中），`.text`-only 反組譯無字串可錨定格式字串位址 [未取得] (build/wiz8.asm:468810)
- 讀檔統一經虛擬檔層（開／讀／定位／關）為既有驗證結論，NSF 應同路（SLF／PATCH 疊層皆適用，德文整包 NSF 經 PATCH.002 覆蓋即旁證） [已驗證] (notes/re/strings.md:10)
- 一次讀整檔或逐段：未取得；DBS／字串庫前例皆為算好大小整塊讀（`count＊stride` 配置），NSF 採同路為推測 [推測] (build/wiz8.asm:448742)
- 對話系統側函式位址可驗：建表 `0x58af60`、切詞 `0x58fdd0`（呼叫點二處）、逐詞繪製 `0x58ffc0`、入列 `0x58b5f0` [已驗證] (build/wiz8.asm:540017)
- 每緩衝 350 詞上限（`0x15e` 比較）在切詞與入列兩側皆在；CJK 無空白整句一詞的行為見 R5 [已驗證] (build/wiz8.asm:533679)
- 索引字串庫 2000-unit 拒收閘存在，但 NSF 文本是否走同一閘未見呼叫關係；NSF 實測最長 505 units，2000 以內 [已驗證] (build/wiz8.asm:412973)
- 對話視窗外的 NPC 台詞（訊息區）出自 NSF 的推定：NPC.DBS 無台詞欄（僅顯示名＋內部別名＋規則） [已驗證] (notes/research/2026-10-07-R6-dbs-msg-format.md:19)
- MSG 池（51 檔 6669 段，本輪獨立重解段數吻合 R6）與 NSF 文本池幾乎不相交：5721 段 20-unit 以上 NSF 文本取 60 字首，僅 11 中（皆短通用短語） [已驗證] (build/data-x/MESSAGES/MARTEN.MSG:0)
- `VOC_*` NSF 內有關鍵字定義直接命名語音行（如 19 字語音鍵，同一檔出現兩次），即帶聲台詞腳本；訊息區文字與其為同一池為推定 [已驗證] (build/data-x/NPC SCRIPTS/VOC_VI_INTRO.NSF:383)
- 綜合上三條：訊息區 NPC 台詞出自 NSF（VOC 類）文本段，既非 NPC.DBS 亦非 MSG；運行期最終確認須實機，列推測 [推測] (build/data-x/NPC SCRIPTS/VOC_DSBOMB.NSF:26)
- 德文翻譯整包換 NSF（159 檔經 PATCH.002）、法文換 3 檔（BALBRAK／FUZZFAS／ZANT），證文字可整段換、長度可變 [已驗證] (notes/research/2026-10-07-fan-translation-de.md:57)
- 法文換的是三個大對話檔（含最大 61KB 的 ZANT），德文 159 檔含 NPC／RPC／VOC 全類；兩者皆未動執行期二進位結構 [已驗證] (notes/research/2026-10-07-fan-translation-fr.md:21)
- 全集同一版式：8＋6＋（12＋文本＋尾）結構 165 檔一致；大小 0 檔無；版本號欄無（魔數恆定） [已驗證] (build/data-x/NPC SCRIPTS/NPC_ZANT.NSF:0)
- 唯一方言檔：其檔前導 TAG 為 `0x68`（餘皆 `0x2f`）、quad 屬 `0x0079` 家族，文法（前導＋標記＋chunk＋關鍵字）相同 [已驗證] (build/data-x/NPC SCRIPTS/NPC_LORD_BRAFFIT.NSF:12)
- XXXX 離群三檔（21976／16992／9840）文法相同，值本身語義未解（疑似共用語音／肖像集 id，跨 NPC／RPC／VOC 共用故非身份） [已驗證] (build/data-x/NPC SCRIPTS/NPC_TESTER.NSF:10)
- 無壓縮、無校驗碼：三個最小檔（366／210／451 位元組）已按上述記錄逐段切分至 EOF 無剩餘；其餘檔的文本＋關鍵字＋前導＋定長記錄覆蓋主體，未見雜湊尾 [已驗證] (build/data-x/NPC SCRIPTS/NPC_BELA_AP.NSF:348)
- 往返風險一：18 位元組記錄的零負載、節點前導的零 u32、關鍵字前導的零 u32 須原樣保留（語義或填充未定，不可清零或重算） [已驗證] (build/data-x/NPC SCRIPTS/VOC_DSBOMB.NSF:138)
- 往返風險二：「空」有三種字面寫法（EMPTY／UNKNOWN／BLANK，皆長度前綴、無 NUL），encode 不可正規化或合併 [已驗證] (build/data-x/NPC SCRIPTS/NPC_ALETHEIDES.NSF:26)
- 往返風險三：04（及 02／03）文本的 chunk 切分點必須原樣保留以達 byte 一致；譯後重切分是否被引擎接受（僅標記計數約束？）未驗 [未取得] (build/data-x/NPC SCRIPTS/VOC_TRANG_INTRO.NSF:26)
- 往返風險四：文本是 unit 計數、關鍵字是位元組計數，兩種長度單位不可混用；讀寫皆須非對齊 u16 存取 [已驗證] (build/data-x/NPC SCRIPTS/NPC_MARTEN.NSF:123)
- 往返風險五：U+0092 等特殊 unit 與 LF／TAB 須原樣保留，encoder 不可做 Unicode 正規化或換行轉換 [已驗證] (build/data-x/NPC SCRIPTS/VOC_VI_INTRO.NSF:26)
- A3 舊結論重驗：對白 u16 長度前綴 UTF-16LE（10306 段）、關鍵字 u16 長度前綴單位元組串（4594 筆），成立 [已驗證] (notes/research/2026-10-07-R5-pronouns-keywords.md:27)
- R5 別名舊結論重驗：MARTEN／MARTAN／MARTIN 並存（位移 123／139／155）及其他多別名群，成立 [已驗證] (build/data-x/NPC SCRIPTS/NPC_MARTEN.NSF:123)

## 規格疑慮

- 節點前導尾 3 位元組主形 `XX 00 00`、14 例變體（`XX YY 01`／`00 01 00`）：XX／YY 語義未解，工具須原樣回寫。
- 31 段無前導第二文本（1 位元組分隔、65–89 units）：分隔位元組語義未解，工具須原樣回寫。
- quadA／B（`0x002F／0x004B／0x004C／0x0079` 家族）與 07 操作數、02-連結小整數的語義未解；往返僅須保位元組，譯文不增刪節點則無需重算。
- NSF 解析器位址與整檔／逐段讀法、譯文變長的引擎側上限未取得；已知最長 505 units 單文本／974 units 四-chunk 總計可作保守參考。
- 04 文本譯後重切分（chunk 數＝標記值須維持）是否被引擎接受未驗；byte 一致驗收下以保留原切分點為準。

## 附錄 A：記錄版式（供 codex 工具用）

```text
file      := u32 0xFFFFFFFF, u32 count, fileleader, node+
fileleader:= u8 0, u8 0, u16 XXXX, u16 TAG        # 6B；TAG 164 檔 0x2f，1 檔 0x68
node      := leader, text, [text2], trailer
leader    := u8 MM, u32 A, u32 B, u8 XX, u8 00, u8 00   # 12B；MM＝text 標記；14 例尾變體
text      := u8 m, chunk × m                          # m∈{1,2,3,4}＝chunk 數
chunk     := u16 nunits, u16[nunits]                   # 明文 UTF-16LE，無 NUL；裸 chunk 無標記
text2     := u8 sep, u8 01, u16 n, u16[n]              # 31 例；n∈65–89；sep 1B 值散佈
trailer   := { kwdef | op13 | op18 | link02 }*
kwdef     := u8 K, u32 A, u32 0, u32 B, u16 n, u8[n]  # K∈{1,2,3}；n＝位元組數；4594 筆
op13      := u8 op, u32 x, u32 1, u32 y                # op∈{01,02,03,04,06}；03:983、06:234
op18      := u8 op, u8[17]                            # op∈{07(210),08(333),11(60),18(36)}；多全零負載
link02    := u8 02, u32 quad, u32 small, u8[...]       # 如 quad＋368／369；语义未解
```

- `count`（1–237）與文本數／關鍵字數無直接等式（不是記錄筆數，疑為節點預算）；工具回寫時原樣保留。
- A／B quad 與 `07` 操作數、`link02` 小整數一律按不透明位元組保留；文件中無絕對位移，改長度只重算 `nunits／n／m` 三處。

## 附錄 B：三檔切分锚點（十進位位移）

- 小檔一（366 位元組，count 2）：檔頭 0–7、前導 8–25、文本一 26（79 units，末 187）、`07` 記錄 187／205／223／241、節點前導 259、文本二 271（37 units，末 348）、`08` 記錄 348 至 366 EOF。
- 小檔二（210 位元組，count 2）：文本一 26（30 units）、`01`＋雙 quad＋`04 00 00` 53–64、文本二 101（17 units，末 138）、`07` 記錄 138／156、`11` 記錄 174、`18` 記錄 192 至 210 EOF。
- 小檔三（451 位元組，count 1）：文本 26（164 units，末 357）、尾 357–380（`01`＋u32＋`01／04`＋`01`＋雙 quad）、關鍵字一定義 383（19 位元組）、中段 389–428、關鍵字二定義 430 至 451 EOF。
- 上述三檔切分後無剩餘位元組；其餘 162 檔同文法（普查如發現段）。

## 命令紀錄

唯讀：`xxd` 目檢、`python3` 位元組解析（未執行遊戲或下載物）、`grep` 行定位。
未寫遊戲目錄，未 commit。解碼普查腳本為一次性 heredoc（未落倉）。

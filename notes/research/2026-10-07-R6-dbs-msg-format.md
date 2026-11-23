# R6：DBS 與 MSG 格式（stage6 6.1）

範圍：`build/data-x/DATABASES/` 12 個 DBS（另有 `HITSOUNDS.TXT` 非 DBS，一行帶過）、51 個 MSG（`build/data-x/MESSAGES/` 15 個场景檔＋`build/data-x/QUOTES/PCS/` 36 個配音檔）。遊戲以 PATCH.001 為準，但 `build/patch-x/DATABASES/` 內兩檔與 DATA.SLF 版 byte 一致（見發現）。報告不貼反組譯碼與遊戲原文，以檔名＋位移／記錄號指稱。

## 發現

- `ITEMS.DBS`：4 位元組檔頭記筆數 819，819 筆定長記錄 stride `0x10d`，`4+819*0x10d` 恰等於檔長 220315；讀檔函式內以 `0x10d` 為單位逐筆讀取 [已驗證] (build/data-x/DATABASES/ITEMS.DBS:0x0) (build/wiz8.asm:448742)
- `ITEMS.DBS` 唯一文字欄是記錄首的 UTF-16LE 名稱欄（明文，非 `0x9696` 變換），寬 32 code units（`+0x40` 起為非零數值欄；最長名稱 30 units，第 347 筆）；記錄其餘部分無 ASCII／UTF-16 文本 [已驗證] (build/data-x/DATABASES/ITEMS.DBS:0x4) (build/wiz8.asm:448742)
- `MONSTERS.DBS`：筆數 595，595 筆定長記錄 stride `0x297`，`4+595*0x297` 恰等於檔長 394489；每筆含 u16 自身索引（`+0x187` 等於記錄號，0／1／2／100／594 筆全中） [已驗證] (build/data-x/DATABASES/MONSTERS.DBS:0x0) (build/wiz8.asm:449172)
- `MONSTERS.DBS` 文字欄僅四組 `0x30` 位元組 UTF-16LE 名欄（`+0x0/+0x30/+0x60/+0x90`，各 24 units；全檔最長 23 units）；記錄其餘部分無文本，玩家可見文字除名稱外無 [已驗證] (build/data-x/DATABASES/MONSTERS.DBS:0x4)
- `LEVELS.DBS`：筆數 60，60 筆定長記錄 stride `0xd8`，`4+60*0xd8` 恰等於檔長 12964；`+0x58` 的 level_id 等於記錄號（第 0／1 筆為 0／1） [已驗證] (build/data-x/DATABASES/LEVELS.DBS:0x0) (build/wiz8.asm:449706)
- `LEVELS.DBS` 名稱欄為記錄首 UTF-16LE（全檔最長 21 units）；關卡名是否顯示於玩家介面（如自動地圖）未定，需 MG 裁定是否在翻譯範圍 [未取得] (build/data-x/DATABASES/LEVELS.DBS:0x4)
- `FACT.DBS`：筆數 807，807 筆定長記錄 stride `0x1d8`，`4+807*0x1d8` 恰等於檔長 380908；每筆 `+0x0` u32 id、`+0x4` 起 256 位元組符號名、`+0x104` 起 106 units UTF-16LE 設計註解 [已驗證] (build/data-x/DATABASES/FACT.DBS:0x0)
- `FACT.DBS` 807 筆中僅 77 筆註解非空，其中 2 筆在首個 NUL 之後仍有殘字，故 encoder 須保留整個 212 位元組定寬欄、不可按首個 NUL 截斷 [已驗證] (build/data-x/DATABASES/FACT.DBS:0x104)
- `FACT.DBS` 註解屬設計者備註（符號名如 `FACT_*` 亦為程式內部鍵），無玩家可見文本 [推測] (build/data-x/DATABASES/FACT.DBS:0x8)
- `SPELLTABLES.DBS`：8 位元組檔頭（筆數 150＋版本 1），150 筆磁碟記錄 stride `0x2c0`，`8+150*0x2c0` 恰等於檔長 105608；磁碟記錄＝`0x101` 位元組前綴＋`0x1bf` 位元組執行期本體 [已驗證] (build/data-x/DATABASES/SPELLTABLES.DBS:0x0)
- `SPELLTABLES.DBS` 法術顯示名為 64 units UTF-16LE（磁碟 `+0x19c`＝執行期 `+0x9b`；全檔最長 23 units）；`0x101` 前綴與執行期 `+0x5b` 資源基名、`+0x14b` 音效基名為單位元組 ASCII 內部路徑，不面向玩家 [已驗證] (build/data-x/DATABASES/SPELLTABLES.DBS:0x19c)
- `SPELLTABLES.DBS` 執行期欄位語義（四系歸屬、骰子、realm、目標型別 `+0x137` 等）見反組譯文件，本機僅驗算術未逐欄驗 [推測] (https://raw.githubusercontent.com/agluszak/wizardry-8-decomp/HEAD/docs/wiz8-data-formats.md)
- `NPC.DBS`：筆數 146，146 筆 `0x309` 位元組定長部＋每筆 u32 規則數＋若干 6 位元組規則（u32 fact id＋u16 述語）；146 筆規則總數 774，解析恰至 EOF（`113446+146*4+774*6`＝118674），無剩餘位元組 [已驗證] (build/data-x/DATABASES/NPC.DBS:0x0)
- `NPC.DBS` 每筆 `+0x58` u32 等於記錄號；`+0x4` 起 UTF-16LE 顯示名（全檔最長 19 units）；`+0x9e` 起單位元組別名（全檔最長 19 位元組，實例見記錄 0 該位移）；顯示名面向玩家但屬名稱（留英文），別名為內部鍵 [已驗證] (build/data-x/DATABASES/NPC.DBS:0x5c)
- `ENCOUNTERTABLES.DBS`：筆數 35，35 個 256 位元組名稱槽；`0x2304` 處 u32 表數 72，之後為變長表區；名稱槽 NUL 之後有大量非零殘留（槽 0 殘留 171 非零位元組，含 `0x12xxxx` 執行期位址形態值） [已驗證] (build/data-x/DATABASES/ENCOUNTERTABLES.DBS:0x2304)
- `ENCOUNTERTABLES.DBS` 表頭含混合編碼負載（單位元組名＋UTF-16 片段＋位址形態值）；内部表結構（`0x108` 表頭＋物種／稀有度／時段／挑戰度平行陣列＋64 位元組腳本名）見反組譯文件，本機未逐欄驗；表名與腳本名皆為內部鍵 [推測] (https://raw.githubusercontent.com/agluszak/wizardry-8-decomp/HEAD/docs/wiz8-data-formats.md)
- `ITEMTABLES.DBS`：筆數 15，15 個 256 位元組分類標籤；`0xF04` 處 u32 表數 114，114 張 `0x1f1` 位元組定長表，`4+3840+4+114*0x1f1` 恰等於檔長 60506；標籤與查表名皆為內部鍵 [已驗證] (build/data-x/DATABASES/ITEMTABLES.DBS:0x0)
- `MISSILETABLES.DBS`：8 位元組檔頭（筆數 36＋版本 1），36 筆記錄每筆 742 位元組，`8+36*742` 恰等於檔長 26720；記錄 `+0x1` 起為單位元組 NUL 結尾名稱；面向玩家與否未見消費端，暫列內部 [推測] (build/data-x/DATABASES/MISSILETABLES.DBS:0x8)
- `HITSOUNDS.TXT` 不是 DBS（首 u32 為 `0x2a2a2a2a` 筆數不合理），無可翻結構，一行帶過 [已驗證] (build/data-x/DATABASES/HITSOUNDS.TXT:0x0)
- 索引字串庫三檔共用一制：5 位元組檔頭（版本 1＋編碼旗標 0）；每筆＝u32 元資料甲＋u32 元資料乙＋u32 code-unit 數＋原文；檔尾＝逐筆絕對位移表＋筆數＋零 dword；三檔位移鏈皆精確銜接（`ITEMDESC` 1000 筆表起 `0x1e8f9`、`SPELLDESC` 160 筆表起 `0x6003`、`SPELLEFFECT` 160 筆表起 `0x785`） [已驗證] (build/data-x/DATABASES/ITEMDESC.DBS:0x0) (build/wiz8.asm:412879)
- 索引字串庫讀取器拒收超過 2000 code units 的字串（無符號大於比較，2000 通過、2001 失敗）；本檔三者最大長度為 474（`ITEMDESC`），上限綽綽有餘 [已驗證] (build/wiz8.asm:412973)
- 索引字串庫三檔當前編碼旗標皆 0＝原文以明文 UTF-16LE 存放（非 `0x9696` 變換）；讀取器僅在旗標非零時逐 unit 做 `~u−0x6969`（即 `0x9696−u`）變換，這解釋了 `tools/strings` 解不開 `SPELLDESC.DBS`：容器不同 [已驗證] (build/wiz8.asm:412983) (tools/strings/strings.go:16)
- `ITEMDESC` 1000 筆中 326 筆非空、`SPELLDESC` 160 筆中 101 筆非空、`SPELLEFFECT` 160 筆全空（最大長度 0）；非空文本全為可列印 ASCII，無 `%`、換行、回車與非 ASCII 字元 [已驗證] (build/data-x/DATABASES/SPELLDESC.DBS:0x11)
- 說明文字與物品／法術記錄的對應鍵在 `ITEMS.DBS` 內找不到（全記錄 u8／u16／u32 逐位移掃描，無值域吻合 1000／819 的索引欄；亦無等於記錄號的 id 欄）；位置對應（物品 i 取 `ITEMDESC[i]`）或按名查皆有可能，對應機制未定 [未取得] (build/data-x/DATABASES/ITEMS.DBS:0x4)
- `SPELLEFFECT.DBS` 160 筆全空且首元資料具固定魔數形態，現無玩家可見內容；是否為預留欄位未定 [推測] (build/data-x/DATABASES/SPELLEFFECT.DBS:0x5)
- `MONSTERS.DBS`／`ITEMS.DBS` 除名稱外無玩家可見文字（全記錄正則掃描僅名稱欄命中）；`NPC.DBS` 除顯示名外餘為內部鍵與規則資料 [已驗證] (build/data-x/DATABASES/MONSTERS.DBS:0x4) (build/data-x/DATABASES/ITEMS.DBS:0x4)
- MSG 全體 51 檔文法一致：u32＝1、u8 msgid＝1，後接若干文本段（8 位元組 `ff` 標記＋u32 code-unit 數＋`0x9696` 編碼 units，以編碼 NUL 結尾），段表之後為脚本尾；全 51 檔 6669 段解析恰至脚本尾，脚本尾內無 8 連 `ff`，故段边界無歧義 [已驗證] (build/data-x/MESSAGES/TEST.MSG:0x0)
- MSG 文本段起点恒在奇數位移：檔頭 `4+1+8+4=17` 為奇數，每段增量 `12+2*N` 為偶數，故首段在 `0x11`、後續恒奇數；decoder 須做非對齊 u16 讀取（`tools/strings` 逐筆 `u32` 長度模型不適用） [已驗證] (build/data-x/MESSAGES/TEST.MSG:0x11) (tools/strings/strings.go:18)
- MSG 每檔僅一則訊息（首 u32 恒 1、msgid 恒 1）：遊戲以檔名引用（虛擬路徑格式字串），檔內無第二層 id；`QUOTES/PCS` 36 檔每檔 146 段、尾 596 位元組，`MESSAGES` 13 檔每檔 105 段、尾 432 位元組（另 `TEST` 3 段尾 24 位元組、`MONASTERY` 45 段尾 192 位元組） [已驗證] (build/data-x/MESSAGES/ARNIKA.MSG:0x0)
- MSG 文本段為敘述碎片（單段可止於词中，下一段續接），6669 段解碼後全為可列印 ASCII＋段尾 NUL，無 `%`、換行控制碼與色碼；換段即換行與否由脚本尾決定，脚本尾語義未解 [推測] (build/data-x/MESSAGES/TEST.MSG:0x11)
- 索引字串庫讀取器（`0x0052ff80`）經統一虛擬檔層開檔／讀／定位／關檔（`0x00404c80／0x404ea0／0x405030／0x404e10`）；物品／怪物／事實／關卡／NPC／法術初始化函式分見 `0x0054a400／0x0054a8a0／0x0054ad00／0x0054ae20／0x0054aac0／0x0054acc10` [已驗證] (build/wiz8.asm:412879) (build/wiz8.asm:448671)
- 怪物讀取為逐筆定位（`4+index*0x297` 讀 `0x297` 位元組，去除 `#` 後綴）；物品初始化以 `count*0x10d` 配置連續陣列；字串庫執行期表基址 `0x0068c09c` 另見既有筆記 [已驗證] (build/wiz8.asm:449172) (build/wiz8.asm:448742) (notes/re/strings.md:5)
- 索引字串庫讀取器把 `nunits` 直接讀入呼叫者緩衝，唯一長度閘即 2000 units；一處呼叫點配置 `0x1200`（4608）位元組工作區，大於 2000 units 所需 4000 位元組，故 2000 以內譯文不溢該點；其餘呼叫點配置未逐一驗 [推測] (build/wiz8.asm:412980) (build/wiz8.asm:291375)
- 定長 DBS 的文字欄有硬上限（物品名 32、怪物名 24×4、法術名 64、NPC 顯示名實測至多 19、事實註解 106、關卡名實測至多 21 units）；譯文超長只能截斷或改 stride，改 stride 則讀取器定位全錯 [推測] (build/data-x/DATABASES/ITEMS.DBS:0x4)
- MSG 文本段為變長串接、無定寬欄，譯文變長不破壞檔形；但行寬／換行由未解的脚本尾與 UI 排版決定，須另驗 [推測] (build/data-x/MESSAGES/TEST.MSG:0x3a5)
- MSG 讀檔函式位址在僅含 `.text` 的反組譯中無字串交叉引用可錨定，未取得 [未取得] (build/wiz8.asm:683409)
- 往返一致風險一：填充殘訊須原樣保留（遭遇表名槽、`SPELLTABLES` 前綴含執行期位址形態值與 `Place holder` 殘字；`FACT` 定寬欄 NUL 後殘字）；encoder 不可重算或清零填充 [已驗證] (build/data-x/DATABASES/ENCOUNTERTABLES.DBS:0x4) (build/data-x/DATABASES/SPELLTABLES.DBS:0x108)
- 往返一致風險二：「空」有兩種寫法（長度 0 與長度 1＋單個 NUL；`ITEMDESC` 記錄 0 屬後者、`SPELLDESC` 記錄 0 屬前者），encode 須按原樣回寫、不可正規化 [已驗證] (build/data-x/DATABASES/ITEMDESC.DBS:0x5) (build/data-x/DATABASES/SPELLDESC.DBS:0x5)
- 往返一致風險三：索引字串庫位移表為自檔首起算的絕對位移，encode 改任一段長度即須重算全表；MSG 无位移表、顺序重建即可 [已驗證] (build/data-x/DATABASES/ITEMDESC.DBS:0x1e8f9) (build/data-x/MESSAGES/TEST.MSG:0x11)
- 未見壓縮與校驗碼：全部 12 DBS 與 51 MSG 的位元組皆已納入上述算術（定長乘法／位移鏈／段串接恰至 EOF），無尾隨雜湊或多餘區段；解碼→編碼僅須保序保位元組 [推測] (build/data-x/DATABASES/NPC.DBS:0x0)
- `PATCH.001` 覆蓋的 `ITEMS.DBS` 與 `SPELLTABLES.DBS` 與 DATA.SLF 版 byte 一致（`cmp` 無差異）；工具驗收應以 PATCH.001 版本為準 [已驗證] (build/patch-x/DATABASES/ITEMS.DBS:0x0)

## 附錄 A：DBS 一覽

| 檔名 | 結構 | 文字欄 | 編碼 | 上限 | 證據 |
|---|---|---|---|---|---|
| `ITEMS.DBS` | 檔頭 u32 筆數 819＋819×`0x10d` 定長 | 記錄首名稱 | UTF-16LE 明文 | 32 units | `ITEMS.DBS:0x0`，讀取 `wiz8.asm:448742` |
| `MONSTERS.DBS` | 筆數 595＋595×`0x297` 定長 | `+0x0/30/60/90` 四組名 | UTF-16LE 明文 | 各 24 units | `MONSTERS.DBS:0x0`，讀取 `wiz8.asm:449172` |
| `SPELLTABLES.DBS` | 8B 檔頭（150＋ver1）＋150×`0x2c0`（`0x101` 前綴＋`0x1bf` 本體） | 磁碟 `+0x19c` 顯示名；前綴／基名為內部 ASCII | UTF-16LE／ASCII | 64 units | `SPELLTABLES.DBS:0x0` |
| `NPC.DBS` | 146×`0x309`＋u32 規則數＋6B 規則×774 | `+0x4` 顯示名；`+0x9e` 內部別名 | UTF-16LE／ASCII | 顯示名實測 ≤19 units | `NPC.DBS:0x0` |
| `LEVELS.DBS` | 筆數 60＋60×`0xd8` 定長 | 記錄首名稱 | UTF-16LE 明文 | 實測 ≤21 units | `LEVELS.DBS:0x0` |
| `FACT.DBS` | 筆數 807＋807×`0x1d8` 定長 | `+0x4` 符號名（內部）；`+0x104` 設計註解 | ASCII／UTF-16LE | 註解 106 units | `FACT.DBS:0x0` |
| `ITEMDESC.DBS` | 5B 檔頭＋1000 變長筆＋位移表＋筆數＋零 | 全筆皆文本（326 非空） | UTF-16LE（旗標 0） | ≤2000 units | `ITEMDESC.DBS:0x0`，閘 `wiz8.asm:412973` |
| `SPELLDESC.DBS` | 同上，160 筆（101 非空） | 全筆皆文本 | UTF-16LE（旗標 0） | ≤2000 units | `SPELLDESC.DBS:0x0` |
| `SPELLEFFECT.DBS` | 同上，160 筆全空 | 無（全空） | — | — | `SPELLEFFECT.DBS:0x0` |
| `ENCOUNTERTABLES.DBS` | 35×256B 名槽＋表數 72＋變長表區 | 名槽／表名／腳本名（皆內部） | ASCII＋UTF-16 混雜 | 定寬槽 | `ENCOUNTERTABLES.DBS:0x2304` |
| `ITEMTABLES.DBS` | 15×256B 標籤＋114×`0x1f1` 定長表 | 標籤／查表名（皆內部） | ASCII | 定寬槽 | `ITEMTABLES.DBS:0x0` |
| `MISSILETABLES.DBS` | 8B 檔頭（36＋ver1）＋36×742 | 記錄 `+0x1` 名（內部） | ASCII NUL 結尾 | 槽內定寬 | `MISSILETABLES.DBS:0x8` |

## 附錄 B：MSG 文法（供 codex 工具用）

```text
file    := u32 1, u8 msgid(=1), segment+, script_tail
segment := 0xff×8, u32 nunits, u16[nunits]   # 0x9696 編碼，末 unit 為 0x9696(NUL)
```

- `nunits` 含結尾 NUL；payload 起点恒奇數，須非對齊讀 u16。
- `script_tail` 起於首個「非 8 連 `ff`」位移，至 EOF；全 51 檔尾內無 8 連 `ff`。
- 各檔段數／尾長：`MESSAGES` 13 檔 105 段尾 432B（`MONASTERY` 45 段尾 192B、`TEST` 3 段尾 24B）；`QUOTES/PCS` 36 檔 146 段尾 596B。
- 例：`TEST.MSG:0x0`（檔頭）→段 0 `0x11`（137 units，`0x89` 記於 `0xd`）→段 1 `0x12f`（148 units）→段 2 `0x263`（161 units）→脚本尾 `0x3a5`（24B 至 `0x3bd` EOF）。

## 命令紀錄

唯讀：`xxd`、`python3`（位元組解析，未执行遊戲或下載物）、`grep`/`awk`（`build/wiz8.asm` 行定位）。未寫遊戲目錄，未 commit。

```bash
for f in build/data-x/DATABASES/*.DBS; do stat -c '%s' "$f"; xxd "$f" | head -8; done
python3 # 定長算術、位移鏈、MSG 文法、字元掃描（見發現各條）
grep -n '52ff80\|54a400\|54a8a0\|54ad00\|54ae20\|54aac0\|4acc10' build/wiz8.asm
grep -n '0x7d0' build/wiz8.asm  # 2000 上限
cmp build/data-x/DATABASES/ITEMS.DBS build/patch-x/DATABASES/ITEMS.DBS
find build/data-x -iname '*.msg' | wc -l  # 51
```

## cc 抽查指引

- `ITEMS.DBS:0x0` 首 u32＝819 且檔長＝`4+819*0x10d`。
- `NPC.DBS` 按「`0x309`＋u32＋6×規則數」步進 146 筆恰至 EOF，規則總數 774。
- `TEST.MSG:0xd` u32＝`0x89`，payload 自 `0x11` 起 274 位元組，末 unit（`0x121`）為 `96 96`。
- `build/wiz8.asm:412973` 有 `cmp … 0x7d0` 接 `ja` 失敗分支。

## 更正（2026-10-08）：名稱欄尾端是記錄資料

上面第 8、10 行把名稱欄整寬當文字欄，不對。欄尾 units 存的是記錄資料，不是殘留：
- `ITEMS.DBS`：32 units 欄的第 30–31 unit 是資料；819 筆裡 808 筆第 31 unit 非零。舊版 dbsname 改名時清零整欄，所有物品的類別與圖示都變成 0（匕首），樂器無法使用（MG 實機截圖回報）。
- `MONSTERS.DBS`：每欄 24 units 的第 23 unit 是資料；2380 欄裡 1176 欄非零。
- `SPELLTABLES.DBS`：欄尾沒有資料。
- 改法（ec0869d）：只寫新字和一個 NUL，其後的原 bytes 保留；新字上限：物品 29、怪物 22、法術 63 units。`scripts/dbsname.test.sh` 斷言 NUL 之後的 bytes 不變，注入 `clear(field)` 會紅。
- 驗收：MG 實機「物品類別跟圖示都正常了」（2026-10-08）。怪物欄尾 unit 23 被清零的部分也由同一修正涵蓋，但沒有另外在實機上驗。

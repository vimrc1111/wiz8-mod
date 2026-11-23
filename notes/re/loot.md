# P5：掉落、寶箱與商店物品來源（靜態 RE）

本材料中，**74 Faust Halberd 有來源引用；96 Maenad's Lance 沒有被下述已解析來源引用**。共列出 136 個未引用 ID；這不是「全遊戲絕對拿不到」名單。任務腳本、合成／職業生成、初始裝備與其他 exe 硬編碼來源未完整納入，地圖／事件可達性也未證明。

只讀 worktree 的 build 材料，不執行遊戲、不下載、不補跑反編譯、不修改遊戲或工具。所有 VA 來自 `build/wiz8.asm`；該檔沒有 raw bytes，**本文所有函式與讀取點的前 16 bytes 均 [未取得]**，不從指令重組。`[已驗證]` 指靜態控制流與現有二進位資料相符，不表示實機測試。以下整數為 little-endian；檔案 offset 與記錄內的相對 offset 分開標示；ID 與槽號從 0 起算。

## 發現

- [已驗證] **ITEMS 是定義庫，不是來源表**：`0x54a400` 讀 u32 筆數後逐筆讀 `0x10d=269` bytes 至 `0x68516c`；本檔 819 筆，長度 `4+819×269=220315`。item i 起點 `4+269i`，名稱從 +0 讀 UTF-16LE；沿用 dbsname 可編輯 29 units 的限制，不將後面欄位當名稱。`0x51c020` 建立實例時將 ID 寫到實例 +0。74 的定義起點 0x4dc6，96 的起點 0x64e4；有定義不保證有取得管道。來源：build/wiz8.asm:448684、build/wiz8.asm:448742、build/wiz8.asm:384724；build/dbs/DATABASES/ITEMS.DBS@0x0；internal/dbsname/dbsname.go:36。

- [已驗證] **ITEMTABLES 外層格式**：載入 `0x54a510`；+0 u32 分組數 G，接 G 個 256-byte、NUL 結束的名稱區，再 u32 表數 T，接 T 個 `0x1f1=497` bytes 記錄。實檔 G=15、T=114；表數位於 0xf04，表 t 起點 `0xf08+497t`，恰到 EOF 60506。載入後表指標陣列是 `0x6836b0`、表數 `0x683f7c`。每表 +0..0xff 是名稱緩衝區，**只取第一個 NUL 前文字**；尾部有舊記憶體樣式殘值，不能全區掃 ID。+0x100 u32 是分組索引，供 `0x57a0e3` 的分組列舉；並非掉落等級或區域的自動選表規則。來源：build/wiz8.asm:448796、build/wiz8.asm:448878、build/wiz8.asm:448937、build/wiz8.asm:511589；build/dbs/DATABASES/ITEMTABLES.DBS@0xf04。

- [已驗證] **ITEMTABLES 槽與骰欄位**：`0x4f88f0` 掃 40 槽，槽 j 起點 `+0x104+5j`：+0 u16 類別／非空標記（0 跳過；詳細類別名稱未對照）、+2 u16 item ID、+4 u8 權重 w。w=0 是固定加入項，不是 0% 掉落。表 +0x1cc u8 啟用等級／價值篩選；+0x1cd 是 4-byte 物品抽取數骰，+0x1d5 是 4-byte 金錢骰，兩者由 `0x445500` 使用；+0x1d1..0x1d4 及 +0x1d9..0x1f0 的語意 [未取得]。全部 114 表有 3376 個非空槽、522 個不同 item ID，均落在 0..818。來源：build/wiz8.asm:334785、build/wiz8.asm:334910、build/wiz8.asm:92698、build/wiz8.asm:92738；build/dbs/DATABASES/ITEMTABLES.DBS@0xf08。

- [已驗證] **RNG 與骰**：`0x40efa0` 的 R(0)=0；一般小正整數 n，在匯入亂數 r∈[0,32767]、乘積不溢位的假設下，`R(n)=floor(r×n/32767) mod n`。r 的匯入名稱與實際分布 [未取得]，不可稱嚴格均勻。`0x517970` 的 4-byte 骰為 `(Z=i16@+0, N=u8@+2, S=u8@+3)`，`D=Z+Σ(1+R(S))`，N=0 回 Z。後文百分比近似僅作讀式說明；cc 應保留實際不等號，或先算 R 的離散分布。來源：build/wiz8.asm:19149、build/wiz8.asm:378002。

- [已驗證] **MONSTERS 掉落來源格式**：`0x54a760` 全讀、`0x54a8a0` 單筆讀，u32 筆數後每筆 `0x297=663` bytes；595 筆、EOF 394489。`0x4e5720` 用怪物實例 +8 的模板 ID 載入此記錄。模板 M 的 8 槽起點 `+0x1f3+10j`，依次為 kind:u8、類別／非空:u16、ref:u16、c:u8、數量骰:4 bytes；kind=0 的 ref 是直接 item ID，kind=1 是 **0-based ITEMTABLES 索引**。+0x243 為金錢骰。現檔有 207 個 kind=0 非空槽（111 個不同 item ID）、142 個 kind=1 非空槽（33 張表）；另有 176 個非空標記但 kind=0xcd 的殘值槽，不能當物品引用，dispatch 也不處理此 kind。模板 +0xd0 bit0 會切去 NPC 分支，故這些是欄位引用統計，不保證每槽實際走普通怪物路徑。來源：build/wiz8.asm:449032、build/wiz8.asm:449091、build/wiz8.asm:307977、build/wiz8.asm:335274、build/wiz8.asm:335358；build/dbs/DATABASES/MONSTERS.DBS@0x0。

- [已驗證] **普通怪物生成掉落：逐槽、先門檻再骰數量**：`0x4f8cb0` 由 `0x4e4ee0` 的事件處理分支在 `0x4e50b0` 呼叫，並非在此看到「全戰鬥合計只擲一次」。每個非空槽接受條件 `R(100)≤c`，所以均勻近似是 `min(c+1,100)/100`，c=0 仍非零；成功後求 Q=D，Q≤0 不產生。kind=0 建立 Q 次物品實例；kind=1 **外圈仍重複 Q 次，每次都以 Q 為抽取數呼叫 `0x4f88f0`**，不能默改成只呼叫一次或宣稱只掉 Q 件。Q=2 時會兩次各要求抽 2 個隨機槽，固定項也隨每次呼叫加入。實例建立另由 `0x51c020` 決定堆疊／充能，不把「實例數」等同箭矢件數。事件碼跳表、所有死亡方式和戰鬥結束通知到此入口的完整鏈 [未取得]。來源：build/wiz8.asm:307252、build/wiz8.asm:307406、build/wiz8.asm:335278、build/wiz8.asm:335291、build/wiz8.asm:335370、build/wiz8.asm:384621。

- [已驗證] **物品池不是單次累積權重抽籤**：`0x4f88f0` 先加入所有非空且 w=0 的固定項；剩餘合格槽形成列表 A。若 +0x1cc≠0 且合格數小於要求數 K，回退為所有非空槽（包含已加入的 w=0 槽）。W 為當時 A 的 w 總和。反覆抽 `j=R(|A|)`，再抽 `R(W)`，當 `R(W)≤w_j` 才輸出並刪除該槽；拒絕則重試、不消耗 K。成功 K 次或 A 空即結束；W 在刪除後**不重算**。重複 ID 的不同槽各自有機會；不放回的是槽而非 ID。固定項不扣 K，回退可能使固定項再中一次。均勻且 W>0 的近似接受權重是 `min(w_j+1,W)`，不是 w_j；精確條件轉移為 `P(next=j|A)=P(R(|A|)=j)·P(R(W)≤w_j) / Σ同項`，W=0 時 R(0)=0。來源：build/wiz8.asm:334785、build/wiz8.asm:334863、build/wiz8.asm:334907、build/wiz8.asm:334937、build/wiz8.asm:334978、build/wiz8.asm:335026。

- [未取得] **等級與區域尚不能轉成全域取得率**：`0x4f88f0` 的 +0x1cc 篩選以 `L=0x4ef420()` 和 item 定義 +0x86 的 u32 數值，查 exe `0x617ce0` 起的 7 組 12-byte 三元組，條件是 `L≥a 且 b≤value<c`；+0x86 稱「價值」仍為推測。表值不在現有 raw 資料，無法列出各級門檻。`0x4ef420` 是有效隊員總等級欄 +0x89 的整數平均，並非最高單人等級。區域由具名寶箱／怪物模板引用決定，不可把 ITEMTABLES 的分組名稱當地區自動對應。ENCOUNTERTABLES 已看到 35×256 名稱區、0x2304 的計數 72，但其餘記錄長度不固定，怪物出現率、區域與等級選怪格式及讀者 [未取得]；未用檔內數字碰撞猜掉落 ID。來源：build/wiz8.asm:335000、build/wiz8.asm:321004；build/dbs/DATABASES/ENCOUNTERTABLES.DBS@0x2304。

- [已驗證] **NPC 主檔與貨單是可變長串流**：`0x54aac0` 先讀 u32 記錄數，每筆主體長 0x309=777；主體 +0 u16 版本／條件值（名稱語意未定）、+4 UTF-16LE 名稱、+0x9d u8 控制是否有尾表。當 `u16(+0)>1 且 u8(+0x9d)=0`，緊接 u32 n 與 n 個 6-byte 貨項：`itemID:u32, 基準庫存量:u8, 固定供貨旗標:u8`。主體 +0x2ca 的磁碟值不作指標解讀，loader 會清除並建立記憶體列表。146 筆全部符合尾表條件，僅 13 筆 n>0，共 774 貨項、376 個不同 ID；最後游標 118674=EOF。例：NPC 2 Anna 起點 0x61e，尾表第一項 0x92b 是 ID 99、量 10、旗標 1。來源：build/wiz8.asm:449414、build/wiz8.asm:449463、build/wiz8.asm:449479、build/wiz8.asm:449504、build/wiz8.asm:471058；build/dbs/DATABASES/NPC.DBS@0x61e。

- [已驗證] **商店生成和補貨另外擲骰**：`0x55a630` 逐貨項重複基準量 q 次，接受 `R(100)<p`，將成功次數交 `0x55a7b0` 加入庫存；候選 p 常數為 0、25、50、75、100。旗標非零直接選分類 4，否則 `0x55aad0` 依物品定義 +0x85 的有號 byte 和平均隊伍等級分類。該函式保留 i8 窄化；可重現為 `a=i8(trunc(i8(trunc(i8(L)×100/30))/5))`，lo=clamp(a,1,16)，hi=clamp(i8(a+4),4,20)：item值>hi 回 0；≥lo 回 3；否則 item值+4≥lo 回 2，剩餘回 1。分類到 p 的跳表 raw 內容 [未取得]，按相鄰分支排列對應 0/25/50/75/100 只可標 [推測]。`0x55afa0`、`0x55ab80` 的補貨還查目前庫存、時間差 0xa8c0／0x15180、旗標，以及 ID 508 的 fact 0x15f 特例；不可將初次單件機率套成每次進店整張貨單固定機率。來源：build/wiz8.asm:471020、build/wiz8.asm:471102、build/wiz8.asm:471491、build/wiz8.asm:471573、build/wiz8.asm:472020、build/wiz8.asm:472160。

- [已驗證] **NPC 死亡物品池不同於商店貨單**：`0x50b9e0` 讀 NPC 主體 +0x64 的 u32 d；d=0 或 d≥表數便不載入，否則取 ITEMTABLES[d−1]（此處是 **1-based**，且上界排除 d=114）。將 40 槽 item 與 w 分別複製到 NPC 動態物件 +0x30 的 i16[40]、+0x115 的 byte[40]，數量骰複製到 +0x10e。`0x4f8cb0` 的 NPC 分支先掉仍存在且 w=0 的項並清除，再擲 K=min(D,剩餘槽數)，用 R(40) 選槽、R(W)≤w 接受、成功清成 −1；這裡 w 與比較為有號值，不能照抄普通池的所有型別。已被取走／偷走的 −1 不再掉。故不能將商店貨單當 NPC 必掉全庫存。Antone 的主體 0xa75、+0x64=59，連至表 58 Antone drop。來源：build/wiz8.asm:361238、build/wiz8.asm:361253、build/wiz8.asm:335119、build/wiz8.asm:335156、build/wiz8.asm:335205；build/dbs/DATABASES/NPC.DBS@0xad9。

- [已驗證] **地上物與寶箱引用在 .STS**：`0x513650` 分派 LVLS 內的 ITEM→`0x514c80`、TRES→`0x43d1f0`。現有 34 個 STS 全部可從 RIFF 包裝逐區塊走到 EOF。區塊頭共 10 bytes：tag[4]、兩個旗標 byte、payload長度 u32@+6；RIFF payload 先 u32 子塊數=1，再 LVLS；LVLS payload 先 u32 子塊數=7、u32 level ID，再子塊（現檔首塊在絕對 0x20）。七類為 STAT/MONS/ITEM/CUBE/MONG/LOCK/TRES，不用任意搜尋四字碼作 parser。34 檔的 ITEM 共讀到 1175 個實例、334 個不同 ID；TRES 共 178 個容器記錄。IFL 是 TGA 動畫檔名清單，不能當 Item File List；RLK／WPT／僅兩個 LVL 尚未解析為物品來源，不能聲稱全 levels 都已排除其他來源。來源：build/wiz8.asm:372289、build/wiz8.asm:372367、build/wiz8.asm:372447、build/wiz8.asm:474238；build/levels/ARNIKA/ARNIKA2.STS@0x0；build/levels/CAMP/BITMAPS/WATERFLOW.IFL@0x0。

- [已驗證] **ITEM 區塊格式與固定擺放**：payload +0 u32 是根物件數，之後每個實例 0xad=173 bytes；+0 u32 實例序號、+9 u32 item ID（其後至 +0x14 是物品實例其餘欄位，數量／充能的完整意義 [未取得]）、+0x15/+0x19/+0x1d 為 f32 世界座標、+0x25/+0x29 為狀態位、+0x31 u32 是後繼存在標記。+0x31 非零就繼續讀下一個 173-byte 節點，直到零才算下一個根；不是磁碟 offset。`0x514c80` 會重建指標。現有檔全部根數／鏈終止／chunk 長度吻合、ID 均在 0..818；例如 Arnika 的 ITEM 在 0x16ee2，第一實例 0x16ef0 指向 ID 588。這是固定擺放的定義引用；載入器此段未擲出現率骰，但可拿取性仍受場景、旗標和劇情限制。來源：build/wiz8.asm:374127、build/wiz8.asm:374143、build/wiz8.asm:374176、build/wiz8.asm:332168、build/wiz8.asm:332182、build/wiz8.asm:384724；build/levels/ARNIKA/ARNIKA2.STS@0x16ee2。

- [已驗證] **TRES 以名稱接到物品池**：payload +0 u32 格式值（現檔為 1）、+4 u32 容器數；每筆 384 bytes，+0 是 128-byte 場景物件名，+0x80 是 256-byte ITEMTABLES 名稱。`0x43d1f0` 將後者讀入場景物件 +0x24c；`0x445500` 經 `0x4f88a0` 名稱比對找表，再骰 K=表+0x1cd、呼叫 `0x4f88f0`，另骰金錢，設物件 +0x350 表示已生成。呼叫前把 +0x354 交匯入 `0x5eb1d8`（重設 seed 的用途為推測，匯入名稱未提供）。現檔名稱以忽略 ASCII 大小寫可接上 177 筆、55 張不同表；唯一不匹配是 RAPAXUPPERFLOOR 的 CONSAFEDOOR→CONSTABLE CHEST，表名字串在 0x10c31。不能替它猜成其他箱。座標要再接場景幾何，本文只提供 STS 路徑和物件名。來源：build/wiz8.asm:82230、build/wiz8.asm:82315、build/wiz8.asm:334723、build/wiz8.asm:92656、build/wiz8.asm:92688；build/levels/RAPAX/RAPAXUPPERFLOOR.STS@0x10c31。

- [已驗證] **74 Faust Halberd 有 7 張池引用，並有場景連結**：表／槽／w 分別為 20/39/2、21/39/1、22/39/2、33/5/3、50/27/1、57/36/1、58/18/7；對應 Chaos、Knowledge、Body、Rapax King、Bayjin Nessie、Marten、Antone drop。前三張啟用尚缺門檻表的等級篩選；33、50、57、58 的 +0x1cc=0。可查驗路徑包括 BAYJIN/BAYJIN2.STS 的 FOURLID（記錄 0x6b1d→表 50）、RAPAX/RAPAXUPPERFLOOR.STS 的 KINGCHEST（0x10d31→33）、SEACAVES/SEACAVE1.STS 的 HIGCHESTLID（0x443f→57）；表 50 有 40 個隨機槽、W=210、K=2+2d2。故不能以這份材料支持「Faust 沒有任何來源」；場景可達、開箱条件與實際一次取得率仍須另證。來源：build/wiz8.asm:334978、build/wiz8.asm:92698；build/dbs/DATABASES/ITEMTABLES.DBS@0x71a5；build/levels/BAYJIN/BAYJIN2.STS@0x6b1d。

- [已驗證] **未引用集合為 136 個 ID**：下節集合 U=`{0..818}−(全部 ITEMTABLES 非空槽 ID ∪ MONSTERS kind=0 非空槽 ID ∪ NPC 貨單 ID ∪ STS ITEM ID)`，聯集大小 683。NPC 死亡池及 TRES 均指向已納入的 ITEMTABLES，展開不增加集合。為保守起見，**全部 114 表（含 Test 等未證明可達者）都納入**；怪物模板也不按是否實際生成而刪除。96 Maenad's Lance 屬 U，74 不屬 U。這個判定是格式欄位級掃描，不是搜尋 ID 的 byte pattern。來源：build/wiz8.asm:334785、build/wiz8.asm:335274、build/wiz8.asm:449504、build/wiz8.asm:374143；build/dbs/DATABASES/ITEMS.DBS@0x64e4。

- [未取得] **任務獎勵與動態來源不在排除證明內**：NPC SCRIPTS 不在材料，任務 give-item 指令格式／讀取函式未取得。合成、角色起始物品、職業升級生成、其他 exe 常量表也未完整枚舉；例如上面 U 內的 572 Bag of Items，`0x4f8cb0` 在 `0x4f90b5` 就用常數 0x23c 建立裝掉落物的容器，已足以反證「U 全部永遠拿不到／不會生成」。P4 的 `0x548e60` 另有自製物品候選表 0x616fe0，尚缺表值。因此 96 目前只能說這四組已解析欄位沒有引用，不能提升為全遊戲不可取得。來源：build/wiz8.asm:335398、build/wiz8.asm:446696。

## 未引用物品

以下每列依據 **A**：已查全部 114 張 ITEMTABLES 的有效非空槽、595 怪物模板的 kind=0 非空槽、146 NPC 的条件式貨單尾表、34 個 STS 的 ITEM 全鏈；NPC 死亡／TRES 透過物品池已涵蓋其引用。採所有表的保守聯集，不做劇情可達性刪選。A 不含腳本、配方、初始裝備、其他硬編碼與未解析關卡格式。英文名照 `build/items-en.tsv` 的 ID 對照，不修正原拼字；415 是空名稱。

**74 Faust Halberd：不列入未引用表**；已查到 7 張池與上述寶箱連結。**96 Maenad's Lance：列入未引用表**，不是已證明全遊戲不可得。

| ID | 英文名 | 依據 |
|---|---|---|
| 32 | Dragon Slayer | A |
| 63 | Faerie Stick | A |
| 64 | Wooden Dowel | A |
| 65 | Notched Shaft | A |
| 71 | Ymmu's Paw | A |
| 85 | Rammbus Staff | A |
| 89 | Crux of Crossing | A |
| 93 | Longstem Spade | A |
| 96 | **Maenad's Lance** | A |
| 107 | Thermal Pineapple | A |
| 119 | Elven Bow | A |
| 145 | Shield of Soaring | A |
| 169 | Gossamer Gown (U) | A |
| 196 | Gossamer Gown (L) | A |
| 198 | Jazeraint Skirt | A |
| 234 | Necromatic Helm | A |
| 245 | Magnetic Boots | A |
| 250 | Goatfoot Boots | A |
| 268 | Ring of Reflextion | A |
| 279 | Blackbelt of 5 Flowers | A |
| 283 | Cross of Protection | A |
| 291 | Scarab Necklace | A |
| 295 | Vitriolic Warder | A |
| 296 | Diamond Ring | A |
| 297 | Rock Crystals | A |
| 298 | Brimstone Nuggets | A |
| 300 | Manta Roots | A |
| 301 | Lion's Claws | A |
| 302 | Ju-Ju Stones | A |
| 303 | Dragon's Teeth | A |
| 308 | Deadman's Hair | A |
| 309 | Stone Pro Airs | A |
| 320 | Pipes of Doom | A |
| 321 | Chromatic Lyre | A |
| 323 | Lyre of Cakes | A |
| 325 | Lute of Sloth | A |
| 328 | Horn of Prometheus | A |
| 330 | Cornu of Demonspawn | A |
| 333 | Helazoid Banner | A |
| 365 | Short Gooda Vine | A |
| 366 | Short Gooda Vine | A |
| 367 | Gooda Vine Rope | A |
| 412 | Bunch of Bananas | A |
| 413 | Salted Munkmeat | A |
| 414 | Moser's Mojo Tea | A |
| 415 | （空名稱） | A |
| 416 | Milk of Magmanasia | A |
| 417 | Dungore's Malt Ale | A |
| 418 | Bottle of Old Jake's | A |
| 424 | Zip Gun | A |
| 431 | Ring of Felix | A |
| 432 | Earthshaker Bomb | A |
| 433 | Tarnished Brass Key | A |
| 435 | Zuzu Petals | A |
| 436 | Rotten Meat | A |
| 446 | Storage Key | A |
| 450 | Drugged Meat | A |
| 455 | Queen's Key | A |
| 458 | Assassin's Cap | A |
| 467 | Fuel Tank | A |
| 476 | Book of Mind Stab | A |
| 489 | Instakiller | A |
| 500 | Mirror Armor | A |
| 501 | Ivory Blade | A |
| 504 | Vampire Chain | A |
| 509 | Canezou Robe | A |
| 510 | Canezou Helm | A |
| 511 | Canezou Dagger | A |
| 571 | Sea Cave Gate Key | A |
| 572 | Bag of Items | A |
| 573 | Bela Powder #trynton | A |
| 580 | Chaos Moliri | A |
| 587 | Bullroarer | A |
| 599 | Omnigun Mk 1 | A |
| 600 | Omnigun Mk 2 | A |
| 601 | Omnigun Mk 3 | A |
| 603 | Omnigun Mk 5 | A |
| 604 | Omnigun Mk 6 | A |
| 605 | Omnigun Mk 7 | A |
| 606 | Omnigun Mk 8 | A |
| 607 | Omnigun Mk 9 | A |
| 608 | Omnigun Mk 10 | A |
| 609 | Omnigun Mk 11 | A |
| 610 | Omnigun Mk 12 | A |
| 612 | Astral Dominae (fake) | A |
| 613 | Screg's Key | A |
| 614 | Solar Powerpaks | A |
| 615 | IUF Level 1 Security Pass | A |
| 617 | IUF Level 3 Security Pass | A |
| 618 | IUF Level 4 Security Pass | A |
| 619 | IUF Level 5 Security Pass | A |
| 620 | IUF Level 6 Security Pass | A |
| 626 | Rocket Launcher | A |
| 627 | Z'Ant Alliance | A |
| 628 | Yamir Alliance | A |
| 629 | Z'Ant Reciprocal Alliance | A |
| 630 | Mook Alliance Offer | A |
| 631 | Mook Alliance Acceptance | A |
| 633 | T'Rang Tracking Device | A |
| 650 | Zynaryx Plate | A |
| 651 | Steelhide Breastplate | A |
| 652 | Featherweight Armor | A |
| 653 | Beastslayer | A |
| 654 | Ebon Staff | A |
| 657 | Stone Idol | A |
| 684 | Jack in the Box | A |
| 685 | Searchlight | A |
| 686 | Holograph Projector | A |
| 687 | Noisemaker | A |
| 688 | Watchdog Alarm | A |
| 689 | Lava Lamp | A |
| 690 | Forcefield Generator | A |
| 691 | X-Ray Scanner | A |
| 692 | Invigorator Belt | A |
| 693 | Magic Mirror | A |
| 695 | Demon in the Box | A |
| 696 | Regenerating Stone | A |
| 697 | Microwave Ray | A |
| 698 | Strobe Light | A |
| 699 | Electroshocker | A |
| 700 | Water Cannon | A |
| 701 | NegatAir | A |
| 702 | Jackhammer | A |
| 709 | Port-O-Potty | A |
| 710 | Electric Eye | A |
| 711 | Loudspeaker | A |
| 724 | Demon Doll | A |
| 726 | Ale | A |
| 727 | Microwave Chip | A |
| 731 | High Voltage Transformer | A |
| 739 | Tripleshot Crossbow | A |
| 741 | Ale | A |
| 747 | Blue Flower | A |
| 748 | Stinger | A |
| 749 | Stinger | A |
| 779 | Trynnie Breastplate | A |

## 給 cc 的本機解析與查詢方法

這裡是工具規格，未將程式碼入版。實作可用任意能讀 little-endian packed 欄位的語言；勿用 host struct 的自然對齊，勿將檔案中的舊指標解參照。

1. 先讀 ITEMS 的 count/stride 並核對英文 TSV 的 ID，建立 819 個空的反向來源列表。ITEMTABLES 按 G、T 定位，為每個非空槽建立 `(tableID, tableName, slot, itemID, w, filterFlag, K-dice, fileOffset)`。保留重複 ID 槽；類別=0 的殘值不得算引用。
2. MONSTERS 按 663-byte 模板解析 8 槽，記 `(monsterID, slot, kind, ref, c, Q-dice, npcFlag, fileOffset)`；kind=0 接 item，kind=1 接 table；其他 kind 報為未處理，不猜。另以 +0xd0 bit0、+0xcd 的 NPC 索引候選接 NPC；該索引全鏈仍需驗證。遇 NPC 模板，查其 NPC 主體 +0x64，注意 1-based 和 loader 的上界條件。
3. NPC 使用串流游標，不以檔案長度除筆數求 stride。主體讀 777 後按條件讀 n×6 貨項。為每項建立 `(npcID, npcName, itemID, q, flag, fileOffset)`。名稱長度以第一個 UTF-16 NUL 為界；只取主體 +4 的名稱，不掃其他文字域。所有尾表即使 q=0 仍保留「欄位引用」，另註無正庫存支持。
4. 每個 STS 驗證 RIFF→LVLS、長度界線與子塊數；保存 LVLS levelID 及相對路徑。ITEM 依根數與 +0x31 讀全鏈，反向接 itemID、世界座標、物件序號、旗標與磁碟 offset；TRES 依 384-byte 記錄讀容器名與表名，忽略 ASCII 大小寫比對後將 table→items 展開。不要把 178 個容器都當不同池；對不上名稱的 1 筆要保留為異常，不悄悄略過。
5. 查詢任一 item ID，列出所有直接物品引用和 table 的逆向引用者，分類為怪物、NPC 死亡、商店、地上物、寶箱，並附具名地點或「模板所在地未取得」。額外列「僅表內引用、沒有已知根」；不要憑表名替未解析 ENCOUNTERTABLES 補地點。若要篩可達來源，另作 rooted graph，不改本票 U 的定義。
6. 機率用參數表示：普通槽門檻 F100(c)=P(R(100)≤c)，直接物品若 Q>0 至少有一實例；池內固定項與隨機項分算。給定候選集合 A、K 和 W，以本文條件轉移遞推刪槽，累計命中目標 ID 的任一槽機率，最後再對 K 的骰分布求和；數個槽有相同 ID 時取聯集，不能只加百分比。kind=1 外圈重複 Q 次要忠實保留，NPC 則以仍存在的槽與清除狀態求解。假設獨立 RNG 才可對不同來源使用 `1−∏(1−p)`，並明標此假設；固定 seed、重試開箱與重新進區不能直接視為獨立。
7. 商店先輸出 `(q,flag,分類函式,跳表映射待補)`。補到 jump table 後，初次至少一份可用 `1−(1−P(R(100)<p))^q`（獨立模型）計算；補貨需輸入現存庫存、fact、時鐘，不能只靠 NPC.DBS 回一個永久不變的概率。地上物回「固定記錄；可拾取條件待驗」，任務回「腳本未取得」。
8. 最後做集合差與自檢：819 定義、114 表、3376 池槽、207 直接怪物槽／142 表槽、774 商店項、1175 地上實例、178 寶箱；唯一物品數分別 522／111／376／334，聯集 683，補集 136。每種 parser 都需游標到 EOF 或 chunk 邊界、所有已採用 item ID 在範圍內；不得把不認識格式的檔當零來源而不註明。下次補到來源後，重算 U 並標版本／材料差異。

## 驗證紀錄

本次用一次性唯讀 Python／struct 解析材料並做邊界與集合斷言；未保存解析工具。ITEMTABLES、MONSTERS、NPC 整檔長度及全部 34 個 STS 的 chunk／ITEM 鏈／TRES 長度均吻合。資料庫記錄數、來源統計與 74／96 的集合歸屬已實算；沒有執行遊戲，機率分布的獨立性與實際可達性未實測。報告驗收命令：`bash scripts/check-research.sh notes/re/loot.md`；結果通過（17 條結論，0 個問題）。

## 規格疑慮

驗收器 `scripts/check-research.sh` 只接受 URL 或 `path:行號` 作為「發現」條目的來源，未接受題目允許的 `path@0x偏移` 單獨出現；本文每條同時附 asm 行號和必要的資料偏移以符合兩者。機檢只檢來源／標記格式，不驗 RE 正確性、引用集合完整性或可取得性。

## cc 抽查（2026-10-08，本機一次性 python -I 照上文格式重解析 build/dbs，工具不入版）

- **74／96／58 的池引用** [已驗證：cc 獨立解析 ITEMTABLES＋MONSTERS kind=0]：74 的 7 張表與槽／w 與上文逐筆相同；96 在兩者皆無；58 Blade Cuisinart 只在表 33「Rapax King Chest」槽 0、w=4。ITEMTABLES 長度 `4+256·15+4+497·114=60506`＝EOF。
- **29 Cane of Corpus** [已驗證：同上]：只在表 71「Don Barlone Drop」槽 27、w=50（W=175，26 個隨機槽，固定項 578、5），K 骰 bytes `01 00 02 03`＝1+2d3。沒有怪物模板以 kind=1 引用表 71，與「NPC 死亡池（主體 +0x64＝72，1-based）」路徑相符 [推測：未讀 Don Barlone 的 NPC 主體 +0x64]。MG 實玩記憶：Don Barlone（鼠人）掉、非必掉（MG 2026-10-08）。
- **骸之杖掉率** [推測：照上文 NPC 分支規則模擬 20 萬次，假設 R 均勻、獨立]：約 78%。

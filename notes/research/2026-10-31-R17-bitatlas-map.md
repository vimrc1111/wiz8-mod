# R17：Wizardry 8 連續座標接 BitAtlas（2026-10-31）

TL;DR：可先交付「Arnika 節點俯視底圖＋連續位置」，但尚不能承諾原版帶材質 automap 的完整重現。
PVL 有打包的三維 automap nodes；X/Z 是水平、Y 是高度；探索位是節點索引位元集，存檔放在各 LVLS 的 AUTO。
分層值來自 MAPFILTERS.TXT，讀入時乘 500；原版像素投影、朝向起點／正負、出口及 NPC 的完整離線解碼仍未取得。
建議 BitAtlas 加可選 image／連續 pose；ddraw 採既有 endFrame 週期取樣。所有結論均為本地 static_analysis，未執行遊戲。

## 發現

### 1. 幾何、檔案與 PNG 證明

- [已驗證] LEVELS.SLF 共 4,956 entries：OCT 34、PVL 34、STS 34、RLK 26、WPT 35、TGA 4,682、IFL 104、PCX 4、LVL 2、無副檔名 1。用既有 `internal/slf.ReadArchive` 檢查目錄、範圍與重疊，沒有全包抽出。逐地圖主檔清單在 build-only 的 `maps.tsv`（47 個 exe 表槽皆列，包括空槽）；並非 47 張都有資料。來源：build/r17/catalog.go:3、build/r17/evidence.txt:1、build/r17/maps.tsv:1。
- [已驗證] 地圖 0 對應 ARNIKA/ARNIKA2，主檔為 OCT／PVL／RLK／STS／WPT，另有共用 BITMAPS 子目錄。這份 SLF 中索引 11、23、28、30、34、35、40–46 的主檔精確路徑比對為空；不能把空項直接當「損壞」或補成其他地圖。原版地圖表是 VA 0x604478、stride 107，現在索引 u32[0x686a70]，無圖為 0xffffffff。來源：build/r17/maps.tsv:2、build/r17/probe.py:23；build/wiz8.asm:56853、build/wiz8.asm:57189。
- [推測] 副檔名用途：OCT 是場景空間／網格資料，PVL 是場景物件及多個序列化區段（含 automap nodes），STS 是初始狀態，RLK 可能為連通資料，WPT 可能為路點，TGA／PCX 是圖像材質，IFL 是動畫影格名單，LVL 可能是編輯／舊版關卡資料。RLK／WPT／LVL 的完整格式本次未取得；不能以名字推定出口／樓梯欄位。PVL/OCT 網格數不符有專用診斷；STS 的區塊格式可與既有 loot 研究交叉核對。來源：build/r17/catalog.tsv:1、build/r17/strings.txt:4；notes/re/loot.md:31。
- [已驗證] 字串入口：`map_partymarker_a.tga` VA 0x64ba48 在 0x582361 被載入，建立渲染物件後存 [0x68f29c]；`map_layerbuttons.sti` VA 0x634af4 位於資源表，沒有直接的立即位址引用。AUTOMAPLARGE／SMALL 未在 exe 可列印字串掃描中命中；因此以隊伍 marker 與 MapFilters 引用接續追查，不假造 large STI 的單一 xref。來源：build/r17/strings.py:5、build/r17/strings.txt:39、build/r17/strings.txt:28；build/wiz8.asm:522024、build/wiz8.asm:522077。
- [已驗證] PVL loader 0x4baff0 在 0x4bb829 呼叫 0x584dd0，失敗診斷明文指向「after automap nodes」。0x584dd0 依序讀 f32 間距至 [0x64b914]、u32 節點數至 [0x68f27c]、配置並讀 n×4 bytes 至 DWORD[0x68f280] 指向的表；外層還讀 0xffffffff 區段結束檢查。不是一張 PNG／STI 的載入格式。來源：build/wiz8.asm:251847、build/wiz8.asm:251868、build/wiz8.asm:525180、build/wiz8.asm:525302。
- [已驗證] 節點每筆 u32 q：`ix=q>>21`、`iy=q&0x3ff`、`iz=(q>>10)&0x7ff`。還原到場景座標：`X=Ox+(ix+0.5)*S`、`Y=Oy+(iy+0.5)*S`、`Z=Oz+(iz+0.5)*S`；O 為 [0x68f240/244/248]，S 為 [0x64b914]。反向打包 0x5852b0 亦依 Z、X、Y 分量除 S 後組位元。浮點轉整數的邊界捨入行為未另證，不把它泛稱 floor。來源：build/wiz8.asm:519542、build/wiz8.asm:519576、build/wiz8.asm:525527。
- [已驗證] Arnika PVL 在 SLF offset 0x443c06，檔長 8,626,343；檔內 0x833337 是 f32 S=2000、u32 n=6983，0x83333f 起為節點表，0x83a05b 是結束值。讀出 6,983 個互異 q；索引範圍 X=3..108、Y=3..12、Z=0..108。已產出 `build/r17/arnika-nodes.png`，672×672，X 向右、Z 向上、不同高度診斷色，所有高度先合併；這是可重跑的原版資料俯視圖證明。來源：build/r17/evidence.txt:3、build/r17/probe.py:30。
- [已驗證] 找到 automap 場景重繪路徑 0x581030：當 dirty flags [0x68f25c/25d] 設定，依 [0x68f26c] 設相機範圍，呼叫引擎相機介面、0x426f80，再處理渲染 surface。探索節點處理 0x5807b0／0x580380 讀節點、測試位元，0x5809f0 以節點世界位置查詢場景物件（0x42f9a0），接著存取物件及渲染狀態。來源：build/wiz8.asm:520454、build/wiz8.asm:520474、build/wiz8.asm:519728、build/wiz8.asm:519951。
- [推測] 原版 automap 主要是對場景幾何做俯視渲染、以節點控制揭露／場景物件顯示；STI 是地圖畫面的 UI 資源，不是每關各一張預繪底圖。證據是同一場景相機、surface、空間查詢與 PVL node 資料流；尚未取得原版地面三角形／牆線段每筆頂點、索引、UV 與材質的完整磁碟結構，不能聲稱節點就是原版所有畫圖幾何。來源：build/wiz8.asm:520474、build/wiz8.asm:519951、build/wiz8.asm:251847；build/r17/evidence.txt:2。
- [未取得] 原版材質俯視 PNG 的完整解碼器。第一輪從 marker／資源字串追到 automap 初始化與相機；第二輪從 PVL node loader／OCT-PVL mismatch 追到節點與場景查詢，未取得 mesh 磁碟欄位／完整渲染契約，依停止條件結束。現有 PNG 是節點佔據診斷圖，不是原版材質圖，也不能視為碰撞／可行走證明。正式 adapter 應先解析 OCT/PVL 網格、TGA/IFL，再離線正投影；原版材質可取得部分仍應使用原圖素，不以本次診斷色當產品完成。來源：build/r17/probe.py:38、build/r17/evidence.txt:5；build/wiz8.asm:519951、build/wiz8.asm:520454。

### 2. 世界座標、投影與方向

- [已驗證] 世界位置讀取函式 0x4534c0 把物件 +0x100/+0x104/+0x108 三個 32-bit 分量複製出來；配合既有介面及浮點使用可讀成 X/Y/Z float32。隊伍世界物件 W=DWORD[0x659c0c]，可直接唯讀 W+0x100 的三個 float；另一位置來源是方向／移動物件 D=DWORD[0x65a0f8] 的 +0x8c/+0x90/+0x94（0x421070），勿混用「全域指標本身」與其欄位。來源：build/wiz8.asm:111878、build/wiz8.asm:44162；notes/re/iface/437f30.tsv:41。
- [已驗證] 水平是 X/Z，高度是 Y。相機中心、左右／上下範圍取三維向量的 +0 與 +8；中間 +4 用相機高度，MapFilters 產生的裁切平面法向是 (0,1,0)。所以題目的「上下樓層 z」應在 Wiz8 讀 Y，報告不延用錯軸。來源：build/wiz8.asm:518946、build/wiz8.asm:519007、build/wiz8.asm:517797、build/wiz8.asm:521134。
- [已驗證] 引擎原點 O 來自場景物件 DWORD[0x6598a4] 的 +0x24/+0x28/+0x2c（三軸 min）和 +0x30/+0x34/+0x38（三軸 max）；0x46ce30 複製到 automap 範圍，0x581911 設 O=min。引擎缺場景時用備援範圍，不能拿備援值當各圖標頭實值。來源：build/wiz8.asm:145452、build/wiz8.asm:521120、build/wiz8.asm:521134。
- [未取得] 每關 bbox 在 OCT/PVL 的精確磁碟 offset；第一輪查看 Arnika 前 160 bytes 與 node 區段，第二輪追場景 bbox getter 0x46ce30 到 automap 消費端，取得記憶體欄位但未取得來源 parser。47 圖換算參數的來源契約可指定為「場景 bbox min/max＋各 PVL 的 S」，不能宣稱目前已離線讀出全表。來源：build/r17/sample.py:4、build/r17/probe.py:30；build/wiz8.asm:145452、build/wiz8.asm:521120。
- [已驗證] 單位可確認為引擎世界單位：MapFilters 的 layer 數字乘 float[0x5ebc40]=500 才放入平面高度；node spacing 可由 PVL 讀，也有建立時依地圖分支設 2000／4000 的路徑。沒有證據把世界單位直接稱公尺、或把所有關卡硬編碼為 S=2000。來源：build/wiz8.asm:517779、build/wiz8.asm:275740、build/wiz8.asm:525556；build/r17/evidence.txt:3。
- [推測] adapter 可自行固定北向底圖：像素 `u=(X-Ox)*p/S`、`v=(Zmax-Z)*p/S`，p 是輸出每節點像素數。再以相同式轉位置和標記，避免目前 floor 導致每 2000 單位跳一格。這是新底圖的明確座標約定，並非已還原的原版 automap 像素公式；本次 PNG 用局部 node index，未帶世界 bbox，不可直接疊實機世界座標。來源：build/r17/probe.py:38；build/wiz8.asm:519542、build/wiz8.asm:521134。
- [未取得] 原版隊伍 marker 的完整 world→screen 矩陣、最後像素原點與軸向正負。已從 marker 0x582361 追到物件 [0x68f29c]、0x57f3c1 的縮放，以及 0x581030 相機設定；第二輪查看 0x57fc70 相機中心與 0x57e660/0x57fb40 的畫面進出保存，矩陣操作跨引擎匯入，未還原完整契約，停止。不把上述 adapter 公式標成遊戲原式。來源：build/wiz8.asm:522024、build/wiz8.asm:518278、build/wiz8.asm:520474、build/wiz8.asm:519007、build/wiz8.asm:517300。
- [已驗證] 朝向候選欄位：0x420dc0 讀 D+4 乘 float[0x5ebcf0]=57.295784（exe 檔內 offset 0x1ebcf0）；0x44fd39 取值後交 0x4538f0，該函式經 0x517c60 正規化後寫 W+0xd4/+0xd8。D+8 另作第二角分量。此倍率與弧度轉度相符，D+4 為弧度、W+0xd4 為度的解釋屬推測；不可直接將 W+0xd4 當弧度輸出，也不能把 pitch 混成 heading。來源：build/wiz8.asm:43886、build/wiz8.asm:107024、build/wiz8.asm:112316。
- [未取得] heading 的弧度零點、順逆時針與 PNG 北向的實機對照。第一輪看既有轉向 0x477b90／0x477cd9，第二輪追 D+4→W+0xd4 與 automap marker，相機／billboard 的朝向對應仍未取得；D+4 的弧度解釋有上述轉換倍率佐證，但零點與符號仍不明，不以四捨五入角度硬填 NESW。首版可 dir=-1，保留 raw heading 供後續校準；校準後才輸出 `round(theta/(π/2)) mod 4`，其中 theta 必須先轉成北起順時針。來源：build/wiz8.asm:160318、build/wiz8.asm:43886、build/wiz8.asm:112316；notes/re/party-move.md:19。

### 3. 同關多層

- [已驗證] 層界資料在 `data-x/AUTOMAP/MAPFILTERS.TXT` 的每關 section，`LAYER=` 可有多筆；Arnika 的 35、−6.5、−14 在檔案本身可讀。0x57eb79 開啟該檔，按目前 map 索引數 section；識別 LAYER 後轉浮點並乘 500，建裁切平面，加入 [0x68f1a8] 列表。層按鈕由 0x580f20 按索引操作，當前層索引存 [0x64b918]。來源：build/wiz8.asm:517670、build/wiz8.asm:517743、build/wiz8.asm:517779、build/wiz8.asm:520354；build/data-x/AUTOMAP/MAPFILTERS.TXT:1。
- [推測] 這是高度裁切式分層，不是每層另一張 level id；layer 平面 Y=數值×500，section 內非 LAYER 的 TGA 名看來是排除不適合俯視的材質／物件過濾條件。adapter 應保留原始順序與世界高度，採同圖的 layer 選擇，或用複合 id `0:layer` 加 place 路由。來源：build/wiz8.asm:517743、build/wiz8.asm:517855、build/wiz8.asm:520354；build/data-x/AUTOMAP/MAPFILTERS.TXT:1。
- [未取得] layer 邊界等號歸屬、兩平面間與最上／最下層的全部顯示規則、自動跟隨高度的切層規則。第一輪看 LAYER parser，第二輪看 0x580f20 和 0x57f469 的相機高度比較，僅能確認裁切與按鈕更新，不能將所有節點按 Y 區間篩掉就宣稱等同原版。PNG 明確先合併高度；正式擴充需新增上下層對照 fixture。來源：build/wiz8.asm:520354、build/wiz8.asm:518307；build/r17/evidence.txt:5。

### 4. 已探索與存檔

- [已驗證] 記憶體探索位元物件 B=DWORD[0x68f288]，bit 資料指標 DWORD[B+0x18]、bit 數 DWORD[B+0x1c]；0x43b620 用 `(i>>5, 1<<(i&31))` 查 u32 位元，0x43b390 設位。位元 i 對應 node 表第 i 筆，而非 `y*w+x` 平面格。DWORD[0x68f28c] 是另一份處理狀態位元集：測試後設位再呼叫 0x5809f0，不能把它當獨立「玩家走過」來源。來源：build/wiz8.asm:79776、build/wiz8.asm:79443、build/wiz8.asm:519788、build/wiz8.asm:519851。
- [已驗證] 存檔是 10-byte chunk header：tag[4]、flags[2]、payload size u32@+6。RIFF payload 先 u32 子塊數；LVLS 先 u32 子塊數及 u32 level id，然後逐子塊走長度。兩份 save 全部走到 EOF，Arnika.SAV 有 5 個 LVLS，Rapax 副本有 32 個；不是靠任意搜 AUTO 字串判格式。來源：build/r17/probe.py:51、build/r17/evidence.txt:6、build/r17/evidence.txt:13。
- [已驗證] Arnika.SAV 的 level 0 AUTO chunk offset=0x8ae91，payload=1057 bytes；另一副本同 level chunk offset=0x1a0ddf，payload=1077。二者 bit count 都是 6983，吻合 PVL 節點數；codec 資料長分別 1029、1049，所以不是把磁碟 payload 直接當 873-byte 未壓縮位圖。存檔讀前／讀後 SHA256 相同。來源：build/r17/evidence.txt:12、build/r17/evidence.txt:45、build/r17/probe.py:59。
- [已驗證] AUTO payload 的已確認包裝：+0 u32 0xdeadd00d、+4 u32 bit count、+8 u32 codec byte count C、+12 C bytes 編碼資料、+12+C u32 0xdeadd00d；其後 +16+C 為 0xf00df00d（文字標記尾段開始）。引擎 0x581e60→0x43aec0 讀探索，再讀文字記錄；寫出為 0x581ce0→0x43b0e0。記憶體解碼結果是 u32 bit vector，最低位先對應 i。來源：build/wiz8.asm:521580、build/wiz8.asm:79074、build/wiz8.asm:79165、build/wiz8.asm:521421；build/r17/probe.py:59。
- [未取得] codec 的完整離線實作／名稱及兩份存檔真正揭露的位元數。第一輪核對 AUTO 邊界與節點數，第二輪追 0x43aec0／0x43b0e0，發現編解碼經 0x5eb74c..0x5eb798 引擎匯入。沒有下載第三方 decoder，也不把編碼 bytes 做 base64 假裝 mapSeen。可先由 hook 讀已解碼的 B+0x18，再以逐 node 的映射輸出。來源：build/wiz8.asm:79165、build/wiz8.asm:79250；build/r17/probe.py:59。
- [推測] BitAtlas 現有 `Map.seen[i]` 可以把同一平面上多高度 node 指到同一 atlas 格，由位元 OR 得揭露；正式多層時需按 layer 分別映射，或以新 image mask 定義 node→像素區域。這只能表示原版揭露狀態，不能以「隊伍正好踩過該 node」替代原版演算法。來源：build/wiz8.asm:521186、build/wiz8.asm:521212；../bitatlas/atlas.ts:25。

### 5. 可以上圖的標記

- [未取得] 換圖出口的完整「觸發區幾何＋目的地圖索引＋落點」記錄。PVL loader 有 triggers 區段、save 有 TRIG／APST；第一輪看 PVL 診斷引用、第二輪對照兩存檔 chunk，尚未解到目的 map 欄位。建議候選來源 PVL trigger 定義與動態 TRIG，不能把每個觸發器都標成出口，也不以 WPT 名稱猜目的地。來源：build/r17/strings.txt:12、build/r17/sample.txt:9；build/wiz8.asm:251847。
- [未取得] 樓梯的可用標記表。已檢查主檔目錄分類與 layer parser／切層；同關階梯可能只是連續坡面／網格，本次未找到獨立 stairs 記錄。分層平面本身不是樓梯位置，不能拿 layer 高度自造 stairs mark。來源：build/r17/maps.tsv:1；build/wiz8.asm:517743、build/wiz8.asm:520354。
- [推測] NPC 靜態初始位置應從 STS 的 MONS 實例接 NPC 定義，活位置則要讀場景怪物／NPC 實例清單；TGA 友好／中立／敵對 marker 是圖素，不含全世界 NPC 座標。本次未取得 MONS→NPC id→世界位置的完整 adapter 欄位契約，故不產可發布 NPC 清單。來源：notes/re/loot.md:17、notes/re/loot.md:31；build/r17/strings.txt:36、build/r17/sample.txt:5。
- [已驗證] 存檔 AUTO 尾段支援文字記錄：0x581f6b 後每筆讀三個 4-byte 值，再讀字數並按 `2*字數` 讀 UTF-16；交 0x581360，記憶體文字物件為 16 bytes，+0/+4 二維座標、+8 layer 值、+0xc 字串指標。文字長度、最多 200 筆等檢查在建立端。來源：build/wiz8.asm:521660、build/wiz8.asm:521690、build/wiz8.asm:520700。
- [未取得] 上述文字的座標單位、內建地名／玩家筆記的分類與原版自動標註完整來源。PVL 有 named position points（載入 0x4bdc90），但未證明與 AUTO 筆記共用；第一輪看 named positions 診斷及載入點，第二輪看 0x581360 與 save 尾段，未取得 join。不可把玩家筆記冒充自動地名。來源：build/r17/strings.txt:9；build/wiz8.asm:251815、build/wiz8.asm:520700、build/wiz8.asm:521690。
- [未取得] 門的完整離線位置／連接邊及即時 open 欄位。候選為 PVL props（門模型／場景物件）及 save APST／LOCK/LCKS 狀態；已檢查 PVL props 診斷與 STS／save 分塊兩輪，沒有建立可靠的門 id→狀態對應。連續空間的門還可能是線段或物件，不能直接填 grid 的 `mapEdges(x,y,d)`。來源：build/r17/strings.txt:16；notes/re/loot.md:31、build/r17/sample.txt:10。

### 6. BitAtlas 缺口與 hook

- [已驗證] 現有 tile 可包一張底圖為單一大 tile，veil 已有類似用法，但位置 xform 先 floor(g/s)、dx/dy/s 驗證整數、mapSeen 是格號、找路基於離散格。它可做粗節點原型，不能無損保留 Wiz8 連續位置、多層或世界門幾何；把位置先取整也不等於真正的連續支援。來源：../bitatlas/DESIGN.md:36、../bitatlas/atlas.ts:23、../bitatlas/atlas.ts:35；build/r17/evidence.txt:4。
- [推測] 最小產品擴充應選獨立 `kind:image`＋可選連續 pose，詳見下節草案。image 底圖仍由原版資料現場建置，未解出材質前明示「節點預覽」。保留舊 tile/grid/rooms 的單位、floor、0..3 dir；直接把既有 dir 改為弧度會破壞舊案，不建議。來源：../bitatlas/atlas.ts:6、../bitatlas/atlas.ts:23、../bitatlas/DESIGN.md:63；build/r17/probe.py:38。
- [已驗證] `mappos.h` 是 C++ `struct MapPos`，update 的 m/px/py/d 都是 int，`snprintf` 用 %d，依相同內容去重＋10 秒心跳，失敗 have=false 下次重試；沒有內建「連續兩次相同」取樣器。現有 proxy.c 由 mingw gcc 編 C 並 `-nostdlib`，使用 Win32 I/O，不能直接 include 該 C++ header，也不能直接輸入 float。來源：../bitatlas/mappos.h:13、../bitatlas/mappos.h:39；hook/ddraw/build.sh:42、hook/ddraw/proxy.c:27。
- [推測] hook 首選既有 `menu_end_frame`（slot VA 0x5eb628，primary renderer DWORD[0x659634]）中原 endFrame 呼叫前的週期取樣，採樣分支放在選單可見性判斷之外；約每 100–250 ms 取樣、最多 4 Hz 寫檔，靜止每 10 秒更新 seq，避免只掛移動更新導致靜止過期。另一候選是主更新 0x56018a 呼叫 0x44fc20 返回後；後者在選單／automap／暫停是否持續執行未取得，不能保證心跳。來源：hook/ddraw/proxy.c:1082、hook/ddraw/proxy.c:1104；build/wiz8.asm:479435。
- [推測] 每次讀 level=[0x686a70]、W=[0x659c0c]、W+0x100/+0x104/+0x108 位置、W+0xd4 heading 候選；可比較 D=[0x65a0f8] 的 +0x8c 位置與 +4 原角，定位轉場／場景不同步。需檢查已載圖、指標頁有效、finite、map 前後一致，換圖時丟棄半套樣本；未知／無圖寫 map=-1。以 exe 所在根目錄建立完整路徑寫 MAPPOS.JS，寫同一行到既有 log。朝向未校準前 dir=-1。來源：build/wiz8.asm:111878、build/wiz8.asm:44162、build/wiz8.asm:112316、build/wiz8.asm:57189；hook/ddraw/proxy.c:27。
- [未取得] endFrame 在視窗最小化／暫停等全部狀態的頻率與 runtime 安全保證，本次也未安裝新 detour。第一輪核對 proxy 既有 overlay hook，第二輪對照主更新，只有靜態候選。實作應保留原呼叫、限制同執行緒讀取，禁止在 DllMain 做長 I/O；codec 未解前可先直接讀記憶體探索位。來源：hook/ddraw/proxy.c:1082、hook/ddraw/proxy.c:1104；build/wiz8.asm:479435、build/wiz8.asm:79776。

## DESIGN.md 修改草案（本票不改共用檔）

這一節是提案，不宣稱現有 validate／viewer 已支援。版本與舊資料行為維持原狀，新增欄位採 opt-in：

| 擴充 | 欄位與規則 | 相容／驗證 |
|---|---|---|
| 底圖 | `Map.kind='image'`；`image:{src,pw,ph}`；w/h 為 atlas 畫布尺寸；src 指向建置產生 PNG | 新 validator 接受 image 分支，檢查正尺寸及本地資源；舊三種 kind 不變。舊 viewer 不保證可讀新 kind，adapter 必須與新版 viewer 成套發佈 |
| 浮點轉換 | `image.worldToImage:[a,b,c,d,tx,ty]`；u=aX+bZ+tx、v=cX+dZ+ty；用有限數、矩陣可逆，不做 floor | 僅 image 啟用；舊 xform 繼續原有整數格語意。畫面與游標逆算使用同一矩陣 |
| 高度／分層 | `image.layers:[{id,label,src,yMin?,yMax?}]`；layer id 與世界 Y 分離；邊界契約下含上不含 | layer 邊界須等後續原版規則校準，不能直接把 MapFilters 當已驗證區間。首版可只給合併層，提供人工切換 |
| 位置 | 保留 `mapPos(1,map,x,y,dir,seq)`；可選新增 `mapPose(1,map,X,Z,Y,theta,layer,seq)`，theta 為北起順時針弧度或 null | 新頁面先登記兩 callback，依同 seq 優先 pose；舊頁面用 mapPos 粗位置。theta 不重用 dir 欄；NaN/Infinity 拒絕，layer 未知可 null |
| 格子降級位置 | mapPos 送整數世界 X/Z（會損失不足一世界單位），dir=-1；新版以 pose 保留浮點 | 若同檔為舊頁提供額外 callback，須用 `typeof mapPose==='function'` guard，避免舊頁丟錯；不把 integer 世界單位當 atlas pixel |
| 探索 | 首版沿用 mapSeen 的 node bits，image 新增 `seenNodes:[{u,v,r,layer?}]`，同 i 對應 PVL i；r 是繪製遮罩半徑，僅顯示用途 | 不把格子 seen[] 偷換語意；不同高度不互相揭露。先定義 node 覆蓋形狀再驗收。重載存檔若探索需回退，現有 OR 累積協定要另加 session/reset 契約；不可漏掉 |
| 動態門／NPC | image 首版只做可靠的靜態 marks；動態支援另定 stable object id、線段／點及 replace 快照 | mapEdges 保留 grid 邊語意，不硬塞浮點門。NPC 移動與開門狀態不寫死 atlas.js |
| 找路與 walk | image 暫不提供 grid BFS；walk 的 image 點／visited 需距離容差欄位，manual 可先用 | 空間近不代表連通，節點也不等於 navmesh；不能把 PNG 非透明像素一律當可走 |

MapPose 若持續移動，不能要求兩個浮點樣本完全相同才寫，否則會長期不更新。建議相同 map／同 layer 連續兩次有效就允許，座標／角度使用量化後的去重鍵（例如 1 世界單位、1 度）但保留原 float 輸出，seq 統一遞增。心跳、log、失敗重試、寫完整檔再替換／避免半寫等規則由新 C helper 與 C++ helper 共用契約；目前 `mappos.h` 是直接 fopen(wb)，原子替換屬建議改進而非既有能力。

## 探勘腳本、證據與限制

所有 `build/...` 引用均相對 `wiz8-mod/`；`../bitatlas/` 是從該目錄解析，不是 repo 根的上一層。VA 為既有 `build/Wiz8.exe` 對應反組譯；字串掃描按既有研究 VA−0x400000 計算。沒有把整段反組譯或原版表／圖貼入報告；本報告只保留欄位、位址、計數與必要小樣本。

從 repo 根重跑：

```bash
cd wiz8-mod
GOCACHE="$PWD/build/r17/go-cache" go run ./build/r17/catalog.go > build/r17/catalog.tsv
python3 build/r17/strings.py > build/r17/strings.txt
python3 build/r17/probe.py
```

`catalog.go` 重用既有 SLF reader，目錄是 build-only JSON／TSV；probe 只對 Arnika PVL 讀整份 8.6 MB 樣本，沒有把 475 MB 封包全讀進記憶體。節點 offset 是本次 fixture 已核對的候選，不是所有 PVL 通用 parser；assert 不符就停止。PNG 是系統 Python 標準庫直接編寫 PNG，無下載、第三方工具或遊戲 exe 執行。存檔以 read-only 讀，chunk EOF、長度、雙 sentinel 與讀前後 hash 均核對。`sample.py` 是第一輪候選掃描／存檔分塊輔助，可重跑；不需要將其輸出入版控。

調查分兩輪封頂：幾何由資源 xref→node／場景；投影由隊伍欄位→相機匯入；分層由文字檔→按鈕消費端；探索由 chunk→bitvector codec；標記由目錄／區段→載入端；協定由 DESIGN／型別→helper／proxy build。各題未取得的欄位即停止，不擴張為完整反編譯。沒有讀寫 /mnt/c，沒有上網，沒有執行遊戲，沒有 commit。

驗收：`bash wiz8-mod/scripts/check-research.sh wiz8-mod/notes/research/2026-10-31-R17-bitatlas-map.md`（repo 根）。此機檢只檢查來源形狀與三態，不驗證 native 相機校準或圖片內容；未取得項不等於機檢已證實可用。

## 建議分單

1. **adapter：Arnika 幾何與底圖**。先把本次硬編碼樣本改為可驗證的 PVL 區段 parser、解出 bbox 與 OCT/PVL 地形／材質，讀 MapFilters，產 atlas.js＋原版材質底圖；擴展到 34 個有檔主地圖，缺檔槽明示。驗收：所有區段長度／索引／bbox 檢查，Arnika 與至少一張多層關卡的 PNG、validate 零錯，node 數與 AUTO bit count 對得上，輸出僅 build／試玩包。不得把節點色塊預覽當材質驗收通過。
2. **ddraw hook：位置、朝向與探索快照**。C／Win32 helper 接 endFrame，讀本文指標鏈，先輸出無朝向位置，再以四方向校準 theta；讀 B 位元輸出 mapSeen，處理換圖與讀檔重設。驗收：測試替身涵蓋 null／NaN／轉場／失敗重試／靜止 10 秒心跳，log 與 JS 同 seq；後續授權實玩再驗 Arnika／多層／戰鬥與選單，不在本票執行遊戲。不要沿用 C++ header 直接 include C 的做法。
3. **BitAtlas 擴充：image＋mapPose**。實作上節 opt-in 欄位、浮點正逆轉換、連續角度、layer 選擇與明確的探索 reset；舊 kind 與協定維持原行為。驗收：舊 atlas／viewer 測試全過，新增負座標、非整數位置、四方向角、跨層、讀檔回退與 file:// 載入案例；PNG 與兩個世界測點疊合，image 不啟用未定義的 grid 路徑搜尋。

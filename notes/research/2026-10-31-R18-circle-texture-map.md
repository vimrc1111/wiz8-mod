# R18：Circle 一張原版材質俯視圖（2026-10-31）

TL;DR：Circle（OCT 634,246 B、PVL 781,083 B）可離線畫原版材質俯視圖。
PVL 前段是 97 筆材質名表＋62 個 mesh 區段（頂點 xyz 三元組已驗證），TGA 皆未壓縮真彩。
`build/r18/render.py`（標準庫，`python3 -I`）取平面樓板區段頂點重建三角網並貼原版
`COSFLOOR_01.TGA`，輸出 `build/r18/circle.png`（900×2143）；疊 PVL 節點中心另存
`build/r18/circle-nodes.png`，同層節點落在地面三角內 35/154（bbox 內 35/57）。
三角形索引與區段→材質對應兩輪未取得（列搜尋範圍），三角連接與 UV 為重建（推測）。

## 發現

### 1. OCT／PVL 網格磁碟結構（loader 讀取順序佐證）

- [已驗證] Circle 主檔位置：OCT 在 SLF 位移 101270530 長 634,246；PVL 位移 101904776 長 781,083。來源：build/r17/catalog.tsv:1041、build/r17/catalog.tsv:1042。
- [已驗證] PVL loader 是 0x4baff0，多區段循序讀取：每次先經 0x404ea0 讀 u32 再比對 `0xffffffff` 區段哨兵，不符就推字串 VA（0x60e1dc／0x60e214／0x60e1ac）走診斷。來源：build/wiz8.asm:251152、build/wiz8.asm:251835、build/wiz8.asm:251866、build/wiz8.asm:5430。
- [已驗證] automap 節點讀取是 0x4baff0 在 0x4bb829 呼叫的 0x584dd0：先讀 f32 間距至 [0x64b914]、u32 節點數至 [0x68f27c]，再按 `n*4` 配置並讀 n 個 u32（`lea ecx,[eax*4]`）。來源：build/wiz8.asm:251848、build/wiz8.asm:525180、build/wiz8.asm:525258。
- [已驗證] 節點位元拆包在 0x58509c 用 `shr eax,0xa` 取跨欄位分量，與 R17 的 `ix=q>>21、iy=q&0x3ff、iz=(q>>10)&0x7ff` 一致。來源：build/wiz8.asm:525350。
- [推測] 0x4baff0 內依序呼叫的 0x4bce20、0x4bd0d0、0x4bdc90、0x4be200 是各 PVL 區段 parser（按呼叫點順序，非逐一反編譯）。來源：build/wiz8.asm:251676、build/wiz8.asm:251784、build/wiz8.asm:251818、build/wiz8.asm:251740。
- [已驗證] Circle PVL 檔頭：檔內位移 0x0 起 u32 為 97（材質筆數，見下）、62、0x0004002c；其後是材質名表（32 B 名＋32 B 貼圖名，間雜編輯器堆積指標垃圾），止於約 0x32f7（其後接全零與 `-1、-1、0、1…` 預設浮點塊）。來源：build/r18/render.py:49。
- [已驗證] Circle PVL 有 62 個 mesh 區段（特徵 `ffffffff 00 01 00 00` 全檔共 62 處）：每段為 `[u32 ffffffff][u32 256][u32 nv][u32 nt][u32 D][u32 -1][u32 X][u32 -1或1][nv 筆 xyz f32]`，頂點在段頭＋32。nv 總和 3,472。來源：build/r18/render.py:49。
- [已驗證] 段頭實例：檔內 0x3342 為 `(nv,nt,D)=(123,326,173)`，0x7072 為 `(29,49,28)`；X 欄位在前段為 -1、後段為遞增 id（62、63、…、96），Y 欄位為 -1 或 1。來源：build/r18/render.py:56。
- [已驗證] 頂點即世界座標三元組：0x3342 段 123 點 x 13938.5..25803.8、y -1457.0..3059.4、z -60450.8..-55821.4；0x7072 段 29 點為等 z 牆帶。來源：build/r18/render.py:60。
- [推測] 頂點之後的小浮點塊是法線／顏色／逐角 UV（無大值、有 `0、1、-0.1` UV 形狀；0x71ee 起 1,988 B 可拆為 29 法線＋29×RGBA＋147 逐角 UV，末端恰接 D 記錄，屬吻合但未證實的切分）。來源：build/r18/render.py:60。
- [已驗證] D 筆 16 B 記錄數恆等於 D（如 0x79b2 起 28 筆 `(0,0,-1)+packed u32`，該段 D=28），內容為面／頂點法線＋打包 u32。來源：build/r18/render.py:60。
- [未取得] 三角形索引陣列。第一輪掃頂點後小浮點塊（含逐角 UV 候選）與段尾 `nv` 個 u32；第二輪掃全檔 u16 小值連續區（0x4356 起 2,668 筆等 128 處）皆非 `<nv` 索引形狀；另證中段塊無 `|v|>5000` 浮點，故非位置湯（soup），索引仍缺。來源：build/r18/render.py:60。
- [未取得] 區段→材質對應。材質名僅 97 筆而 D 最大逾 380，D 非材質索引；0x75257／0x783dd 的逐物件貼圖表與材質表同序但夾堆積垃圾，未解出 join 欄。來源：build/r18/render.py:49。
- [已驗證] Circle 節點區段在檔內 0xbe497：間距 2000.0、396 筆、尾哨兵 `0xffffffff`；396 個 q 互異；索引範圍 X 2..19、Y 2..5、Z 0..71。來源：build/r18/render.py:68。
- [推測] OCT（634,246 B）主體是 BSP／碰撞類浮點資料（4 對齊字中 92% 為有限浮點、無貼圖名），僅在檔內 0x5ad1a 嵌一段同格式 mesh（265,264,0），其餘磁碟結構未取得。來源：build/r18/render.py:49。

### 2. 材質：貼圖名→TGA、IFL、TGA 格式

- [已驗證] 三角引用的是無路徑基名（如 `CosPillar_01.tga`），按下層 BITMAPS 子目錄規則對到同地圖 TGA：`CIRCLE\BITMAPS\COSFLOOR_01.TGA` 在目錄第 956 行（位移 90387114，196,652 B）。來源：build/r17/catalog.tsv:956。
- [已驗證] IFL 是 CRLF 影格名單：`PART-0004.IFL` 60 B 列 4 格，`RED_ENERGY001A.IFL` 280 B；靜態取第一格（慣例，未見反例）。來源：build/r17/catalog.tsv:1。
- [已驗證] TGA 皆未壓縮真彩 type 2、無色盤：24-bit（描述 0x00，左下原點）如 256×256（18＋196608＋26＝196652）；32-bit（描述 0x08）如 128×128（18＋65536＋26＝65580）。來源：build/r18/render.py:76。
- [已驗證] TGA 尾 26 B 是 v2 footer（8 零＋`TRUEVISION-XFILE.`＋零），像素區嚴格等於寬×高×通道。來源：build/r18/render.py:76。

### 3. 產圖：render.py、circle.png（選層說明）

- [已驗證] `build/r18/render.py` 只用系統 Python 標準庫（`python3 -I` 可跑）：自帶最小 SLF 目錄解析、TGA 解碼、PNG 編寫（zlib＋crc），不讀寫 /mnt/c、不執行遊戲、不下載。來源：build/r18/render.py:1。
- [推測] 選層：畫 L0 主樓板帶（y -100..700 的平面區段，yrange≤900，共 101 點），不用最上層；因頂層房間皆牆＋地組裝段、無孤立平面段，硬畫會混入牆面。來源：build/r18/render.py:178。
- [推測] 三角連接是 XZ Delaunay 重建（187→邊長≤9000 保留 160），平面 UV 以 2000 世界單位一磚貼 `COSFLOOR_01.TGA` 最近點取樣；法向＋Y 門檻因俯視重建自然成立，未用原版每三角法向（索引未取得故無從驗）。來源：build/r18/render.py:108。
- [已驗證] 輸出 `build/r18/circle.png` 900×2143；世界 x 1095.5..21720.2、z -43655.3..5473.0；每像素 22.916 世界單位；著色像素佔 0.239。來源：build/r18/render.py:196。
- [已驗證] 重跑命令：`cd wiz8-mod && python3 -I build/r18/render.py --node-iy 3 --max-edge 9000`。來源：build/r18/render.py:176。

### 4. 對照：circle-nodes.png、節點落在地面三角比例

- [已驗證] 原點 O 用全幾何 bbox 中心對節點格中心擬合（平移のみ，S＝2000 固定）：O＝(-10542.8,-134710.7)；跨度核對：節點 z 144000 對幾何 z 143774（差 226），x 36000 對 39721（側牆／空隙）。來源：build/r18/render.py:267。
- [推測] 樓層 pattern：平面段均高 {165（L0）、2050（L1）} 與節點 iy {3（足165）、4（足2165）、5（足4165）} 差一常數 Oy≈-5835，L0 樓板對 iy＝3。來源：build/r18/render.py:267。
- [已驗證] `build/r18/circle-nodes.png` 疊 iy＝3 的 154 節點（紅點）；落在地面三角內 35/154＝0.227，限樓板 bbox 內 35/57。來源：build/r18/render.py:279。
- [推測] 比例未達高標主因是樓板僅覆蓋 L1 南段（ix 5.3..15.6、iz 45.0..69.6），其餘 L1 地面藏於組裝段（牆地混合，未分離）；形狀（跨度差 226、層高 pattern）仍支持座標系一致。來源：build/r18/render.py:279。

## 建議

1. **adapter 推到 34 張地圖的做法**：每圖以本報告 §1 段 walker 取全部頂點＋0xbe497 式節點區（掃描 S／n／哨兵，不寫死位移）；TGA 按 `<MAP>\BITMAPS\<基名>`（大小寫不敏感）解，IFL 取首格；bbox 用全頂點跨度、O 用本報告中心擬合；MAPFILTERS 有 `LAYER=` 才分層（Circle 的 `[LEVEL_COSMIC_CIRCLE]` 為空段，不分層，見 build/data-x/AUTOMAP/MAPFILTERS.TXT:10）；底圖存 image＋worldToImage 平面矩陣。
2. **還缺的欄位**：三角形索引陣列格式與位置；區段→材質 join（X／Y 欄是否為 key）；逐角 UV 與平面 UV 取捨（原版 UV 未用）；OCT BSP 是否需跳過碰撞面；各圖 O 與 bbox 的引擎真值（目前為跨度擬合）；IFL 動畫在底圖的取格政策。
3. **驗收線**：後續圖先核三數——節點數＝AUTO bit count、z 跨度差＜1000、平面段樓高 pattern＝節點 iy pattern；三者合才收底圖，不以診斷色充數。

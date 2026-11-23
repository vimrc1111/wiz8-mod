# R18d：Circle 原版地面網格

TL;DR：**已找到並畫出地面，在 PVL `0x6a7da–0x7447e` 的追加 35 個 version `0x102` 網格中，格式同 R18c。** 真地板使用材質 3 的 `CosFloor_05.tga` 與材質 4 的 `CosGradient_01.tga`，貼圖字串在材質記錄 `+0xa1`，不是 `+0x29`；只檢查 CosTile 或主貼圖欄會漏掉它。33 段共 183 個向上地面三角形已按原版 TGA、真 UV 產圖，另畫 R18c 牆面深色線。沿用舊原點假設，iy＝3 節點落面 **93/154＝60.39%**（R18c 13/154，R18 重建 35/154）。OCT 頂層可走到檔尾，其碰撞網格沒有 UV；PVL 中段物件／設定鏈的內部分界仍標「未取得」，不宣稱已完整解開整個 PVL。

## 發現

### 1. PVL：0x6a7da 之後

- [已驗證] `0x488240` 按檔頭數量讀 **97 段**，不是只讀 62 段；後 35 段低版本 byte＝2、高 byte＝1，均由同一 `0x49e9a0` 讀取。`mesh.py` 從 `0x6a7da` 逐段累加、核對索引及結尾哨兵，精準到 `0x7447e`。35 段逐一的起點、長度、V/U/T 與地面數見 floor.log 第 3–37 行。來源：build/wiz8.asm:182699、build/wiz8.asm:212095；build/r18d/mesh.py:8、build/r18d/floor.log:3。
- [已驗證] 下表是半開區間 `[起點,終點)`，長度為 bytes；含分隔哨兵的地方明列。第一列起點沿用 R18c 的共享前哨兵，`0x7447e` 是第 97 段末哨兵，`0x74482` 再有一個 dispatcher 哨兵。已知邊界、數量、零尾均有斷言。來源：build/r18d/pvl-layout.py:1、build/r18d/pvl-layout.log:1。

| 三態 | 起點–終點 | 長度 | 讀取函式／內容 | 是否含三角網格 |
|---|---|---:|---|---|
| 已驗證 | `0x6a7da–0x74482` | 40104 | `0x488240 → 0x49e9a0`；35 段含前後哨兵 | 是，含地面 |
| 已驗證 | `0x74482–0x74486` | 4 | `0x4baff0` 額外哨兵 | 否 |
| 已驗證 | `0x74486–0x74a68` | 1506 | `0x4bbad0`；u16＝25、25×60 B、哨兵 | 否，燈資料 |
| 已驗證 | `0x74a68–0x74a70` | 8 | `0x4bc140`；u32＝0、哨兵 | 否 |
| 已驗證 | `0x74a70–0x74a78` | 8 | `0x4bc380`；u32＝0、哨兵 | 否 |
| 已驗證 | `0x74a78–0x74a80` | 8 | dispatcher `0x4bb2c4`；u32＝0、哨兵 | 否 |
| 未取得內部分界 | `0x74a80–0xbdd33` | 299699 | 由 count＝36 的 `0x4bc5e0(flag=0)` 開始，之後為下述物件／設定鏈；末端三個哨兵 | 是，至少兩個柱模型；其餘未逐一分類 |
| 已驗證 | `0xbdd33–0xbe497` | 1892 | `0x4bdc90`；u32＝12、12×157 B、哨兵 | 否，命名記錄 |
| 已驗證 | `0xbe497–0xbead3` | 1596 | `0x584dd0`；S＝2000、count＝396、396×u32、哨兵 | 否，automap 節點 |
| 已驗證資料／未取得消費者 | `0xbead3–0xbeb1b` | 72 | 全零尾部；本單未再追讀取者 | 否 |

- [未取得] `0x74a80–0xbdd33` 尚未把 36 個物件及後續設定逐項切開，表中合併列出外界，不能視為單一 36-object 區的精確長度。已追鏈：`0x4bc5e0 → 0x44bf50 → 0x44aee0 → 0x4a05c0 → 0x4b5b30 → 0x485b20 → 0x485c10`；dispatcher 隨後還呼叫 `0x4bc5e0(flag=1)`、`0x4bc850`、條件式 `0x4bc9d0`、`0x441a20`、`0x4bce20`、`0x4bd0d0`。地面已在前方取得，依停止條件不再擴解此鏈。來源：build/wiz8.asm:251426、build/wiz8.asm:251460、build/wiz8.asm:251493、build/wiz8.asm:251541、build/wiz8.asm:100304、build/wiz8.asm:214551、build/wiz8.asm:243546、build/wiz8.asm:179150。
- [已驗證] 上述鏈內兩份含 CosTile 的材質表不是地面證據：模型 `0x75d3e`、`0x77ca0` 各有 19 頂點、30 面，全部用局部材質 10 `Cos_pillarvein.tga`，至 `0x776f2`／`0x79654`。通過模型尺寸、索引與材質界限檢查；未畫入地面。來源：build/r18d/models.py:10；build/wiz8.asm:179490。

### 2. OCT 頂層與讀取者

- [已驗證] scene 載入時先在 `0x44f82a` 呼叫 OCT loader **`0x42bc10`**，再在 `0x44fa77` 呼叫 PVL loader `0x4baff0`。OCT 是 634246 B、版本 u16＝34、245 B header；`0x42bd00` 讀 0xf5 B，`0x42d2a0` 把 header 欄位搬入 runtime 結構。以下界址按 header 數量累加，哨兵全部相合，最後到 `0x9ad86`＝檔長。來源：build/wiz8.asm:106633、build/wiz8.asm:106797、build/wiz8.asm:58906、build/wiz8.asm:58985、build/wiz8.asm:60663；build/r18d/layout.py:1、build/r18d/layout.log:1。

表內 H 指 OCT header 欄位位移。除指定子讀取器外，磁碟讀取均由 `0x42bc10` 執行；「未取得」表示記錄內部語意尚未逐欄確認，不能將它當成已排除所有幾何。

| 三態 | 起點–終點 | 長度 B | 格式／讀取者 | 是否含三角網格 |
|---|---|---:|---|---|
| 已驗證 | `0–0xf5` | 245 | header | 否 |
| 已驗證 | `0xf5–0xf9` | 4 | 哨兵 | 否 |
| 未取得內部語意 | `0xf9–0x2c91` | 11160 | H+0x6a＝310，stride 36 | 未取得；未識別為貼圖網格 |
| 未取得內部語意 | `0x2c91–0xae69` | 33240 | H+0x6e＝831，stride 40 | 未取得；未識別為貼圖網格 |
| 已驗證尺寸 | `0xae69–0x17961` | 51960 | H+0x82＝12990，u32 陣列 | 無獨立 xyz/UV 網格 |
| 已驗證尺寸 | `0x17961–0x21161` | 38912 | 19×8×64，u32 grid | 無獨立 xyz/UV 網格 |
| 已驗證尺寸 | `0x21161–0x2602d` | 20172 | H+0x72＝5043，u32 陣列 | 無獨立 xyz/UV 網格 |
| 已驗證尺寸 | `0x2602d–0x2668d` | 1632 | H+0x92＝816，u16 陣列 | 無獨立 xyz/UV 網格 |
| 已驗證尺寸 | `0x2668d–0x326c5` | 49208 | H+0x86＝12302，u32 陣列 | 無獨立 xyz/UV 網格 |
| 已驗證 | `0x326c5–0x326c5` | 0 | H+0x8a＝0，u16 陣列 | 否 |
| 未取得內部語意 | `0x326c5–0x3345d` | 3480 | H+0x96＝15，stride 232 | 未取得 |
| 已驗證 | `0x3345d–0x33461` | 4 | 哨兵 | 否 |
| 已驗證尺寸 | `0x33461–0x33a81` | 1568 | (97+1)×16 B 物件記錄 | 無獨立 xyz/UV 網格 |
| 已驗證 | `0x33a81–0x33a9a` | 25 | packed list；count 62、payload 9 B、標籤 0xdeadd00d | 否 |
| 已驗證尺寸 | `0x33a9a–0x33b18` | 126 | 63×u16 offsets | 無獨立 xyz/UV 網格 |
| 已驗證尺寸 | `0x33b18–0x33b4c` | 52 | 26×u16 refs | 無獨立 xyz/UV 網格 |
| 已驗證尺寸 | `0x33b4c–0x33bca` | 126 | 63×u16 offsets | 無獨立 xyz/UV 網格 |
| 已驗證尺寸 | `0x33bca–0x33cf4` | 298 | 149×u16 refs | 無獨立 xyz/UV 網格 |
| 已驗證 | `0x33cf4–0x33cf8` | 4 | 哨兵 | 否 |
| 已驗證尺寸 | `0x33cf8–0x3b730` | 31288 | `0x458ce0`；3911×8 B | 無獨立 xyz/UV 網格 |
| 已驗證尺寸 | `0x3b730–0x3b740` | 16 | `0x458ce0` 子 header | 否 |
| 已驗證 | `0x3b740–0x3b762` | 34 | packed list；count 37、payload 18 B | 否 |
| 已驗證 | `0x3b762–0x3b766` | 4 | 哨兵 | 否 |
| 已驗證 | `0x3b766–0x3b7ce` | 104 | `0x449010 → 0x449240` 碰撞 header | 宣告下面的碰撞網格 |
| 已驗證 | `0x3b7ce–0x43c9a` | 33996 | 2833×xyz f32 | 是，碰撞位置 |
| 已驗證 | `0x43c9a–0x9ad3e` | 356516 | 4691×76 B 面記錄 | 是，碰撞三角形，沒有 UV |
| 已驗證尺寸 | `0x9ad3e–0x9ad82` | 68 | 一筆附屬記錄；另三陣列 count＝0 | 無獨立 xyz/UV 網格 |
| 已驗證 | `0x9ad82–0x9ad86` | 4 | 結尾哨兵 | 否 |

- [已驗證] OCT 76 B 面記錄的位置索引是 `+24/+28/+32` 三個 u32，法向在 `+36` 的 xyz f32；前三個 u32 **不是面材質編號**。`0x4498c0` 的消費方式與此相符；此區沒有逐角 UV 或貼圖表，不能把面旗標 5 當 CosTile 材質 5。來源：build/wiz8.asm:97868、build/wiz8.asm:98390；build/r18d/layout.py:41、build/r18d/validate.py:25。

### 3. 地面格式與判定依據

- [已驗證] 地面幾何格式 **同 R18c**：共享哨兵 h 後七個 u32；h+32 開始 V×xyz f32，接 U×uv f32；Y 欄為負才有 V×u32 頂點材質，再接 T×3u32 UV 索引、T×3u32 位置索引、T×u32 面材質、兩段 V×12 B、T×16 B 法向、V×4 B 版本附屬流、哨兵。下一個 h＝`h+32+40V+8U+44T+A`，A＝Y<0 時 4V。版本 0x102 的高 byte 同為 1，故最後 V×4 仍存在。來源：build/r18d/mesh.py:8；build/wiz8.asm:212095、build/wiz8.asm:212347、build/wiz8.asm:212385、build/wiz8.asm:212504。
- [已驗證] 地板不是 CosTile：材質 3 記錄起點 `0x388`，`+0xa1`＝PVL `0x429` 為 `CosFloor_05.tga`；材質 4 起點 `0x4b2`，貼圖名在 `0x553`，為 `CosGradient_01.tga`。兩筆 `+0x29` 都空。`0x4b8a70` 先嘗試 `+0x29`，再 `+0x51/+0x79/+0xa1`，確實能選到第四貼圖欄；render 重現相同優先序。來源：build/r18d/floor.py:19；build/wiz8.asm:247876、build/wiz8.asm:247919、build/wiz8.asm:247961、build/wiz8.asm:248003。
- [已驗證] 35 段中保留原版面材質 3／4 且幾何單位法向 Ny≥0.7：33 段、183 面，其中材質 3＝105 面、材質 4＝78 面。最小 Ny 分別 0.999244538、0.999999985；不翻面、不補面。高度範圍 `[-2486.821777,449.505096]`，包含原版不同高度的地板，未沿用 R18c 的 [-100,700] 篩選。材質 6 的高處晶體段 86／87 不畫。來源：build/r18d/floor.py:10、build/r18d/floor.py:24、build/r18d/floor.log:1、build/r18d/floor.log:38。
- [已驗證] XZ 鄰接不是只靠圖像觀察：與 R18c 材質 1／2／7／9 且 abs(Ny)<0.7 的牆面頂點比對，材質 3 地板的 315 個逐角位置有 104 個距牆頂點≤1 世界單位；材質 4 為 12/234，兩者最小距離均為 **0**。這些是逐角出現次數，沒有冒充不重複頂點數。來源：build/r18d/floor.py:30、build/r18d/floor.py:34、build/r18d/floor.log:38。
- [已驗證] 具體定位例：第 64 段 h＝`0x6ac9a`、長 5448 B，62 面中 38 面為材質 3；第 68 段 h＝`0x6c81a`、長 5200 B，60 面中 36 面為材質 3；第 82–85 段 h＝`0x71216/0x71406/0x715d2/0x7171a`，共 12 面材質 3，Y＝-72.664146。其餘地面分散於同一追加 35 段，完整段界可重跑 floor.py。來源：build/r18d/floor.log:5、build/r18d/floor.log:9、build/r18d/floor.log:23。

### 4. 產圖、比例與檢核

- [已驗證] 已產出 `build/r18d/circle.png`、`build/r18d/circle-nodes.png`，均 **900×2970**。原始位置／UV 索引、逐面材質查表、原版兩張 TGA，最近點 repeat 取樣、重疊取最高 Y；8370 條牆面三角邊畫深色。沒有畫天花板或做光照、迷霧 adapter／足跡。來源：build/r18d/render.py:10、build/r18d/render.py:47、build/r18d/render.py:65、build/r18d/render.log:1。
- [已驗證] iy＝3 共 154 節點，逐點在地面三角形的 XZ 投影內測試，**93/154＝0.6038961039**。此數據計算於牆線／節點覆蓋前，不使用像素或 bbox 判定；原版多高度地面在 XZ 取聯集。畫布包含全部節點，地面著色率 0.1570819304。來源：build/r18d/render.py:79、build/r18d/render.log:7。
- [推測] 節點原點沿用 R18c 的 bbox 擬合 `O=(-10542.7543945,-134710.711426)`、S＝2000，未新取引擎原點碼證。93/154 因此是**固定舊原點下，全部已選地面 XZ 聯集的覆蓋率**，不能解讀為 iy＝3 可通行率；R18c 13/154 另有高度帶限制，比例差異同時包含補得地面與取消舊高度篩選的影響。來源：build/r18d/render.py:32、build/r18d/render.py:79；notes/research/2026-10-31-R18c-circle-mesh-walker.md:49。
- [已驗證] 重用 R18c `load()`、SLF seek 讀取器與 TGA 解碼，原版檔及貼圖只在記憶體讀取；新腳本、摘要 log、兩張合成圖均只在 `build/r18d/`。無資產副本／完整抽出材質表／整段反組譯碼落地；未 commit，未讀寫 /mnt/c，未執行遊戲或 exe，未下載、未上網，未派子代理。來源：build/r18d/mesh.py:1、build/r18d/floor.py:16、build/r18d/render.py:1。

- [已驗證] 額外幾何檢核：35 段共 341 面，非退化面的索引幾何法向與儲存法向點積誤差均＜1e-6，UV 有限；OCT 4691 面位置索引均＜2833；兩張 PNG header 尺寸正確。來源：build/r18d/validate.py:1、build/r18d/validate.log:1。

## 建議

後續迷霧底圖可使用這份材質 3／4 的原版地面，保留逐角真 UV；材質解析必須包含四欄 fallback。先核定節點的引擎原點及高度對應，再把覆蓋率當作迷霧定位依據。本單已找到地面並產圖，停止擴解物件鏈與其他地圖。

產物：[原版地面＋牆線](../../build/r18d/circle.png)、[iy＝3 節點疊圖](../../build/r18d/circle-nodes.png)。這是原版幾何與貼圖的靜態俯視圖，未重現材質動畫、混色或遊戲畫面效果。

重跑：

```bash
python3 -I wiz8-mod/build/r18d/floor.py
python3 -I wiz8-mod/build/r18d/pvl-layout.py
python3 -I wiz8-mod/build/r18d/layout.py
python3 -I wiz8-mod/build/r18d/render.py
python3 -I wiz8-mod/build/r18d/validate.py
bash wiz8-mod/scripts/check-research.sh wiz8-mod/notes/research/2026-10-31-R18d-circle-floor-mesh.md
```

機檢只驗「發現」條列的來源與三態，沒有驗證網格、貼圖、圖片或覆蓋率。幾何／界址另由腳本斷言及 log 支持。PVL 合併區與 OCT 三組記錄的內部語意尚未取得，已逐項標示；不以機檢通過冒充全檔解碼完成。

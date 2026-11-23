# R18c：Circle PVL mesh 段真正讀取者

TL;DR：找到真讀取者 **0x49e9a0**，並非整塊讀入後指標走訪：它藉 helper 讀變長陣列。R18 的 `nt` 是 UV 筆數、`D` 才是三角數；材質表是 **44 筆**，不是 97。62 個版本 0x100 段共 4,701 三角，真 u32 索引、逐面材質與原始 UV 已解，段界全通過。兩張圖已產出；向上面與高度門檻保留 290 三角，iy＝3 節點落面 **13/154＝8.44%**，低於 R18 重建版 35/154，尚不足當作完整迷霧底圖。

## 發現

### 1. 真讀取者與搜尋紀錄

- [已驗證] 呼叫鏈為 `0x4baff0 → 0x488240 → 0x49e9a0`，呼叫點分別是 `0x4bb09e`、`0x4883f8`。0x49e9a0 使用 0x404ea0 與共用 helper 讀變長陣列，並非原先假設的整塊記憶體 mesh walker。來源：build/wiz8.asm:251212、build/wiz8.asm:182699、build/wiz8.asm:212101、build/wiz8.asm:212219。
- [已驗證] 第一輪全 asm 限定 grep：`cmp.*,0x100$` 95 命中；`cmp.*,(0xffffffff|0x100)$` 1,952 命中；聯集 `imul.*0xc$` 與 PCRE2 `lea.*\[([a-z]+)\+\2\*2\]` 共 4,094 命中。看過 0x4c135f／0x4c1504、0x4d5aaa／0x4d5e2d 上下文，未確立 mesh。段長首四筆擬合 `32+44nv+8nt+44D`，其餘 45 筆有殘差。第二輪沿 dispatcher 前段呼叫找到 0x488240／0x49e9a0，解出可選 nv×4；兩題均取得結論即停止追查。搜尋與已追函式清單：build/r18c/search-summary.txt:1；擬合：build/r18c/fit.py:6、build/r18c/fit.log:1。
- [已驗證] 0x100 是格式／版本打包值：0x49eadc 取 AH 存到讀取器物件 `+0`，低 byte 存回 `+0x3c`，未直接比較 0x100。這解釋常數交集搜尋未命中；版本非零會在段尾額外讀 nv×4。來源：build/wiz8.asm:212151、build/wiz8.asm:212159、build/wiz8.asm:212504。
- [已驗證] 本次不重追題列十個已排除函式；asm 僅 rg 搜尋與局部上下文，未保存反組譯函式整段。來源：build/r18c/search-summary.txt:1。

### 2. 段頭、陣列順序與大小（Q1）

以下 `h` 沿用 R18 掃描到的 `ffffffff 00010000` 起點。**h 的哨兵屬前一區的結尾**；0x49e9a0 從 h＋4 開始讀七個 u32，最後另讀本段結尾哨兵。令 V＝nv、U＝舊 nt、T＝舊 D、A＝最後欄 Y＜0 時 V×4，否則 0。

- [已驗證] 七欄對讀取器欄位為：h＋4 版本→`+0x3c`；h＋8 V→`+0x40`；h＋12 U→`+0x10`；h＋16 T→`+0x44`；h＋20→`+4`；h＋24 X→`+8`；h＋28 Y→`+0xc`。Circle h＋20 都是 -1，Y 是 -1 或 1。來源：build/wiz8.asm:212095、build/wiz8.asm:212104、build/wiz8.asm:212111、build/wiz8.asm:212118、build/wiz8.asm:212126、build/wiz8.asm:212133、build/wiz8.asm:212140；build/r18c/render.py:23。

| 順序 | 起點／筆數 | 每筆／格式 | 讀取碼證（build/wiz8.asm） |
|---|---|---|---|
| 位置 | h＋32；V | 12 B，xyz f32 | :212219、:74054 |
| UV | 位置後；U | 8 B，uv f32 | :212246、:74081 |
| 逐頂點材質 | UV 後；Y＜0 才有 V 筆 | 4 B，u32；否則全用 Y | :212253、:212276、:212321 |
| UV 三角索引 | 前項後；T | 12 B，3×u32 | :212347、:74054 |
| 位置三角索引 | UV 索引後；T | 12 B，3×u32 | :212367、:74054 |
| 逐面材質 ID | 位置索引後；T | 4 B，u32 | :212385、:212392 |
| 頂點附屬流 A | 面材質後；V | 12 B | :212440、:212446 |
| 頂點附屬流 B | 流 A 後；V | 12 B | :212463、:212468 |
| 面法向記錄 | 流 B 後；T | 16 B，前三個 f32 是法向 | :212483、:212488、:74041 |
| 版本附屬頂點流 | 法向後；V（版本非零） | 4 B | :212504、:212512 |
| 結尾哨兵 | 上項後 | 4 B，ffffffff | :212540、:212548 |

- [已驗證] helper 0x4374e0 用 `lea n,[n+n*2]` 再 `shl 2` 讀 n×12；0x437510 讀 n×8；0x4374c0 讀 n×16。上表完整順序與指標累加在離線 walker 中重現，無以擬合代替碼證。來源：build/wiz8.asm:74054、build/wiz8.asm:74081、build/wiz8.asm:74041；build/r18c/render.py:24。
- [已驗證] 下一個共享哨兵位移為 `h＋32＋40V＋8U＋44T＋A`；61 個相鄰段全數精準相接，第 62 段在 0x6a7da 恰接結尾哨兵。第一段 h＝0x3342、下一段 0x7072；最後段 h＝0x69ede。已驗證範圍限這 62 個 0x100 特徵段，未擴追其餘物件版式。來源：build/r18c/render.py:37、build/r18c/validate.py:5、build/r18c/validate.log:1。
- [未取得] 頂點附屬流 A／B、最後 V×4 流及面法向記錄第四個 u32 的確切語意；大小與順序已釘選，但不把它們強命名為顏色、光照或平面距離。來源：build/wiz8.asm:212440、build/wiz8.asm:212463、build/wiz8.asm:212483、build/wiz8.asm:212504。

### 3. 真索引、段→材質與 UV（Q2、Q3）

- [已驗證] 位置索引起點為 `h＋32＋12V＋8U＋A＋12T`，是 little-endian u32，每三角三值；UV 索引在其前 12T。62 段全部位置索引＜V、UV 索引＜U，共 3,472 位置、8,752 UV、4,701 三角；第一段兩索引起點分別 0x4542（UV）、0x4d5e（位置），第二段為 0x73ea／0x753a，末段為 0x6a17a／0x6a2a6。來源：build/r18c/validate.py:5、build/r18c/validate.log:1；build/wiz8.asm:212347、build/wiz8.asm:212367。
- [已驗證] 材質筆數是 PVL 位移 8 的 u16＝44，記錄從 0xa 起；每筆版本 byte＝4，磁碟長 0x12a＝298 B（先讀 0x11a，再補 0x10，餘筆同 stride）。記錄＋1 是 40 B 名稱，＋0x29 是 40 B 主貼圖名，＋0x51／＋0x79 是其他貼圖欄；0x489980 對各字串尾清零，並非 R18 所說的 32＋32 B 表。檔頭 97 不可用作材質筆數。來源：build/r18c/materials.py:5、build/wiz8.asm:182224、build/wiz8.asm:182264、build/wiz8.asm:182279、build/wiz8.asm:182292、build/wiz8.asm:184323、build/wiz8.asm:184326、build/wiz8.asm:247876、build/wiz8.asm:247919、build/wiz8.asm:247961。
- [已驗證] join 是**逐面**：兩組索引後 T 個 u32 ID，0x49ee15 以 ID×4 索引傳入的材質物件表，再交給該面的材質槽；物件表由 0x487e10 經 0x4b8a70 按記錄順序建成，後者讀記錄＋0x29 貼圖名。62 段所有逐面／逐頂點 ID 均＜44。不能用段頭 X 或單一固定地板貼圖取代它。來源：build/wiz8.asm:212392、build/wiz8.asm:212412、build/wiz8.asm:182487、build/wiz8.asm:247876；build/r18c/materials.py:15、build/r18c/render.py:27。
- [未取得] 段頭 X（前段 -1、後段 62～96）的確切語意；本次讀取器僅見存到 `+8`，未見它參與面材質查表。這不影響已取得的逐面材質 join。來源：build/wiz8.asm:212133、build/wiz8.asm:212412。
- [已驗證] UV 是位置陣列後 U 筆 f32×2，經獨立 UV 索引逐角對應；不是逐位置一對 UV，也不是平面投影重建。離線以原始 UV 重複取樣，TGA 解碼重用 R18 helper。來源：build/wiz8.asm:212246、build/wiz8.asm:212347；build/r18c/render.py:25、build/r18c/render.py:34、build/r18c/render.py:59。
- [已驗證] 16 B 面記錄前三個 f32 與依原始索引順序算的幾何單位法向同向：4,701／4,701 的點積在 0.999999917～1.000000089，逐筆斷言誤差＜1e-6。法向方向無憑圖形反轉。來源：build/r18c/normals.py:9、build/r18c/normals.log:1。

### 4. Circle 產圖與節點覆蓋（Q4）

- [已驗證] 產出 `build/r18c/circle.png`、`build/r18c/circle-nodes.png`，均 900×2953；真索引、原始 UV、逐面原版 TGA，採最近點與 UV repeat，重疊處取最高 Y，沒有 Delaunay 或補面。來源：build/r18c/render.py:30、build/r18c/render.py:47、build/r18c/render.py:59、build/r18c/render.log:1。
- [已驗證] 門檻：幾何法向 `Ny／|N|≥0.7`，三頂點 Y 全在 [-100,700]；4,701 面中排除不向上 4,030 面、離高度帶 381 面，保留 37 段 290 面，退化面／缺 TGA 面皆 0。實際用到材質 1 的 `CosPillar_01.tga`（262 面）與材質 8 的 `CosCeiling_01.tga`（28 面），均按各面 ID，非手選貼圖；檔名含 Ceiling 不用作剔除依據。來源：build/r18c/render.py:47、build/r18c/render.log:2。
- [推測] 當前「地面」指上述幾何門檻的候選面，會包含牆頂／裝飾上表面，未證明可行走；僅向上與高度條件不足以完整抽出迷霧需要的樓板。成圖有大片空白，未用擴邊、補洞或反轉面來提高覆蓋。來源：build/r18c/circle.png:1、build/r18c/render.log:2。
- [已驗證] iy＝3 仍是 154 節點；以與 R18 相同的全 62 段幾何 bbox 中心擬合原點 O＝(-10542.7544,-134710.7114)、S＝2000，做世界 XZ 點在真三角內檢查，得 **13/154＝0.084416**，R18 重建版是 35/154。畫布包含全部 154 節點；著色像素率 0.038601，不以包圍盒命中率冒充落面率。來源：build/r18c/render.py:66、build/r18c/render.log:7；notes/research/2026-10-31-R18-circle-texture-map.md:49。
- [推測] 節點原點與 iy＝3 對應此高度帶仍沿用 R18 擬合，尚無引擎座標原點碼證；13/154 只衡量這個固定假設下的候選地面。它低於重建版，證明此次解碼成功不等於底圖完整。來源：build/r18c/render.py:66、build/r18c/render.log:7。
- [已驗證] 重跑：`python3 -I wiz8-mod/build/r18c/render.py`；幾何檢核：`python3 -I wiz8-mod/build/r18c/validate.py`、`python3 -I wiz8-mod/build/r18c/normals.py`。SLF 以唯讀 seek 取 Circle 與所需 TGA，在記憶體解碼；無原版資產／抽出表／原版圖落地，不執行 exe、遊戲、下載、網路或 /mnt/c。新腳本與產物均留 build/r18c，未 commit。來源：build/r18c/probe.py:1、build/r18c/render.py:1。

## 建議

保留此版真索引／逐面材質解碼，替換 R18 的 Delaunay 假設；此次只交 Circle 研究產物，不把 8.44% 覆蓋率宣稱為可用小地圖。後續若另立單，再核定地面語意、物件選取與節點原點；本單到兩題解出並產圖即停止，不擴追其他地圖、牆、天花板、光照、adapter 或足跡。

產物：[circle.png](../../build/r18c/circle.png)、[circle-nodes.png](../../build/r18c/circle-nodes.png)。

機檢只檢查來源與三態，不驗圖片、幾何正確性或覆蓋率；幾何另由上述斷言與 normals.log 支持。

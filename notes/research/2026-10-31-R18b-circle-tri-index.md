# R18b Circle 網格的三角形索引與段→材質（從 loader 碼讀）

- 日期：2026-10-31
- 範圍：只追 PVL mesh 區段 parser；只畫 Circle
- 上游：R18 `wiz8-mod/notes/research/2026-10-31-R18-circle-texture-map.md` §1 段結構、§建議 2；重用 `wiz8-mod/build/r18/render.py`（段 walker、TGA、PNG 編寫）
- 方法：只讀 `wiz8-mod/build/wiz8.asm` 的 0x4baff0 及其呼叫的 0x4bce20、0x4bd0d0、0x4bdc90、0x4be200 及其直接下層讀檔呼叫（0x404ea0 類）；不從 hex 猜
- 狀態：研究中（骨架先行，結論隨追碼追加）

## TL;DR

- 第一輪讀碼結論：0x4bce20／0x4bd0d0／0x4bdc90／0x4be200 四者皆非 mesh 段 parser（固定長度記錄或無檔案讀取）；mesh 段（含三角索引）應由 0x4baff0 內另一些哨兵守衛的 0x4bc* 處理函式讀取，已超出本單讀碼範圍。
- 第二輪檔案交叉驗證結論：Circle PVL 檔頭為 (97, 62, 262188)，62 段 nv＝19～260 變長；檔頭材質名／貼圖名對存在（0x33 NoTexture；0x135 名＋0x15d 貼圖名），但全檔無 80／81 步長貼圖表，故 0x4bce20 的緊湊二進位表另在他處；三角索引、段→材質 join、UV 皆未取得（各追兩輪），Q4 不以重建充數、未產圖。

## 發現

### R1：四個被呼叫函式的讀取序列（讀 0x4baff0 內呼叫點＋各函式內 0x404ea0 尺寸序列）

- 0x4baff0 是 PVL 多區段 dispatcher：先讀 u32 與 `0xffffffff` 哨兵比對（不符即推字串 VA 走診斷），再呼叫各區段處理函式；0x4bce20、0x4be200、0x4bd0d0、0x4bdc90、0x584dd0 在 0x4baff0 內的呼叫點依序為 asm 行 251676、251740、251784、251818、251848。來源：wiz8-mod/build/wiz8.asm:251676 [已驗證]
- 0x4bce20 的讀取順序：u32 筆數（4B）→ 每筆 1B 標籤＋0x40（64）B＋0x10（16）B；固定長度、無 nv／nt 變長陣列，不是 mesh 段，應為檔頭材質名表。來源：wiz8-mod/build/wiz8.asm:253754 [推測]
- 0x4bd0d0 的讀取順序：u32 筆數（4B）→ 每筆 1B 型別標籤（1..4）＋固定 0x216／0x218／0x21d／0x225 B 記錄；固定長度約 534～549B，不是 mesh 頂點／索引變長陣列。來源：wiz8-mod/build/wiz8.asm:253929 [推測]
- 0x4bdc90 的讀取順序：u32 筆數（4B）→ 每筆 1B 標籤（須為 1）＋0x80 B＋7 個 u32／f32（其中三個 f32 乘上單位換算常數 0x5ec150 後存回）；小記錄，非 mesh。來源：wiz8-mod/build/wiz8.asm:254616 [推測]
- 0x4be200 全函式無 0x404ea0 檔案讀取：純浮點運算＋配置（0x1c 頭＋每欄陣列），再逐欄呼叫 0x4be100；是記憶體內格子／BSP 建構器，不是檔案 parser。來源：wiz8-mod/build/wiz8.asm:255160 [已驗證]
- 讀取尺寸序列抽取腳本：wiz8-mod/build/r18b/readseq.py:1（輸出即上四條的尺寸證據） [已驗證]

### R2：檔案位元組交叉驗證（唯讀 SLF，不落地原版檔）

- Circle PVL 檔頭三 u32 為 (97, 62, 262188＝0x4002c)，檔長 781083 B。來源：wiz8-mod/build/r18b/verify.py:16 [已驗證]
- 檔頭區有材質名／貼圖名對：檔內位移 0x33 為 `NoTexture`；0x135 `Vert Tile Marble Grey` 對 0x15d `CosPillar_01.tga`（名在前、貼圖名在後，與 R18 的 32B＋32B 描述一致）。來源：wiz8-mod/build/r18b/verify.py:19 [已驗證]
- 步長擬合（碼定版式→檔驗證）：E＝80／81、表頭 S＝0x0～0x3f、97 筆全滅（0／97）；全檔掃 count＋81 步長貼圖表得 0 候選；放寬 count＝10～400、E∈{80,81} 仍 0 張表。來源：wiz8-mod/build/r18b/matfit.py:1 [已驗證]
- 上條的另一半證據（全檔＋放寬掃描皆零）：來源：wiz8-mod/build/r18b/tabscan2.py:1 [已驗證]
- 綜合：0x4bce20 的每筆 0x40＋0x10 緊湊二進位表不是檔頭名表（名表由範圍外程式碼讀走）；其筆數／檔位在本單範圍內無法釘選。來源：wiz8-mod/build/wiz8.asm:253754 [推測]
- mesh 段 nv＝19～260 變長（62 段；前兩段與 R18 一致：0x3342 為 123／326／173，0x7072 為 29／49／28），故四個固定長度讀取器都不可能是 mesh parser。來源：wiz8-mod/build/r18b/verify.py:30 [已驗證]
- 0x4baff0 內另有六組哨兵守衛的處理函式呼叫（本體讀碼範圍內列名、未讀本體）：0x4bc140、0x4bc380、0x4bc5e0（兩處）、0x4bc850、0x4bc060、0x4bc9d0，呼叫點在 asm 行 251316、251349、251426、251460、251493、251522、251541；mesh 段 parser 應在其中。來源：wiz8-mod/build/wiz8.asm:251316 [已驗證]

### 要答 verdict（Q1～Q4）

- Q1 mesh 段頭 8 u32 讀取順序與各陣列（筆數來源／每筆大小／物件欄位）：兩輪未取得。R1 讀完範圍內四函式（讀取順序見 R1），R2 證實皆非 mesh；真 parser 在上條 0x4bc* 清單中（範圍外）。來源：wiz8-mod/build/wiz8.asm:251316 [未取得]
- Q2 三角形索引（位置／u16 或 u32／每三角值數；全段索引＜nv、段尾恰接下段頭）：兩輪未取得。範圍內無變長索引陣列讀取；R18 的頂點後小浮點塊與全檔 u16 連續區兩輪掃描亦非索引形狀。來源：wiz8-mod/notes/research/2026-10-31-R18-circle-texture-map.md:25 [未取得]
- Q3 材質表貼圖名欄位：檔頭名表以後半 32B 為貼圖基名（如 0x15d `CosPillar_01.tga` 對同筆前半材質名），R18 的 `<MAP>\BITMAPS\<基名>` 規則維持。來源：wiz8-mod/build/r18b/verify.py:19 [已驗證]
- Q3 段→材質 join（段頭 X／±1 欄或段內欄位對 97 筆表）：兩輪未取得；X 欄後段為 62～96 遞增 id 仍只是 R18 觀測，無碼證 join。來源：wiz8-mod/notes/research/2026-10-31-R18-circle-texture-map.md:21 [未取得]
- Q3 UV 位置（逐頂點或逐角）：兩輪未取得；0x4bdc90 的 0x80B＋7×f32 小記錄含單位換算浮點但非 mesh 附屬，無法對到段。來源：wiz8-mod/build/wiz8.asm:254616 [未取得]
- Q4 `build/r18b/circle.png`／`circle-nodes.png`：未產出。真索引、真 UV、各段自有 TGA 三者缺二，不以 R18 式 Delaunay＋單一貼圖重建充數；待真 parser（0x4bc*）解出後再畫，屆時法向＋Y 門檻與 iy＝3 節點落肉比例寫報告。來源：wiz8-mod/build/r18b/verify.py:1 [未取得]

## 建議

1. 下一單把讀碼範圍擴到 0x4bc140、0x4bc380、0x4bc5e0、0x4bc850、0x4bc060、0x4bc9d0 六者（本單已列呼叫點），以同樣手法抽 0x404ea0 尺寸序列，找 `[8×u32][nv×12B][nt×?][D×16B]` 變長版式；dispatcher 哨兵讀＋`cmp 0xffffffff`＋診斷字串的模式可直接對段。
2. 0x4bce20 的 80B／筆二進位表（筆數 u32＋每筆 0x40＋0x10，另有 1B 前綴讀取）值得同法釘選檔位：它可能是材質屬性表（色／旗標），與檔頭名表 join 後即為段→材質的另一半。
3. 本單產物只留 `build/r18b/` 下四個標準庫腳本（readseq／matfit／tabscan＋tabscan2／verify），皆唯讀 SLF、無原版檔落地；R18 的 `circle.png` 仍是目前唯一底圖。

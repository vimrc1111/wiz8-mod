# R18d 小地圖續：Circle 地面網格在哪（OCT 或 PVL 0x6a7da 之後），畫出原版地面

驗收: bash wiz8-mod/scripts/check-research.sh wiz8-mod/notes/research/2026-10-31-R18d-circle-floor-mesh.md
驗收時長: 1
規模: 小（只找地面網格；只畫 Circle）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 同 R18（不 commit；`/mnt/c` 不讀不寫；不執行遊戲或 exe；不下載、不上網；repo 不放原版檔、抽出的表或圖、反組譯碼整段；腳本與產物只放 `wiz8-mod/build/r18d/`）
白名單: wiz8-mod/notes/research/2026-10-31-R18d-circle-floor-mesh.md
預期紅: research:2026-10-31-R18d-circle-floor-mesh
執行器: codex
預估讀入: R18c 報告全文（段格式、44 筆材質表、0x49e9a0 讀取器）；`wiz8-mod/build/r18c/render.py`（load()，重用）；`wiz8-mod/build/wiz8.asm` 只從 0x4baff0／0x488240 dispatcher 與 OCT loader 附近 grep；Circle OCT 634 KB、PVL 781 KB；約 1 MB
停止條件: 找到地面網格並產圖、寫完報告就結束；兩輪沒進展寫「未取得」（列追過的區段、函式），照樣寫完報告結束

## 先做
**第一個動作**：建報告骨架（檔頭 TL;DR、`## 發現`、`## 建議`），每得一條結論就追加。白名單檔 1260 秒沒變動會被看門狗砍。

## 背景（cc 複核 R18c 的結論）
- R18c 解出的 62 段（PVL 0x3342–0x6a7da）材質只有柱、牆、磚牆、天花板、壁畫；**沒有地板**。44 筆材質表裡未被這 62 段用到的有 `CosTile_01.tga`（材質 5）、`chrome.tga` 與書、行星類。→ 地面在別處。
- PVL 0x6a7da 之後到檔尾（781,083 B）尚未解，其中 0xbe497 起是 automap 節點（R17／R18）。OCT（634,246 B）完全未解。
- 0x4baff0 → 0x488240 → 0x49e9a0 是 PVL 物件段讀取鏈；dispatcher 0x488240 應還有別的分支。

## 要答（每條附 `build/wiz8.asm:<行>` 或 `build/r18d/<檔>`／檔內位移與三態）
1. PVL 0x6a7da 之後的區段清單（起點、長度、讀取函式），以及 OCT 的頂層結構與讀取函式（從 OCT 檔名字串或 dispatcher 追）。每段標「是否含三角網格」。
2. 地面網格：在哪個區段、頂點／索引／材質／UV 的格式（能用 R18c 的段格式就寫「同 R18c」）。判斷依據要有：材質為 `CosTile_01` 或其他地板貼圖、法向朝上、與 R18c 牆段在 XZ 上相鄰。
3. 產圖 `build/r18d/circle.png`：地面網格（各面原版 TGA、真 UV）＋R18c 牆段畫成深色線，`circle-nodes.png` 疊 iy＝3 節點；報告寫節點落在地面內的比例（R18c 是 13/154，R18 重建版是 35/154）。

## 不做
天花板、光照、其他地圖、adapter、足跡（MG：只要迷霧）。

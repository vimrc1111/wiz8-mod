# R18 小地圖：Circle 一張原版材質俯視圖（證明 OCT/PVL 網格＋TGA 材質可離線畫）

驗收: bash wiz8-mod/scripts/check-research.sh wiz8-mod/notes/research/2026-10-31-R18-circle-texture-map.md
驗收時長: 1
規模: 小（只做最小的一張地圖 Circle；一份報告＋build/ 下腳本與一張 PNG）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單以外的檔；`/mnt/c` 不讀不寫（材料在 `wiz8-mod/build/r17/orig/LEVELS.SLF`、`wiz8-mod/build/data-x/`）；不執行遊戲或 exe；不下載、不執行第三方工具；不上網；repo 不放原版檔、抽出的表或圖、反組譯碼整段（報告寫 VA、檔內位移、≤24 B hex、欄位名）；腳本與產物只放 `wiz8-mod/build/r18/`
白名單: wiz8-mod/notes/research/2026-10-31-R18-circle-texture-map.md
預期紅: research:2026-10-31-R18-circle-texture-map
執行器: muse
預估讀入: R17 報告 `wiz8-mod/notes/research/2026-10-31-R17-bitatlas-map.md`（§發現 1–3、末段建議）；`wiz8-mod/build/r17/catalog.tsv`（SLF 目錄：offset、長度、路徑）、`probe.py`（Arnika PVL 節點讀法，重用）、`catalog.go`（SLF reader 用法）；`wiz8-mod/tools/slf/`；`wiz8-mod/build/wiz8.asm`（19 MB，只 grep OCT／PVL loader 附近）；LEVELS.SLF 只讀 Circle 的 OCT（634,246 B）、PVL（781,083 B）與它引用的 TGA；約 2 MB
停止條件: 產出 PNG 並寫完報告就結束；網格格式找兩輪沒進展就寫「未取得」（列搜過的範圍與已解出的欄位），照樣寫完報告結束

## 先做
**第一個動作**：建報告骨架（檔頭 TL;DR、`## 發現`、`## 建議`），每得一條結論就追加。白名單檔 1260 秒沒變動會被看門狗砍。

## 背景
- 目標是 BitAtlas 用原版材質畫 wiz8 底圖（原版 automap 是引擎對場景幾何即時俯視渲染，沒有預繪圖可抽；R17 §發現 20–21）。R17 只解出 PVL automap 節點（佔據診斷圖），網格與材質未取得。
- 本單只拿最小的一張：Circle（`CIRCLE\CIRCLE.OCT`、`CIRCLE\CIRCLE.PVL`，SLF 位移見 catalog.tsv）。
- 世界座標 X/Z 水平、Y 高（R17）。

## 要答（每條附 `build/wiz8.asm:<行>`、`build/r18/<檔>` 或檔內位移來源與三態 `[已驗證]`／`[推測]`／`[未取得]`）
1. OCT／PVL 網格的磁碟結構：頂點（float 三元組）、三角形索引、UV、材質／貼圖名或索引各在哪個區段、每筆欄位與長度；用 loader 的讀取順序（`build/wiz8.asm` 從 R17 已知的 PVL loader 0x4baff0、OCT/PVL mismatch 診斷字串往回追）佐證，不只靠 hex 猜。
2. 材質：三角形引用的貼圖名怎麼對到 LEVELS.SLF 裡的 TGA（路徑規則、共用 BITMAPS 子目錄、IFL 動畫取第一格）；TGA 格式（未壓縮／RLE、位深）。
3. 產圖：`build/r18/render.py`（系統 Python 標準庫，`python3 -I`）把 Circle 的地面朝上三角形（法向 Y 分量 > 門檻）以正投影畫到 PNG，貼原版材質（最近點取樣即可）；多樓層先畫最上一層或分檔輸出各 Y 區間，報告寫你選哪種。輸出 `build/r18/circle.png`，報告寫像素尺寸、世界範圍、每像素世界單位。
4. 對照：同一張 PNG 疊 PVL 節點中心點（R17 probe 的節點解碼套到 Circle）另存 `build/r18/circle-nodes.png`，報告寫節點落在地面三角形內的比例（驗證座標系一致）。

## 不做
- 牆、天花板、物件、門、NPC、光照；其他地圖；BitAtlas adapter；足跡（MG 2026-10-31：「足跡我覺得不用，久了會很紊亂」「有迷霧很夠了」）。

## 報告格式
沿用 R17：檔頭 3–5 行 TL;DR；`## 發現` 每條附來源與三態；`## 建議`：要推到全部 34 張地圖時 adapter 怎麼做、還缺哪些欄位。

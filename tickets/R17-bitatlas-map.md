# R17 研究：Wizardry 8（連續 3D 座標）怎麼接上 BitAtlas 地圖服務

驗收: bash wiz8-mod/scripts/check-research.sh wiz8-mod/notes/research/2026-10-31-R17-bitatlas-map.md
驗收時長: 1
規模: 中（靜態研究＋資料格式探勘；一份報告＋build/ 下探勘腳本）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單以外的檔；不寫入遊戲目錄（`/mnt/c` 不讀不寫，材料已複製進 `build/`）；不執行遊戲或 exe；不下載、不執行第三方工具；不上網；repo 不放原版檔、從原版抽出的表或圖、反組譯碼整段（報告寫 VA、≤24 B hex、檔名、欄位名）；探勘腳本與產物只放 `build/r17/`；存檔副本只讀
白名單: wiz8-mod/notes/research/2026-10-31-R17-bitatlas-map.md
預期紅: research:2026-10-31-R17-bitatlas-map
執行器: codex
預估讀入: （下列路徑除驗收、白名單外皆相對 `wiz8-mod/`；impl-run 在 repo 根跑）本單；`../bitatlas/DESIGN.md`（121 行，§三.1 格式、§四 MAPPOS／mapEdges／mapSeen、§五 viewer）；`../bitatlas/atlas.ts:1–40`（型別）；`../bitatlas/mappos.h`（C 端寫位置）；`notes/re/misc.md:65`（地圖索引 0x686a70、地圖表 0x604478）；`notes/re/party-move.md`（隊伍物件 0x652dac、連續座標）；`notes/re/iface/437f30.tsv`（0x4534c0 取世界座標、0x659c0c 隊伍世界物件）；`tools/slf/`（既有 SLF 讀檔器，重用）；`hook/ddraw/` 檔頭（代理 DLL 怎麼掛）；`build/wiz8.asm`（19 MB，只 grep 相關位址）；`build/data-x/AUTOMAP/`；`build/r17/orig/LEVELS.SLF`（475 MB，只列目錄與抽樣）；約 1 MB 文字
停止條件: 下列 6 題都有結論（含「未取得」與搜過的範圍）就寫檔結束；同一題找兩輪沒進展就寫「未取得」換下一題

## 背景
- BitAtlas（`../bitatlas/`）是 rpg-mod 共用地圖服務：每款寫 adapter（原版資料 → `atlas.js`，過 `validate`），遊戲端 hook 寫試玩包根目錄 `MAPPOS.JS`（`mapPos(1,map,x,y,dir,seq)`），共用 `viewer.html` 每秒重讀並標位置。現有 Map kind 只有 `grid`／`tile`／`rooms`，座標一律整數格；`Map.xform{dx,dy,flip,s}` 把遊戲座標換成圖座標（veil 用 s 把房內像素縮成格）。原則：地圖用該遊戲自己的圖素畫，原版給不了的才用向量圖層替代；會變的狀態（開門、已探索）走 hook 寫檔，不在遊戲內疊層。
- Wiz8 不是格子遊戲：隊伍位置是連續浮點三維座標（party-move.md），地圖 47 張（level-id.md）。遊戲自己有 automap（`data-x/AUTOMAP/`：大小地圖 STI、隊伍／怪物／文字標記 TGA、圖層按鈕、`MAPFILTERS.TXT`）。
- 已有代理 `ddraw.dll`（`hook/ddraw/`，C）可在程序內讀記憶體、寫檔；結案時已部署在 Steam 目錄。
- 目標：MG 實玩時開 `攻略\地圖.html` 看得到每張地圖＋自己所在位置與朝向（＋最好有已探索範圍、出口／樓梯、NPC）。

## 要答（每條附 `build/wiz8.asm:<行>`、`build/r17/<檔>` 或檔內位移來源與三態）
1. 地圖幾何從哪來：`LEVELS.SLF` 裡每張地圖有哪些檔（列副檔名與每類用途推測），哪一份是 automap 畫圖用的資料——預先畫好的點陣圖？還是從地形網格（地面三角形、牆線段）即時畫？找 automap 繪製函式（從 `AUTOMAPLARGE.STI`／`MAP_PARTYMARKER_A.TGA` 字串的引用追），寫出它讀哪個資料結構、每筆欄位。給一張地圖（Arnika，索引 0）從原版產出俯視圖的可行做法，`build/r17/` 寫可重跑腳本產一張 PNG 證明（不入 repo）。
2. 座標換算：世界座標 (x,y,z) 哪兩軸是水平、單位，automap 怎麼把隊伍世界座標投影到圖上（縮放、原點、軸向翻轉，從繪製隊伍標記的碼讀）；朝向角從哪個欄位讀、弧度起點與方向，對到 BitAtlas dir（0–3＝NESW，或建議加連續角度）。給每張地圖換算參數的來源（地圖檔標頭的包圍盒？固定常數？）。
3. 多層：同一張地圖內的上下樓層（z）automap 怎麼分層（`MAP_LAYERBUTTONS`），層的界線從哪讀。
4. 已探索範圍：automap 的「走過才揭露」存在哪（記憶體位址、存檔區段），格式（點陣？格子？多邊形？）。可抽樣讀 `build/r17/saves/` 兩份存檔副本。
5. 可上圖的標記：地圖間出口（換圖觸發區、目的地圖索引）、樓梯、NPC 位置、文字標記（automap 自動標註的地名）、門；各給「從哪份資料得到」或「未取得」。
6. BitAtlas 格式缺口：照上面結論，Wiz8 用現有 `tile` kind（把俯視圖切格當圖素、xform.s 縮放浮點座標）夠不夠？不夠的話提出最小擴充（例：新 kind `image`＝一張底圖＋浮點座標；`mapPos` dir 改連續角度），寫成 DESIGN.md 的修改草案（列欄位與向下相容性，不改 DESIGN.md 本身）。ddraw.dll 端：寫 MAPPOS.JS 的掛點建議（哪個週期函式、讀哪些位址），對照 `../bitatlas/mappos.h` 介面是否直接可用。

## 報告格式
沿用 `notes/re/party-move.md`：檔頭 3–5 行 TL;DR；`## 發現` 每條附來源與三態（`[已驗證]`／`[推測]`／`[未取得]`）；末段「建議分單」：adapter／ddraw hook／BitAtlas 擴充各一張實作單的範圍與驗收想法。

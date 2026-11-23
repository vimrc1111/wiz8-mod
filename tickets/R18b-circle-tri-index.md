# R18b 小地圖續：Circle 網格的三角形索引與段→材質（從 loader 碼讀，不從 hex 猜）

驗收: bash wiz8-mod/scripts/check-research.sh wiz8-mod/notes/research/2026-10-31-R18b-circle-tri-index.md
驗收時長: 1
規模: 小（只追 PVL mesh 區段 parser；只畫 Circle）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 同 R18（不 commit；`/mnt/c` 不讀不寫；不執行遊戲或 exe；不下載、不上網；repo 不放原版檔、抽出的表或圖、反組譯碼整段；腳本與產物只放 `wiz8-mod/build/r18b/`）
白名單: wiz8-mod/notes/research/2026-10-31-R18b-circle-tri-index.md
預期紅: research:2026-10-31-R18b-circle-tri-index
執行器: muse
預估讀入: R18 報告 `wiz8-mod/notes/research/2026-10-31-R18-circle-texture-map.md`（§1 段結構、§建議 2）；`wiz8-mod/build/r18/render.py`（段 walker、TGA、PNG 編寫，重用）；`wiz8-mod/build/wiz8.asm` 只讀 0x4baff0 與它呼叫的 0x4bce20、0x4bd0d0、0x4bdc90、0x4be200 及其直接下層讀檔呼叫（0x404ea0 類）；Circle PVL 781 KB；約 1 MB
停止條件: 三角索引與段→材質都有結論就產圖、寫完報告結束；任一題追兩輪沒進展寫「未取得」（列追過的函式與讀取順序），照樣寫完報告結束

## 先做
**第一個動作**：建報告骨架（檔頭 TL;DR、`## 發現`、`## 建議`），每得一條結論就追加。白名單檔 1260 秒沒變動會被看門狗砍。

## 背景
R18 已解出 62 個 mesh 段的段頭 `[ffffffff][256][nv][nt][D][-1][X][±1]` 與 nv 筆頂點，但三角索引與段→材質沒找到，底圖用 Delaunay＋單一 `COSFLOOR_01` 重建（約四分之一節點落在畫出的地板外）。這次改從碼讀：找哪個 parser 在讀到 `nv` 之後還讀了哪些陣列、各幾筆、每筆幾 bytes（`nt` 很可能是三角數），按讀取順序切段，不靠形狀猜。

## 要答（每條附 `build/wiz8.asm:<行>` 或 `build/r18b/<檔>`／檔內位移與三態）
1. 哪個函式讀 mesh 段（段頭 8 個 u32 的讀取順序），段頭之後依序讀的每個陣列：筆數來源（nv／nt／D／常數）、每筆大小、存進物件的哪個欄位。
2. 三角形索引：位置、型別（u16／u32）、每三角幾個值；驗證：Circle 全部 62 段的索引都 < nv、每段讀完剛好接下一段段頭。
3. 段→材質：段頭 X／±1 欄或段內哪個欄位對到 PVL 檔頭 97 筆材質表，材質表每筆哪個欄是貼圖名；UV 在哪（逐頂點或逐角）。
4. 產圖 `build/r18b/circle.png`：用真索引、真 UV、各段自己的 TGA，畫法向 Y 朝上的三角（門檻寫報告）；另存 `circle-nodes.png` 疊 iy＝3 節點，報告寫節點落在畫出地面內的比例（R18 是 35/154）。

## 不做
牆、天花板、光照、其他地圖、adapter、足跡（MG：只要迷霧）。

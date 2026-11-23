# R18c 小地圖續：找 PVL mesh 段的真正讀取者（整塊讀入後指標走訪），解三角索引與段→材質

驗收: bash wiz8-mod/scripts/check-research.sh wiz8-mod/notes/research/2026-10-31-R18c-circle-mesh-walker.md
驗收時長: 1
規模: 小（只追 mesh 段消費者；只畫 Circle）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 同 R18（不 commit；`/mnt/c` 不讀不寫；不執行遊戲或 exe；不下載、不上網；repo 不放原版檔、抽出的表或圖、反組譯碼整段；腳本與產物只放 `wiz8-mod/build/r18c/`）
白名單: wiz8-mod/notes/research/2026-10-31-R18c-circle-mesh-walker.md
預期紅: research:2026-10-31-R18c-circle-mesh-walker
執行器: codex
預估讀入: R18 報告 §1（段頭、頂點）、R18b 報告（已排除的函式）；`wiz8-mod/build/r18/render.py`（段 walker、TGA、PNG，重用）；`wiz8-mod/build/wiz8.asm` 只 grep（見下）；Circle PVL 781 KB；約 1 MB
停止條件: 三角索引與段→材質都有結論就產圖、寫完報告結束；任一題追兩輪沒進展寫「未取得」（列追過的函式、grep 樣式與命中數），照樣寫完報告結束

## 先做
**第一個動作**：建報告骨架（檔頭 TL;DR、`## 發現`、`## 建議`），每得一條結論就追加。白名單檔 1260 秒沒變動會被看門狗砍。

## 背景（已排除，不重追）
- R18b：0x4bce20、0x4bd0d0、0x4bdc90、0x4be200 非 mesh parser。
- cc 複核：0x4baff0 底下 6 個哨兵守衛函式 0x4bc060／0x4bc140／0x4bc380／0x4bc5e0／0x4bc850／0x4bc9d0 都只用 0x404ea0 讀固定小欄位（4、0x14、0xc、1…），沒有依 nv／nt 算長度的讀取 → 不是 mesh 段。
- 推測：mesh 區段整塊讀進記憶體（或 file map）後以指標走訪，所以沒有逐陣列的讀檔呼叫。

## 找法（建議，可換）
1. 用段頭常數找走訪者：同一函式內同時有 `cmp …,0xffffffff`（或 `-1`）與 `0x100`（256）比較，且有 `×12`（`lea r,[r+r*2]` 後 `shl 2`／`imul …,0xc`）指標前進。
2. 從 R18 段長反推：每段 = 段頭 32 B + nv×12 + ?；用 Circle 62 段實測「段頭到下一段頭」的距離減 nv×12，對 nt 做線性擬合（nt×6？nt×12？＋逐頂點 UV nv×8？），再回碼找對應常數印證。資料擬合只當線索，碼佐證才標 `[已驗證]`。
3. 段→材質：段頭 X 欄是否 < 97（材質表筆數）；對得上就用材質表貼圖名。

## 要答（每條附 `build/wiz8.asm:<行>` 或 `build/r18c/<檔>`／檔內位移與三態）
1. mesh 段走訪函式 VA、段內各陣列順序、筆數來源、每筆大小。
2. 三角索引：位置、型別、每三角幾值；驗證 62 段索引全 < nv、每段走完剛好接下一段段頭。
3. 段→材質、UV 位置。
4. 產圖 `build/r18c/circle.png`（真索引、各段 TGA、法向 Y 朝上，門檻寫報告）＋`circle-nodes.png` 疊 iy＝3 節點，報告寫節點落在地面內比例（R18 重建版 35/154）。

## 不做
牆、天花板、光照、其他地圖、adapter、足跡（MG：只要迷霧）。

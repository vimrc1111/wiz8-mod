# P5 RE：掉落、寶箱與商店的物品來源；列出拿不到的物品

驗收: bash scripts/check-research.sh notes/re/loot.md
規模: 中（codex gpt-6-astra medium；靜態 RE＋資料格式，一份筆記；不寫碼入版）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔；不寫入遊戲目錄；不執行遊戲；不貼反編譯碼或組語區塊（位址、前 16 bytes、常數、自己的話與數學式可寫）；不下載任何東西；不把原版資料內容整表抄進筆記（列 id、欄位意義、統計與少數例子即可）
白名單: notes/re/loot.md
預期紅: research:loot
執行器: codex

## 背景
MG 2026-10-08：「遊戲裏頭好幾把神兵都拿不到，例如浮士德之戟、Maenad's Lance 之類的女武神神兵」「所以掉落率也能反組譯吧」。
同系列 P／P2／P3／P4（`notes/re/formulas.md`、`formulas-2.md`、`growth.md`、`abilities.md`）。RNG 入口 `0x40efa0`（P2）。物品名稱欄格式見 `tools/dbsname`（ITEMS.DBS 819 筆，名稱欄 29 units）。
材料（worktree 內 build/，gitignored，只讀；沙箱沒有遊戲目錄與 exe，不跑 `re/run.sh`）：
- `build/wiz8.asm`、`build/decomp/`
- `build/dbs/`：DATA.SLF 的 DATABASES\*.DBS 原檔（ITEMTABLES、ENCOUNTERTABLES、MONSTERS、ITEMS、NPC、LEVELS 等）
- `build/levels/`：LEVELS.SLF 內除 TGA／OCT／PVL 以外的成員（STS、RLK、WPT、IFL、LVL），目錄結構照原路徑；`build/levels/list.tsv` 為全部成員清單（偏移、長度、路徑）
- `build/items-en.tsv`：物品 id、英文名、目前中文

## 要找的（依優先序；找不到的標 [未取得] 並寫查到哪裡）
1. 物品來源有哪幾類、各在哪個檔：怪物掉落（ITEMTABLES？MONSTERS 欄位？）、固定寶箱／地上物（levels 哪種檔）、商店貨單（NPC.DBS？）、任務獎勵（NPC SCRIPTS 不在材料內，標 [未取得] 即可）。每類寫出記錄格式（欄位位移、意義），以及 exe 中讀它的函式位址。
2. 掉落規則：一場戰鬥結束怎麼決定掉什麼——掉落機率、物品池、等級／區域對應、隨機數怎麼用（位址＋式子）。
3. 用上面解出的格式，列出 ITEMS.DBS 中**沒有被任何已解析來源引用**的物品 id＋英文名；特別標出 74 Faust Halberd、96 Maenad's Lance。若某類來源沒解出，明寫結論只涵蓋哪幾類。
4. 每個物品可從哪些來源取得（id → 來源類別與機率／地點），寫成方法：要 cc 本機跑的話，給出解析步驟，cc 寫工具。

## 報告格式
`notes/re/loot.md`：
- `## 發現`：每條 `- `，寫位址＋前 16 bytes（asm 沒有 raw bytes 就標 [未取得]）、檔案格式、式子。來源寫 `build/wiz8.asm:<行號>` 或 `build/dbs/<檔>@0x<偏移>`，每條附 `[已驗證]`／`[推測]`／`[未取得]`。
- `## 未引用物品`：id、英文名、依據（哪幾類來源已查）。
- `## 規格疑慮`：沒有就寫「無」。

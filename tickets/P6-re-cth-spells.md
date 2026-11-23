# P6 RE：完整命中（CTH）、法術傷害、SP 回復、表值欄位

驗收: bash scripts/check-research.sh notes/re/combat-2.md
規模: 中（codex gpt-6-astra medium；靜態 RE＋資料格式，一份筆記；不寫碼入版）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔；不寫入遊戲目錄；不執行遊戲；不貼反編譯碼或組語區塊（位址、前 16 bytes、常數、自己的話與數學式可寫）；不下載任何東西；不把原版資料內容整表抄進筆記（列欄位意義、統計與少數例子即可）
白名單: notes/re/combat-2.md
預期紅: research:combat-2
執行器: codex

## 背景
MG 2026-10-08：「在合理的情況下，我們是否是可以單靠公式就算出以等級 28 破關，什麼職業的潛在傷害力最高」。本票補齊計算器所缺的公式與表格式；計算器是下一票。
前案：`notes/re/formulas.md`（P，F16／F19 CTH 未完、低等級 scale 常數未取得、cc 補查清單在「規格疑慮」）、`formulas-2.md`（P2，法術強度、Power Cast、範圍未完）、`abilities.md`（P4，SP 回復倍率 0x1a／0x1b 數值未取得）、`growth.md`（P3）。已知事實不重寫，只引用條號並補缺。
**這次有 raw bytes**：前案的「缺原始 bytes／資料區」已解決，前 16 bytes 與常數都要實際讀出來。
材料（worktree 內 build/，gitignored，只讀；沙箱沒有遊戲目錄，不跑 `re/run.sh`）：
- `build/Wiz8.exe`：原檔。區段：.text VA 0x401000 檔偏移 0x1000；.rdata VA 0x5eb000 檔偏移 0x1eb000；.data VA 0x5ff000 檔偏移 0x1ff000（imagebase 0x400000）。
- `build/wiz8.asm`、`build/decomp/`；`build/decomp/p6.c` 是本票新反編譯的 24 個函式（前案點名的 0x4e8370、0x53d870、0x547940、0x5479b0、0x4ee130、0x4ee220、0x541c00、0x541ec0、0x542ee0、0x542fc0、0x542e10、0x543dc0、0x53e250、0x5420b0、0x543270、0x519ac0、0x4ff790、0x4fb4c0、0x54e950、0x544b60、0x503100、0x552250、0x553350、0x5535d0）
- `build/dbs/DATABASES/*.DBS`：含 ITEMS、MONSTERS、SPELLEFFECT、SPELLTABLES、SPELLDESC 等原檔
- `build/items-en.tsv`：物品 id、英文名、目前中文

## 要找的（依優先序；找不到的標 [未取得] 並寫查到哪裡）
1. **完整 CTH**：攻擊值全分支（F16 續）→ 與怪物防禦（AC？）怎麼比 → 命中判定（F19 續，≤5／>95 特例），以及揮擊次數中每揮是否重擲。寫成可直接實作的整數式。
2. **低等級 scale 常數**：`0x5ee800`、`0x5ed490` 等 float 的實值，確認 `F(n,m)` 與 Lightning Strikes／背刺門檻在 L≤20 的精確式。
3. **法術傷害**：Power Cast 到效果量的完整映射（P2 未取得部分）；法術模板 F 的記錄格式在哪個 DBS、欄位位移（傷害骰、PL 上限、SP 消耗、領域、目標類型／範圍）；傷害如何套抗性（六領域）與怪物的抗性欄位。舉 2–3 個攻擊法術當例子（id、骰、成本）。
4. **SP 回復**：基礎式（每回合／每步？讀哪些屬性、等級）＋ 0x1a／0x1b 倍率實值。
5. **表值欄位**：ITEMS.DBS 武器的傷害骰、命中加成、攻擊模式、攻速修正、需求職業位移；MONSTERS.DBS 的 HP、AC、抗性、閃避相關欄位位移。寫格式＋讀它的 exe 位址，不整表抄。
6. 給下一票（計算器）的介面：哪些輸入、哪些式子已可精確、哪些仍是近似（照前案第 5 條的格式）。

## 報告格式
`notes/re/combat-2.md`：
- `## 發現`：每條 `- `，寫位址＋前 16 bytes、檔案格式、式子。來源寫 `build/wiz8.asm:<行號>`、`build/Wiz8.exe@0x<檔偏移>` 或 `build/dbs/<檔>@0x<偏移>`，每條附 `[已驗證]`／`[推測]`／`[未取得]`。
- `## 計算器介面`：第 6 項。
- `## 規格疑慮`：沒有就寫「無」。

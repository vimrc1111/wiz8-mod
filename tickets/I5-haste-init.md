# I5 研究：Haste 與先攻對近戰輸出的影響（goals/stage16.md）

驗收: bash scripts/check-research.sh notes/re/haste-init.md
規模: 小（研究，一份報告）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單以外的檔；不寫入遊戲目錄；不執行遊戲或 exe；不下載、不執行第三方工具；不送 email 或真實姓名（User-Agent 用「wiz8-mod personal research (non-commercial)」）
白名單: notes/re/haste-init.md
預期紅: research:haste-init
執行器: muse

## 你是誰、要做什麼
你是研究執行方，只做靜態分析與查資料，不寫碼。專案：Wizardry 8 數值試算器，現在每回合近戰期望傷害（`calc fight`，notes/calc.md:15）不含 Haste、也不把先攻算進分數。MG：「top tier 的武器全部都加 init，而且靈能者法術書（武僧、靈能者）跟工匠還有 haste 可以用」。下一步要把 Haste 當情境輸入、把先攻納入分數，需要規則。範圍只限數值（不碰地圖、移動、投射物）。

## 材料（worktree 內實體檔，唯讀）
- `build/Wiz8.exe`（v1.2.4）、`build/wiz8.asm`、`build/data-x/DATABASES/`（ITEMS.DBS 格式 notes/re/combat-2.md:85；法術表格式見 notes/re/ 下既有筆記）。
- 已知公式：notes/re/formulas.md（攻擊次數、揮擊數、Init 相關條目）、notes/re/combat-timing.md、notes/research/2026-10-09-R13-combat-speed-community.md。

## 待答問題
1. Haste 的效果：改哪個角色欄位（攻擊次數、揮擊數、先攻、行動延遲）、數值與法力等級的關係、持續時間。
2. Haste 的來源：哪些法術（含 Psionic 法術書，Monk／Psionic 可學）、哪些物品或工匠（Gadgeteer）道具；職業／種族可用性。
3. 先攻（Init）怎麼影響一回合內的出手：只決定順序，還是也影響攻擊次數或回合內行動數？武器 Init 加成（物品 +0x48，notes/re/combat-2.md:88）進哪條公式。
4. 對「擊殺回合數」的影響：先出手能否在怪物行動前殺掉（只寫規則，不做模擬）。
5. 給下一張實作票的文字規格：Haste 情境輸入怎麼改 calc fight 的哪些輸入；先攻要不要、怎麼進分數。

找不到時照實寫 `[未取得]` 與原因。

## 證據規格（機檢）
- 結論寫在 `## 發現` 段，每條一行、以 `- ` 開頭；每條含來源 `build/wiz8.asm:<行號>`（或 `Wiz8.exe:0x<偏移>`、網址），並標 `[已驗證]`、`[推測]`、`[未取得]` 之一。
- `[已驗證]` 只給你在材料上實際讀到的；只憑命名或社群說法的標 `[推測]`。
- 報告不放反組譯碼或還原的原始碼；不轉錄 DBS 表。
- 不要把「沒找到」寫成「不存在」。

## 交付回覆
最後回覆附「規格疑慮」段；沒有就寫「無」。

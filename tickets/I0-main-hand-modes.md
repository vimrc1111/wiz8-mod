# I0 研究：玩家主手可選攻擊模式（goals/stage16.md）

驗收: bash scripts/check-research.sh notes/re/attack-modes.md
規模: 小（研究，一份報告）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單以外的檔；不寫入遊戲目錄；不執行遊戲或 exe；不下載、不執行第三方工具；不送 email 或真實姓名（User-Agent 用「wiz8-mod personal research (non-commercial)」）
白名單: notes/re/attack-modes.md
預期紅: research:attack-modes
執行器: muse

## 你是誰、要做什麼
你是研究執行方，只做靜態分析與查資料，不寫碼。專案：Wizardry 8 數值試算器（`calc rank` 列舉角色開局×武器，算每回合近戰期望傷害）。下一張票要列舉「主手攻擊模式」，需要知道玩家在戰鬥選單能替主手選哪些模式（mode 0..8，表見 `notes/re/combat-2.md:34` C06）。

## 材料（worktree 內實體檔，唯讀）
- `build/Wiz8.exe`（v1.2.4，檔內偏移＝VA−0x400000）、`build/wiz8.asm`、`build/data-x/DATABASES/ITEMS.DBS`（記錄格式 `notes/re/combat-2.md:85`）。
- 已知：物品 +0x4e u16 是允許模式遮罩（`notes/re/combat-2.md:94`）；副手模式自選 `0x542ca0`（`notes/re/backstab.md:35` B15）；狂暴＝mode 3，原版物品遮罩只有 Bloodlust 有 bit 3（`notes/re/backstab.md:37` B16）。但實玩戰士系能對一般武器選狂暴，所以主手 UI 可選集合 ≠ 物品遮罩。

## 待答問題
1. 主手（玩家手動選）可選模式集合怎麼算：物品遮罩之外，哪裡加入其他模式（職業 feature、技能門檻、等級）？函式位址與條件。
2. 狂暴（mode 3）：哪些職業能選、要什麼條件（技能、等級）？
3. 其他模式（0..8 各是什麼，名稱 [推測] 即可）是否也有職業／技能來源。
4. 空手（徒手）主手的可選模式來源（H+0x31，`notes/re/backstab.md:35`）。
5. 給下一張票的規則：`MainModes(class, level, skills, item) -> 模式集合` 的文字規格。

找不到時照實寫 `[未取得]` 與原因，不要拿物品遮罩湊答案。

## 證據規格（機檢）
- 結論寫在 `## 發現` 段，每條一行、以 `- ` 開頭；每條含來源 `build/wiz8.asm:<行號>`（或 `Wiz8.exe:0x<偏移>`、網址），並標 `[已驗證]`、`[推測]`、`[未取得]` 之一。
- `[已驗證]` 只給你在材料上實際讀到的；只憑命名或社群說法的標 `[推測]`。
- 報告不放反組譯碼或還原的原始碼：只寫位址、自己的話描述行為；不轉錄 DBS 表。
- 不要把「沒找到」寫成「不存在」。

## 交付回覆
最後回覆附「規格疑慮」段；沒有就寫「無」。

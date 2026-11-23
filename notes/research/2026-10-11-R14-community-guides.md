# R14 社群攻略／組隊／戰鬥（2026-10-11）

> MG 2026-10-11：「派 Haiku 5.5 去搜尋 wiz8 的 walkthrough、party guide 或 combat skill……要求清楚關鍵字要有不要自己生成或幻覺……當作參考資料，縮減我們實際需要模擬的範圍」。
> 三個 Haiku 平行搜（組隊／戰鬥／法術與流程），引文原檔在 `2026-10-11-R14-web/`；cc 判讀。社群說法只當線索（goals/stage10.md B1），沒過 D3 的不算數。

## 引文核對（機檢）
- Haiku 的 WebFetch 會經小模型轉述，引文不保證逐字。cc 另以 curl 抓原頁、去標籤、正規化空白與引號後，逐條比對子字串：`python3 -I scripts/vq-quotes.py <md> <頁面快取目錄>`，結果 `2026-10-11-R14-web/vq-*.txt`。
- 對上：組隊 67/68、法術 151/154、戰鬥 123/127（戰鬥檔兩個 URL 原寫壞，cc 修正後重比）。對不上的 8 條多為格式殘留（`\"`、`[...]`、行尾逗號）；未核實者不採用。
- 疑似注入：Haiku 回報兩個 Steam 帖（…1737760452986202911、…2145343448810179395；另一份報 …846965611335606332）的 fetch 結果含「逐字引用」指示。cc 對 curl 原頁搜 `verbatim`／`ignore previous`／`language model`／`quote it` 零命中；判讀為 fetch 摘要器回應我方提示，非頁面內容 [推測]。
- 沒找到：GameFAQs 的 Wizardry 8 FAQ、抗性表、爆擊／即死機率、彈藥數、AC 數值。官方 hint book PDF（mocagh.org，>10MB）未抓。

## 與我方已驗證公式對照
| 社群說法 | 來源 | 對照 |
|---|---|---|
| 單手 BAR＝`Floor(Floor((CC+2*WPN)/3)*2+Floor(DEX/2)+SUM(ProfessionLvLUpMul*Lvl))/3)+60` | combat.md「BASE ATTACK RATING FORMULA」 | 結構同 `notes/re/formulas.md` F10（`60+tz((floor(DEX/2)+2K+P+M)/3)`）、F06 職業加權等級；逐項等價未驗 [推測] |

## 可轉成 calc 題目的可檢說法（待 calc 對應節點 D4 後出題；答案以 calc 為準，社群說法只當預期）
- 法術傷害＝每 PL 法力 1/4＋法術等級 1/2＋PL＋施法者等級 1/2，再乘 Power Cast 技能百分比（combat.md「At last, I understand spell damage!」）。對應法術核心循環群 4fa4d0。
- 抗性：`Effective=(目標等級−SAS)*3+基礎抗性`，≤0 全傷；`Final=Eff/2+rand(0..Eff/2)`，傷害按 Final% 減（同帖）。
- 雙持：0 點 Dual Weapons 主手 50%、副手 25% 技能；超過 100 無效（同帖有人反駁）（combat.md「w8 dual weapons skill」）。
- 睡眠／麻痺／網住的目標吃近戰雙倍傷害，遠程與法術不加（combat.md LP Archive）。
- 戰士等級不計入破法術抗性（party.md）。
- 每回合攻擊／揮擊次數取決於速度、敏捷、戰鬥與武器技能、等級（combat.md「dexterity speed or senses」）。
- 起始職業決定法力回復（party.md，表格未擷取）。
- 武士近戰、忍者投擲有爆擊即死（combat.md Cyberly；SNIPPET 未核）。

## 玩家共識（縮小 `calc party` 先算的範圍，不當結論）
- 近戰最強、戰士傷害最高；Valkyrie 優於 Lord；Bishop 不適合新手（combat.md QT3、GOG）。
- 2–3 近戰＋至少 2 施法者＋遠程／輔助；近戰先堆 STR，第二屬性 DEX（party.md、combat.md）。
- 用法：`calc party` 預設先算這些組合給 MG 看，枚舉範圍不縮；共識與 calc 結果不符時列出，不改 calc。

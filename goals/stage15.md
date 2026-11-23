# 階段 15：最佳化試算 v0（近戰）

> MG 2026-10-12：「執行到可以開始做最佳化試算差多少」「距離目標可以試算大概還差幾個 task」「之後大單拆小一點派給 gpt-6.1 sol，astra 等顯式要求再派」。
> 承 goals/stage12.md 路線 A（A1 `calc fight` 已完成）。v0 只用近戰分數；法術（A2）、生存（A3）、六人隊（A5）是 v1。數值規劃範圍照舊。

## 票
| 票 | 內容 | 驗收（會紅） | 執行器 |
|---|---|---|---|
| H1 | `PlanSkills`：依技能優先序配技能點（創角 15、升級每級 9；每項創角 ≤5、升級 ≤3；手動裸值 ≤75；只配本職業可用技能；進階技能 34..40 在對應裸屬性 ≥100 的等級起才可配）。規則出處 `notes/re/growth.md`:13、:16、:18、:25 | `TestPlanSkills` | codex sol |
| H2 | `calc rank`：列舉種族 × 職業 × 性別（只在限制職業時分）× 屬性優先序（使用者給的屬性集合的全排列）× 等級清單，技能由 H1 依範本優先序配，武器／怪物／難度取範本；分數＝`calc fight` 的 `round.damage`。輸出每組合一列＋各等級分數的 Pareto 前緣 | `TestRank`（小列舉逐筆對 `Fight` 獨立重算） | codex sol |
| H3 | 試算 v0：cc 用 H2 跑一份（對 Dark Savant 與一隻中期怪，L1／L10／L28），入 `notes/research/` | 結果檔＋重跑指令 | cc |

## 假設（試算報告要列出）
- 練技（使用自然成長）是玩法行為，不建模；範本可給 `Train`（技能→額外裸值）當情境輸入，預設 0。
- 自動初始技能 [未取得]（growth.md:20「不含自動初始技能／練技」）：取 0，H1 註明。
- 裝備屬性加成、負重、背刺、Lightning Strikes、combatSkill 0x11 沿用 calc fight 未建模清單。

## 完成定義
1. H1、H2 收貨，各自四段注入。
2. H3 結果入檔，報告三段給 MG。
3. check-formulas、calc-qa、party 綠；CI exit 0 並 push。

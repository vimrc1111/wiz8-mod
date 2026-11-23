# R11 研究：Wizardry 8 遊戲機制公式（戰鬥、技能、職業特技）

驗收: bash scripts/check-research.sh notes/research/2026-10-08-R11-formulas.md
規模: 中（研究，一份報告）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單以外的檔；不寫入遊戲目錄；不下載、不解壓、不執行任何程式或下載物（只讀網頁、文件與公開原始碼頁面）；報告不貼反編譯碼（位址或檔名＋行號＋一句話描述）；不送 email 或真實姓名
白名單: notes/research/2026-10-08-R11-formulas.md
預期紅: research:2026-10-08-R11-formulas
執行器: muse

## 背景
MG 想不靠修改器重練，直接用公式比較隊伍與角色配置（Steam 版 v1.2.4）。例：武士（Samurai）的「Lightning Strikes／閃電連擊」發動機率跟哪些屬性、技能、裝備、負重相關；28 級、滿技能、拿最好武器時哪個職業／種族攻擊力最強。
已知可能來源：社群機制文件（zimlab.com、tk421.net、GameFAQs FAQ、Steam／GOG 討論、Wizardry 8 wiki）、社群反組譯專案 https://github.com/agluszak/wizardry-8-decomp 。

## 問題（每條結論附來源 URL 或 `file:line@commit`，與三態標記；公式寫原式，不轉述成大概）
1. 命中與傷害：命中率、每回合攻擊次數、傷害（武器骰＋力量加成＋技能）、爆擊／致命一擊（Critical／Kill）公式。
2. 職業特技：Samurai Lightning Strikes（閃電連擊）、Monk／Ninja 的多重攻擊與 Critical、Ninja 投擲、Fighter／Lord／Valkyrie 的特技（Berserk、Power Strike 等）——各自的觸發條件、機率公式、受哪些屬性／技能／裝備／負重影響。
3. 負重（Load／Encumbrance）對速度、攻擊次數、特技機率的影響門檻。
4. 屬性與技能：各屬性影響什麼（Strength→傷害、Speed→先攻與攻擊次數…）、技能 100 上限之後（125 與 Power Skills）、種族差異。
5. 法術：法術威力與 Power Level、施法成功率、抗性公式。
6. 有沒有人做過「最強職業／配置」的數值比較或模擬器（附網址），結論為何。
7. 各來源的可信度：實測／反組譯／傳聞，分開標；互相矛盾的列出兩邊。

## 報告格式
`notes/research/2026-10-08-R11-formulas.md`，含 `## 發現` 段，每條 `- ` 結論附來源與 `[已驗證]`／`[推測]`／`[未取得]`。另含 `## 規格疑慮` 段（沒有就寫「無」）。

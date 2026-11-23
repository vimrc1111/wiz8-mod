# R13 研究：Wizardry 8 戰鬥加速的社群做法與遊戲內建設定

驗收: bash scripts/check-research.sh notes/research/2026-10-09-R13-combat-speed-community.md
規模: 中（研究，一份報告）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單以外的檔；不寫入遊戲目錄；不執行遊戲或任何下載物；不下載可執行檔；報告不轉錄整張網頁表格；不送 email 或真實姓名（User-Agent 用「wiz8-mod personal research (non-commercial)」）
白名單: notes/research/2026-10-09-R13-combat-speed-community.md
預期紅: research:2026-10-09-R13-combat-speed-community
執行器: muse

## 背景
Wizardry 8（Steam 版 Wiz8.exe v1.2.4）。玩家普遍覺得回合制戰鬥慢：每個行動有動畫、戰鬥訊息逐條停頓。我們打算在自己的 `ddraw.dll` 代理裡加「戰鬥加速」hook（倍率可調、可關），原版檔案不改。另有一份靜態 RE 票在查 exe 內部，本票只查外部資料。

## 問題（每條結論附 URL 與取得日期、三態標記）
1. 遊戲內建：選項選單裡和戰鬥速度相關的設定（例如訊息延遲、動畫、戰鬥模式 continuous／phased），各自影響什麼；有無 ini／登錄檔隱藏設定。
2. 社群既有加速做法：Cheat Engine speedhack、修改版 exe、已知 patch／mod（例如 Wizardry 8 相關的 fan patch、Christian Coder 的 mod 是否處理過）、dgVoodoo／nGlide 的影格設定對速度的影響。每項寫做法原理（改計時 API、改常數、改設定）與已知副作用。
3. 已知副作用：全域加速會不會讓即時部分（探索中怪物移動、回復、遊戲時間）也加快；音效、語音被截斷；訊息讀不及。
4. 社群有沒有提過「訊息延遲」或「動畫時間」對應的 exe 位址或常數（有就附出處原文位置，不轉錄大段）。
5. 開放題：玩家最常抱怨的是哪一段慢（動畫、訊息、怪物回合、施法特效），以便決定加速哪一段。

## 報告格式
`notes/research/2026-10-09-R13-combat-speed-community.md`，含 `## 發現` 段，每條 `- ` 結論附來源與 `[已驗證]`／`[推測]`／`[未取得]`。另含 `## 規格疑慮` 段（沒有就寫「無」）。

# 階段 9：戰鬥加速

> MG 2026-10-08：「中文化收尾後再開研究票，我覺得加個戰鬥加速就差不多了」。MG 2026-10-09：「開工，先派研究票，你來執行到需要我測試爲止」。
> 分工見 CLAUDE.md「派工」。做到「hook 寫好、CI 綠、裝好待 MG 試玩」為止，試玩前不需要 MG。

## 原則
- 同中文化：原版檔案一個 byte 不動。加速做在我們的 `ddraw.dll` 代理裡，用簽章比對 hook（同 `hook/ddraw/build.sh` 的簽章列）。
- 倍率可調、可關（設 1＝原版行為）；只動戰鬥，不動探索與即時時間流（休息、回復、遊戲曆照原式，`notes/re/combat-2.md:83`）。
- 怪物移動倍率一起做進 hook（MG 2026-10-09：「一起做進 hook」）：效果同社群 Wiz8Fast（`notes/research/2026-10-09-R13-combat-speed-community.md:14`），但在記憶體裡改讀到的值，不改 `Wiz8.CFG`（那是 MG 的設定檔，遊戲內改選項會覆寫）。倍率可調，過高會「瞬移」，上限交 MG 試玩定。
- 影音不做（resume「下一步建議」：畫面交 nGlide／dgVoodoo）。

## 步驟
- 9.1 研究（平行）：
  - R13（muse，網路）：社群既有做法、遊戲內建設定、已知副作用。票 `tickets/R13-combat-speed-community.md`。
  - S（codex astra medium，靜態 RE）：exe 內戰鬥的等待從哪來（動畫、訊息停頓、回合間延遲、計時 API），候選 hook 點與簽章。票 `tickets/S-re-combat-timing.md`。
- 9.2 cc：收貨、抽查（R6／R8 報告有錯的教訓：關鍵位址 cc 自己對一次 asm），寫實作規格 `tickets/T-combat-speed-hook.md`（含會紅的驗收）。
- 9.3 codex（或 cc）實作；CI exit 0；四段注入。
- 9.4 cc：自動 run 只驗不崩、原版三檔雜湊不變；裝進遊戲目錄，交 MG 試玩（倍率、戰鬥 log 讀不讀得到、有沒有漏訊息）。

## 完成定義
- 研究兩份過 check-research，cc 抽查結果記在收貨 commit。
- hook 入版、CI exit 0、注入紅／還原綠。
- MG 試玩回饋記進 resume（倍率定案或要改）。

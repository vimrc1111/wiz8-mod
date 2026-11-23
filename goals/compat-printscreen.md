# 相容支線 C：PrintScreen 不崩

> MG 2026-10-07 定案（「可以，定案，先派 PrintScreen 給 codex」）。來源：next-goal-draft v2。分工見 CLAUDE.md「派工」。

- 現況：存出 Wiz800000.JPG 後，崩在系統 ddraw 10.0.26100.8115 的 +0x33159（0xC000041D）。
  - 無代理、有代理都能重現 → 原版的相容問題。
- 作法：在代理攔下存圖時對 surface 的 Lock／GetDC 路徑，改由我們自己存檔。具體作法交給反組譯結果決定。
- 驗收（2026-10-07 達成，run 063150；條件：Win11 PrintScreen 開剪取工具的設定須關閉，見 notes/re/shot.md）：實機按 PrintScreen 3 次，遊戲繼續活到 60 秒，有圖檔，目錄 diff clean。
  - 負對照：不放代理時照樣會崩（已有 run 042256 的紀錄）。


## 反組譯事實
見 `notes/re/shot.md`。

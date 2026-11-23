# Y4e 52b4f0 其他角色的 SP 改為輸入，calc 不寫死 case 初值（Y4d 收貨的 cc 發現）

驗收: bash scripts/case-index.test.sh 52b4f0 && ! grep -nE '10000|100\*character' internal/calc/n_52b4f0.go
規模: 中（codex gpt-6.1-sol medium）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔（含 re/emu/emu.py、scripts/）；不刪 boundary 既有列；不縮 random 範圍閃差異；不寫入遊戲目錄；不執行遊戲；不下載任何東西；筆記不貼組語或反編譯碼；不把原版 bytes、從 exe 讀出的表寫進 repo
白名單: re/emu/cases/0052b4f0.json, internal/calc/n_52b4f0.go, notes/re/iface/52b4f0.tsv
預期紅: 第二段 grep（現在 n_52b4f0.go 寫死 case 合成值）
執行器: codex

## 背景
Y4d（`tickets/Y4d-char-index-52b4f0.md`）讓 52b4f0 依 characterIndex 寫所選角色、其他角色留不同的合成值；但 `internal/calc/n_52b4f0.go` 把那些合成值（-10000-100*character-realm）寫死在 calc 函式裡。`calc f` 是給玩家算公式的，函式裡不能有 case 夾具。
範例：`internal/calc/n_55cc00.go`（Y4b）把其他角色的狀態當成輸入（rosterReverseDifficultyState0..7）。

## 要做的
1. 其他角色的各領域 SP 初值改成 case 的輸入（依角色 index 尋址或逐筆欄位皆可，格式見 `notes/re/emu.md`「case 格式」），彼此不同、與所選角色不同；Go 從輸入取值，不含 case 常數。
2. iface 依新輸入更新。`go test -count=1 -run 'TestEmu$' ./internal/calc` 全綠。

## 規格疑慮
寫在最終回覆。

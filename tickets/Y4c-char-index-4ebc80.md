# Y4c 4ebc80 case 依角色 index 尋址（取代 Y4 第 3 項的 4ebc80；grok zoWN 發現 3）

驗收: bash scripts/case-index.test.sh 4ebc80
規模: 中（codex gpt-6.1-sol medium）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔（含 re/emu/emu.py、scripts/）；不刪 boundary 既有列；不縮 random 範圍閃差異；不寫入遊戲目錄；不執行遊戲；不下載任何東西；筆記不貼組語或反編譯碼；不把原版 bytes、從 exe 讀出的表寫進 repo
白名單: re/emu/cases/004ebc80.json, internal/calc/n_4ebc80.go, notes/re/iface/4ebc80.tsv
預期紅: case-index
執行器: codex

## 背景
規格本體：`tickets/Y4-case-index.md` 第 3 項（emu 的 index 尋址已做完，格式見 `notes/re/emu.md`「case 格式」的 `index` 條；index<0 不寫、≥count 報錯）。範例：`re/emu/cases/0051cce0.json`（itemIndex 尋址，Y4a）。
稽核摘要（grok zoWN 發現 3）：本節點 case 用 `also` 把同一輸入值複寫到八名角色，原版依角色 index 讀「那一位」；因為八人值相同，讀錯人也不會 diff，D3 測不到「選了誰」。

## 要做的
1. 「所選角色」的欄位依角色 index 尋址（`index` 指向角色 index 輸入），拿掉全部 `also`。
2. 其他角色的對應欄位要與所選角色不同（另給輸入或固定不同值），讓「讀錯人」會產生 diff。
3. 若 Go 節點語意與原版不符（例如讀錯角色），照原版改。iface 依新輸入更新。
4. `go test -count=1 -run 'TestEmu$' ./internal/calc` 全綠（其他節點不得變紅）。

## 規格疑慮
寫在最終回覆。

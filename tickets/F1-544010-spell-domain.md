# F1 544010 spellIndex 輸入域擴到 540170 實傳的範圍（540170 收貨 d9b4171 的 cc 發現）

驗收: go test -count=1 -run 'TestEmu$' ./internal/calc && bash scripts/nodes/node-accept.sh 544010 && go run ./tools/calc f 00544010 $(bash scripts/nodes/f1-probe.sh) >/dev/null
規模: 中（cc 子 agent）
禁止項: 不 commit；不改白名單外任何檔（含 re/emu/emu.py、scripts/nodes/ 既有檔）；不刪 boundary 既有列；不縮 random 範圍閃差異；不寫入遊戲目錄；不執行遊戲；筆記不貼組語或反編譯碼；不把原版 bytes、從 exe 讀出的表寫進 repo
白名單: re/emu/cases/00544010.json, internal/calc/n_544010.go, notes/re/iface/544010.tsv, notes/re/cov-exempt.d/544010.tsv, scripts/nodes/f1-probe.sh
預期紅: 第三段（f1-probe.sh 尚不存在；建立後輸出 spellIndex＝8 的合法輸入向量，現行 calc 在 input 84 limit 2 panic）
執行器: cc

## 背景
`internal/calc/n_544010.go` 輸入檢查把第 84 欄（spellIndex，M+0x2e5）限 0..2，因為 case 的法術表 arena 只放 3 筆。540170 對 M+0x2e5 實傳怪物攻擊模式 0..8（`notes/re/iface/540170.tsv:139`），正常遊玩的值會 panic＝calc 輸入域比原版窄（goals/stage10.md B4「不准縮輸入範圍」）。

## 要做的
1. 確認 544010 對 M+0x2e5 的實際取值域（呼叫端 540170 為 0..8；若原版本身另有上限，以原版為準並寫進 iface 附證據行）。
2. case 的法術表 arena 擴到涵蓋整個取值域（各筆內容不同），boundary 加 spellIndex 上下界各一列，random 範圍覆蓋全域。
3. calc 的 limit 改成原版的域；法術表內容是輸入，不在 Go 裡寫死。
4. `scripts/nodes/f1-probe.sh` 印出一組 spellIndex＝域上界的合法輸入（單行空白分隔）。
5. node-accept 544010 綠；540170 的 node-accept 也重跑一次（它把 544010 當 D4 位址樁）。

## 規格疑慮
寫在最終回覆。

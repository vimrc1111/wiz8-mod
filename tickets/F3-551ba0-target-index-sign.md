# F3 551ba0 目標 index 未符號延伸（calc-qa Q4 作答 agent 發現）

驗收: go test -count=1 -run 'TestEmu$' ./internal/calc && bash scripts/node-accept.sh build/f3.list && go run ./tools/calc f 00551ba0 $(bash scripts/nodes/f3-probe.sh) >/dev/null
規模: 小（cc 子 agent）
禁止項: 同 F1（tickets/F1-544010-spell-domain.md）
白名單: internal/calc/n_551ba0.go, re/emu/cases/00551ba0.json, notes/re/iface/551ba0.tsv, scripts/nodes/f3-probe.sh
預期紅: 第三段（f3-probe.sh 印出怪物目標 characterIndex＝−1 的合法輸入；現行 n_551ba0.go:405 以 u32 讀 idx、:491 以 int64(idx) 傳給 sdr，−1 變 4294967295，在 5327e0 夾具檢查 panic）
執行器: cc

## 要做的
1. 確認原版對目標 +4 的取值與符號（怪物目標時為何值），calc 照原版延伸。
2. boundary 補怪物目標 index＝−1（或原版實際值）的列，0 差異。
3. `build/f3.list` 由執行者建立（`551ba0<TAB>calc 函式名`，衍生、不入 git）。

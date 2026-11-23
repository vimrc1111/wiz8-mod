# F2 540170 兩個 cov-exempt 塊在 calc 沒有 panic（540170 收貨 d9b4171 的 cc 發現）

驗收: go test -count=1 -run 'TestEmu$' ./internal/calc && bash scripts/nodes/node-accept.sh 540170
規模: 小（cc 子 agent）
禁止項: 同 F1
白名單: internal/calc/n_540170.go, notes/re/cov-exempt.d/540170.tsv
預期紅: 無機檢（規則「每個豁免塊 calc 都要 panic」目前只靠收貨抽查；本票只修這兩塊）
執行器: cc

## 背景
`notes/re/cov-exempt.d/540170.tsv` 的 54037e（失手名單配置回空指標）、540844（怪物指標為零的記錄塊）在 calc 沒有對應分支、也不 panic。規則：每個 cov-exempt 塊，calc 在對應條件成立時 panic。

## 要做的
1. 在 calc 模型能表達該條件的位置加 panic（訊息寫原版行為）。
2. calc 模型中該條件確實無從成立（例：配置失敗不在輸入域）時，不硬加死碼；改在該列理由補一句「calc 無對應狀態：<為何>」，最終回覆列出。

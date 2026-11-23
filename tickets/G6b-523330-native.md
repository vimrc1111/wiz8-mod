# G6b 54df00 原生比對轉綠：523330 callee 副作用＋52e690 記錄容量（G6 a0c9e89 的後續；goals/stage14.md）

驗收: bash scripts/node-native.sh 54df00 && { bash scripts/node-accept.sh tickets/nodes/g6b.list >/dev/null 2>&1 || { echo 'FAIL g6b-emu'; false; }; } && { bash scripts/calc-qa.test.sh >/dev/null 2>&1 && bash scripts/party.test.sh >/dev/null 2>&1 || { echo 'FAIL g6b-calc'; false; }; }
規模: 大
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔（含 re/emu/emu.py、scripts/、fnative_test.go）；不刪 boundary 既有列；不縮 random 範圍或輸入域閃差異；不把初始角色類別等改成略過重建的哨兵值；不忽略 RNG；不放寬任何驗證；不寫入遊戲目錄；不執行遊戲或 exe；筆記與程式註解不貼組語或反編譯碼；不把原版 bytes、從 exe／DBS 讀出的表寫進 repo
白名單: internal/calc/n_523330.go, notes/re/iface/523330.tsv, re/emu/native/0054df00.json, re/emu/cases/0054df00.json, internal/calc/n_54df00.go, notes/re/iface/54df00.tsv
預期紅: node-native
執行器: codex

## 背景（G6 執行方回報，見 notes/re/iface/54df00.tsv 的 native 列、notes/re/emu.md「G6」段）
- 原生 523330 經 50e650→4ed9d0→52a2f0／52a3e0 重算 HP／體力；Go 的 523330（characterConditionRemoveNode）只記 50e650 呼叫，F 沒有這些副作用。原 boundary 索引 3：原生 HP=7、F=52（c1_b11）。
- 完整原生域會經 HP 重算觸發 522fe0→52abf0→52f110 的死亡回應，原 case 的 52e690（排除類語音樁）8 次記錄容量不夠；原 random 索引 19（cond=14）可重現。

## 要做的
1. 523330 的 F 補上 HP／體力重建副作用：50e650 以下若已有 D4 的 Go 節點就呼叫它們（`internal/calc/n_*.go` 先搜），沒有才在 n_523330.go 內寫；夾具上限照「原版上限或〔未取得〕＋正常遊玩可達值」，理由寫 iface。
2. 52e690 記錄容量：從原生路徑最大呼叫次數定（依據寫 iface），改在 native 夾具或原 case；原 case 改動後 54df00 的 TestEmu 仍 diff=0。
3. 523330 自己的 TestEmu 與 54df00（`tickets/nodes/g6b.list`）照綠；523330 的 F 呼叫端（503100、4ea5c0、524720、5236a0、52abf0、56c6d0）走 F，calc-qa／party 也在驗收內：若因修正使 calc 答案改變而紅，不改測試期望值，停下寫在規格疑慮（列哪題、舊值、新值）。worktree 內 build/emu 是完整副本，`bash scripts/check-formulas.sh` 不新增不符。
4. 四段注入：在 523330 F 的新副作用路徑注入差一 → node-native FAIL → 副本還原 → PASS；貼進回覆。

## 規格疑慮
寫在最終回覆；沒有就寫「無」。

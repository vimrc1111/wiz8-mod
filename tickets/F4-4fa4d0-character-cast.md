# F4 4fa4d0 法術全流程對人物施法者跑不起來（calc-qa Q8 作答 agent 發現）

驗收: go test -count=1 -run 'TestEmu$' ./internal/calc && bash scripts/node-accept.sh build/f4.list && go run ./tools/calc f 004fa4d0 $(bash scripts/nodes/f4-probe.sh) >/dev/null
規模: 中（cc 子 agent）
禁止項: 同 F1
白名單: internal/calc/n_4fa4d0.go, internal/calc/n_4fb4c0.go, internal/calc/n_54ba00.go, internal/calc/n_54c930.go, internal/calc/n_4ff220.go, internal/calc/n_53e250.go, re/emu/cases/004fa4d0.json, re/emu/cases/004ff220.json, re/emu/cases/0053e250.json, notes/re/iface/4fa4d0.tsv, notes/re/iface/4ff220.tsv, notes/re/iface/53e250.tsv, notes/re/cov-exempt.d/4fa4d0.tsv, notes/re/cov-exempt.d/4ff220.tsv, scripts/nodes/f4-probe.sh
預期紅: 第三段（f4-probe.sh 印出「Mage L10、Fire 50、PL3 Fireball（SPELLTABLES id 36）、人物施法者」的合法輸入；據回報現行 calc 會 panic）
執行器: cc

## 背景（回報，未驗）
- 0x4ff220 夾具上限 spellIndex≤3、sourceMonsterID≤7：比原版輸入域窄（同 F1 的形狀）。
- 0x53e250 的人物來源（marCharSource，n_53e250.go:199）把 +8 寫 0xffffffff，之後人物施法必 panic。
先重現兩項，分清「calc 與原版不符」或「夾具窄於原版域」，再以原版為準修；不縮輸入範圍閃差異。

## 要做的
1. 重現；每項寫出原版的實際域（附位址）。
2. 擴夾具／修 calc，各受影響節點 node-accept 綠（清單 `build/f4.list` 由執行者建立）。
3. f4-probe.sh 指向 Q8 的情境；最終回覆給出 calc 算出的成功率。

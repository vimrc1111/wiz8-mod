# F5 F3／F4 白名單補正（cc 切單漏列；F3／F4 執行 agent 回報）

驗收: 同 F3、F4 的驗收命令，三段全綠（含 probe）
規模: 中（cc 子 agent，接續 F3／F4 的 worktree ../wiz8-mod-F3）
禁止項: 同 F1
白名單（追加於 F3、F4 之上）: re/emu/cases/00551eb0.json, internal/calc/n_551eb0.go, notes/re/iface/551eb0.tsv, re/emu/cases/004ff7f0.json, internal/calc/n_4ff7f0.go, notes/re/iface/4ff7f0.tsv, re/emu/cases/004fedc0.json, internal/calc/n_4fedc0.go, notes/re/iface/4fedc0.tsv, re/emu/cases/004fef90.json, internal/calc/n_4fef90.go, notes/re/iface/4fef90.tsv, notes/re/cov-exempt.d/4ff7f0.tsv, notes/re/cov-exempt.d/4fedc0.tsv, notes/re/cov-exempt.d/4fef90.tsv
執行器: cc

## 裁決（cc）
- F3：551eb0 的 charIdx 改有號，boundary 補 −1（怪物目標）列。
- F4：照執行 agent 的修法：四個 case（4ff7f0、4ff220、4fedc0、4fef90）的法術物件移到空 arena、筆數 150（原版 0x5001e0 拒 ≥0x96），random／boundary 擴到 0..149；4ff220 的來源欄依來源種類接受 0xffffffff 哨兵（原版 53be00／53be50 寫入），n_4fb4c0.go 生成段的 marLoc 同步。
- 4fef90 維持 公式 D4：本票只擴夾具域，不改歸類（落點輸出照舊當數值輸出）。
- 受影響節點全部 node-accept 重跑（清單放 build/）。

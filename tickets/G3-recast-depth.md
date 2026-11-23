# G3 法術 recast 深度夾具上限（4fa4d0 SCC、4fac40）（goals/stage14.md，後續票 4a）

驗收: { go run ./tools/calc f 004fa4d0 $(bash scripts/nodes/g3-probe.sh) >/dev/null || { echo 'FAIL g3-probe'; false; }; } && bash scripts/node-accept.sh tickets/nodes/g3.list
規模: 大
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔（含 re/emu/emu.py、scripts/node-accept.sh、scripts/nodes/ 既有檔，除非白名單列出）；不刪 boundary 既有列；不縮 random 範圍或輸入域閃差異；不放寬任何驗證；不寫入遊戲目錄；不執行遊戲或 exe；筆記與程式註解不貼組語或反編譯碼；不把原版 bytes、從 exe／DBS 讀出的表寫進 repo
白名單: re/emu/cases/004fa4d0.json, re/emu/cases/004fb4c0.json, re/emu/cases/0054ba00.json, re/emu/cases/0054c930.json, re/emu/cases/004fac40.json, internal/calc/n_4fa4d0.go, internal/calc/n_4fb4c0.go, internal/calc/n_54ba00.go, internal/calc/n_54c930.go, internal/calc/n_4fac40.go, notes/re/iface/4fa4d0.tsv, notes/re/iface/4fb4c0.tsv, notes/re/iface/54ba00.tsv, notes/re/iface/54c930.tsv, notes/re/iface/4fac40.tsv, notes/re/cov-exempt.d/4fa4d0.tsv, notes/re/cov-exempt.d/4fb4c0.tsv, notes/re/cov-exempt.d/54ba00.tsv, notes/re/cov-exempt.d/54c930.tsv, scripts/nodes/g3-probe.sh
預期紅: g3-probe
執行器: codex

## 背景（c9e44d9 收貨時回報）
- 法術 SCC（4fa4d0／4fb4c0／54ba00／54c930）的 recast 深度夾具上限 3：0x408860（sound_play，排除樁）的 record 只記 3 次，0x4fac40 的樁記錄 4 筆。正常遊玩的深度受 SP／cost 限制，可以超過 3；calc 超過就 panic＝calc 輸入域比原版窄。

## 要做的
1. 從 `build/wiz8.asm` 找 recast 深度的原版上限（有就附行號）；沒有硬上限就寫 `[未取得]`，夾具上限取「SP 上限÷最小 cost」能到的深度，理由寫 iface。
2. case 的樁記錄改成 `n` 次（`notes/re/emu.md` record／ret 的 `n`）涵蓋上面的上限；calc 的 limit 同步。
3. `scripts/nodes/g3-probe.sh` 印一組 recast 深度 5 的合法輸入（單行空白分隔，給 `calc f 004fa4d0`）。
4. boundary 加深度上限列。node-accept 五節點綠。
5. 四段注入：把 calc limit 改回 3 → 第一段 FAIL（panic）→ 副本還原 → PASS。

## 規格疑慮
寫在最終回覆；沒有就寫「無」。

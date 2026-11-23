# G5 547cb0 怪物筆數夾具上限 8 → 原版上限（goals/stage14.md，後續票 5）

驗收: { go run ./tools/calc f 00547cb0 $(bash scripts/nodes/g5-probe.sh) >/dev/null || { echo 'FAIL g5-probe'; false; }; } && bash scripts/node-accept.sh tickets/nodes/547cb0.list
規模: 中
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔（含 re/emu/emu.py、scripts/node-accept.sh、scripts/nodes/ 既有檔，除非白名單列出）；不刪 boundary 既有列；不縮 random 範圍或輸入域閃差異；不放寬任何驗證；不寫入遊戲目錄；不執行遊戲或 exe；筆記與程式註解不貼組語或反編譯碼；不把原版 bytes、從 exe／DBS 讀出的表寫進 repo
白名單: re/emu/cases/00547cb0.json, internal/calc/n_547cb0.go, notes/re/iface/547cb0.tsv, notes/re/cov-exempt.d/547cb0.tsv, scripts/nodes/g5-probe.sh
預期紅: g5-probe
執行器: codex

## 背景（49494d2 收貨時回報）
- `n_547cb0.go:44`：calc 輸入限怪物筆數 ≤8（夾具清單物件只放 8 筆）。正常遊玩一群可以超過 8 → calc 會 panic，輸入域比原版窄。calc-qa／party 會走到 547cb0（`notes/calc-reach.txt`）。
- 原版 P∈(-N,0) 時無限迴圈，calc 以 panic 表示，這條保留。

## 要做的
1. 從 `build/wiz8.asm` 找怪物群／清單的原版上限（附行號寫 iface）；找不到就 `[未取得]`，取正常遊玩能到的值並寫理由。
2. case 的清單物件擴到上限（各筆內容不同），boundary 加筆數上限列，random 範圍覆蓋全域；calc limit 同步。
3. `scripts/nodes/g5-probe.sh` 印一組怪物筆數＝上限的合法輸入（單行空白分隔）。
4. 四段注入：calc limit 改回 8 → 第一段 FAIL → 副本還原 → PASS。

## 規格疑慮
寫在最終回覆；沒有就寫「無」。

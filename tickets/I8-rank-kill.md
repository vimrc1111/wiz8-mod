# I8 calc rank：多怪＋擊殺率指標（goals/stage16.md I8）

驗收: { go test -count=1 -v -run 'TestRankKill$' ./internal/calc 2>&1 | grep -q -- '--- PASS: TestRankKill' || { echo 'FAIL TestRankKill'; false; }; } && { go test -count=1 ./internal/calc ./tools/calc >/dev/null 2>&1 || { echo 'FAIL calc-package'; false; }; }
規模: 小
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔（含 build.go、fight.go、n_*.go 節點檔）；不放寬或刪既有測試；不寫入遊戲目錄；不執行遊戲或 exe；註解不貼組語或反編譯碼；不把原版 bytes、從 exe／DBS 讀出的表或名稱寫進 repo
白名單: internal/calc/rank.go, internal/calc/rank_kill_test.go, notes/calc.md:15
預期紅: TestRankKill
執行器: codex

## 背景
- MG 2026-10-12 要六人隊「擊殺率最高」且「要計算致命一擊的即死」。現行 rank 每格只算 `round.damage`（rank.go `rankDamage`）。
- `Fight` 已輸出 `round.instakill`（一回合內至少一次即死，已含怪物抗性：cc 實測 564 為 0、411 約 0.22）。
- 怪物 HP 骰：`LoadMonsters`（load.go:64）的 `Monster.HP`（`Dice{Z,N,S}`，模板 +0xd6；建立實例時擲 Z＋NdS，見 notes/re/combat-2.md C21、notes/re/loot.md 骰）。FightWorld 已有 `w.mons` 原始記錄，可直接讀 +0xd6（照 load.go `readDice`）。
- 10-08 筆記（notes/research/ranking-history.md:184）的擊殺率模型：每回合擊殺數＝1／期望回合數。

## 要做的
1. `RankTemplate` 加 `Monsters []int`（選填）與 `Metric string`（選填，`""`／`"damage"`＝現行，`"kill"`）。`Monsters` 非空時取代 `Monster`（此時 `Monster` 必須為 0，否則報錯）；每個 id 檢查範圍；重複 id 報錯。
2. 每格先對每隻怪算：模式加權（同現行 main×off 分布）的 D＝Σ w·round.damage 與 p＝Σ w·round.instakill（每回合混模式，兩者各自加權平均）。
3. 每隻怪的格值：
   - `damage`：D。
   - `kill`：H＝Z＋N·(S+1)/2（float），D>0 時 n＝⌈H/D⌉（H≤0 時 n＝1），E＝Σ_{r=1..n}(1−p)^{r−1}，格值 1/E；D≤0 時 p>0 格值 p、p＝0 格值 0。
   多怪時格值＝各怪格值的算術平均。輸出格式不變（Damage 欄放格值）。
4. 去重鍵要分辨 Monsters 清單與 Metric（現行 `monster` 欄不足）。
5. `TestRankKill`（新檔，讀 build 原版檔，缺檔 `t.Skip`）：範本 Fighter（class 0）L28、Races 兩個、單一 Loadout（任一 Fighter 可用近戰武器，從表挑）、Train {0:100,16:100,19:100,34:100}、AttrMin 100、Haste、`Monsters` [564, 411]、`Metric` "kill"、不給 Commands（固定 Mode 0）：
   (a) 每格等於測試自己對兩隻怪各呼叫 `Fight`（同 Attrs／Skills）讀 round.damage／round.instakill、自己讀 MONSTERS.DBS +0xd6 骰算 H、照上式算的平均（容差同 TestRank）；
   (b) 564 那隻的 p 為 0、411 的 p>0（斷言前提成立，否則測試無鑑別力）；
   (c) 同範本 `Metric` "damage" 的格值＝兩怪 round.damage 平均；
   (d) `Monsters` 非空且 `Monster`≠0 報錯；`Metric` 未知值報錯；
   (e) 加 Commands ["attack","berserk"] 的 kill 格：至少驗一格等於測試自己用 `RankModeDistribution` 加權 D、p 後的值。
6. notes/calc.md:15：rank 的 `Monsters`／`Metric` 說明與擊殺率式（標 [推測]：傷害取期望值不擲骰、怪回合內死亡不建模）。
7. 四段注入（只動 rank.go）：kill 式的 ⌈H/D⌉ 改成 ⌊H/D⌋（或 p 改成 0）→ 驗收印 FAIL TestRankKill → 副本還原 → PASS；貼進回覆。

## 規格疑慮
（執行方填）

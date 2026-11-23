# I2 calc rank：配點順序列舉＋去重快取＋平行（goals/stage16.md）

驗收: { go test -count=1 -v -run 'TestRankSkillPrios$' ./internal/calc 2>&1 | grep -q -- '--- PASS: TestRankSkillPrios' || { echo 'FAIL TestRankSkillPrios'; false; }; } && { go test -count=1 -v -run 'TestRank$|TestRankLoadouts$' ./internal/calc 2>&1 | grep -c -- '--- PASS' | grep -qx 2 || { echo 'FAIL TestRank-regress'; false; }; }
規模: 小
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔（含 fight.go、build.go）；不放寬或刪既有測試；不寫入遊戲目錄；不執行遊戲或 exe；註解不貼組語或反編譯碼；不把原版 bytes、從 exe／DBS 讀出的表或名稱寫進 repo
白名單: internal/calc/rank.go, internal/calc/rank_test.go, internal/calc/rank_skillprio_test.go, tools/calc/main.go, notes/calc.md:136
預期紅: TestRankSkillPrios
執行器: codex

## 背景
- I1 已讓 rank 支援 `Loadouts`（先讀 rank.go 現況）。目前一個範本只有一個 `SkillPrio`。v0 試算發現配點順序決定雙持吃不吃虧（notes/research/ranking-history.md:610 補跑段），要列舉。
- 一次 Fight 約 24 ms；許多配點順序在某等級給出相同技能向量，可去重。

## 要做的
1. 範本加選填 `SkillSet`（不重複技能 id，全排列當 SkillPrio）與 `SkillPrios`（明列多組）；三者（`SkillPrio`／`SkillSet`／`SkillPrios`）恰好給一個，否則報錯。武器組的技能用 `{main}`、`{off}` 佔位符（JSON 用 -1／-2）展開成該武器組兩手的武器技能；同一技能出現兩次時去掉後者。
2. 輸出加 `sprio` 欄（逗號分隔實際技能 id）。
3. 去重：同一 (race, class, level, 屬性結果, 技能結果含 Train, 武器組, 模式, 怪物, 難度) 只跑一次 Fight，結果共用；`# fights <實跑數> <總格數>` 印在 skipped 之前。
4. 平行：外層用 worker pool（`-j N`，預設 GOMAXPROCS）；輸出順序必須與 `-j 1` 逐 byte 相同。
5. `TestRankSkillPrios`（新檔，讀 build 原版檔，缺檔 `t.Skip`）：小範本（1 族、1 職業、1 武器組、SkillSet 3 項、2 等級）；測試自己展開 6 個排列、自己呼叫 PlanSkills＋Fight 獨立重算逐格相等（容差同 TestRank）；`-j 1` 與 `-j 4` 的 TSV 逐 byte 相同；實跑數＝測試自己算的相異輸入數。
6. notes/calc.md:136：欄位、佔位符、去重規則，並用 rank-h3-564.json 實測 `-j 1` 與預設 `-j` 的 real 秒數（寫量到的數字與機器負載）。
7. 四段注入：去重鍵漏掉 level → 驗收印 FAIL TestRankSkillPrios → 副本還原 → PASS；貼進回覆。

## 規格疑慮
寫在最終回覆；沒有就寫「無」。

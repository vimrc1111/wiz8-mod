# H2 calc rank：開局組合列舉＋近戰分數（goals/stage15.md）

驗收: { go test -count=1 -v -run 'TestRank$' ./internal/calc 2>&1 | grep -q -- '--- PASS: TestRank' || { echo 'FAIL TestRank'; false; }; } && { go run ./tools/calc rank -exe build/Wiz8.exe -dbs build/data-x/DATABASES notes/calc-qa/rank-savant.json > /dev/null || { echo 'FAIL rank-cli'; false; }; }
規模: 中
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔（含 fight.go、build.go、既有測試）；不放寬或刪既有測試；不寫入遊戲目錄；不執行遊戲或 exe；註解不貼組語或反編譯碼；不把原版 bytes、從 exe／DBS 讀出的表或名稱寫進 repo
白名單: internal/calc/rank.go, internal/calc/rank_test.go, tools/calc/main.go, notes/calc.md:136, notes/calc-qa/rank-savant.json
預期紅: TestRank
執行器: codex

## 背景
- `calc fight`（internal/calc/fight.go:1100 `(*FightWorld).Fight`，場景 `FightScenario` :257）算單角色一回合近戰期望；說明 notes/calc.md:15。
- `PlanAttrs`（build.go:53）依屬性優先序配屬性；`PlanSkills`（H1，build.go）依技能優先序配技能。
- 目標：列舉開局選擇，看各等級近戰傷害，找出不被支配的組合。

## 要做的
1. 範本 JSON（`DisallowUnknownFields`）：`Attrs`（要排列的屬性索引集合）、`Levels`（等級清單）、`SkillPrio`、`Train`（技能 id→額外裸值，選填，加在 PlanSkills 結果上、裸值上限 100）、`Main`、`Off`（可 null）、`Mode`、`OffMode`、`Monster`、`Difficulty`、`Races`／`Classes`（選填，預設全部 0..10／0..14）。
2. `Rank(w *FightWorld, t RankTemplate) ([]RankRow, error)`：種族 × 職業 × `Attrs` 全排列（排列當 Prio 前綴）。PlanAttrs 回 error（職業門檻付不起）或主手武器技能本職業不可用的組合**跳過並計數**，不當錯誤。每列對每個等級跑一次 Fight，取 `round.damage`。性別不列舉（growth.md:15 性別對屬性加減 [未取得]；只限制職業），文件註明。
3. `calc rank -exe -dbs [-top N] <template.json>`：TSV，欄位 `race class prio dmg_L<lv>…`；之後空行、`# pareto` 段（各等級 dmg 都不輸的列不被支配；同值全保留）；最後一行 `# skipped <n>`。`-top N` 依最後一個等級排序只印前 N 列（pareto 段照印全部）。
4. `notes/calc-qa/rank-savant.json`：Fighter 類可用的範本（武器與怪物 id 照 notes/calc-qa/fight-l28-06.json，Levels [1,10,28]，Attrs [0,4,5,3]，SkillPrio 依該武器技能、16、18、34）。
5. `notes/calc.md:136`：用法、欄位、假設（練技不建模只給 Train、初始技能 0、性別不列舉、calc fight 未建模清單沿用）、全列舉實測耗時（寫實際量到的秒數與機器狀態，不估）。
6. `TestRank`：小範本（Races 2 個、Classes 2 個、Attrs 2 個、Levels 2 個）；每一列用**獨立路徑**重算（測試自己組 FightScenario＋PlanSkills＋Fight，不呼叫 Rank 內部 helper）逐欄相等；列數＝合法組合數（測試自己數）；pareto 段與測試內 O(n²) 支配判定相同。測試讀 build/ 原版檔，缺檔 `t.Fatal`。
7. 四段注入：Rank 少列舉最後一個排列 → 驗收印 FAIL TestRank → 副本還原 → PASS；貼進回覆。

## 規格疑慮
寫在最終回覆；沒有就寫「無」。

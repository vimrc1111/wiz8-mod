# I6 calc fight／rank：Haste 與屬性覆寫（goals/stage16.md）

驗收: { go test -count=1 -v -run 'TestFightHasteAttrs$' ./internal/calc 2>&1 | grep -q -- '--- PASS: TestFightHasteAttrs' || { echo 'FAIL TestFightHasteAttrs'; false; }; } && { go test -count=1 ./internal/calc >/dev/null 2>&1 || { echo 'FAIL calc-package'; false; }; }
規模: 小
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔（含 build.go）；不放寬或刪既有測試；不寫入遊戲目錄；不執行遊戲或 exe；註解不貼組語或反編譯碼；不把原版 bytes、從 exe／DBS 讀出的表或名稱寫進 repo
白名單: internal/calc/fight.go, internal/calc/fight_haste_test.go, internal/calc/rank.go, tools/calc/main.go, notes/calc.md:15
預期紅: TestFightHasteAttrs
執行器: codex

## 背景
- Fight 的屬性目前只來自 `PlanAttrs(Race, Class, Level, Prio)`（fight.go 約 469 行），沒有覆寫入口。I4 要兩種輸入：MG 實盤的七項屬性原值（notes/calc-qa/mg-party-l25.md），以及「屬性 Train 到 100」。
- Haste：MG 實機施法後全隊 SPD 顯示 125（notes/re/haste-init.md 末段）。建模為 SPD＝max(SPD, 125)；持續回合不建模。
- rank 現況先讀 rank.go（I1–I3 已收）。

## 要做的
1. `FightScenario` 加選填 `Attrs *[7]int`（STR INT PIE VIT DEX SPD SEN，各 0..125，超出報錯）：非 nil 時取代 PlanAttrs 結果（Race／Class／Level 仍用於其他計算）。加 `Haste bool`：屬性決定後 SPD＝max(SPD, 125)。輸出格的屬性來源說明依實際來源標（PlanAttrs／Attrs／Haste），不要都寫 PlanAttrs。
2. calc fight 的 JSON 接受 `Attrs`、`Haste`（沿用 DisallowUnknownFields）。
3. `RankTemplate` 加選填 `AttrMin int`（0..125；PlanAttrs 之後每項＝max(值, AttrMin)，把結果當 `Attrs` 傳給 Fight）與 `Haste bool`（照傳）。去重鍵要用修改後的屬性並含 Haste。
4. `TestFightHasteAttrs`（新檔，讀 build 原版檔，缺檔 `t.Skip`）：
   (a) 一個情境以 `Attrs` 明給 SPD 97 加 `Haste`，與同情境 `Attrs` 明給 SPD 125 不加 Haste，全部輸出逐格相等；
   (b) `Attrs` 明給 SPD 130 加 Haste，SPD 格仍為 130；
   (c) `Attrs` 為 nil 時與現況（不給 Attrs）逐格相等；
   (d) rank 範本 `AttrMin: 100, Haste: true` 的每格 round.damage，等於測試自己用 PlanAttrs＋max＋Haste 算出 Attrs 再呼叫 Fight 的值（容差同 TestRank）；
   (e) 不合法 Attrs（-1、126）與 AttrMin 126 報錯。
5. notes/calc.md:15：兩個新欄位、Haste 建模假設與來源（notes/re/haste-init.md）。
6. 四段注入：Haste 改成無條件 SPD＝125 → 驗收印 FAIL TestFightHasteAttrs → 副本還原 → PASS；貼進回覆。

## 規格疑慮

# H1 PlanSkills：技能配點規劃（goals/stage15.md）

驗收: { go test -count=1 -v -run 'TestPlanSkills$' ./internal/calc 2>&1 | grep -q -- '--- PASS: TestPlanSkills' || { echo 'FAIL TestPlanSkills'; false; }; }
規模: 小
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔；不放寬或刪既有測試；不寫入遊戲目錄；不執行遊戲或 exe；註解不貼組語或反編譯碼；不把原版 bytes、從 exe／DBS 讀出的表寫進 repo（測試在執行時讀 build/Wiz8.exe）
白名單: internal/calc/build.go, internal/calc/skillplan_test.go
預期紅: TestPlanSkills
執行器: codex

## 背景
- `PlanAttrs`（internal/calc/build.go:53）依屬性優先序配屬性點；技能目前只能由情境手給（`FightScenario.Skills`）。最佳化列舉要依等級自動配技能。
- 規則（已查證，見 notes/re/growth.md）：創角技能預算 15（:13），每次升級 9（:16）；單項額度創角 5、升級 3，手動裸值上限 75，技能可用旗標必須非零（:18）；進階技能 s=34..40 對應屬性 i=s−34，裸屬性 ≥100 才啟用（:25）。
- `ExeTables.Avail[s][class]` 非零＝本職業可用（internal/calc/exe.go:18）。

## 要做的
1. 在 build.go 加 `PlanSkills(x *ExeTables, race, class, level int, attrPrio, skillPrio []int) ([41]int, error)`：
   - 逐級模擬：L1 用創角 15 點（每項 ≤5），之後每級 9 點（每項 ≤3）；每級依 skillPrio 順序貪婪配到該項額度或 75，配不完的點往下一項；全部配滿仍有餘點就丟棄（不跨級累積，與 growth.md:16 每級重設預算一致；若你讀材料發現會累積，寫進規格疑慮，不自行改規則）。
   - 進階技能在該級屬性（用 `PlanAttrs(x, race, class, lv, attrPrio)`）裸值 ≥100 時才可配。
   - 不可用技能（Avail 為 0）跳過；skillPrio 有重複或越界回 error。
   - 自動初始技能取 0，函式註解寫 `[未取得]` 與 growth.md:20 出處。
2. `skillplan_test.go` 的 `TestPlanSkills`：期望值**手算**寫死在表裡（不呼叫被測函式產生期望），至少涵蓋：L1 三項各 5；L2 再各 3；某項到 75 後點數流到下一項；不可用技能被跳過；進階技能 34 在 STR 到 100 之前為 0、之後開始累積（等級從 PlanAttrs 結果推導，不寫字面等級）；錯誤輸入回 error。測試讀 `build/Wiz8.exe`，缺檔時 `t.Fatal`（不 Skip）。
3. 四段注入：把升級每項額度 3 改 4 → 驗收印 FAIL TestPlanSkills → 副本還原 → PASS；貼進回覆。

## 規格疑慮
寫在最終回覆；沒有就寫「無」。

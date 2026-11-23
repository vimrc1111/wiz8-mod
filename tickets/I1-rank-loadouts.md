# I1 calc rank：武器組列舉＋裝備合法性（goals/stage16.md）

驗收: { go test -count=1 -v -run 'TestRankLoadouts$' ./internal/calc 2>&1 | grep -q -- '--- PASS: TestRankLoadouts' || { echo 'FAIL TestRankLoadouts'; false; }; } && { go test -count=1 -v -run 'TestRank$' ./internal/calc 2>&1 | grep -q -- '--- PASS: TestRank' || { echo 'FAIL TestRank'; false; }; }
規模: 小
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔（含 fight.go、build.go）；不放寬或刪既有測試；不寫入遊戲目錄；不執行遊戲或 exe；註解不貼組語或反編譯碼；不把原版 bytes、從 exe／DBS 讀出的表或名稱寫進 repo
白名單: internal/calc/rank.go, internal/calc/rank_test.go, internal/calc/rank_loadout_test.go, tools/calc/main.go, notes/calc.md:136
預期紅: TestRankLoadouts
執行器: codex

## 背景
- `calc rank`（internal/calc/rank.go，說明 notes/calc.md:136）目前一個範本只有一組 `Main`／`Off`，且不檢查裝備限制（只檢查主手技能）。
- 物品記錄欄位（notes/re/combat-2.md:85–:100）：+0x41 手部位元（bit2＝雙手武器、bit3＝可當副手，notes/research/ranking-history.md:116）、+0x46 武器技能、+0x76 u16 職業遮罩（bit＝職業 id）、+0x78 u32 種族遮罩（bit＝種族 id）、+0x7c u8 性別遮罩、+0x7d..+0x80 兩組（屬性 id u8、需求 u8）、+0x81..+0x84 兩組（技能 id u8、需求 u8）。需求 id 的「無」值要從資料判斷（例如 0xff 或需求 0），寫進 notes。

## 要做的
1. 範本加選填 `Loadouts`：`[{Name, Main, Off}]`（`Main` 必填、`Off` 可 null）。有 `Loadouts` 時不得同時給頂層 `Main`／`Off`（報錯）；沒有時沿用頂層（舊範本行為不變，`TestRank` 照過）。
2. 合法性：每個 種族×職業×武器組 先檢查兩手的職業遮罩、種族遮罩、兩手武器技能本職業可用；副手要 +0x41 bit3；主手 +0x41 bit2 時副手必須 null。不合法 → 跳過並**按原因計數**。性別遮罩不列舉，只把「此武器組限單一性別」寫進該列的 `note` 欄。
3. 屬性／技能需求：逐等級用 PlanAttrs／PlanSkills（含 Train）結果比；不足的等級該格輸出 `NA`，不跑 Fight。Pareto 把 NA 當比任何數都小。
4. TSV 欄位變成 `race class loadout prio dmg_L<lv>… note`；`# skipped` 改成每個原因一行 `# skipped <原因> <n>`。舊範本輸出的 loadout 欄印 `-`。
5. `TestRankLoadouts`（新檔 rank_loadout_test.go，讀 build 原版檔，缺檔 `t.Skip`）：用 ITEMS.DBS 執行期挑出符合條件的物品（例如種族遮罩只有一族的、雙手的、不可當副手的），測試**自己**依上述欄位判斷合法性並計數，與 Rank 的列與 skipped 計數逐項相等；至少涵蓋：種族不合、職業不合、副手不可、雙手配副手、需求不足出 NA 各一。不得把物品 id 或名稱寫死在測試或文件裡（執行期由欄位找）。
6. notes/calc.md:136 補欄位與規則。
7. 四段注入：把副手 bit3 檢查拿掉 → 驗收印 FAIL TestRankLoadouts → 副本還原 → PASS；貼進回覆。

## 規格疑慮
寫在最終回覆；沒有就寫「無」。

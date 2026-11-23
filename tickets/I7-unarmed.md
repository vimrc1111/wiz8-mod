# I7 calc fight／rank：空手（Monk 等）主手（goals/stage16.md）

驗收: { go test -count=1 -v -run 'TestFightUnarmed$' ./internal/calc 2>&1 | grep -q -- '--- PASS: TestFightUnarmed' || { echo 'FAIL TestFightUnarmed'; false; }; } && { go test -count=1 ./internal/calc ./tools/calc >/dev/null 2>&1 || { echo 'FAIL calc-package'; false; }; }
規模: 中
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔（含 build.go、n_*.go 節點檔）；不放寬或刪既有測試；不寫入遊戲目錄；不執行遊戲或 exe；註解不貼組語或反編譯碼；不把原版 bytes、從 exe／DBS 讀出的表或名稱寫進 repo
白名單: internal/calc/fight.go, internal/calc/fight_unarmed_test.go, internal/calc/rank.go, tools/calc/main.go, notes/calc.md:15
預期紅: TestFightUnarmed
執行器: codex

## 背景
- MG 2026-10-12：「Monk 空手可以先開票，跟先前計算的一起列，因爲這也是近戰。記得 Monk 的空手是可以 125 的」。I4（notes/research/ranking-history.md:682）把 Monk 空手列為未建模。
- 空手規則已在節點裡：`n_4ee220.go` 約 255 行 `id == -1` 分支（武器技能固定 14、骰取 exe 0x6172f0 表第 ⌊有效技能/11⌋ 項、主手低 16 位加 2、先攻 ⌊K/10⌋、模式遮罩 32／技能 14 有效值 ≥5 時 96）。規則來源 notes/re/formulas.md F26、notes/re/attack-modes.md:18、:32。
- 主技能加成：`EffectiveSkill`（build.go:146）主技能 raw＋max(1,raw/4)，上限 125。
- 現況：`fightChar` 要求 Main（fight.go 約 486 行「scenario needs Main」），主手 itemID0 固定填索引 0（fight.go 約 623 行）；rank 也要求 Main（rank.go 約 232 行）；tools/calc 兩處（main.go:172、:419）同樣要求。副手無時已有「空物品格」做法（fight.go 約 629、716、900 行），主手空手照同樣方式接。
- `RankModeDistribution(mask *uint16, unarmedSkill int)`（rank.go:593）已支援 mask＝nil 的空手。

## 要做的
1. `FightScenario.Main` 可為 nil＝主手空手。Off 非 nil 而 Main 為 nil 報錯（不支援空主手＋副手武器）。主手空手時餵給各節點的主手物品欄走空物品格（itemID0＝-1 等，照節點輸入域；不改節點檔）。輸出格來源說明標「Main 空手」。
2. calc fight JSON 接受不給 `Main`（或 `null`）；tools/calc 兩處檢查同步。
3. rank：`Loadouts` 的 `Main` 可省略＝空手（Off 也必須省略）；裝備合法性對空手不查物品，只查技能 14 對該職業可用（Avail，同 I1 的武器技能檢查），不可用＝NA 計入 `weapon-skill`；模式分布用 `RankModeDistribution(nil, 有效技能14)`。去重鍵要能分辨空手。
4. `TestFightUnarmed`（新檔，讀 build 原版檔，缺檔 `t.Skip`）。情境 Monk（class 6）L28、Haste、Attrs 全 100、技能 14 raw 100 與其他近戰技能 100、Monster 564：
   (a) `EffectiveSkill(x, 6, 14, 100)` 的值，等於 4ee220 輸出的空手技能格（從表推導，不寫字面 125；若 Monk 主技能不是 14 照實寫進規格疑慮，不改規則）；
   (b) 4ee220 主手骰格＝從 exe 表（測試自己讀 0x6172f0 第 ⌊有效技能/11⌋ 項）低 16 位加 2 的值；
   (c) 技能 14 raw 3（有效 <5）時模式遮罩格 32，raw 100 時 96；
   (d) round.damage 有限且 > 0；
   (e) rank 範本 Monk、空手 Loadout、AttrMin 100、Haste、Train {14:100,16:100,19:100,34:100}：每格 round.damage 等於測試自己組 Attrs／Skills 呼叫 Fight 的值（容差同 TestRank）；職業不可用技能 14 的列（挑一個 Avail[14]＝0 的職業，從表找）為 NA；
   (f) Main nil＋Off 非 nil 報錯。
5. notes/calc.md:15：空手輸入、空物品格接法、未建模清單移除 Monk 空手。
6. 四段注入（只動白名單內 fight.go）：空手主手的 itemID0 改回填 0（主手當成物品 0）→ 驗收印 FAIL TestFightUnarmed → 副本還原 → PASS；貼進回覆。

## 規格疑慮
（執行方填）

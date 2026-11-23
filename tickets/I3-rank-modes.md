# I3 calc rank：攻擊模式列舉（goals/stage16.md）

驗收: { go test -count=1 -v -run 'TestRankModes$' ./internal/calc 2>&1 | grep -q -- '--- PASS: TestRankModes' || { echo 'FAIL TestRankModes'; false; }; } && { go test -count=1 -v -run 'TestRank$|TestRankLoadouts$|TestRankSkillPrios$' ./internal/calc 2>&1 | grep -c -- '--- PASS' | grep -qx 3 || { echo 'FAIL TestRank-regress'; false; }; } && { go test -count=1 ./internal/calc >/dev/null 2>&1 || { echo 'FAIL calc-package'; false; }; }
規模: 小
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔（含 fight.go、build.go）；不放寬或刪既有測試；不寫入遊戲目錄；不執行遊戲或 exe；註解不貼組語或反編譯碼；不把原版 bytes、從 exe／DBS 讀出的表或名稱寫進 repo
白名單: internal/calc/rank.go, internal/calc/rank_test.go, internal/calc/rank_mode_test.go, internal/calc/rank_loadout_test.go, internal/calc/rank_skillprio_test.go, tools/calc/main.go, notes/calc.md:136
預期紅: TestRankModes
執行器: codex

## 背景
- I1、I2 已收（先讀 rank.go 現況）。目前 `Mode`／`OffMode` 是範本固定值。
- 規則（I0，`notes/re/attack-modes.md`「給下一張票的規則」）：玩家不選模式 0..8，只有兩個指令。
  - Attack：主手、副手都在每次揮擊開始依該手物品 +0x4e 遮罩自選；分布見 `notes/re/backstab.md:35`（B15：啟用模式排序後 P(mj)＝#{q∈0..8 : q mod k＝j}/9）。空手：{5}，空手技能 14 有效值 ≥5 時 {5,6}。
  - Berserk：只有具能力 0x14 的職業（attack-modes.md 所記）且主手可近戰；主手固定模式 3，不看遮罩；副手照 Attack 規則。

## 要做的
1. 範本加選填 `Commands`（`["attack","berserk"]` 子集，預設兩者）；每列逐一列舉合法指令，Berserk 不合法的職業跳過該指令（計入 skipped 理由）。範本頂層 `Mode`／`OffMode` 與 `Commands` 互斥；舊範本（無 Commands）沿用固定模式以保持 TestRank。
2. 該列 round.damage＝Σ_{i,j} Pmain(mi)·Poff(mj)·Fight(mi, mj)（Berserk 時 Pmain 集中在 3）；單持時只對主手加權。P 由一個可單測的函式算。
3. 輸出加 `command` 欄。去重鍵加入指令。
4. `TestRankModes`（新檔，讀 build 原版檔，缺檔 `t.Skip`）：(a) B15 分布函式對 k＝1..9 手算表逐項相等；(b) 小範本用執行期找到的一件多模式主手＋一件多模式副手，測試自己展開兩手分布、自己加權 Fight 重算，Attack／Berserk 各逐格相等（容差同 TestRank）；(c) 非 Berserk 職業的 berserk 列被跳過。
5. notes/calc.md:136 補規則。
6. 四段注入：分布改成等機率 1/k → 驗收印 FAIL TestRankModes → 副本還原 → PASS；貼進回覆。

## 規格疑慮
寫在最終回覆；沒有就寫「無」。

## 既有測試（I2 收貨教訓）
- 加 `command` 欄會改 TSV 表頭與欄位索引：rank_loadout_test.go、rank_skillprio_test.go 只准同步表頭字串與欄位索引（`command` 欄值可加等值斷言），不得刪除或放寬其他斷言。
- 交件前跑全包 `go test ./internal/calc`（已併入驗收）。

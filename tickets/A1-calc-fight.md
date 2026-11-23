# A1 實作：calc fight＝以 D4 節點組合的單角色近戰查詢（goals/stage12.md A1）

驗收: bash scripts/fight.test.sh && bash scripts/party.test.sh && bash scripts/calc-qa.test.sh && go test -count=1 ./internal/calc ./tools/calc
規模: 大（場景→記憶體欄位的世界建構＋節點 F 串接＋輸入登記）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 scripts/fight.test.sh、scripts/party.test.sh、scripts/calc-qa.test.sh、notes/calc-qa.md、notes/calc-qa/fight-q3.json、notes/calc-qa/fight-q4.json；不改既有 n_*.go 的 Emu 行為（TestEmu 必須照舊綠）；不寫入遊戲目錄；不執行遊戲；不放寬或跳過測試；筆記不貼反組譯碼或反編譯碼；不改 calc build／calc melee 既有輸出
白名單: tools/calc/main.go、internal/calc/fight.go、internal/calc/fight_test.go、notes/calc-fight-inputs.tsv、notes/calc.md:15、notes/calc-qa/fight-l28-01.json、notes/calc-qa/fight-l28-02.json、notes/calc-qa/fight-l28-03.json、notes/calc-qa/fight-l28-04.json、notes/calc-qa/fight-l28-05.json、notes/calc-qa/fight-l28-06.json、notes/calc-qa/fight-l28-07.json、notes/calc-qa/fight-l28-08.json、notes/calc-qa/fight-l28-09.json、notes/calc-qa/fight-l28-10.json、internal/calc/n_53d870.go、internal/calc/n_53e250.go、internal/calc/n_541c00.go、internal/calc/n_5420b0.go、internal/calc/n_543270.go
預期紅: fight
執行器: codex

## 背景
Wizardry 8 mod 的數值計算器。`internal/calc` 的 `Nodes` 是 381 個已 D4（與原版 exe 模擬器逐列相同）的公式節點：`Node{In,Out,F,Emu}`，輸入是具名欄位向量（名稱前綴對應記憶體物件＋偏移，例 `C.fa5`＝角色記錄 +0xfa5、`W0.4`＝主手武器、`M.HP`＝怪物；定義在 `re/emu/cases/<8 位址>.json` 的 `in` 與 `notes/re/iface/<位址>.tsv`）。`F` 呼叫真的子節點 Go 函式；`Emu` 把子節點回傳當輸入（`r<addr>_*`、`w<addr>.*`）。`calc f <8 位址>` 不給輸入時印出輸入名。
近戰相關節點：0x53d870 attack_swings、0x53e250 melee_attack_resolution、0x541c00 player_arhit、0x5420b0 weapon_damage、0x543270 attack_effects（即死在其槽 5）、0x4ee220 calc_attacks；子節點從各節點 F 的呼叫追。
既有 `calc build`（`internal/calc/build.go`）是階段 10 前的手寫公式，結果只當對照，不是答案來源。
材料：`build/Wiz8.exe`、`build/data-x/**/ITEMS.DBS`、MONSTERS.DBS（`internal/calc/load.go` 已能讀）。題目與已知答見 `notes/calc-qa.md` Q1–Q4；L28 排名見 `notes/research/ranking-history.md:104`。

## 要做
1. `calc fight -exe <Wiz8.exe> -dbs <dir> <scenario.json>`：
   - 場景格式＝`notes/calc-qa/fight-q4.json`（`calc build` 的欄位，加 `Monster`＝MONSTERS.DBS id、`Difficulty`＝0/1/2）。屬性用既有 `PlanAttrs`；技能經 `EffectiveSkill`。
   - 由場景建一份「世界」（角色記錄、兩手武器與物品記錄、怪物記錄、戰鬥全域），每個節點的輸入向量依其輸入名從世界讀出。子節點結果一律跑子節點 F，不吃 `r*`／`w*` 輸入。
   - 輸出 TSV `key\tvalue`（小數 6 位），至少：`round.swings`、`main.hit`、`main.dmg_per_hit`、`main.dice_groups_mean`、`main.instakill_trigger`、`main.instakill`（皆為每下命中的條件值）、`off.*`（有副手時同上）、`round.damage`、`round.instakill`（一回合內至少一次即死）。分布取節點輸出的完整分布計算期望，不抽樣。
   - `-nodes`：只印這次用到的節點 8 位址，一行一個。
2. `notes/calc-fight-inputs.tsv`（表頭 `node\tinput\tsource\tvalue\treason`）：用到的每個節點的每個輸入名恰一列；`source` ∈ {scenario, dbs, exe, node, fixed}；`fixed`＝場景無關的定值（站位、UI、計時、幾何等數值規劃範圍外的量，見 `decisions/2026-10-11-numeric-only-scope.md`），必填 value 與 reason。
3. F 模式缺口：上述路徑節點的 F 若 panic 或與 Emu 不同（`go test` 有現成 F/Emu 比對的就沿用），在白名單內的 n_*.go 修 F（不動 Emu），修不了的寫進 `notes/calc.md:15`「F 缺口」段：節點、條件、對 fight 輸出的影響。
4. `notes/calc-qa/fight-l28-01..10.json`：L28 排名筆記「前十名（對 Dark Savant）」表逐列建檔（Dark Savant＝MONSTERS.DBS id 564；物品 id 由 ITEMS.DBS 名稱查；技能配置照筆記「設定」段）。
5. `notes/calc.md:15`：用法、輸出鍵、世界建構規則、F 缺口、未建模項；另附前十名 `round.damage` 對照筆記原數字的表（只列數字，不下結論）。

## 已知限制（不用處理）
- 遠程、法術、承傷不在本票（A2／A3）。
- 練技成長、負重、裝備屬性加成照 `calc build` 現況可以不建模，但要在 calc-fight.md 列出。

## 交付回覆
用到的節點；fixed 輸入數與最大宗理由；F 缺口；前十名對照表；規格疑慮段，沒有就寫「無」。

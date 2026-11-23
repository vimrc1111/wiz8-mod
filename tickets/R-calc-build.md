# R 實作：calc build（28 級戰士的配點、手部快取與每回合期望傷害）

驗收: out=$(WIZ8_EXE=$PWD/build/Wiz8.exe WIZ8_DBS=$PWD/build/dbs/DATABASES go test -count=1 -v ./internal/calc/ 2>&1); echo "$out"; echo "$out" | grep -q -- '--- PASS: TestBuildRealData' && echo "$out" | grep -q -- '--- PASS: TestExeTables' && test -z "$(gofmt -l .)" && go vet ./... && go build -o /dev/null ./tools/calc && go test -count=1 ./...
規模: 中（codex gpt-6.1-sol；Go 標準庫）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 internal/calc/build_test.go、calc_test.go 與白名單外任何檔；不加第三方依賴；不寫入遊戲目錄；不執行遊戲；不放寬或跳過測試；不把原版資料內容抄進 repo（測試只放既有測試已有的少數值）
白名單: internal/calc/build.go、internal/calc/round.go、internal/calc/round_test.go、tools/calc/main.go
預期紅: TestPlanAttrs、TestEffectiveSkill、TestBuildDualL28、TestBuildBerserkL28、TestBuildDualL5、TestBuildRealData
執行器: codex

## 背景
MG 2026-10-08：「等反組譯解析都回來後，其實就可以計算一個 28 級的戰士有多少升級點跟技能點，然後選什麼種族會拿到更多可以分配的點數，那個時候就知道拿這些武器的強度」。`goals/stage8.md` 8.2。
公式來源：`notes/re/growth.md`（創角與升級配點）、`notes/re/formulas.md` F06–F14、F28、`notes/re/combat-2.md` C04–C06、`notes/re/abilities.md`（主技能加成）、`notes/re/backstab.md`（副手模式不繼承）。筆記與測試衝突時以 `build_test.go` 為準，在規格疑慮列出。
材料：`build/Wiz8.exe`、`build/dbs/DATABASES/`（gitignored，原版，只讀）。已有：`LoadExe`（`internal/calc/exe.go`）、`AttackValues`、`MeleeRound`、`LoadItems`。

## 做法
- 型別與簽章照 `internal/calc/build.go` 骨架，填實作。
- `PlanAttrs(x, race, class, level, prio)`：
  - 起點：`Apre[i]=max(Race[r][i], ClassMin[p][i])`，`D=Σmax(0,ClassMin−Race)`，`B=60−D`；B<0 回 error。
  - 創角：每項上限 `K=min(B,max(3,⌈B/3⌉))`。
  - 升級：2..level 每級 6 點，每項 ≤3。
  - 裸值 ≤100。
  - 分配順序：prio 列的屬性先，其餘依索引序補上；每一步給 `min(上限, 100−目前值, 剩餘)`。
- `EffectiveSkill`：`s==MainSkill[class]` 時為 `raw+max(1,⌊raw/4⌋)`，否則 raw；夾 0..125。
- `Hands`：照 0x4ee220 的手部重建，雙持＝兩手都有武器。
  - 先求三個技能值，全部用 EffectiveSkill：`W`＝本手武器技能，`B`＝Close Combat 16，`D`＝Dual Weapons 18。
  - `u=⌊2(B+2W)/3⌋`，`d=20`；雙持時 `u+=⌊W_other/2⌋+D`，`d=35`（F07 雙持旗標 C+0x12b5，0x4ee44e；原寫 d=25、漏 D，grok 稽核 cXan F1 抓到，2026-10-08 更正）；`K=⌊10u/d⌋`。
  - 修正 M：雙持時主手 `−10−tz((100−D)/4)`、副手 `−20−tz((100−D)/4)`；單持為 0。
  - `P=PMult[class]·Level`。
  - 負重 E、e 都取 0（未建模，寫進結果說明）。
  - `a=tz((⌊(DEX+SPD)/2⌋+M+P+K)/3)`。
  - `v=tz((SPD+10·SpeedMod+M+P+K)/3)`：SpeedMod 是本手物品 +0x48 的值。
  - `BARraw=60+tz((⌊DEX/2⌋+2K+P+M)/3)`。
  - HitMod：
    - 物品 HitMod；
    - F13 力量項（副手在 STR>50 時 c=2）；
    - `tz((DEX−50)/10)`；
    - F14 感知項；
    - STR 裸值 ≥100 時加 `1+⌊Eff(34)/20⌋`（Power Strike）。
  - DmgPct：只算 F13 的力量項。F28 的特殊物品 10×Z 分支未建模，寫進說明。
- `MeleeBuild`：
  - 每手呼叫 `AttackValues`：BARraw、HitMod、Mode、WeaponSkill、SPD 帶入；`Delta=DeltaActual=(INT+DEX+SPD+SEN)−Attr4Sum`；DefHit、DefPen 取自 Target；Difficulty=1；其餘欄 0／false。
  - 再呼叫 `MeleeRound`：Dice、`G=1+[mode==3]`、BonusPct=DmgPct、DA、A、V、OffHand；`SlotSum=Item.Effects[5]`、Skill=0、LSLevel=0。
  - 主手用 mode，副手用 offMode。
- `tools/calc/main.go`：新增 `calc build -exe <Wiz8.exe> -dbs <dir> <scenario.json>`。
  - JSON 欄位：`Race, Class, Level, Prio[], Skills{id:raw}, Main{Item,Skill}, Off{Item,Skill}|null, Mode, OffMode, Target{DefHit,DefPen,Attr4Sum,DA}`；Item 是 ITEMS.DBS 零起算 id。
  - 輸出純文字表：最終屬性、兩手 HandCache、ARhit／ARpen、揮擊數、期望傷害、合計，以及不支援項。
  - usage 寫明不支援：負重、裝備屬性加成、特殊物品傷害分支、練技成長、背刺、Kill 抗性。
- 自寫 `round_test.go` 可加 CLI 解析的小案例；不改 `build_test.go`。

## 報告
`.claude/pm-diffs/R-calc-build.out` 由 impl-run 收；最後回覆列「規格疑慮」（沒有就寫無）。

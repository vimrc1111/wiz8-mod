# Q 實作：internal/calc＋tools/calc（近戰與直接攻擊法術的期望傷害計算器）

驗收: out=$(WIZ8_DBS=$PWD/build/dbs/DATABASES go test -count=1 -v ./internal/calc/ 2>&1); echo "$out"; echo "$out" | grep -q -- '--- PASS: TestRealData' && test -z "$(gofmt -l .)" && go vet ./... && go build -o /dev/null ./tools/calc && go test -count=1 ./...
規模: 中（codex gpt-6.1-sol；Go 標準庫）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 internal/calc/calc_test.go 與白名單外任何檔；不加第三方依賴；不寫入遊戲目錄；不執行遊戲；不放寬或跳過測試；不把原版資料內容抄進 repo（測試只放 calc_test.go 已有的少數值）
白名單: internal/calc/calc.go、internal/calc/load.go、internal/calc/round.go、internal/calc/round_test.go、tools/calc/main.go
預期紅: TestScale、TestDice、TestSwingProb、TestPhysDamage、TestKillThreshold、TestRoundCounts、TestSpell、TestResist、TestLoadSpellsSynthetic、TestRealData
執行器: codex

## 背景
MG 2026-10-08：「在合理的情況下，我們是否是可以單靠公式就算出以等級 28 破關，什麼職業的潛在傷害力最高」→「開，先做近戰跟法術」。遠程不做（`notes/re/combat-2.md` C10 未取得）。
公式來源：`notes/re/formulas.md`（F04、F07–F09、F20–F24）、`formulas-2.md`（法術強度、PL 縮放、:19 技能即死）、`combat-2.md`（C02、C04–C09、C11–C14、C20–C22）、`abilities.md`（物理減傷）。筆記與測試衝突時以 `calc_test.go` 為準並在規格疑慮列出。
材料：`build/dbs/DATABASES/`（gitignored，原版 DBS，只讀；驗收用 WIZ8_DBS 指向它）。

## 做法
- 骨架已在 `internal/calc/calc.go`、`tools/calc/main.go`（型別照骨架，函式填實作；可拆到白名單內其他檔）。
- 機率一律均勻近似：`R(n)` 視為 0..n−1 等機率，`R(0)=0`。分布型別 `type Dist map[int]float64`（值→機率），`Mean()`；用列舉／卷積算精確期望，不用 Monte Carlo。
- `calc_test.go` 用到的 API（簽章照測試推導）：`Scale`、`Dice{Z int16; N, S uint8}`、`DiceDist`、`Groups`、`HitProb`、`GateProb`、`PhysDamage(d Dist, bonusPct, DA int)`、`KillThreshold(slotSum, skill int) float64`（slotSum 先 mod 256）、`Attacks(a) (main, off int)`、`SwingCap(v)`、`LSExtraSwings(L) float64`、`SpellStrength(L, PL, manaPerPL, level int, type7 bool)`、`PowerCastB(skill int, on bool)`、`SpellDamage(d Dice, PL, b int) Dist`、`Resist(d Dist, Rj, Lt, S int) Dist`、`LoadSpells`／`LoadItems`／`LoadMonsters`。
  - `Spell{Name string; Dice Dice; ManaPerPL, Level int32; Realm uint32; Range, Type int32}`（C11 磁碟位移）。
  - `Item{Name string; Dice Dice; Effects [16]uint8; SpeedMod, HitMod int8; ModeMask, ClassMask uint16}`（C20；名稱 +0 UTF-16LE 到 NUL）。
  - `Monster{Name string; HP Dice; Attr [5]uint8; AC int8; Resist [6]uint8; Level uint8; Absorb uint8}`（C21；Absorb＝+0xcf，F23）。
  - 載入檢查檔長＝檔頭筆數×stride＋檔頭，不符回 error。
- `round.go`：組合成「一回合期望」：
  - 近戰 `MeleeRound(in MeleeInput) MeleeResult`：輸入已算好的 ARhit A、ARpen P（C04–C06 的組合另提供 `AttackValues` 函式，輸入照 C04／C06 列的欄位）、a、v、武器骰、g（含 mode 3）、傷害加成%、目標 DA、即死 slotSum／技能、LS 職業等級（非武士填 0）。輸出：每回合期望揮擊數、期望有效揮擊數、期望傷害、每揮即死門檻期望（**只到門檻，不算抗性後成功率**：F25 未取得，欄名寫明）。失手換目標（C08）不支援，結果欄註明。
  - 法術 `SpellCast(sp Spell, L, PL, b int, target Monster)`：單目標期望傷害、每 PL 成本×PL、強度 S（C12 依 id 扣 d）。
  - 自寫 `round_test.go`：每個組合至少一個手算案例，註解寫推導。
- `tools/calc/main.go`：`calc melee <scenario.json>`、`calc spell -dbs <dir> -id <n> -pl <n> -L <n> -pc <skill> -monster <id>`；輸出純文字表。scenario 欄位與 MeleeInput 同名。
- 不支援的寫明：遠程、AoE 名單、施法成功率、Kill 抗性、狀態效果、F01 精確 RNG。

## 報告
`.claude/pm-diffs/Q-calc.out` 由 impl-run 收；最後回覆列「規格疑慮」（沒有就寫無）。

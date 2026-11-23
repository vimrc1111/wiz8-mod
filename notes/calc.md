# calc 工具說明

> calc fight／rank／party 與掉落查表的用法與限制。題庫在 notes/calc-qa.md（機檢讀取，不併）。
> 2026-11-01 由 4 份原檔逐行併入（標題降一級、內文不改）；原檔見 tag `wiz8-notes-pre-merge`（85393087）。

目錄：
- 第 15 行起：calc fight：單角色近戰一回合（A1）（原 `notes/calc-fight.md`）
- 第 136 行起：calc rank：開局列舉與近戰分數（原 `notes/calc-rank.md`）
- 第 229 行起：calc party：創角 Pareto 前緣（原 `notes/calc-party.md`）
- 第 263 行起：I12 掉落查表（原 `notes/loot-where.md`）


---
<!-- 原檔 notes/calc-fight.md -->
## calc fight：單角色近戰一回合（A1）

以 D4 節點的 F 組合，對一隻怪物算一回合近戰的期望值。分布用完整列舉（CRT `rand()` 權重），不抽樣。

### 用法

```
calc fight -exe build/Wiz8.exe -dbs build/data-x/DATABASES [-nodes] <scenario.json>
```

- 場景格式同 `notes/calc-qa/fight-q4.json`：`calc build` 欄位（Race、Class、Level、Prio、Skills、Main、Off、Mode、OffMode），加 `Monster`（MONSTERS.DBS id）與 `Difficulty`（0／1／2）。不認得的欄位會報錯。
- `Main` 省略或 null 表示主手空手，空手 Mode／OffMode 0（含省略）轉為拳擊模式 6；模式 5 為另一空手模式。兩手皆空時 004ee220 開啟第二手（副手拳），Fight 依第二手 enabled 出手並輸出 `off.*`（I11）。`Off` 必須省略或 null；空主手＋副手武器會報錯。技能由 `Skills[14]` 經 `EffectiveSkill` 決定，包含職業主技能加成。
- rank 的 `Loadouts` 同樣可省略 `Main` 與 `Off`；空手只查技能 14 的職業可用性，不查物品需求。不可用的列為 NA，計入 `weapon-skill`；`{main}` 技能占位符展開為 14，Attack 使用有效技能 14 的空手模式分布。
- `-nodes`：只印用到的節點位址，一行一個。
- `Attrs`（選填）：七項原值，順序 STR、INT、PIE、VIT、DEX、SPD、SEN，各為 0..125；非 null 時取代 `PlanAttrs`，Race／Class／Level 仍供其他計算使用。省略或 null 沿用配點。
- `Haste`（選填，預設 false）：屬性決定後令 SPD＝max(SPD,125)。依 [MG 實機對照](re/haste-init.md) 的全隊施法後 SPD 125 建模；這是情境假設，持續回合不建模。
- rank 範本另有 `AttrMin`（選填，預設 0，範圍 0..125）：`PlanAttrs` 後逐項取 max(值,AttrMin)，再套用 `Haste`，用於裝備需求與 Fight。技能配點維持 `PlanSkills` 規則；去重使用修改後屬性與 Haste。
- rank 範本 `Monsters`（選填）：非空時取代單一 `Monster`，此時 `Monster` 必須為 0；每個 id 必須在 MONSTERS.DBS 範圍內，且不可重複。省略或空清單沿用 `Monster`。
- rank 範本 `Metric`（選填）：省略、`""` 或 `"damage"` 為每回合傷害期望，`"kill"` 為每回合擊殺數；其他值報錯。每格先對每隻怪按主手 × 副手模式分布，分別加權 `D＝Σ w·round.damage` 與 `p＝Σ w·round.instakill`，再算指標；多怪格值為各怪指標的算術平均。TSV 格式不變，`Damage`／`dmg_L*` 欄承載選定指標，去重鍵包含怪物清單與指標。
- [推測] 擊殺率模型：怪物 HP 取 MONSTERS.DBS 模板 +0xd6 的骰期望 `H＝Z＋N·(S+1)/2`。`D>0` 時，`H>0` 取 `n＝⌈H/D⌉`，`H≤0` 取 `n＝1`；期望回合數 `E＝Σ(r=1..n) (1−p)^(r−1)`，擊殺率為 `1/E`。`D≤0` 時，`p>0` 回傳 `p`，`p＝0` 回傳 0。這是假設每回合重新混合模式、即死率固定的近似：傷害取期望值不擲骰，怪物回合內死亡不建模，後續揮擊仍計入 D 與 p。
- 輸入登記：`notes/calc-fight-inputs.tsv`，由 `internal/calc/fight_test.go` 的 TestFightInputsTSV 產生與比對（重產：`UPDATE=1 go test -run TestFightInputsTSV ./internal/calc`）。

### 輸出鍵（`key<TAB>value`，小數 6 位）

| 鍵 | 意義 |
|---|---|
| `round.swings` | 一回合揮擊數期望（兩手合計，含 53d870 的回合數分布） |
| `<hand>.hit` | 每揮命中機率（hand＝main／off；off 只在第二手 enabled 時輸出：有副手武器或空手） |
| `<hand>.dmg_per_hit` | 每下命中的傷害期望（5420b0 回傳） |
| `<hand>.dice_groups_mean` | 每下命中的骰組數期望 |
| `<hand>.instakill_trigger` | 每下命中觸發即死（槽 5 擲過、進入 551ba0）的期望次數 |
| `<hand>.instakill` | 每下命中的即死機率（stats_50 增量） |
| `round.damage` | 一回合傷害期望 |
| `round.instakill` | 一回合內至少一次即死 |

每手的值依揮擊數加權（53d870 各結果的機率 × 揮擊數）。

### 世界建構規則

- 角色：`PlanAttrs` 依 Prio 配點或由 `Attrs` 覆寫，之後套用 `Haste`；輸入格的來源說明依實際使用標記 PlanAttrs／Attrs／Haste。技能只取場景的 `Skills`，經 `EffectiveSkill`，沒列的為 0。場景武器的 `Skill` 欄不讀，有武器時技能由物品 +0x46 經節點決定，空手固定技能 14。
- 手部：0x4ee220 迭代到不動點（最多 8 次）。meleeExpert＝STR ≥ 100。
- 物品：有武器的主副手讀 ITEMS.DBS 記錄。主手空手時 004ee220 的 itemID0＝−1，物品欄值為 0；0053d870 的 item6 與 005420b0 的 it6＝2，依子節點值域指向既有空物品格（骰字 0、技能欄 255）。來源標記「Main 空手」。沒有副手時沿用空物品格接法。
- 空手：004ee220 決定技能 14、有效技能對應骰表、主手骰低 16 位加 2、先攻與模式遮罩；不另複製節點規則。Monk 主技能加成直接讀原版職業表。
- 怪物：讀 MONSTERS.DBS 的記錄；`Difficulty` 由場景決定。
- 子節點一律跑 F；`r*`／`w*`（Emu 重播格）都是 fixed，理由寫「Emu 重播格，F 路徑不讀」。
- 不在數值範圍內的輸入（隊形格、正面相鄰、執行期姿勢）一律 fixed，並附理由。
- 543270 沒有建模的格子，取 calc-qa Q4 情境列（`re/emu/cases/00543270.json` boundary 81）。
- 每一下的流程照 53e250 的順序，在 `fight.go` 裡串接（見「未建模」）：
  - 先算命中，再算 fumble、部位；
  - 5420b0 回傳 > 0 才進 543270；
  - 槽 5 的 chance 是物品 +0x55，加上 Critical Strike 的 ⌊K/25⌋，餘數以 R(25) 進位。
- 輸入列：見下表。

| 節點 | 列數 |
|---|---|
| 004ee220 | 119 |
| 0053d870 | 96 |
| 00541c00 | 73 |
| 00541ec0 | 80 |
| 005420b0 | 100 |
| 00543270 | 991 |

共 1459 列，fixed 1191 列。

fixed 的主要理由：

| 理由 | 列數 |
|---|---|
| 543270 Q4 情境列 | 956 |
| 執行期戰鬥狀態取開戰初值 | 45 |
| 目標是怪物，角色目標欄不讀 | 31 |
| 空物品格 | 18 |
| 幾何（MG 2026-10-11） | 25 |

### F 缺口與列舉調整

只改各節點 F 的 real 路徑，Emu 不動（TestEmu 照舊綠）：

| 節點 | 條件 | 處理 | 對輸出的影響 |
|---|---|---|---|
| 0x53d870 | 沒有副手時，item7＝−1 會 panic（「item index outside four supplied records」） | 改用 item7＝2（空物品格），子節點 519ac0 的值域是 0..3 | 單持時揮擊數照常；空格是否等同原版的 −1，未取得 |
| 0x5420b0 | 逐顆擲骰，路徑數是 s^n | 列舉器有 `DiceTotal` 時，整組骰一次卷積分支 | 分布相同（TestFightDiceTotal 比對逐顆的動差） |
| 0x543270 | 數量骰 15d2：Q4 一次呼叫超過 2.5 分鐘未完 | 數量骰改走 `DiceTotal`，再以零骰的骰字送回 551a20 | 分布相同 |
| 0x543270 | `R(100) < c` 與 `R(100) < 50` 各分 100 支 | 列舉器有 `Below` 時改成兩支（CRT 權重加總） | 分布相同；Q4 的 `calc fight` 全程約 1.2 秒 |

### 未建模

- 0x53e250 沒有直接呼叫，在 `fight.go` 照順序串接各子節點。T74（主手連動副槽）≠ −1 時報錯。
- 戰鬥技能 0x11 報「未建模」錯。
- 每回合只走 ac < 2 的路徑；0x545a00 那條沒有接。
- fumble 清單只取 n＝1。
- 特效：槽 5 以外都是 0；chance6 的 0x13；0x1f8 回血。
- 怪物：回合內死亡（後續揮擊照打）；執行期修正值；T94。
- Lightning Strikes 旗標（Samurai feature 0x10）。
- 背後與側面幾何：固定正面相鄰。
- 負重、裝備屬性加成、練技成長。
- 場景的 `Skill` 欄不讀。

### 前十名 round.damage 對照（Dark Savant id 564，Difficulty 1，Mode 0）

| # | 檔案 | 組合 | calc fight | 筆記 |
|---|---|---|---|---|
| 1 | fight-l28-01 | Rogue Faerie 雙持 LS＋Sprite's Dagger | 336.999549 | 424 |
| 2 | fight-l28-02 | Rogue Human 單持 LS | 302.634329 | 368 |
| 3 | fight-l28-03 | Ninja Faerie 雙持 Cane of Corpus＋Rod of Sprites | 275.321897 | 321 |
| 4 | fight-l28-04 | Samurai Faerie 雙持 LS＋Rod of Sprites | 269.213805 | 318 |
| 5 | fight-l28-05 | Lord Human 雙持 LS＋Diamond Eyes | 276.360194 | 312 |
| 6 | fight-l28-06 | Fighter Human 雙持 LS＋Diamond Eyes | 268.051533 | 308 |
| 7 | fight-l28-07 | Ranger Human 雙持 LS＋Diamond Eyes | 263.390479 | 305 |
| 8 | fight-l28-08 | Valkyrie Human 雙持 LS＋Diamond Eyes | 257.851305 | 301 |
| 9 | fight-l28-09 | Ninja Faerie 單持 Cane of Corpus | 251.095193 | 292 |
| 10 | fight-l28-10 | Samurai Human 單持 LS | 247.979894 | 277（含 Lightning Strikes） |

物品 id：LS（`*Light* *Sword*`）62、Sprite's Dagger 430、Cane of Corpus 29、Rod of Sprites 70、Diamond Eyes 25。

技能照筆記「設定」段：
- 單持：武器 100、CC 100、PS 58。
- 雙持「高」：再加副手技能 100、DW 100。

---
<!-- 原檔 notes/calc-rank.md -->
## calc rank：開局列舉與近戰分數

```sh
go run ./tools/calc rank -exe build/Wiz8.exe -dbs build/data-x/DATABASES notes/calc-qa/rank-savant.json
go run ./tools/calc rank -exe build/Wiz8.exe -dbs build/data-x/DATABASES -top 10 notes/calc-qa/rank-savant.json
```

exe／DBS 僅讀取，不執行遊戲。JSON 使用 `DisallowUnknownFields`，拒絕未知欄位、null 範本及多個 JSON 值。

| 欄位 | 意義 |
| --- | --- |
| `Attrs` | 無重複的屬性索引集合（0..6：STR INT PIE VIT DEX SPD SEN）；全排列作為 PlanAttrs 的 Prio 前綴，其餘屬性由 PlanAttrs 補上。空集合有一個空排列。 |
| `Levels` | 非空的正整數等級清單；依輸入順序輸出。 |
| `SkillPrio` | 單一無重複技能順序；與 SkillSet／SkillPrios 恰好給一個，非 null；空清單表示不配技能。 |
| `SkillSet` | 無重複技能集合，全排列當 SkillPrio，順序按輸入遞迴交換；空集合有一個空排列。 |
| `SkillPrios` | 明列多組無重複技能順序，按輸入順序列舉；空清單表示沒有候選。 |
| `Train` | 選填，技能 id → 額外裸值（0..100）；在每個等級 PlanSkills 配完後加上，總裸值封頂 100，不回饋配點過程。 |
| `Loadouts` | 選填清單 `[{Name,Main,Off}]`，Main 必填、Off 可 null；不得同時給頂層 Main／Off（包含 JSON null）。Name 不能含 TAB 或換行；空清單表示不列舉。 |
| `Main`、`Off` | `{Item,Skill}`；Main 必填，Off 可 null。Item 為零起算 id。與 Fight 一致，實際武器技能讀物品資料，Skill 僅為資訊欄。 |
| `Commands` | 選填 `"attack"`／`"berserk"` 的無重複子集；按輸入順序列舉。`null` 使用兩者，`[]` 不列舉；省略時保留舊固定模式行為。與頂層 Mode／OffMode 互斥，包含顯式 0。 |
| `Mode`、`OffMode` | 舊範本各手固定模式 0..8，未給時為 0。 |
| `Monster`、`Difficulty` | 怪物 id、Fight 難度 0..3。 |
| `Races`、`Classes` | 選填，預設全 0..10／0..14；指定清單不可重複。空清單表示不列舉。 |

技能 id 使用 0..40，三種來源皆可用 `{main}`／`{off}` 佔位符（JSON 寫 `-1`／`-2`）。每個武器組以物品資料的實際兩手武器技能展開，展開後同技能只留第一次；`{off}` 需要副手。SkillSet 的唯一性是在展開前檢查，展開後相等的候選列仍保留。

列舉順序為種族、職業、武器組（Loadouts 輸入順序）、指令（Commands 輸入順序）、技能順序、屬性排列（按 Attrs 輸入順序遞迴交換）。未給 Loadouts 時沿用頂層 Main／Off，loadout 欄印 `-`。每列以單一職業由 L1 配點到指定等級，合法等級執行 Fight，分數取 `round.damage`。

裝備欄位從 ITEMS.DBS 執行期讀取：每筆 0x10d bytes；+0x76 u16 職業遮罩、+0x78 u32 種族遮罩，各以 id 對應 bit；兩手 +0x46 武器技能皆須本職業可用；主手 +0x41 bit2 表示雙手武器，必須沒有副手；副手 +0x41 bit3 必須為 1。+0x7c 性別遮罩不列舉，兩手遮罩交集限單一性別時，note 印「此武器組限單一性別」。

跳過按種族 × 職業 × 武器組計數一次，不乘屬性排列或等級；只記第一個失敗原因，順序為 `creation`（PlanAttrs 付不起職業門檻）、主手職業／種族／技能、副手職業／種族／技能、雙手限制、副手位元。遮罩／技能原因分別是 `class-mask`、`race-mask`、`weapon-skill`，手部原因是 `two-handed`、`offhand`。其他 Fight 錯誤回報並中止。

+0x7d..0x80 為兩組屬性 id u8／需求 u8，+0x81..0x84 為兩組技能 id u8／需求 u8。原版資料掃描確認空需求常用 id=0xff、需求=0，也存在 id=0、需求=0；id=0xff 或需求=0 均視為無需求。其餘逐等級比對 PlanAttrs 裸屬性與 PlanSkills 加 Train 後的裸技能（封頂 100），兩手任一需求不足，該格輸出 `NA`，不執行 Fight；有效範圍外的需求 id 亦不能滿足。需求不足不增加 skipped，列仍保留。

TSV 欄為 `race class loadout command prio sprio dmg_L<lv>… note`（實際以 TAB 分隔）；prio 是逗號分隔屬性索引，sprio 是展開並去重後的實際技能 id。Fight 內部平行加總有末位誤差；rank 傷害固定到 13 位有效數字（符合 1e-12 相對容差），API／排序／Pareto 共用這個值，TSV 仍用 `%.17g` 輸出。主段之後空行、`# pareto`、完整前緣列，接著 `# fights <實跑數> <總格數>`，最後每個非零原因一行 `# skipped <原因> <n>`（原因字典序）。前緣不重複表頭。支配必須各等級分數 ≥ 且至少一欄 >；完全同值全保留。NA 在 API 的 Damage 中以負無限表示，排序及 Pareto 視為比任何傷害數值小。`-top N` 僅將主段按清單最後一個等級分數降序取前 N 列，同分保留原序；0 印全部。Pareto 段永遠用全部列。

練技行為不建模，只能透過 Train 給定情境。自動初始技能未取得，裸值起始 0。性別不列舉：growth.md 的性別屬性加減未取得，性別只涉及職業限制；本工具列出職業的數值候選，不判定性別資格（與 stage15 的早期文字不同，以 H2 票為準）。裝備屬性加成、負重、背刺、Lightning Strikes、combatSkill 0x11 等限制完整沿用 [calc fight 未建模清單](calc-fight.md#未建模)。裝備合法性與需求依上列規則檢查。

交付範本使用 fight-l28-06 的武器與怪物 id，Levels `[1,10,28]`、Attrs `[0,4,5,3]`、SkillPrio `[0,16,18,34]`；限定 `Classes:[0]`（Fighter），種族全部。沒有 Train，副手技能若未列在 SkillPrio 則維持 0。

### 全列舉實測

2026-10-12，AMD Ryzen 7 5800X3D，16 logical CPUs，31 GiB RAM；共享主機，量測前 load average 14.07／20.27／16.20，量測後 17.75／19.87／16.61，另有小範本測試及編譯工作。原預設 Go cache 不可寫，執行時改用 `/tmp/h2-go-cache`。

```sh
GOCACHE=/tmp/h2-go-cache /usr/bin/time -p go run ./tools/calc rank -exe build/Wiz8.exe -dbs build/data-x/DATABASES notes/calc-qa/rank-savant.json > /tmp/h2-rank-full.tsv
```

交付範本完整列舉 11 × 1 × 24 = 264 組，792 次 Fight，skipped 0：real **3.78 秒**、user 3.80 秒、sys 0.47 秒；含 go run，cache 已暖，不含把範本縮成抽樣。

另實測省略 Classes 的全種族／全職業嘗試（3960 個候選組合），real **47.21 秒**、user 117.44 秒、sys 12.95 秒（含冷 cache 編譯）；到 race 0、class 7、prio `[4,5,3,0]`、L1 時，既有 Fight 回報 `weapon damage: hb0 outside the replay domain`。該次未完成，47.21 秒不是全部 3960 組的完成耗時。白名單不允許修改戰鬥核心，所以範本限定 Fighter，Rank 對此錯誤仍回報失敗。

`TestRank` 讀 build 的原版檔，缺檔 SKIP（同 fight_test，CI 副本沒有原版）；驗收行 grep `--- PASS`，SKIP 不會被當成通過。測試自行列舉與組 FightScenario、PlanSkills、Fight 重算；組合欄位精確比對，傷害採 1e-12 × max(1, |expected|) 容差，因 Fight 的浮點加總順序與逐項重算不同（781152d 起 fightExpect 依前兩層分支排序加總，同輸入重跑逐 byte 相同，`TestFightDeterministic`）。I2 起 Rank 將傷害固定到 13 位有效數字；Pareto 對固定後分數作精確支配比較。TSV 全段／top 與 O(n²) 前緣判定均有檢查。

`TestRankLoadouts` 從原版 ITEMS.DBS 欄位動態挑選武器，獨立解讀遮罩、手部位元與需求，逐項比對列、傷害／NA、note 和每個 skipped 原因；另驗證 Train 能補足技能需求、NA 的 Pareto 支配與相等點保留，以及衝突／缺 Main 範本拒絕。缺原版檔時 SKIP，驗收仍要求 PASS。既有 TestRank 保留獨立 Fight 與 Pareto／top／TSV 驗證，配合新合法性與欄位更新預期。

原負傷害百分比回歸測試的武器組在該職業遮罩下不合法；改以 Fight 直接驗證原先全部 24 個屬性排列，避免被 Rank 的合法性過濾擋住戰鬥回歸路徑。

### I2 去重與平行實測

單次 Rank 呼叫先列舉所有格子，按第一次出現的順序建立唯一 Fight 工作，再以外層 worker pool 執行；不跨呼叫快取。鍵包含 race、class、level、PlanAttrs 的七項結果、PlanSkills 加 Train 封頂後的全部 41 項技能、主副手物品與實際技能、副手是否存在、指令、Mode／OffMode、Monster、Difficulty。屬性／技能配點順序本身、loadout 名稱不在鍵中；相同結果共用一次 Fight，候選列仍全部保留。NA 不跑 Fight；總格數為通過 creation／裝備合法性篩選後所有列 × Levels，包含 NA，不受 top 或 Pareto 影響。

`-j N` 必須為正整數，預設 `runtime.GOMAXPROCS(0)`；唯一工作在固定索引寫結果，全部完成後按原列舉順序填回。top 同分順序及 Pareto 順序固定。既有 Fight 內部平行保留。

2026-10-12，AMD Ryzen 7 5800X3D，16 logical CPUs，共享主機。序列量測前 load average **24.90／23.75／18.80**；預設平行量測前 **24.91／23.85／18.97**，後 **24.83／23.85／19.00**；同時有 Go 編譯／小範本測試及其他主機工作。使用暖 cache，先建 `/tmp/i2-calc`，以下 real 不含編譯時間，完整使用未改動的 `rank-h3-564.json`：

```sh
GOCACHE=/tmp/i2-go-cache go build -o /tmp/i2-calc ./tools/calc
/usr/bin/time -p /tmp/i2-calc rank -exe build/Wiz8.exe -dbs build/data-x/DATABASES -j 1 notes/calc-qa/rank-h3-564.json > /tmp/i2-j1.tsv
/usr/bin/time -p /tmp/i2-calc rank -exe build/Wiz8.exe -dbs build/data-x/DATABASES notes/calc-qa/rank-h3-564.json > /tmp/i2-default.tsv
cmp /tmp/i2-j1.tsv /tmp/i2-default.tsv
```

`-j 1` real **5.82 秒**（user 5.34、sys 0.60）；預設 `-j` real **2.21 秒**（user 5.11、sys 0.48）。兩份 TSV 逐 byte 相同；皆印 `# fights 1156 2664`，skipped 為 class-mask 108、creation 18、race-mask 2。這份範本仍使用單一 SkillPrio，已可由屬性結果相同而減少 Fight。

`TestRankSkillPrios` 使用一族、一職業、一武器組、三項 SkillSet、兩個等級與兩個屬性排列。測試自行以三重索引迴圈展開六個技能排列，獨立呼叫 PlanSkills／PlanAttrs／Fight，逐格比對容差與自行序列化的相異輸入數；比對 `-j 1`／`-j 4` 完整 TSV，並檢查 SkillPrios 明列及佔位符合併。另以 L100／101 和 Train 100 讓技能／屬性相同，確保漏掉 level 的快取鍵一定被抓到。原版檔缺失則 Skip。

驗收障礙：白名單外的 `internal/calc/rank_loadout_test.go` 中 TestRankLoadouts 未指定任何技能來源，且斷言舊七欄 TSV。新規格要求恰好一個技能來源與新增 sprio，因此原測試無法同時符合新規格；保留該檔及斷言，回歸驗收會印 FAIL TestRank-regress。

### I3 指令與模式分布

新範本以 `"Commands":["attack","berserk"]` 列舉兩條指令，或只給其中一條。舊範本無 Commands 時仍走原先 Mode／OffMode，command 欄為空。規格同時要求「預設兩者」及「無 Commands 保持舊固定模式」，因此此版本以 `Commands:null` 表示指令模式的預設兩者；Go 呼叫者以非 nil 清單啟用指令列舉。

Attack 的每手模式從物品 +0x4e u16 遮罩的 bit 0..8 依序排序成 m0..m(k−1)。B15 規定 P(mj)＝#{q∈0..8：q mod k＝j}/9，並非一般的 1/k。`RankModeDistribution` 可獨立測試；nil 遮罩表示空手，有效技能 14 <5 時只用模式 5，≥5 時用模式 5、6（5/9、4/9）。有效值應先經 EffectiveSkill。空手武器組的副手拳同樣用 nil 遮罩分布，與主手獨立（I11）。Fight 仍要求 Main 物品，且缺副手表示單持，因此 Rank 暫不提供空手武器組；沒有修改 fight.go／build.go。

Berserk 只允許執行期 ClassFeat 有能力 0x14，且主手物品射程 +0x47 ≤1 的職業／武器組；主手固定模式 3，忽略物品遮罩。副手仍使用 Attack 分布。不合法指令以 `# skipped berserk <n>` 計數，每個通過 creation／裝備檢查的種族 × 職業 × 武器組計一次，不乘配點排列或等級。

每格對兩手模式的笛卡兒積呼叫 Fight，以 Pmain(mi) × Poff(mj) 加權 `round.damage`；單持只加權主手。最後總分才固定到 13 位有效數字。遮罩 bit 0..8 全為零會回報錯誤，不自行添加模式。去重鍵包含指令，所以 Attack 與 Berserk 分開建立工作；相同結果的配點候選仍共用加權工作。`# fights` 計數實際模式組合的 Fight 呼叫數，`總格數` 包含每條合法指令的格子及 NA。

`TestRankModes` 手算 k＝1..9 的每個模式機率，另檢查稀疏遮罩、空手門檻與空遮罩。原版檔存在時，執行期尋找多模式近戰主副手，獨立以九個 q 展開分布、重算 Fight，逐格驗證 Attack／Berserk 的雙持及單持分數（1e-12 × max(1, |expected|)），檢查指令 TSV、平行輸出及非 Berserk 職業的 skipped。缺原版檔會 Skip；驗收要求 PASS。

---
<!-- 原檔 notes/calc-party.md -->
## calc party：創角 Pareto 前緣

用法（exe 僅讀取，不執行）：

```sh
go run ./tools/calc party -exe /path/to/Wiz8.exe -metric attr:0,l100:6
```

`-exe` 必填。省略 `-metric`＝B3 缺省：依序輸出 attr:0..6、l100:0..6 共 14 段單一指標前緣，段間空行，每段等於 `-metric` 該指標的輸出（`scripts/party.test.sh` 斷言）。給了 `-metric` 就不能是空值。指標以逗號分隔，每項為 `attr:i` 或 `l100:i`，`i=0..6`，依序為 STR、INT、PIE、VIT、DEX、SPD、SEN。重複指標可用，欄位仍依輸入順序保留。缺少或不合法的指標會非零退出，錯誤說明送到 stderr。

stdout 為 TSV，第一列是 `race class gender attrs` 加上原字串指標名（欄位以 tab 分隔）；`attrs` 是七個逗號分隔的起始裸屬性。每列代表一名角色的創角選項，並非完整多人隊伍。只印全域 Pareto 前緣：不存在另一可行選項在每項指標至少一樣好，且至少一項更好。同一指標值向量只保留任一合法達成者，因此不會列出所有並列種族、職業、性別或配點。

### 規則與來源

- 種族 0..10、職業 0..14、性別 0..1；職業 2 僅允許性別 1。限制見 `notes/re/growth.md:11`、`:12`、`:14`。
- 種族初值 R 與職業門檻 M 由既有 `LoadExe` 在執行時從使用者自己的 exe 讀取；不在程式或 fixture 內嵌遊戲表。這也沿用該讀表器既有的 exe 版本與資料合理性檢查。
- `D=Σmax(0,M−R)`、`B=60−D`、`Apre=max(R,M)`；B<0 不可選。依 `notes/re/growth.md:12`–`:14`。
- 自由點 `Σx≤B`，每項 `0≤x[i]≤min(K,100−Apre[i])`，`K=min(B,max(3,ceil(B/3)))`，`Astart=Apre+x`。只對指標提及的屬性配自由點，其餘 x=0。依 `notes/re/growth.md:17`。
- `attr:i=Astart[i]` 越大越好；`l100:i=1+ceil(max(0,100−Astart[i])/3)` 越小越好。依 `notes/re/growth.md:16`、`:17`、`:21`。後者假定沒有轉職欠額或其他永久改值，且每級對該屬性投入 3 點；多項 l100 是各自獨立的最早等級，並不保證能在每級總額 6 點下同時達成。

枚舉利用指標隨裸屬性單調改善的性質，只訪問已在所選屬性花滿可用額度的配點邊界；未用完可用額度的點可被這些點涵蓋。以全部合法起始組合的預算、單項上限判斷支配，並合併相同指標值。這些剪枝保留完整前緣，不抽樣、不截斷。指標涉及很多不同屬性時，前緣本身及枚舉量仍可能很大。

### 驗證與後續

`TestParty*` 使用人造 R/M 表，比對獨立暴力枚舉得到的完整前緣，涵蓋混合方向支配、l100 取整、重複向量與指標、性別限制、負預算、零預算、K 與 100 上限，以及 TSV 與錯誤輸出。驗收指令為 `bash scripts/party.test.sh`；該腳本在有本機 exe 時另以獨立計算比對，沒有 exe 時會 SKIP。

後續加入初始技能、開局法術、性別差異（除本版已有的職業限制；研究狀態見 `notes/re/growth.md:15`）、近戰評分指標，以及多人隊伍的共同限制。本版不宣稱已完成 stage10 B3 的全部最佳化功能。

### 規格疑慮

- `goals/stage10.md` B3 描述未指定指標時輸出各指標前緣；本票第 3 項明確要求缺省 `-metric` 時報錯。原版依本票實作必填；2026-10-12 cc 改依 B3 補上缺省行為（規格優先於票），`-metric` 帶空值仍報錯。

---
<!-- 原檔 notes/loot-where.md -->
## I12 掉落查表

唯讀工具，僅用 Python 標準庫。輸入是使用者自有的解出資料庫目錄；不執行遊戲，也不寫入輸入目錄或保存資料庫內容、名稱與查詢結果。

```sh
python3 -I scripts/loot-where.py item build/data-x/DATABASES 34
python3 -I scripts/loot-where.py item build/data-x/DATABASES excaliber
python3 -I scripts/loot-where.py monster build/data-x/DATABASES 283
bash scripts/loot-where.test.sh
```

最後一個參數為十進位 ID 或不分大小寫的名稱子字串；多筆名稱命中依 ID 逐筆列出。無命中、ID 越界、缺檔或格式錯誤均向 stderr 印錯誤並非零退出。載入時檢查四檔的筆數、長度、讀取邊界、名稱終止、已處理引用的範圍和 EOF；NPC 尾表按條件串流讀取。ITEMTABLES 依分組數定位表數，並非硬編碼固定表數位址。缺少預設資料庫目錄時測試印 `SKIP loot-where`。

輸出為 TSV，表頭列明欄位。文字中的反斜線、tab、換行以跳脫文字表示。所有顯示的 ID、表號及槽號從 0 起算。

- `ITEM`、`MONSTER`：查詢命中的 ID 與名稱。
- `TABLE`：表號、名稱、槽號、item ID、該槽 w、整張表非空槽數、非空槽 w 總和、等級篩選旗標、抽取數骰。同一物品重複槽逐槽保留；不能把行數一概當成不同表數。
- `MONSTER_SOURCE`：怪物 ID、名稱、槽號、kind、途徑（`direct` 或表號）、c、近似槽門檻機率、數量骰。
- `NPC_DEATH`：NPC ID、名稱、表號。磁碟死亡池 d 僅在 `0 < d < 表數` 時採用，顯示為 d−1。
- `NPC_SHOP`：NPC ID、名稱、item ID、基準量、固定供貨旗標。量為零的欄位引用仍保留。
- `SLOT`：怪物的八槽，各列槽號、kind、類別／非空值、ref、目標名或狀態、c、近似槽門檻機率、數量骰。類別為零顯示 empty；未知 kind 顯示 unhandled kind，不猜引用。
- `TABLE_ITEM`：怪物 kind 1 引用表的表號、槽號、item ID、物品名、w；另附完整 `TABLE` 欄位。

骰顯示為 `Z+NdS`，表示有號基值 Z 加 N 次 1 至 S 的骰；N=0 時只有基值。表中的 w=0 是固定加入項，並非零機率。w／總和僅是欄位資訊，**不計算表抽中機率**：表不是一般累積權重抽籤，還有固定項、不放回槽、接受門檻与回退等規則，且 +0x1cc 的等級篩選門檻尚未取得。

怪物槽的近似機率為 `min(c+1,100)/100`，依接受條件 `R(100) ≤ c` 並假設 R(100) 均勻。實際 RNG 分布未確認；此數字只表示槽門檻，不是特定物品取得率。數量骰可能非正而不產生物品；kind 1 的數量 Q 還控制外圈重複 Q 次、每次要求抽 Q 次。工具保留骰而不推算件數或合成掉率。

NPC 商店量與旗標是候選貨單資訊，不保證實際庫存；死亡池還受已拿走／偷走物品等狀態影響。怪物模板引用不證明實際生成、所在區域或死亡路徑。地上物與寶箱擺放的 `.STS` 未納入；任務、合成、初始裝備、其他硬編碼來源與可達性也未建模。因此沒有列出的來源不能解讀成全遊戲無法取得，亦不能由表名推定寶箱位置。

格式與限制依據：[掉落格式筆記](re/loot.md)。

已知表外來源（MG 2026-10-13：「Ivory Blade 是鐵匠製作品。Fang 是某一張地圖打完怪可以拿，這兩者必出」）：Ivory Blade 501 為鐵匠合成、Fang 59 為某地圖固定取得，兩者必得；工具查無來源屬上述未建模範圍。

The Avenger 61（2026-10-13 查表）：只在表 57 Marten Chest（w 3／總和 197，同表有 Zatoichi Bo、Faust Halberd）；沒有怪物、NPC 引用表 57。MG：寶箱不會重生，所以 legit 只有一把；其他寶箱是否也用表 57 未查（`.STS` 未解析）。

MG 遊戲中沒印象、本工具也查無來源的 7 個 id（2026-10-13）：85、93、96、465、489、748、749（Rammbus Staff、Longstem Spade、Maenad's Lance、Flamequencher Wand、Instakiller、Stinger×2）。I9c 起從武器池排除。jeffludwig.com 武器表 231 把裡有 49 把在掉落表、怪物、商店都查不到（近戰 29、其他 20）：
- 近戰：Beastslayer、Cleaver、Woodsman's Axe、Canezou Dagger、Chain、Vampire Chain、Vulcan Hammer、Longstem Spade、Maenad's Lance、Mindblast Rod、Spiked Spear、Crux of Crossing、Ebon Staff、Faerie Stick、Flamequencher Wand、Notched Shaft、Rammbus Staff、Staff of Ash、Winterwand、Wooden Dowel、Ymmu's Paw、Bloodlust、Diamond Epee、Dragon Slayer、Fang、Instakiller、Ivory Blade、Stinger×2。
- 其他：Boomerang Shuriken、Elven Bow、Tripleshot Crossbow、Flamethrower、Rocket Launcher、Zip Gun、Solar Powerpaks、Earthshaker Bomb、Thermal Pineapple、Omnigun Mk 1–3、5–12。
- 名稱比對用正規化（網站拼法 Nunchaka、Seige Arbalest、Chieftan's Sceptre 對到資料檔名稱）。查無來源不等於拿不到（見上方未建模範圍；Fang、Ivory 即屬表外）。

# 階段 16：最佳化試算 v1（近戰，武器組 × 配點順序 × 攻擊模式）

> MG 2026-10-12：「開票，武器組、配點順序、攻擊模式都列舉。真正的比賽是 仙靈忍者 coc + ros／盜賊（不確定什麼種族） 光劍 + 盜賊匕首／武士（不確定什麼種族） 村正 + enchanted wakaxxxx／戰士（不確定什麼種族） ivory blade + diamond eyes／其他混種職業也要拿自己專屬或遊戲中 top tier 的武器」。
> 承 stage15（v0 全職業共用 LS＋Diamond Eyes、Mode 0、單一 SkillPrio，見 notes/research/ranking-history.md:610）。派工照 CLAUDE.md 第 7 條：sol，小票。

## 票
| 票 | 內容 | 驗收（會紅） | 執行器 |
|---|---|---|---|
| I0 | 研究：玩家主手 UI 可選攻擊模式＝物品 +0x4e 遮罩之外還有什麼（狂暴 mode 3 由職業給？哪些職業？其他模式？），附 `build/wiz8.asm:<行>` | 結果檔每條有來源；cc 抽查 | muse |
| I1 | rank 範本 `Loadouts`（每組 Main／Off＋名稱）＋裝備合法性（職業 +0x76、種族 +0x78、副手 +0x41 bit3、雙手 +0x41 bit2、兩手武器技能本職業可用；屬性／技能需求 +0x7d..84 逐等級，不足該格 NA） | `TestRankLoadouts` | codex sol |
| I2 | 配點順序列舉（`SkillSet` 全排列或 `SkillPrios` 明列）＋相同 Fight 輸入去重快取＋外層 goroutine 平行（輸出順序固定） | `TestRankSkillPrios` | codex sol |
| I3 | 指令列舉（Attack／Berserk），兩手模式按 B15 遮罩分布加權（規則見 I0 結果） | `TestRankModes` | codex sol |
| I5 | 研究：Haste（效果、來源：Psionic 法術書 Monk／Psionic、Gadgeteer）與先攻對輸出／擊殺的影響；之後依結果開實作票 | 結果檔每條有來源；cc 抽查 | muse |
| I6 | Haste（SPD＝max(SPD,125)）與屬性覆寫：FightScenario.Attrs／Haste、rank AttrMin／Haste（I3 收後派，rank.go 重疊） | `TestFightHasteAttrs` | codex sol |
| I7 | 空手主手（Monk 等）：FightScenario／rank Loadout 的 Main 可省略＝空手，技能 14（MG 2026-10-12：「Monk 空手可以先開票，跟先前計算的一起列…Monk 的空手是可以 125 的」）；收貨後 cc 把空手加進 I4 範本重跑 | `TestFightUnarmed` | codex sol |
| I8 | rank 多怪（`Monsters`）＋擊殺率指標（`Metric:"kill"`：每怪 1／期望回合數，含 round.instakill，HP＝模板 +0xd6 骰期望），多怪取平均（MG 2026-10-12：「怎麼配種族、職業跟裝備擊殺率最高？要計算致命一擊的即死哦」） | `TestRankKill` | codex sol |
| I9 | 六人隊最佳化（cc）：L20 以上 56 隻（同 2026-10-08 擊殺率定義）等權、I4 主情境 L28；隊伍值＝成員擊殺率和 [推測：各打各的怪]；武器每件一把（錐刺短劍 id 3 不限、復仇者 61 排除、空手不限）；種族自由（照物品種族遮罩）。兩範本：A 戰賊忍＋{僧女領侍遊}選 3；B 詩工戰賊忍＋選 1（MG 2026-10-12） | 結果入 notes/research/，每組合列最佳隊 | cc |
| I9b | I9 全隊實跑改 Expert 8 怪（cc；MG 2026-10-13「不如跑隊伍最佳化」；基準依 MG 同日「export 的怪才夠打…Expert 才是比較基準」→ 不跑 Normal 56 怪，偏離 I9 原設定）：設定同 baseline-align 第七部分（8 怪 564、114、184、171、129、158、120、192，Difficulty 2，L28，屬性 0,4,5,3，AttrMin 0，不開 Haste，指令 attack，Train 同第七部分），11 種族全開，武器池沿用 I9 範本（篩選來自 Normal 56 怪的 I4，池邊界限制照記）；計算器含 I11 | 完成定義：(1) `scripts/i9-party.py` 能重產 Expert 範本（重跑指令入筆記，不靠 scratch）；(2) 10 職業 `calc rank` 全 rc 0、每檔列數＝種族×配裝（NA 列計入），列數由腳本斷言；(3) A 範本 10 組、B 範本 5 組各列最佳隊（成員職業／種族／配裝／值、合計），限制：光劍等每件一把、Stiletto 不限、Avenger 排除；(4) 抽查：第七部分已算過的格（如女 SoD、僧 SoD）值相同，不同就停下歸因；(5) 對照甲隊（I10）合計與差；任一最佳隊成員落在池邊界（該職業第 TOP 名）要註明；(6) 實際耗時記入；結果入 notes/research/ranking-history.md:827；CI exit 0 後 push；完成 tg [wiz8-mod] | cc |
| I12 | 掉落查表（MG 2026-10-13「只要能知道什麼武器在哪裏打或是什麼寶箱會出…就是刷而已」「掉落查表那票可以平行開」）：`scripts/loot-where.py` 依 notes/re/loot.md 格式查物品來源（表、怪物、NPC 死亡池、商店）與怪物掉落；.STS 擺放與表等級篩選不納入 | tickets/I12-loot-where.md 驗收；codex 執行，cc 收貨重跑 | codex |
| I9c | I9b 的武器池改用 Expert 8 怪重篩後重跑全隊（cc；MG 2026-10-13 設 goal）：篩選＝人類 L28、屬性順序 0,4,5,3、Expert 8 怪、單持主手值（同 I4 做法，副手以單持值排名），每職業主手前 6、副手前 8 ∪ MG 裝備 ∪ Stiletto，Avenger 排除（**修訂 MG 2026-10-13**：查到 Avenger 在 Marten Chest 表 57 後 MG「那其實都要計算了，這些不是空中樓閣」→ Avenger 改為計算、每件一把；排除清單改為 MG 沒印象且查無來源的 85、93、96、465、489、748、749）；其餘設定同 I9b（Fighter attack＋berserk） | 完成定義：(1) `scripts/i9-party.py` 新增篩選與建範本指令，範本在 build/i9e，重跑只靠 repo 指令；(2) 10 職業 rc 0、check 全過，結束碼與耗時存 build/i9e/run.log 並印出；(3) A 10 組、B 5 組最佳隊（職業／種族／配裝／值／合計），存 build/i9e/party.md 並印出；(4) 腳本比對與 I9b（build/i9x）共有格逐位相同，印比對格數與不同格數，有不同即停下歸因；(5) 新舊池差（每職業進出武器）、15 隊換人／換配裝與合計差、對甲隊合計；(6) 成員武器在新池最後一名→主手前 8／副手前 10 重跑該職業一次，仍在邊界記入筆記；(7) 耗時、notes/research/ranking-history.md:921、CI exit 0 後 push、tg [wiz8-mod] | cc |
| I10 | 基準對齊（cc；MG 2026-10-12：「基準要對後續才有用，我們要先對齊基準」）：(1) 照 2026-10-08 設定（L28、屬性 STR DEX SPD VIT、不開 Haste、無 AttrMin、技能 武器／CC／DW／CS 100、PS 58、56 隻 L20+ 等權、Normal、戰士狂暴）用 calc rank kill 重算 notes/research/ranking-history.md:325–327 兩隊成員，加僧 81／女 86 兩格（MG：「僧侶配座頭市之棍比較強」）；(2) 同成員對 8 怪（564、114、184、171、129、158、120、192，MG 2026-10-12 選定）Expert 逐怪擊殺率＋平均；(3) 判斷兩隊配裝與答案要不要改（不跑全隊最佳化） | notes/research/ranking-history.md:404：每格 10-08 值／calc 值／差／歸因（公式或設定差，歸不出寫未取得）；8 怪表；結論段；CI exit 0 後 push；完成 tg [wiz8-mod] | cc |
| I11 | 空手副手拳（cc；MG 2026-10-13：「好，開 I11 修」）：兩手皆空時 004ee220 輸出的第二手 enabled＝1（2 下、+50%、骰 4+4d4，cc 探針），Fight 卻以 `c.off == nil` 跳過第二手 → 改以第二手 enabled 決定是否出手；空手 OffMode 0 比照主手轉拳擊模式 6；rank Attack 的空手副手模式分布同主手（`RankModeDistribution(nil, 有效技能 14)`）。對照 MG 面板（2026-10-13，拳法 125 的僧）：踢擊 20–44×3、拳擊 12–30×2，命中差 3 | `TestFightUnarmed` 加：空手 Monk L28 有 `off.*`、第二手 2 下、骰範圍乘傷害加值＝面板 20–44／12–30、round.damage＝兩手和；拿武器（81 雙手、501 單持）無 `off.*` 且 round.damage 與修前相同。四段注入：還原跳過條件 → FAIL。**驗收更正（2026-10-13 cc，e846fa4）**：「round.damage＝兩手和」不成立為斷言（Fight 不輸出各手傷害），改為空手有 `off.*`、81／501 無 `off.*`、59＋25 有 `off.*`；「拿武器 round.damage 與修前相同」只以修前修後探針比對（scratch，未入測試）。既有 Attack 分布斷言改主副手模式交叉加權（5/9、4/9）。注入兩處：迴圈條件 → FAIL `:124`；rank 副手分布 → FAIL `:208` | cc |
| I4 | 試算 v1：cc 編武器組清單（MG 四組＋各混種職業專屬／top tier），三情境（主：屬性與用到的技能 Train 到 100；次：MG 實盤技能 notes/calc-qa/mg-party-l25.md 走 calc fight；參考：只配點），對 564、411，L1／10／28 | 結果檔＋重跑指令 | cc |

I1、I2 可平行（各自 worktree，白名單重疊 rank.go → I2 在 I1 收貨後 rebase 再派，或循序）。I3 等 I0。I5 結果出來後 cc 決定 Haste／先攻實作票，I4 等它們。

> MG 2026-10-12（I 票開票後）：「理論上 str、dex、spd 跟 dual wield 都滿的情況下，雙持不可能輸給單持，更何況 top tier 的武器全部都加 init，而且靈能者法術書（武僧、靈能者）跟工匠還有 haste 可以用」→ I4 加 Train 到 100 情境、I5。

## 成本
v0 實測一次 Fight 約 24 ms（3960 組×3 等級 287 s）。v1 單一武器組：種族 ≤11 × 屬性排列 × 配點排列 × 主手模式 × 3 等級；I4 範本要把屬性排列縮到 2（STR 先／DEX 先）、SkillSet ≤5 項，先跑一組量實際秒數再全跑。

## 假設（沿用 v0，I4 報告列出）
練技不建模（Train 情境另列）、初始技能 0、性別不列舉（但物品 +0x7c 性別遮罩記錄）、calc fight 未建模清單（notes/calc.md:91）。

## 完成定義
1. I1–I3 收貨（各四段注入，main 重跑驗收）；I0 收貨並抽查。
2. I4 結果入 notes/research/，三段報告：MG 四組對決＋各職業最佳＋Pareto。
3. check-formulas、calc-qa、party 綠；CI exit 0 並 push。

> MG 2026-10-12：「單靠升級配點，反而不是實際情況，因爲任何一個技能在使用中就會漸漸提升」＋四張 L25 面板 → 只配點降為參考情境；技能隨使用成長不建模，改以 Train 100 與實盤快照當輸入。

> MG 2026-10-12（Haste 實機後）：「即便 speed 只是目前速度 x 1.1 也沒差，因爲最可怕的敵人都在後期，當時你的 speed 也高了，就當作 speed 是 125。所以爲什麼我堅信雙持總輸出更高，因爲有 speed 跟高階武器的 init 加持。況且你可以看到破關前的雙持技能都很高。load 最高還能到 125」
> → I4 主情境改：SPD 一律 125（Haste 常駐，不建模持續回合）；屬性其餘與用到的技能 Train 100，另跑一組技能 125 當上限情境（「load 最高 125」cc 讀作技能值含裝備可到 125 [推測：待 MG 更正]）。雙持對單持是 I4 報告的第一個問題，結果照實報，不預設結論。

> MG 2026-10-12：「每個職業都有一個技能可以有加成 x 1.25。lord 的就是雙持技能，我的基底技能沒有練到 100，所以加成上去不到 125」
> → 更正上一段：125 是主技能 ×1.25 的上限，不是全技能。撤掉「技能 125」上限情境；Train 100 下主技能經 build.go 的 b+⌊b/4⌋ 自然得 125（與 MG 說法一致）。Lord 主技能＝雙持（MG）。

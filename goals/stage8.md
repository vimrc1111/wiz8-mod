# 階段 8：28 級戰士的種族 × 武器強度表

> MG 2026-10-08：「等反組譯解析都回來後，其實就可以計算一個 28 級的戰士有多少升級點跟技能點，然後選什麼種族會拿到更多可以分配的點數，那個時候就知道拿這些武器的強度」。MG 離席前要 cc 開新 goal 讓事情往下走。分工見 CLAUDE.md「派工」。

## 已有（不重做）
- 點數：`notes/re/growth.md`（每級 6 屬性點／9 技能點、每級每項 ≤3、裸值 ≤100、28 級共 222／258）。
- 屬性→近戰：`notes/re/formulas.md` F07–F14、F20–F24、F28；`notes/re/combat-2.md` C02–C07；背刺／副手狂暴 `notes/re/backstab.md`。
- 計算器：`internal/calc`（票 Q，`calc melee`／`calc spell`）。
- 交叉核對：`notes/research/ranking-history.md:385`（種族列序、效果欄意義）。

## 步驟
- 8.1 cc：從玩家自有 `Wiz8.exe` 讀表，loader 寫在 `internal/calc`。表內容不入版控，測試靠 `WIZ8_EXE` 環境變數，未設就 skip，做法同 `TestRealData`。要讀的表：
  - 種族初值 `0x614cf0`；
  - 職業門檻 `0x614e24`；
  - 職業主技能 `0x6161dc`；
  - 技能控制屬性 `0x6155b0`；
  - 職業×技能可用矩陣 `0x615840`；
  - feature 表 `0x61507c`。
  - 位址未讀過的先讀 asm 確認長度。
- 8.2 codex（gpt-6.1-sol；卡兩輪就改 astra medium）票 R：`calc build`。
  - 輸入：種族、職業＝Fighter、L=28、配點策略、技能分配、主副手武器 id、攻擊模式（一般／狂暴）、參考怪物 id。
  - 輸出：每回合期望傷害，以及中間值（attack 數、揮擊數、ARhit、ARpen、命中率）。
  - 配點策略預設「STR、DEX、SPD 先到 100，其次 VIT」：逐級合法（每級 Σ≤6、每項 ≤3），不是七項各自最早等級相加。
  - 技能只算手動 258 點，不算練技成長：結果標成下限。
  - 驗收：cc 先寫 golden 測試，至少兩組手算，兩組都要含門檻跨越（v=67／100、a=50／75）。
- 8.3 cc：產出種族 × 武器表 `notes/research/2026-10-?-L28-fighter.md`。
  - 種族：11 族中能當 Fighter 的全部。
  - 武器，依 MG 所列：
    - Light Sword＋Canezou Dagger
    - Light Sword＋Stiletto
    - Ivory Blade
    - Fang
    - Excaliber
    - Giant's Sword（只 Mook）
    - Dread Spear
  - 模式：一般／狂暴。
  - 目標：AC 0 假想怪，加一隻後期怪（id 在 note 寫明）。
  - 每格標 `[已驗證：calc golden]`／`[推測]`。
  - 抽 3 格手算核對。
- 8.4 grok 稽核 8.2 的實作（`audit-run.sh`），只當參考；發現要轉成會紅的測試才收（地板 §10.8）。

## 完成定義
- `bash scripts/ci-run.sh wiz8-mod bash scripts/ci-check.sh` exit 0。
- `WIZ8_EXE`／`WIZ8_DBS` 設定下，`go test ./internal/calc/` 的真資料測試 PASS。
- 8.3 的 note 過 `scripts/check-research.sh`，已 push。
- resume.md 更新：做了什麼、哪些格是下限或推測、待 MG 什麼。

## 邊界（沿用）
- 原版資料一個 byte 不動。exe／DBS 抽出的表值、反編譯碼都不入版控，note 只寫結論與少量引用值。
- 不跑實機遊戲（MG 不在，不動 Saves）。
- 已知缺口照標，不擋本階段：
  - 遠程（C10）；
  - Kill 抗性（F25）：即死欄只列門檻，不算成功率；
  - 練技成長；
  - 裝備屬性加成：只算物品資料裡有的。

## 停損
- 同一項連續 3 次失敗、且原因不是基礎設施 → 記「待 MG」，換下一項。
- 表位址讀出來和 jeffludwig 交叉核對對不上，且不是已知的兩處差異 → 停，記待 MG，不猜。
- codex 額度用完 → 小票派 muse，大票 cc 接手。

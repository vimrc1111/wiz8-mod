# 階段 14：公式計算修補票（後續票＋F 模式）

> MG 2026-10-12：「先全面把修補票完成，Muse/Codex 額度都回來了」。範圍是 resume.md「後續票」8 條（:323–:330）與「F 模式待追」。
> F-only 範圍沿用 MG 2026-10-11「照你建議的，D4 完再說」：只補 calc party／calc-qa 會走到的路徑。

## 範圍：calc 實際走到哪些節點
`bash scripts/calc-reach.sh` → `notes/calc-reach.txt`（148 檔；Go coverage，init 不算，檔案級過度包含）。
待追節點與它的交集（F-only 做這些）：551ba0 551eb0 54df00 543270 4eae20 545090 5560a0 5526f0 547cb0 4e9f90 4e9490 4e8ea0。
不在交集（本階段不做 F-only，理由＝calc-qa／party／fight 走不到）：556b10 5517c0 50b590 550c10 54dbe0 544d30 4ebfe0 4fa4d0 54ba00 4fb4c0 502d00 51dde0 4e9b20 540170。

## 票
| 票 | 後續票 | 內容 | 驗收（會紅） | 執行器 |
|---|---|---|---|---|
| G1 | 3 | 550c10／54fa30／545090 的 case 改以位址樁換 4c6c30，放開 +0x1dc 位元 0x400 | `scripts/nodes/g1-check.sh`＋node-accept 三節點 | codex sol |
| G2 | 2 | 590950 後綴模型：wcscat 到另一區、長度只看格式文字；補 551eb0 case 走到格式文字只有單一 `'` | `scripts/nodes/g2-probe.sh`（執行方建）＋node-accept 551eb0 | codex sol |
| G3 | 4a | 法術 recast 深度夾具上限 3／4 → 依原版上限或 n 記錄 | `scripts/nodes/g3-probe.sh`（深度≥5 合法輸入，calc 不 panic）＋node-accept | codex astra |
| G4 | 4b | SpellTables.dbs 解析格式後統計 +0x12b 模式，驗 4fa5cc 豁免前提「只有 0..3」 | `TestSpellMode12bDomain`（本機有原版才跑） | muse |
| G5 | 5 | 547cb0 怪物筆數夾具上限 8 → 原版一群上限 | `scripts/nodes/g5-probe.sh`（上限筆數合法輸入）＋node-accept 547cb0 | codex sol |
| G6 | F-only | 模擬器原生模式（公式類位址樁不裝，排除類照裝）＋`TestFNative` 用 calc F 比；試點 54df00 | `TestFNative` 對 54df00 diff=0 | codex astra |
| G7… | 1、F 模式 | G6 收貨後，交集其餘節點逐一上原生 case；F panic／F≠原版即修 F 或 callee 夾具 | 各節點 `TestFNative` diff=0 | codex／muse |
| — | 7 | 5ba740 iicDsrItem 斷言 | `TestIicDsrItemMatchesCase`（cc 做，dd74262） | cc |
| — | 8 | 5e3800 類別 8 未初始化欄 | 已在 `internal/calc/n_5e3800.go` 檔頭（:12–:14），不另開 | — |
| — | 6a／6b | 544010 域、540170 豁免塊 | F1 7f979b6、F2 ba105d8 已收 | — |

6c（540170 F panic）不在交集，本階段不做。

## 共通
- 原版不入 repo；票的材料（`build/Wiz8.exe`、`build/wiz8.asm`、模擬器 venv、`build/data-x`）放 worktree 內實體檔。
- 不縮輸入域、不放寬驗證、不刪 boundary 既有列閃差異。
- 夾具上限改動：上限取原版的（附 `build/wiz8.asm:<行>` 證據），找不到原版上限就寫 `[未取得]` 並取正常遊玩能到的值，理由寫進 iface。
- 每票四段注入（注入→FAIL→副本還原→PASS）貼進回覆。

## 完成定義
1. G1–G6 收貨；G7 交集節點全部 `TestFNative` diff=0，或逐節點寫合格「未取得」（原因＋卡在哪個 callee）。
2. `bash scripts/check-formulas.sh`、`bash scripts/calc-qa.test.sh`、`bash scripts/party.test.sh` 仍綠。
3. CI exit 0 並 push。

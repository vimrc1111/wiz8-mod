# N-4fa4d0 節點 character_spell_cast 到 D4（stage10 B1／B2）

驗收: bash scripts/node-accept.sh tickets/nodes/4fa4d0.list
規模: 大（循環群 4fa4d0 4fb4c0 54ba00 54c930，cc）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔（含 re/emu/emu.py、re/emu/run.sh、internal/calc 既有檔、登記簿、scripts/）；不寫入遊戲目錄；不執行遊戲；不下載任何東西；筆記與註解不貼組語或反編譯碼；不把原版 bytes、從 exe 讀出的表寫進 repo
白名單: re/emu/cases/004fa4d0.json, notes/re/iface/4fa4d0.tsv, internal/calc/n_4fa4d0.go, notes/re/cov-exempt.d/4fa4d0.tsv, re/emu/cases/004fb4c0.json, notes/re/iface/4fb4c0.tsv, internal/calc/n_4fb4c0.go, notes/re/cov-exempt.d/4fb4c0.tsv, re/emu/cases/0054ba00.json, notes/re/iface/54ba00.tsv, internal/calc/n_54ba00.go, notes/re/cov-exempt.d/54ba00.tsv, re/emu/cases/0054c930.json, notes/re/iface/54c930.tsv, internal/calc/n_54c930.go, notes/re/cov-exempt.d/54c930.tsv
預期紅: node-accept
執行器: cc

## 背景
goals/stage10.md B：每個公式節點要到 D4——介面全命名（D1）、`internal/calc` 純函數（D2）、模擬器跑原版函式與 calc 同輸入 0 差異（D3）、輸入走過全部基本塊（D4）。機制已就緒（`notes/re/emu.md`），範例是骰子節點 517970：`re/emu/cases/00517970.json`、`notes/re/iface/517970.tsv`、`internal/calc/nodes.go` 的 `diceNode`。
本節點：`0x4fa4d0` character_spell_cast（1886 bytes，86 個基本塊）。登記簿描述：讀角色+0x6cf、+0x6d5、學派與領域技能、職業等級、領域SP及法術+0x49、+0x56，算Power Cast加成和失敗風險交0x4fb4c0，輸出施法結果、扣SP並給技能使用點；build/wiz8.asm:337721；notes/re/combat-2.md:59。
子呼叫：2b6,40efa0,4acba0,4f97a0,4f9ae0,4fac40,4fb4c0,4fe480,4ff3b0,4ff7f0,5001e0,517ec0,52b480,52c320,52e690,53be00,553f10,5905c0,590950（模擬器會原樣跑子函式；calc 要重現組合後的行為，可呼叫既有 calc 函數或其他節點的 F）。
材料：`build/wiz8.asm`（objdump）、`build/decomp/cand-b02.c`（Ghidra 反編譯，只供閱讀）、`build/re/funcs.tsv`、`build/re/fields.tsv`、`build/Wiz8.exe`、`build/venv`；既有筆記 `notes/re/*.md`。

## 要做的
1. `notes/re/iface/4fa4d0.tsv`：先跑驗收看 D1 列出的每個子呼叫、資料引用、欄位運算元，逐一寫 `call|data|field<TAB>目標<TAB>名稱或「無關」<TAB>理由`。命名用遊戲語意（例：`C+0xe91` 寫成角色欄位名），「無關」要寫為什麼與規則無關。
2. `re/emu/cases/004fa4d0.json`：呼叫約定、輸入（參數、ecx、arena 物件欄位、全域）、輸出、邊界一批＋隨機一批。輸入要涵蓋函式實際讀的每個與規則有關的欄位。子函式讀到的記錄（角色、怪物、物品、全域表）也要在 arena／全域擺好；從 exe .data 來的表模擬器已映射，不用擺。
3. `internal/calc/n_4fa4d0.go`：純函數 `characterSpellCastNode(in []int64, r Rand) []int64`，用 `func init() { Nodes["004fa4d0"] = Node{...} }` 登記（不改 nodes.go）。常數寫具體值並註明出處（exe 位址或筆記行）；資料表執行時從使用者自己的 Wiz8.exe 讀（見 `internal/calc/exe.go`），不內嵌。既有 calc 函數與原版不符時不改它：本檔另寫，並在最終回覆列出差異。不准有 TODO。
4. 走不到的基本塊（任何合法輸入都到不了，例：防禦性錯誤分支）寫 `notes/re/cov-exempt.d/4fa4d0.tsv`：`4fa4d0<TAB>塊hex<TAB>理由`。「難構造」不是理由。
5. 差分有差異：以原版為準修 calc，不准縮輸入範圍閃差異。

## 規格疑慮
缺的 import 回傳值依呼叫情境而定（查表、取物件）時，先用 case 的 `"stubs"` 欄（`notes/re/emu.md` 搜 stubs）在 case 檔裡指定，不算改模擬器。仍需要新的內建樁或改模擬器才做得下去：停下，在最終回覆寫 `規格疑慮：` 列出缺的樁（`unstubbed import` 那行）或需求，不自己改 emu.py。

## 循環群（一張票做完全部成員）
本票清單 `tickets/nodes/4fa4d0.list` 有 4 行：這些節點互相呼叫（呼叫圖的強連通分量），任何一個都不會單獨就緒，所以一起做。每個成員各自一套白名單檔（case、iface、n_<addr>.go、cov-exempt），各自登記 `Nodes["<addr8>"]`。calc 裡成員互相直接呼叫 Go 函數（相互遞迴）；模擬器照原樣跑整群。驗收一次跑整份清單，每個成員都要到 D4。
其他成員：
- `0x4fb4c0` spell_failure_backfire_and_effect（6928 bytes，317 塊；子呼叫 2b6,408860,40efa0,421150,421680,421750,437f30,467430,489bb0,4a5ab0,4ac9d0,4aca00,4aca60,4ad080,4ad430,4ad630,4ad8a0,4adb20,4addf0,4ca4c0,4e5550,4e5620,4e5720,4e5950,4e6020,4ecc80,4ed260,4ed950,4fb360,4fd030,4fea50,4ff220,4ff790,4ffbd0,501e50,501f80,501fd0,517950,517a90,517f90,52e5c0,52e690,536170,53bea0,53bf10,543260,544630,5471d0,5474b0,54ba00,54ff20,5519c0,58ac00,5905f0,590950,59af40,5e1c10,5e1ce0,5e2c70）：讀法術ID、來源、目標、PL、加成與失敗率，用同一次100骰判失敗及反噬，另依法術+0x51及加成計效果量與強度，輸出結果碼、實際PL及法術實例效果記錄；build/wiz8.asm:338733、build/wiz8.asm:339179；notes/re/formulas-2.md:35、notes/re/formulas-2.md:37、notes/re/combat-2.md:59。
- `0x54ba00` spell_effect_dispatch（3267 bytes，164 塊；子呼叫 2b6,408860,40efa0,4506c0,451280,4ac9d0,4aca00,4ae080,4f0e80,4fa4d0,4fb4c0,506280,50e700,517970,517a70,517f90,520b40,52af50,52b550,52b910,52c320,53bea0,54c930,54cbf0,54cdd0,54d100,54d380,54d5c0,54dbe0,54df00,54e3f0,54e610,54e950,54ec80,54f190,54f520,54f8c0,54fa30,5500c0,5507d0,5508e0,550a00,550c10,5510b0,551500,5517c0,553910,562a50,587c80,58a9c0,58aad0,5905c0,5905e0,5905f0,590bd0,5af2d0）：讀效果編號、+0xb4骰與+0xd4加成等參數分派規則，其中金錢損失由骰量計算後寫回0x685189；build/wiz8.asm:451409，build/wiz8.asm:451428。
- `0x54c930` spell_reflection（689 bytes，42 塊；子呼叫 2b6,4e5550,4e5620,4fb4c0,4fea50,517ec0,523a80,524390,5474b0,590950,590b40）：讀角色+0xa95/+0xa9d或怪物+0xd7/+0xdf反射效果，限制效果+0xd0強度並重送法術，消耗對應反射強度欄位；build/wiz8.asm:452065。


## cc 普查裁定（2026-10-11，可逆預設）
- 外部非 D4 公式依賴：零（普查）。已登記 `排除` 的子呼叫一律位址樁；4ae080 的樁回傳 case 提供的可寫暫存指標。
- 原生跑：4ecc80（非函式入口，callee 皆排除）、4fb360、4fea50、524390（calc 已有 monsterCacheRebuild）、50e8c0、551500（含 552410 心智抗性擲骰）。
- 停下回報：4fea50 內 4feb91 的 vtable 呼叫目標擋路；或 5080f0 寫入規則狀態。
- 位置／座標類輸入當 case 輸入，calc 只重現算術（同 503990、544d30 先例）。

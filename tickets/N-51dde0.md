# N-51dde0 節點 item_use_power_selection 到 D4（stage10 B1／B2）

驗收: bash scripts/node-accept.sh tickets/nodes/51dde0.list
規模: 大（codex gpt-6-astra medium）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔（含 re/emu/emu.py、re/emu/run.sh、internal/calc 既有檔、登記簿、scripts/）；不寫入遊戲目錄；不執行遊戲；不下載任何東西；筆記與註解不貼組語或反編譯碼；不把原版 bytes、從 exe 讀出的表寫進 repo
白名單: re/emu/cases/0051dde0.json, notes/re/iface/51dde0.tsv, internal/calc/n_51dde0.go, notes/re/cov-exempt.d/51dde0.tsv
預期紅: node-accept
執行器: cc

## 背景
goals/stage10.md B：每個公式節點要到 D4——介面全命名（D1）、`internal/calc` 純函數（D2）、模擬器跑原版函式與 calc 同輸入 0 差異（D3）、輸入走過全部基本塊（D4）。機制已就緒（`notes/re/emu.md`），範例是骰子節點 517970：`re/emu/cases/00517970.json`、`notes/re/iface/517970.tsv`、`internal/calc/nodes.go` 的 `diceNode`。
本節點：`0x51dde0` item_use_power_selection（2420 bytes，112 個基本塊）。登記簿描述：讀角色技能 +0x1a3 列、物品 +0x3e、+0x42、+0x63、+0x64 與充能，按技能風險及 100 範圍亂數逐級降低使用強度，輸出效果結果與動作成本參數；build/wiz8.asm:387810、build/wiz8.asm:387812。
子呼叫：2b6,40efa0,4d9080,4e5150,4e5550,4e5620,5061a0,50b800,517a70,517ec0,51d610,51d800,51dcd0,51e760,51ee70,52e690,538670,53be00,53be50,53bea0,53bf10,553f10,58aad0,5905c0,590950,5f,65（模擬器會原樣跑子函式；calc 要重現組合後的行為，可呼叫既有 calc 函數或其他節點的 F）。
材料：`build/wiz8.asm`（objdump）、`build/decomp/cand-b04.c`（Ghidra 反編譯，只供閱讀）、`build/re/funcs.tsv`、`build/re/fields.tsv`、`build/Wiz8.exe`、`build/venv`；既有筆記 `notes/re/*.md`。

## 要做的
1. `notes/re/iface/51dde0.tsv`：先跑驗收看 D1 列出的每個子呼叫、資料引用、欄位運算元，逐一寫 `call|data|field<TAB>目標<TAB>名稱或「無關」<TAB>理由`。命名用遊戲語意（例：`C+0xe91` 寫成角色欄位名），「無關」要寫為什麼與規則無關。
2. `re/emu/cases/0051dde0.json`：呼叫約定、輸入（參數、ecx、arena 物件欄位、全域）、輸出、邊界一批＋隨機一批。輸入要涵蓋函式實際讀的每個與規則有關的欄位。子函式讀到的記錄（角色、怪物、物品、全域表）也要在 arena／全域擺好；從 exe .data 來的表模擬器已映射，不用擺。
3. `internal/calc/n_51dde0.go`：純函數 `itemUsePowerSelectionNode(in []int64, r Rand) []int64`，用 `func init() { Nodes["0051dde0"] = Node{...} }` 登記（不改 nodes.go）。常數寫具體值並註明出處（exe 位址或筆記行）；資料表執行時從使用者自己的 Wiz8.exe 讀（見 `internal/calc/exe.go`），不內嵌。既有 calc 函數與原版不符時不改它：本檔另寫，並在最終回覆列出差異。不准有 TODO。
4. 走不到的基本塊（任何合法輸入都到不了，例：防禦性錯誤分支）寫 `notes/re/cov-exempt.d/51dde0.tsv`：`51dde0<TAB>塊hex<TAB>理由`。「難構造」不是理由。
5. 差分有差異：以原版為準修 calc，不准縮輸入範圍閃差異。

## 規格疑慮
缺的 import 回傳值依呼叫情境而定（查表、取物件）時，先用 case 的 `"stubs"` 欄（`notes/re/emu.md` 搜 stubs）在 case 檔裡指定，不算改模擬器。仍需要新的內建樁或改模擬器才做得下去：停下，在最終回覆寫 `規格疑慮：` 列出缺的樁（`unstubbed import` 那行）或需求，不自己改 emu.py。

# N-4e6ce0 節點 combat_difficulty_rating 到 D4（stage10 B1／B2）

驗收: bash scripts/node-accept.sh tickets/nodes/4e6ce0.list
規模: 大（codex gpt-6-astra medium）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔（含 re/emu/emu.py、re/emu/run.sh、internal/calc 既有檔、登記簿、scripts/）；不寫入遊戲目錄；不執行遊戲；不下載任何東西；筆記與註解不貼組語或反編譯碼；不把原版 bytes、從 exe 讀出的表寫進 repo
白名單: re/emu/cases/004e6ce0.json, notes/re/iface/4e6ce0.tsv, internal/calc/n_4e6ce0.go, notes/re/cov-exempt.d/4e6ce0.tsv
預期紅: node-accept
執行器: cc

## 背景
goals/stage10.md B：每個公式節點要到 D4——介面全命名（D1）、`internal/calc` 純函數（D2）、模擬器跑原版函式與 calc 同輸入 0 差異（D3）、輸入走過全部基本塊（D4）。機制已就緒（`notes/re/emu.md`），範例是骰子節點 517970：`re/emu/cases/00517970.json`、`notes/re/iface/517970.tsv`、`internal/calc/nodes.go` 的 `diceNode`。
本節點：`0x4e6ce0` combat_difficulty_rating（937 bytes，68 個基本塊）。登記簿描述：不確定：讀怪物+27/+2b血量、模板經驗與角色+89等級及NPC關係，算敵我相對強度寫6850b4並設定NPC物件+f0/+f1與6837a1，強度用於音樂選擇但NPC旗標的規則影響待核；build/wiz8.asm:310205、build/wiz8.asm:310248。
子呼叫：406c00,48fc10,4e5620,4e5720,4e6780,50b800,510180,5101b0,52e5c0,556000,5e1c22,5e1f6c,5e2c70（模擬器會原樣跑子函式；calc 要重現組合後的行為，可呼叫既有 calc 函數或其他節點的 F）。
材料：`build/wiz8.asm`（objdump）、`build/decomp/cand-b00.c`（Ghidra 反編譯，只供閱讀）、`build/re/funcs.tsv`、`build/re/fields.tsv`、`build/Wiz8.exe`、`build/venv`；既有筆記 `notes/re/*.md`。

## 要做的
1. `notes/re/iface/4e6ce0.tsv`：先跑驗收看 D1 列出的每個子呼叫、資料引用、欄位運算元，逐一寫 `call|data|field<TAB>目標<TAB>名稱或「無關」<TAB>理由`。命名用遊戲語意（例：`C+0xe91` 寫成角色欄位名），「無關」要寫為什麼與規則無關。
2. `re/emu/cases/004e6ce0.json`：呼叫約定、輸入（參數、ecx、arena 物件欄位、全域）、輸出、邊界一批＋隨機一批。輸入要涵蓋函式實際讀的每個與規則有關的欄位。子函式讀到的記錄（角色、怪物、物品、全域表）也要在 arena／全域擺好；從 exe .data 來的表模擬器已映射，不用擺。
3. `internal/calc/n_4e6ce0.go`：純函數 `combatDifficultyRatingNode(in []int64, r Rand) []int64`，用 `func init() { Nodes["004e6ce0"] = Node{...} }` 登記（不改 nodes.go）。常數寫具體值並註明出處（exe 位址或筆記行）；資料表執行時從使用者自己的 Wiz8.exe 讀（見 `internal/calc/exe.go`），不內嵌。既有 calc 函數與原版不符時不改它：本檔另寫，並在最終回覆列出差異。不准有 TODO。
4. 走不到的基本塊（任何合法輸入都到不了，例：防禦性錯誤分支）寫 `notes/re/cov-exempt.d/4e6ce0.tsv`：`4e6ce0<TAB>塊hex<TAB>理由`。「難構造」不是理由。
5. 差分有差異：以原版為準修 calc，不准縮輸入範圍閃差異。

## 規格疑慮
缺的 import 回傳值依呼叫情境而定（查表、取物件）時，先用 case 的 `"stubs"` 欄（`notes/re/emu.md` 搜 stubs）在 case 檔裡指定，不算改模擬器。仍需要新的內建樁或改模擬器才做得下去：停下，在最終回覆寫 `規格疑慮：` 列出缺的樁（`unstubbed import` 那行）或需求，不自己改 emu.py。

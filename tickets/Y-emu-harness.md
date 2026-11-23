# Y 模擬器差分機制：unicorn 跑原版函式 × calc 節點重放（stage10 B2）

驗收: bash scripts/emu.test.sh
規模: 大（codex gpt-6-astra medium；Python 模擬器＋Go 節點登記＋calc f）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔；不改 scripts/emu.test.sh、scripts/check-formulas.sh；不寫入遊戲目錄；不執行遊戲；不下載任何東西（unicorn 2.1.4 已裝在 build/venv）；筆記不貼組語或反編譯碼；不把原版 bytes、從 exe 讀出的表寫進 repo（只能寫進 build/）
白名單: re/emu/run.sh, re/emu/emu.py, re/emu/cases/00517970.json, re/emu/cases/00552250.json, internal/calc/nodes.go, internal/calc/nodes_test.go, tools/calc/main.go, tools/calc/main_test.go, notes/re/emu.md
預期紅: emu
執行器: codex

## 背景
目標（goals/stage10.md B）：全遊戲規則公式寫成 `internal/calc` 純函數，用模擬器跑**原版函式本身（含子函式）**比對到 0 差異（D3），且輸入走過全部基本塊（D4）。本票只做機制＋兩個種子節點，之後每個公式節點只要「加一個 case 檔＋一個 calc 節點」。
材料：`build/Wiz8.exe`（PE，image base 0x400000）、`build/wiz8.asm`（objdump，行首 `  addr:`）、`build/re/funcs.tsv`（Ghidra 函式表：入口 大小 前16 呼叫 記錄基址 資料引用 基本塊起點，逗號分隔）、`build/venv/bin/python`（unicorn 2.1.4）。
已知：亂數 `0x40efa0`：cdecl/stdcall 參數 n（u32），回傳 0..n−1；內部 `r=rand()`（0..32767），`(r*n/32767) mod n`（`notes/re/formulas-2.md:11`）。骰子 `0x517970`：參數是指向 `{i16 Z, u8 N, u8 S}` 的指標，結果 `Z+Σ_{N 次}(1+R(S))`（N=0 時只回 Z）。抗性削減 `0x552250`：`notes/re/formulas-2.md:106`、calc 現有 `Resist`。

## 要做的
1. `re/emu/run.sh <addr>|--selftest-import`：用 `build/venv/bin/python -I re/emu/emu.py`（venv 不在就試 `~/.local/state/wiz8-mod-emu/venv`），cwd 不影響結果。
2. `re/emu/emu.py`：
   - 依 PE 節區映射 `build/Wiz8.exe`（含 .data 初值）；堆疊、arena 另開。
   - 每個 import（IAT 槽）都指到樁；沒寫樁的被呼叫就停下，stderr 印 `unstubbed import 0x<IAT 位址> <dll>!<name>`，exit 非 0。必要的樁至少：`_ftol`（x87 ST0 朝零截斷 → edx:eax，彈出 ST0）。`--selftest-import` 跑一支會碰到沒樁 import 的 exe 函式（自己挑，寫進 notes），驗證上述行為。
   - `0x40efa0` 換成可控樁：從本條路徑的亂數序列取值，記下每次的 n 與回傳值。
   - 讀 `re/emu/cases/<addr>.json`：呼叫約定（cdecl／stdcall／thiscall）、輸入名稱與它在記憶體的位置（參數、ecx、arena 物件欄位、全域）與大小／有號、輸出（eax、記憶體欄位）、邊界輸入一批＋隨機輸入一批（種子固定、數量寫在檔裡）。格式自己定，寫進 `notes/re/emu.md`。
   - 每組輸入對亂數做 DFS 全列舉（每次呼叫分支 0..n−1）；路徑數超過檔內上限就改成固定種子抽樣，該行標 `"sampled":true`。
   - 輸出 `build/emu/<addr>.jsonl`：每條路徑一行 `{"in":[…],"rng":[[n,v],…],"out":[…],"sampled":bool}`；`build/emu/<addr>.hit`：本次所有路徑命中的基本塊起點（只算該函式自己的塊，hex 小寫無 0x，與 funcs.tsv 第 7 欄同格式），一行一個。
   - 執行步數設上限，超過就紅（不准無限迴圈靜默）。
3. `internal/calc/nodes.go`：節點登記 `Nodes map[string]Node`，鍵是 8 位 hex 位址；`Node{Addr, In, Out []string, F func(in []int64, r Rand) []int64}`；`Rand` 介面 `R(n int) int`。種子節點 `00517970`（骰子）、`00552250`（抗性削減）用既有 calc 函數組出，不另寫一套公式。`Dist(node, in)` 在 Go 端 DFS 列舉 Rand 得到完整分布（權重用真實 `rand()` 0..32767 映射到 0..n−1 的計數，不當均勻）。
4. `internal/calc/nodes_test.go`：
   - `TestEmu`：對 `build/emu/*.jsonl` 每一行，用腳本化 Rand 重放（同時斷言每次要求的 n 相同、次數相同），比 `out`；全對寫 `build/emu/<addr>.d3` 為 `cases=<行數> diff=0`，有錯寫 `cases=N diff=K` 並 FAIL。沒有 jsonl 就 `t.Skip` 印 SKIP。
   - `TestEmuWrong`：只在 `EMU_WRONG=1` 時跑：用「骰子結果加 1」的錯版節點重放 `00517970.jsonl`，斷言 diff>0（抓得到錯）；不寫 .d3。
5. `tools/calc` 加 `calc f <節點> <輸入…>`：印 `dist` 開頭的分布（值:機率，每行一個）、平均、該節點的輸入輸出名稱。
6. `notes/re/emu.md`：case 格式、樁清單、selftest 用的函式、已知上限。

## 規格疑慮
做不到的寫在 `notes/re/emu.md` 末段 `## 規格疑慮`，並在最終回覆列出。

# Z D1 欄位運算元：每個 [reg±off] 都要在 iface 命名（stage10 B1）

驗收: bash scripts/check-formulas.test.sh
規模: 小（codex gpt-6.1-sol medium；bash＋awk）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔；不改 scripts/check-formulas.test.sh；不寫入遊戲目錄；不下載任何東西
白名單: scripts/func-fields.sh, scripts/check-formulas.sh
預期紅: fields, fields:d1, fields:missing, check-formulas
執行器: codex

## 背景
`scripts/check-formulas.sh` 的 D1 目前只查子呼叫與絕對位址資料引用；goals/stage10.md B1 要求函式讀寫的**每個記憶體運算元**都列出並命名。結構欄位（`[esi+0x24]` 這種）還沒查，本票補上。

## 要做的
1. `scripts/func-fields.sh <asm> <funcs.tsv>`：對 funcs.tsv 每個函式（第 1 欄入口 hex、第 2 欄大小，範圍 `[入口, 入口+大小)`），從 objdump 格式的 asm（行首 `  <hex>:\t`）收該範圍內所有方括號記憶體運算元，輸出 `入口<TAB>運算元,運算元…`（排序、去重；沒有就輸出空第二欄）。正規化：
   - 去空白；無位移的 `[esi]` 寫 `esi+0x0`；其餘照原文（`eax+ecx*4+0x10`、`edi-0x4`）。
   - 不算：`esp` 基底（堆疊）；函式開頭是 `push ebp` 接 `mov ebp,esp` 時，`ebp` 基底也不算（框架）；沒有暫存器的絕對位址（已由資料引用欄處理）。
   - 要快：真檔 `build/wiz8.asm` 約 65 萬行、funcs.tsv 數千列，一次掃完（不准每個函式重讀一次 asm）。
2. `scripts/check-formulas.sh`：新增輸入 `FIELDS`（預設 `build/re/fields.tsv`，即 func-fields.sh 的輸出），檔頭註解補說明與產生方式。
   - 有任何 D1 以上節點而 FIELDS 不存在 → `FAIL fields …`（指出產生指令）。
   - D1 以上節點的每個欄位運算元，iface 檔要有 `field<TAB>運算元<TAB>名稱或「無關」<TAB>理由` 列，缺 → `FAIL d1 <addr> 欄位 <運算元> …`。
   - `scripts/formula-candidates.sh` 沒參數時會用 Ghidra 重產 funcs.tsv；fields.tsv 由 cc 另行產生，check-formulas 不自動產生。
3. 不放寬既有任何檢查；既有夾具案例照舊要綠。

# G6 F 模式機械比對：模擬器原生模式＋TestFNative，試點 54df00（goals/stage14.md）

驗收: { test -f scripts/node-native.sh || { echo 'FAIL node-native'; false; }; } && bash scripts/node-native.sh 54df00
規模: 大
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔（含 scripts/node-accept.sh、scripts/nodes/ 既有檔，除非白名單列出）；不刪 boundary 既有列；不縮 random 範圍或輸入域閃差異；不放寬任何驗證；不寫入遊戲目錄；不執行遊戲或 exe；筆記與程式註解不貼組語或反編譯碼；不把原版 bytes、從 exe／DBS 讀出的表寫進 repo
白名單: re/emu/emu.py, notes/re/emu.md, internal/calc/fnative_test.go, scripts/node-native.sh, re/emu/cases/0054df00.json, re/emu/native/0054df00.json, internal/calc/n_54df00.go, notes/re/iface/54df00.tsv
預期紅: node-native
執行器: codex

## 背景
- 節點驗收（node-accept、TestEmu）只比 `Node.Emu`：已 D4 的公式 callee 在上層 case 以位址樁替代，回傳值取自 case 輸入（`decisions/2026-10-09-compositional-stubs.md`）。`Node.F`（calc 正式查詢用、呼叫 callee 真 Go 函式）沒有任何機械比對；收貨時 agent 的 scratch 統計顯示多個節點 F panic 或 F≠Emu（resume.md「F 模式待追」）。
- 這張票補上 F 的比對：模擬器跑「整鏈原生」（D4 公式 callee 不樁，照原版跑），calc 用 F 比。

## 要做的
1. `re/emu/emu.py` 加原生模式（例 `--native <addr>`）：位址樁中登記簿「公式」類一律不裝、照原版跑；「排除」類與 IO 樁照裝。輸出到 `build/emu-f/<addr>.jsonl`（不碰 `build/emu/`）。
   原生模式下不會被寫的輸出（公式樁的 record／count 位置）與用不到的輸入（公式樁的 ret 位置）要能從 case 機械判斷；jsonl 每列標出哪些輸出欄要比，或另寫 mask，二擇一，寫進 `notes/re/emu.md`。
   原 case 不夠原生跑時（callee 讀的世界狀態沒放），case 可另放 `re/emu/native/<addr>.json`（覆寫／追加欄位的方式自定，寫進 emu.md），不改原 case 的輸入輸出順序。
2. `internal/calc/fnative_test.go`：`TestFNative` 讀 `build/emu-f/*.jsonl`，用 `Nodes[addr].F`（不是 Emu）重放，只比要比的欄；寫 `build/emu-f/<addr>.d3f`（cases／diff）。沒有 jsonl 就 Skip。RNG 照 TestEmu 的嚴格規則（多叫、少叫都算 diff）。
3. `scripts/node-native.sh <addr>…`：對每個位址跑原生模式→TestFNative；印 `ok node-native <addr> cases=N` 或 `FAIL node-native <addr> …`；任何一個 diff>0 或 jsonl 空就非零。
4. 試點 54df00（4 個公式樁）：F panic 或 F≠原版時，先判定是 F（或 callee 夾具上限）錯還是比對產物；F 錯就修 `n_54df00.go`，callee 的錯寫在規格疑慮、不跨白名單修。
5. 原有 `bash re/emu/run.sh --selftest-stubs` 等自測、`go test -run 'TestEmu$' ./internal/calc` 照綠。
6. 四段注入：在 n_54df00.go 的 F 路徑（不是 Emu 共用段）注入一個差一錯誤 → node-native FAIL → 副本還原 → PASS。

## 規格疑慮
寫在最終回覆；沒有就寫「無」。

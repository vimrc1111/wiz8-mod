# Y3 模擬器第三批：wcscpy、SR.DLL srClass 建構子、x87 ST0 回傳值

驗收: bash scripts/emu.test.sh
規模: 中（codex gpt-6.1-sol medium；re/emu/emu.py 為主）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔；不改 scripts/emu.test.sh；不寫入遊戲目錄；不執行遊戲；不下載任何東西；筆記不貼組語或反編譯碼；不把原版 bytes、從 exe 讀出的表寫進 repo
白名單: re/emu/emu.py, re/emu/run.sh, notes/re/emu.md, internal/calc/nodes_test.go
預期紅: emu
執行器: codex

## 背景
節點票第四波三張紅在模擬器（樁寫法見 Y2：`re/emu/emu.py` 的 vswprintf、srAssertFail 與 `--selftest-stubs`）：
- 553540：`unstubbed import 0x5eb1ec MSVCRT.DLL!wcscpy`。
- 43ba10：`unstubbed import 0x5eb6e0 SR.DLL!??0srClass@@IAE@XZ`（srClass 的 protected 預設建構子，thiscall）。
- 4efb60：原版回傳值在 x87 ST0（浮點），case 的 `out` 只能讀 eax／ecx／記憶體。

## 要做的
1. `wcscpy(dst, src)`（cdecl，UTF-16LE，含結尾 0）回傳 dst。
2. `srClass::srClass()`：先對照 build/wiz8.asm 的呼叫點，確認衍生類建構子在呼叫它之後自己寫 vtable／欄位；確認的話樁只回傳 ecx（this）、不寫記憶體，並在 notes 寫明依據（引 `build/wiz8.asm:<行>`，不貼碼）。若發現衍生類依賴基底建構子寫入的欄位，照實寫進 `## 規格疑慮`，樁照讀到的語意補上。
3. case `out` 欄位支援 `{"at":{"reg":"st0"}, "size":8}`：讀 x87 ST0，轉成 IEEE754 double，輸出其 64 位元以**有號 int64** 解讀的整數（Go 端以 `int64(math.Float64bits(x))` 比對）。確認 Go 側 TestEmu 解 JSON 時這種 >2^53 的整數不經 float64 失真；有失真就改 `internal/calc/nodes_test.go` 的解碼（只改解碼，不改比對規則）。
4. `--selftest-stubs` 再印三行：
   - `wcscpy "hello" ok`：合成呼叫 wcscpy 複製 L"hello"，讀回內容相同且回傳值＝dst 才印 ok。
   - `srClass ok`：合成 thiscall 呼叫，回傳 eax＝傳入的 ecx 才印 ok。
   - `st0 0x400c000000000000`：合成碼把 3.5 載入 ST0 後回傳，經 `out` 的 st0 讀法讀出，印 16 位 hex（無號表示）。
5. `notes/re/emu.md` 樁清單補上以上三項的語意與上限。

## 規格疑慮
做不到的寫在 `notes/re/emu.md` 末段 `## 規格疑慮`，並在最終回覆列出。

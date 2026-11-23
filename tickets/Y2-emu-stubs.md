# Y2 模擬器第二批樁：vswprintf、GetTickCount、FS 段、srAssertFail、未映射存取報位置

驗收: bash scripts/emu.test.sh
規模: 中（codex gpt-6.1-sol medium；只改 re/emu/emu.py）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔；不改 scripts/emu.test.sh；不寫入遊戲目錄；不執行遊戲；不下載任何東西；筆記不貼組語或反編譯碼；不把原版 bytes、從 exe 讀出的表寫進 repo
白名單: re/emu/emu.py, re/emu/run.sh, notes/re/emu.md
預期紅: emu
執行器: codex

## 背景
節點票（tickets/N-*.md）跑原版函式時卡在沒樁的 import 或段暫存器：
- 553540：`unstubbed import 0x5eb1f0 MSVCRT.DLL!vswprintf`（戰鬥訊息格式化，輸出不影響公式，但要能跑過）。
- 43ba10：讀 `GetTickCount`（寶物種子＝時鐘＋亂數）；另在某處 `Invalid memory read (UC_ERR_READ_UNMAPPED)`，疑為 `fs:[0]`（SEH 鏈）；目前錯誤訊息不含位置。
- 501530：`SR.DLL!?srAssertFail@@YAXPBD0J0ZZ`（引擎的 assert，cdecl 可變參數；參數依序推測為 條件字串、檔名、行號、訊息格式…，請對照 build/wiz8.asm 的呼叫點自行確認）。
既有樁的寫法見 `re/emu/emu.py`（`_ftol`、`qsort`、`--selftest-qsort` 的合成呼叫端）與 `notes/re/emu.md`「樁清單與 selftest」。

## 要做的
1. `vswprintf(buf, fmt, va_list)`（MSVCRT 舊版，無 count 參數；寬字元 UTF-16LE）：支援 `%d %i %u %x %X %c %s %S %%`，旗標 `-`、`0`、寬度、`l`/`h` 長度修飾；MSVC 寬版語意：`%s`＝寬字串、`%S`＝窄字串。回傳寫入字元數（不含結尾 0）。不支援的轉換明確報錯，不靜默。
2. `GetTickCount`：回傳值由 case 控制（case 檔可給 `"tick": <u32>` 或把它當一個輸入；預設 0），每次呼叫回同一值，寫進 notes。
3. FS 段：映射一塊 TIB，`fs:[0]` 讀得到（初值 0xffffffff＝SEH 鏈尾），`fs:[0x18]` 指向 TIB 自己；寫入 `fs:[0]` 可行。
4. `srAssertFail`：模擬停止、exit 非 0，stderr 印 `srAssertFail stop <檔名>:<行號>`（檔名讀第二個參數的 C 字串、行號第三個）。節點票的 case 要避開 assert 路徑；assert 專屬基本塊由節點票登記 cov-exempt。
5. 所有未映射讀寫／取指的錯誤訊息加 `unmapped eip=0x<8hex> addr=0x<8hex>`（取到就印；在 hook 裡記最後 EIP 即可）。
6. `run.sh --selftest-stubs`（合成呼叫端寫在 arena，不用 exe bytes），stdout 依序印：
   - `vswprintf "<結果>"`：fmt＝`L"hp=%d x=%x n=%S w=%s %04d%%"`，參數 −7、0x1f、窄 "ab"、寬 L"AB"、42 → 結果須為 `hp=-7 x=1f n=ab w=AB 0042%`。
   - `GetTickCount 12345`：case 設 12345 後由合成呼叫端呼叫 import 取得。
   - `fs0 ok`：合成碼 `mov eax, fs:[0]` 得 0xffffffff，`mov fs:[0], ebx` 後再讀得 ebx。
   - `srAssertFail stop foo.cpp:77`：合成呼叫 srAssertFail("x", "foo.cpp", 77, "msg")，在子行程或捕捉例外後印這行（selftest 本身仍 exit 0）。
   - `unmapped eip=0x…`：合成碼讀 0x00000004，捕捉錯誤後印該訊息的那一行。
7. `notes/re/emu.md` 樁清單補上以上各樁的語意與上限。

## 規格疑慮
做不到的寫在 `notes/re/emu.md` 末段 `## 規格疑慮`，並在最終回覆列出。

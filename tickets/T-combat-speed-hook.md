# T 實作：戰鬥中怪物移動倍率（stage9 9.3）

驗收: bash hook/ddraw/speed.test.sh && bash scripts/check-sigs.sh && bash hook/ddraw/build.sh && bash hook/ddraw/check.sh
規模: 小（3 處讀取改址＋一支背景執行緒＋純函數）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 hook/ddraw/speed.test.sh、scripts/check-sigs.sh 與白名單外任何檔；不寫入遊戲目錄；不執行遊戲；不寫 `Wiz8.CFG`、不寫遊戲記憶體 0x006850ff；不加 CRT 或第三方依賴；不放寬或跳過測試；筆記不貼反組譯碼（只寫位址、前 16 bytes、行為，用自己的話）
白名單: hook/ddraw/speed.h、hook/ddraw/proxy.c、hook/ddraw/build.sh、notes/re/speed.md
預期紅: speed-rule
執行器: codex

## 背景
Wizardry 8（Wiz8.exe v1.2.4，i386）中文化＋現代相容 mod。我們的 ddraw 代理 `hook/ddraw/proxy.c` 被遊戲載入，以簽章比對後改寫遊戲碼（`install()`／`hook_call()`；簽章由 build.sh 從 `notes/re/*.md` 表格產生 `hooks.h`，`scripts/check-sigs.sh` 每次 CI 對真 exe 驗）。跑驗收前 `export WIZ8_EXE=$PWD/build/Wiz8.exe SYS_DDRAW=$PWD/build/sys-ddraw.dll`（沙箱看不到 /mnt/c）。材料：`build/wiz8.asm`（objdump -d -M intel）、`build/Wiz8.exe`（檔內偏移＝VA−0x400000）。戰鬥計時的既有 RE 見 `notes/re/combat-timing.md`。

已確認的事實（cc 對過 asm 與社群工具 Wiz8Fast）：
- `0x006850ff`：f32，遊戲選項「怪物移動速度」（預設 2.5，存於 `Wiz8.CFG` 檔內偏移 0x4f；社群工具 Wiz8Fast 就是改這 4 bytes 讓怪物走得更快）。
- 執行期讀它的地方只有三處：`0x004546fb`、`0x00454720`（`fmul dword [0x6850ff]`，乘進怪物位移量）、`0x004554dd`（`mov eax,[0x6850ff]`，抄進物件 +0x124）。另有選項畫面 `0x005a6f71`／`0x005a6f7f` 讀寫它——**不動**。
- 戰鬥旗標：`BYTE [0x00683f94]` 非 0＝戰鬥中。

## 要做
1. `hook/ddraw/speed.h`：純函數 `float speed_eff(float base, int pct, int in_combat)`：
   - `in_combat` 為 0 → 回 `base`（不動探索）；
   - 否則回 `base * clamp(pct) / 100`，`clamp` 把 pct 限在 [10, 5000]；
   - `base` 非有限值或 ≤0 → 回 `base` 原值（不放大壞值）。
   規則以 `hook/ddraw/speed.test.sh` 為準（cc 寫，你不改）。
2. `notes/re/speed.md`：三處讀取各一列 `| 0x<VA> | <前 16 bytes> | 行為 |`（格式同 `notes/re/draw.md`，check-sigs 會驗）；另寫 +0x124 這份副本何時被抄、誰讀它（若它在戰鬥開始前就被抄走、之後不再更新，寫明影響）。
3. `proxy.c`：
   - 倍率：讀遊戲目錄 `wiz8mod.ini` 的 `[combat] monster_move_pct`（`GetPrivateProfileIntA`，KERNEL32；路徑用 exe 所在目錄）；檔案或鍵不存在 → 400。啟動時 log 一行 `speed monster_move_pct=<值>`。
   - 新增我方 4-byte 對齊的 `volatile float speed_move`。三處讀取把指令裡的位址運算元改指 `&speed_move`（只改那 4 bytes，指令長度不變）；簽章不符照慣例 log `sig-mismatch speed <位址>` 不裝，三處要嘛全裝要嘛全不裝。
   - 背景執行緒（DllMain 裡 `CreateThread`，不等它）每 50 ms：`speed_move = speed_eff(*(float *)0x006850ff, pct, *(BYTE *)0x00683f94)`。裝 patch 前先寫一次初值，保證任何時刻 `speed_move` 都是有效值。
   - 第一次觀察到戰鬥旗標由 0 變非 0 時 log 一行 `speed combat base=<原值*1000> eff=<有效值*1000>`（整數，不用 CRT 的浮點格式化），之後不再印。
4. `build.sh`：照現有 Y 群的做法把三處簽章從 `notes/re/speed.md` 產進 hooks.h。

## 驗收（CI 驗不到的部分）
執行期由 cc 實機驗：log 有 `speed monster_move_pct=`、進戰鬥有 `speed combat`、遊戲活到時限、遊戲目錄 diff 乾淨。手感由 MG 試玩。

## 交付回覆
列出改了哪些位元組（位址、原 4 bytes、新指向）；+0x124 的調查結果；附「規格疑慮」段，沒有就寫「無」。

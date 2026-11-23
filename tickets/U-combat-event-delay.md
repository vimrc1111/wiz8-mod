# U RE＋實作：戰鬥事件固定停頓倍率（stage9 第二輪）

驗收: bash hook/ddraw/speed.test.sh && bash scripts/check-sigs.sh && bash hook/ddraw/build.sh && bash hook/ddraw/check.sh
規模: 中（codex gpt-6-astra medium；先補 RE 列舉寫入點，再改立即值）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 hook/ddraw/speed.test.sh、scripts/check-sigs.sh 與白名單外任何檔；不寫入遊戲目錄；不執行遊戲；不寫 `Wiz8.CFG`；不動 T 已做的三處 0x6850ff 讀取；不加 CRT 或第三方依賴；不放寬或跳過測試；筆記不貼反組譯碼或反編譯碼（只寫位址、前 16 bytes、常數、行為，用自己的話）
白名單: hook/ddraw/speed.h、hook/ddraw/proxy.c、hook/ddraw/build.sh、notes/re/speed.md
預期紅: delay-rule
執行器: codex

## 背景
Wizardry 8（Wiz8.exe v1.2.4，i386）mod。`hook/ddraw/proxy.c` 是遊戲載入的 ddraw 代理，以簽章比對後改寫遊戲碼；簽章由 build.sh 從 `notes/re/*.md` 表格產生 `hooks.h`，`scripts/check-sigs.sh` 每次 CI 對真 exe 驗。上一票 T 已把戰鬥中怪物移動做成倍率（`speed.h` 的 `speed_eff`、`proxy.c` 的 `install_speed`、`notes/re/speed.md`），照它的寫法擴充。
跑驗收前 `export WIZ8_EXE=$PWD/build/Wiz8.exe SYS_DDRAW=$PWD/build/sys-ddraw.dll`（沙箱看不到 /mnt/c）。材料：`build/wiz8.asm`（objdump -d -M intel）、`build/Wiz8.exe`（檔內偏移＝VA−0x400000）、`build/decomp/`（Ghidra 反編譯，只讀）。既有 RE：`notes/re/combat-timing.md:43`、`:156`、`:169`。

已知（S 報告，cc 未逐條重驗）：攻擊函式 0x53e250 在 0x53eeda、攻擊結果路徑 0x543110 在 0x54373b 以 GetTickCount 記事件物件 +0x34 時戳並設 +0x30＝800；戰鬥結束 0x4e9f90 在 0x4ea0c5 記時戳並設 1500；共同消費端 0x52e01b 比較經過時間與 +0x30，未到就返回。

## 要做
1. RE：列出所有在戰鬥路徑上把**固定毫秒常數**寫進這類事件物件 +0x30 的指令（至少上述三處；另找施法、怪物攻擊、閃避／受擊等是否有同形寫入）。每處在 `notes/re/speed.md` 加一列 `| 0x<指令VA> | <前 16 bytes> | 行為 |`（簽章從該指令起算），寫明原常數、所屬函式、為何判定在戰鬥路徑上。非戰鬥路徑也寫這種事件的，列出但不改。
2. `speed.h`：純函數 `unsigned delay_scaled(unsigned ms, int pct)`：回 `ms * 100 / clamp(pct)`，clamp 同 `speed_eff` 的 [10, 5000]，整數運算向下取整、不溢位（ms 上限 600000）。規則以 `speed.test.sh` 的 delay-rule 為準。
3. `proxy.c`：
   - 倍率讀 `wiz8mod.ini` 的 `[combat] event_speed_pct`，缺則 300；啟動 log 一行 `speed event_speed_pct=<值>`。
   - 只改指令裡的 32 位元立即值為 `delay_scaled(原值, pct)`（指令長度不變；若某處立即值不是 imm32 或常數不在指令內，寫進規格疑慮、不改那處）。安裝時算一次即可，不需執行緒。簽章不符 log `sig-mismatch delay <位址>`，整組不裝；與 T 組互不影響。
   - 每處改寫 log 一行 `delay 0x<位址> <原值>-><新值>`。
4. `build.sh`：新列用新的 kind（例 `D`），照 T 的 `#ifdef` 作法產進 hooks.h。

## 驗收（CI 驗不到的部分）
cc 實機：log 有 `speed event_speed_pct=` 與各 `delay` 列、遊戲活到時限、遊戲目錄 diff 乾淨。手感由 MG 試玩（攻擊後的停頓是否變短、戰鬥音效／語音是否被切掉）。

## 交付回覆
改了哪些立即值（位址、原值、新值＠pct=300）；找到但沒改的寫入點與理由；附「規格疑慮」段，沒有就寫「無」。

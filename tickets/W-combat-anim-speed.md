# W 實作：戰鬥中怪物攻擊／施法動畫倍率（stage10 A2，V 候選 1）

驗收: bash hook/ddraw/speed.test.sh && bash scripts/check-sigs.sh && bash hook/ddraw/build.sh && bash hook/ddraw/check.sh
規模: 小（一處 call 改址＋一支 wrapper＋純函數）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 hook/ddraw/speed.test.sh、scripts/check-sigs.sh 與白名單外任何檔；不寫入遊戲目錄；不執行遊戲；不動 T、U 已裝的 patch；不改全域 timer 0x6598b8、不改 busy、不改 FPS 資料；不加 CRT 或第三方依賴；不放寬或跳過測試；筆記不貼反組譯碼或反編譯碼
白名單: hook/ddraw/speed.h、hook/ddraw/proxy.c、hook/ddraw/build.sh、notes/re/speed.md
預期紅: anim-rule
執行器: codex

## 背景
Wizardry 8（Wiz8.exe v1.2.4，i386）mod。`hook/ddraw/proxy.c` 是 ddraw 代理，簽章比對後改寫遊戲碼；簽章由 build.sh 從 `notes/re/*.md` 表格產生 hooks.h，`scripts/check-sigs.sh` 對真 exe 驗。T（怪物移動倍率，`speed_eff`、`install_speed`）與 U（事件停頓，`delay_scaled`、`install_delay`）已做，照它們的寫法擴充。
跑驗收前 `export WIZ8_EXE=$PWD/build/Wiz8.exe SYS_DDRAW=$PWD/build/sys-ddraw.dll`。材料：`build/wiz8.asm`、`build/Wiz8.exe`（檔內偏移＝VA−0x400000）。RE 依據：`notes/re/combat-anim.md`（候選 1）。

已知（cc 對過 asm）：
- 0x4a6e20 是動畫時間→影格更新者，thiscall（ecx＝怪物物件 B），一個 float 參數 m，`ret 4`。
- 怪物更新 0x4c2100 在 0x4c2b5d 用 5-byte `call rel32` 呼叫它（前一指令 `mov ecx,esi`，返回 0x4c2b62）。
- 序列：A＝DWORD[B+0x1d8]，s＝signed BYTE[A+0xa4]。
- 戰鬥旗標：BYTE[0x683f94]≠0。

## 要做
1. `speed.h`：純函數 `float anim_m(float m, int pct, int in_combat, int seq)`：seq 屬 {9,0xa,0xb,0xc,0xd,0xe,0xf,0x10,0x11,0x19} 時回 `speed_eff(m, pct, in_combat)`，否則回 m。規則以 `speed.test.sh` 的 anim-rule 為準。
2. `notes/re/speed.md`：新增一列 `| 0x004c2b5d | <前 16 bytes> | 行為 |`，新 kind（例 `A`），寫明原 call 目標 0x4a6e20、呼叫慣例、序列集合來源（combat-anim.md）。
3. `proxy.c`：
   - 倍率讀 `wiz8mod.ini` 的 `[combat] anim_speed_pct`，缺則 200；啟動 log 一行 `speed anim_speed_pct=<值>`。
   - wrapper 以 `__fastcall`＋不用的 edx 參數模擬 thiscall（callee 清 4 bytes），取 B→A→seq，呼叫原 0x4a6e20 時傳 `anim_m(m, pct, BYTE[0x683f94], seq)`。A 為 NULL 時直通原 m。
   - 只改 0x4c2b5e 起 4 bytes 的 rel32 指向 wrapper；簽章不符 log `sig-mismatch anim 0x004c2b5d` 不裝；裝好 log `hooked anim 0x004c2b5d`。pct＝100 時也裝（wrapper 直通），行為不變。
4. `build.sh`：新 kind 照 D 的 `#ifdef` 作法產進 hooks.h。

## 已知限制（不用處理）
- 戰鬥進出若在同一序列中途，小數相位可能跳不到一格（V 候選 1 說明）；本票不做序列邊界鎖定。
- 近戰傷害由外層輪詢「過半格」放行，倍率大時傷害可能延到動畫末格（V 推測）；手感由 MG 試玩。

## 驗收（CI 驗不到的部分）
cc 實機自動 run：log 有 `speed anim_speed_pct=`、`hooked anim`、無 sig-mismatch、活到時限、遊戲目錄 clean。手感 MG 試玩。

## 交付回覆
改了哪些 bytes（位址、原 rel32、新目標）；規格疑慮段，沒有就寫「無」。

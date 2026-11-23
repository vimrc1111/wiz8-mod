# B1 實作：遊戲內選單（內掛）第一票＝戰鬥倍率即時調整（goals/stage11.md、goals/stage12.md B1）

驗收: bash hook/ddraw/menu.test.sh && bash hook/ddraw/speed.test.sh && bash scripts/check-sigs.sh && bash hook/ddraw/build.sh && bash hook/ddraw/check.sh
規模: 中（純函數狀態機＋視窗訊息子類化＋畫面疊加＋即時套用）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 hook/ddraw/menu.test.sh、hook/ddraw/speed.test.sh、scripts/check-sigs.sh 與白名單外任何檔；不寫入遊戲目錄；不執行遊戲；不改既有 T／U／W patch 的位址與簽章；不加 CRT 或第三方依賴；不放寬或跳過測試；筆記不貼反組譯碼或反編譯碼；作弊類功能不做
白名單: hook/ddraw/menu.h、hook/ddraw/proxy.c、hook/ddraw/speed.h、hook/ddraw/build.sh、notes/re/menu.md
預期紅: menu-rule
執行器: codex

## 背景
Wizardry 8（Wiz8.exe v1.2.4，i386）mod。`hook/ddraw/proxy.c` 是 ddraw 代理 DLL（DllMain 取真 ddraw 的匯出並轉發；簽章比對後改寫遊戲碼；log 走 `log_line`）。
跑驗收前 `export WIZ8_EXE=$PWD/build/Wiz8.exe SYS_DDRAW=$PWD/build/sys-ddraw.dll`。材料：`build/wiz8.asm`、`build/Wiz8.exe`（檔內偏移＝VA−0x400000）。
現況：三個倍率只在啟動讀 `wiz8mod.ini` `[combat]`（`speed_init`）。
- `speed_pct`：`speed_worker` 每 50ms 重算，改了就生效。
- `anim_speed_pct`：`anim_update` 每次呼叫讀，改了就生效。
- `event_speed_pct`：`install_delay` 只在安裝時把五個 imm32 改成 `delay_scaled(原值,pct)`，要即時生效得保存原值、改值時重寫（同樣的 VirtualProtect／FlushInstructionCache 流程）。

## 要做
1. `menu.h`（純函數，主機 gcc 可編，不 include windows.h）：介面與規則以 `menu.test.sh` 為準。
   - `MenuItem { const char *key; int def; …標籤欄位自訂 }`、`menu_items[MENU_N]`（項目表；之後加項＝加一列）、`MenuState { int open, sel, dirty; int val[MENU_N]; }`、`enum { MK_OTHER, MK_TOGGLE, MK_UP, MK_DOWN, MK_LEFT, MK_RIGHT, MK_RESET, MK_RESET_ALL, MK_ESC }`、`int menu_key(MenuState *, int key)`。
   - 標籤用中文：怪物移動、事件停頓、攻擊動畫（字必須在 `atlas.h` 的 `atlas_chars` 內；不在就在交付回覆列出，不改 atlas）。
2. `proxy.c`：
   - `speed_init` 的三個預設值改從 `menu_items` 取（單一來源）。
   - 熱鍵：`[menu] hotkey_vk`，缺則 0x79（F10）。啟動 log `menu hotkey_vk=<值>`。
   - 按鍵：在背景執行緒等到遊戲主視窗出現後，以 `SetWindowLongA(GWL_WNDPROC)` 子類化（不改遊戲 bytes）；WM_KEYDOWN／WM_SYSKEYDOWN 映射成 MK_*（方向鍵、Home＝本項原版、End＝全部原版、Esc、熱鍵，其餘 MK_OTHER），`menu_key` 回 1 的訊息不傳給原 WndProc（對應的 WM_KEYUP／WM_CHAR 也吞掉）。裝好 log `hooked menu wndproc`；失敗 log `ERROR menu wndproc <GetLastError>`。
     若遊戲讀鍵不走視窗訊息（DirectInput／GetAsyncKeyState），在 `notes/re/menu.md` 寫查到的依據（位址＋用途，不貼碼），並改成在該處擋；這情況下新 hook 照既有慣例在 `notes/re/*.md` 表格登記簽章（build.sh 產進 hooks.h），對不上 log `sig-mismatch menu <位址>` 不裝。
   - 疊加：選單開著時，在遊戲每幀畫完、送上畫面之前，把半透明底框＋三列「標籤 數值%」（游標列反白）＋一列操作提示畫進要呈現的 surface；字形用 `atlas.h` 的 1-bit 字形（像素格式從 surface 描述取，至少支援 16-bit 565／555）。
     呈現路徑先查清楚再選：proxy 包裝 `DirectDrawCreate*` 回傳物件、在 surface vtable 的 Flip／Blt（到 primary）前畫，或遊戲自己的每幀呈現函式（`notes/re/shot.md` 的 srGERD lockBuffer 物件可做線索）。選定路徑與依據寫進 `notes/re/menu.md`。裝好 log `hooked menu overlay <路徑名>`；選單第一次開啟 log `menu open`。
   - 即時套用：值改變時立刻寫入 `speed_pct`／`event_speed_pct`／`anim_speed_pct`；event 重寫五個 imm32（由保存的原值重算，不由目前值重算），log `delay live pct=<值>`。選單關閉且 dirty 時 `WritePrivateProfileStringA` 寫回 `[combat]` 三鍵，log `menu saved`。
   - 執行緒安全：WndProc 與 worker 不同執行緒，共享值用對齊的 32-bit 寫入（或 Interlocked）。
3. `speed.h`：只在需要共用 clamp 時才動；`speed.test.sh` 必須照舊綠。
4. `notes/re/menu.md`：熱鍵、按鍵對照、呈現路徑依據、ini 鍵、已知限制。

## 已知限制（不用處理）
- 全螢幕獨占模式下視窗訊息與畫面行為由實機確認（MG 試玩）。
- 非戰鬥時改倍率只是存值，進戰鬥才看得到效果。

## 驗收（CI 驗不到的部分）
cc 實機自動 run：log 有 `menu hotkey_vk=`、`hooked menu wndproc`、`hooked menu overlay`，無 sig-mismatch，活到時限，遊戲目錄 clean。開選單、調倍率、畫面與手感 MG 試玩。

## 交付回覆
呈現路徑與依據；擋鍵方式；改了哪些遊戲 bytes（應只有既有 delay 五處的即時重寫）；atlas 缺字；規格疑慮段，沒有就寫「無」。

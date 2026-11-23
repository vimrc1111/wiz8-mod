# P13 實作：探索中隊伍移動倍率 party_move_pct（goals/stage13.md 13.2–13.3）

驗收: bash hook/ddraw/speed.test.sh && bash hook/ddraw/menu.test.sh && bash scripts/check-sigs.sh && bash hook/ddraw/build.sh && bash hook/ddraw/check.sh
規模: 小（一處 5-byte detour＋naked stub＋純函數＋選單一列）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改驗收腳本（含 speed.test.sh、menu.test.sh）與白名單外任何檔；不寫入遊戲目錄；不執行遊戲；不改既有 T／D／A／M／W／N patch 的位址與簽章；不改共用時間物件 T+0x28、不改 0x00603aac／0x00603ab0／0x00609ea4 常數、不碰轉向；不加 CRT 或第三方依賴；不放寬或跳過測試；筆記不貼反組譯碼或反編譯碼
白名單: hook/ddraw/speed.h、hook/ddraw/proxy.c、hook/ddraw/menu.h、hook/ddraw/build.sh、notes/re/party-move.md
預期紅: party-rule、menu-rule
執行器: codex

## 背景
Wizardry 8（Wiz8.exe v1.2.4，i386）mod。`hook/ddraw/proxy.c` 是 ddraw 代理，簽章比對後改寫遊戲碼；簽章由 build.sh 從 `notes/re/*.md` 表格產生 hooks.h，`scripts/check-sigs.sh` 對真 exe 驗。T（怪物移動）、D（事件停頓）、A（攻擊動畫）已做，照它們的寫法擴充；選單（B1）項目表在 `menu.h`。
跑驗收前 `export WIZ8_EXE=$PWD/build/Wiz8.exe SYS_DDRAW=$PWD/build/sysddraw.dll`。材料（worktree 內實體檔）：`build/wiz8.asm`、`build/Wiz8.exe`（檔內偏移＝VA−0x400000）。RE 依據：`notes/re/party-move.md`（只用其「唯一建議改寫點」）。

已知（cc 對過 exe 與 asm）：
- 0x0041f78f：3-byte 指令把 x87 頂端（引擎本次 dt）存到隊伍移動物件 P 的 +0x14（esi＝P），接著 2-byte 指令把 ebp 複製到 eax；兩者共 5 bytes，下一指令在 0x0041f794。前 16 bytes `d95e148bc58b088d5634890a8b480489`。
- P+0x14 是隊伍自己的 dt 副本，後續輸入、限速、碰撞、後處理都讀它；共用時間物件不受影響。
- 戰鬥旗標：BYTE[0x683f94]≠0。

## 要做
1. `speed.h`：純函數 `float party_dt(float dt, int pct, int in_combat)`：in_combat 非 0、dt 非有限或 ≤0 時回 dt；否則 `dt*clamp(pct,10,5000)/100`。規則以 `speed.test.sh` 的 party-rule 為準。
2. `notes/re/party-move.md`：在「候選 hook 簽章」表把 0x0041f78f 那一列的行為欄開頭改成新 kind（例 `Q；`），寫明覆蓋 5 bytes、返回 0x0041f794、原兩條指令在 stub 內重做；其他列不動。先自己確認 0x0041f794 之後到下一個設旗標指令前不讀 EFLAGS（stub 會動旗標），結論寫進該列。
3. `proxy.c`：
   - 倍率讀 ini 的 `party_move_pct`（與其他三項同段、同讀法），缺則 100；啟動 log 一行 `speed party_move_pct=<值>`。
   - naked stub：照原樣存 dt 到 [esi+0x14]；若 BYTE[0x683f94]==0，以一個由 `party_dt` 規則算好的 float 係數（menu apply 與啟動時更新，aligned 4、volatile）縮放 [esi+0x14]；重做原第二條指令；jmp 0x0041f794。x87 堆疊進出平衡，不動 esi／ebp／其他通用暫存器（eax 照原指令被設）。係數的計算必須呼叫 `party_dt`（例 `party_dt(1.0f, pct, 0)`），不另寫一份規則。
   - 只在 16 bytes 簽章相符時把 0x0041f78f 起 5 bytes 換成 `jmp rel32`；不符 log `sig-mismatch party 0x0041f78f` 不裝；裝好 log `hooked party 0x0041f78f`。pct＝100 時也裝（係數 1.0），行為不變。
4. `menu.h`：項目表最後加一列 `{"party_move_pct", 100, L"隊伍移動", MENU_MIN, MENU_MAX, MENU_STEP, MENU_APPLY(party)}`；`menu_apply_party` 即時更新係數，寫回 ini 走既有 dirty 流程。疊加框四列要畫得下（看 menu 的高度計算，必要時改常數）。
5. `build.sh`：新 kind 對到 `notes/re/party-move.md`，照 A 的 `#ifdef` 作法產進 hooks.h。

## 已知限制（不用處理）
- 垂直物理與距離累積（負重等）會一起被縮放；手感、穿牆、漏觸發由 MG 試玩。
- 轉向不在本票。

## 驗收（CI 驗不到的部分）
cc 實機自動 run：log 有 `speed party_move_pct=`、`hooked party`、無 sig-mismatch、活到時限、遊戲目錄 clean。

## 交付回覆
改了哪些 bytes（位址、原 bytes、新 jmp 目標）；EFLAGS 確認結論；規格疑慮段，沒有就寫「無」。

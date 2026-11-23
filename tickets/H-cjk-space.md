# H 實作：訊息佇列中英間距（stage5，MG 2026-10-07 試玩）

驗收: bash hook/ddraw/space.test.sh && bash hook/ddraw/wrap.test.sh && bash hook/ddraw/build.sh && bash hook/ddraw/check.sh
規模: 小（一個純函數＋改一個既有入口 hook）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 hook/ddraw/space.test.sh 與白名單外任何檔；不寫入遊戲目錄；不執行遊戲；不加 CRT 匯入或第三方依賴；不放寬或跳過測試；不貼反編譯碼或遊戲原文
白名單: hook/ddraw/space.h、hook/ddraw/proxy.c
預期紅: space-rule
執行器: codex

## 背景
Wizardry 8 中文化，ddraw 代理 `hook/ddraw/proxy.c` 以 detour 改寫遊戲函式。英文名字（角色、怪物、法術、物品）留英文，由遊戲在執行期用模板（`%s`）拼進中文句子；
同一個 `%s` 有時代入英文名、有時代入中文（技能名、「隊伍」），譯文模板無法一律加空白 → 戰鬥 log 有的「Sprite施放Iceball」黏在一起、有的「受到 5 點」有空白。
修法：在訊息佇列入口把整句補間距，之後的量寬、斷行、入列都看到補好的字串。

已知（cc 已核對）：
- 0x0058ac00＝訊息佇列分行入口（`notes/re/wrap.md`），cdecl 五參數；**第 2 個參數（入口時 `[esp+8]`）是 UTF-16 字串指標**，第 1 個是字型調色盤索引。
  該函式在入口後才建 0x2010 位元組區域緩衝（`mov eax,0x2010; call __chkstk`），所以其內部行緩衝上限 4096 字。
- 入列 0x0058af60 以 `wcslen`＋`malloc`＋`wcscpy` **複製**字串，不留呼叫端指標 → 換成我們的緩衝是安全的，呼叫返回後緩衝可重用。
- 該入口已有 hook：`WRAP_ENTRY(M, 0x0058ac00)`（proxy.c，`wrap_enter_M` 記 `wrap hit` 後走跳板）。`WrapFrame` 的佈局見 proxy.c 的 `WRAP_THUNK`（pushfl／pushal 之後；入口時原堆疊在 frame 之後）。

## 要做
1. `hook/ddraw/space.h`（新檔，`#pragma once`，不准用 libc；proxy.c 與 space.test.sh 共用）：
   `static inline int space_cjk(unsigned short *dst, int cap, const unsigned short *src)`
   - 漢字／假名／諺文（U+3040–U+9FFF、U+AC00–U+D7AF、U+F900–U+FAFF）與 ASCII 英數字 `[0-9A-Za-z]` **直接相鄰**時，兩字之間補一個 U+0020；其他一律照抄（全形標點 U+3000–U+303F、U+FF00–U+FFEF、ASCII 標點、LF、已有空白都不補）。
   - 回傳輸出長度（不含 NUL）；放不下（含 NUL 超過 cap）回 -1，且不寫 `dst[cap]` 以後。space.test.sh 是規格的一部分，衝突以它為準。
2. proxy.c：`wrap_enter_M`（只改 M，C／D 不動）在走跳板前，若字串含 CJK（`wrap_has_cjk`），以 `space_cjk` 寫進一塊 static 緩衝（4096 字），成功且長度有變才把 `[esp+8]` 換成緩衝指標；失敗或沒變就照原指標。
   - 重入：入口被重入（遞迴或別的執行緒）時不得覆蓋使用中的緩衝；用旗標或 InterlockedExchange，搶不到就照原指標，不等。
   - 第一次真的換掉指標時 log 一行 `space hit 0x0058ac00`（之後不再印）。
3. 純 ASCII 字串的行為與原版完全相同（不進 space_cjk）。

## 驗收（CI 驗不到的部分）
cc 實機驗：戰鬥 log「名字 施放 法術」有空白、`space hit` 有出現、遊戲活到時限、遊戲目錄 diff 乾淨。

## 交付回覆
附「規格疑慮」段；沒有就寫「無」。

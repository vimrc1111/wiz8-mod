# F 實作＋逆向：中文任意斷行（stage5 5.3）

驗收: bash hook/ddraw/wrap.test.sh && bash scripts/check-sigs.sh && bash hook/ddraw/build.sh && bash hook/ddraw/check.sh
規模: 大（逆向找出遊戲的量寬／換行函式＋detour）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 hook/ddraw/wrap.test.sh、scripts/check-sigs.sh 與白名單外任何檔；不寫入遊戲目錄；不執行遊戲；不加 CRT 或第三方依賴；不放寬或跳過測試；筆記不貼反編譯碼或遊戲原文（只寫位址、前 16 bytes、行為，用自己的話）
白名單: hook/ddraw/wrap.h、hook/ddraw/proxy.c、hook/ddraw/build.sh、notes/re/wrap.md
預期紅: wrap-rule
執行器: codex

## 背景
Wizardry 8（Wiz8.exe v1.2.4，i386，`/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8/Wiz8.exe`，唯讀）中文化。字串全程 UTF-16。
我們的 ddraw 代理 `hook/ddraw/proxy.c` 被遊戲載入，以 detour 改寫遊戲函式（`install()`；簽章由 build.sh 從 `notes/re/*.md` 表格產生 `hooks.h`，check-sigs.sh 每次 CI 對真 exe 驗）。
已知事實見 `notes/re/draw.md`：0x00407650／0x004077d0 格式化繪字、**0x00407010 字串寬度**（short* UTF-16, 字型槽）、字表與字形記錄。中文字形寬＝字高（全形）。
問題：遊戲換行只在空白斷，中文句子沒有空白 → 整句當成一個詞，溢框或整句被推到下一行。戰鬥訊息一句會分多次畫（實機 run 045348），所以不能在繪字層處理，要在遊戲自己的版面／換行邏輯。
反組譯：`build/wiz8.asm`（objdump -d -M intel，已附）；也可自行對 exe 跑 objdump。0x00407010 有 144 處呼叫。
已看過、**不是**換行的：0x0058fdd0（對白框取詞：從某位置略過空白、取到下一個空白為止的一個詞，回傳起訖與 x 範圍；疑似點擊關鍵字用，5.5 要用，**不准改壞它的行為**）。

## 要做
1. `hook/ddraw/wrap.h`：`wrap_can_break(prev, next)` 回 1＝可在兩字之間換行。規則（wrap.test.sh 是規格的一部分）：
   - 任一邊是空白 → 可；
   - 任一邊是 CJK（含 U+2E80–U+9FFF、U+3000–U+303F、U+3040–U+30FF、U+AC00–U+D7AF、U+F900–U+FAFF、U+FF00–U+FFEF）→ 可，**但** next 是避頭標點（，。、！？：；」』）〉》… 等全形收尾標點）或 prev 是避尾標點（「『（〈《）→ 不可；
   - 其餘（拉丁字母、數字、`%s` 之間）→ 不可。
2. 逆向：找出遊戲所有「依寬度決定換行」的函式（至少：一般訊息框、戰鬥訊息、對白框、物品／技能說明文字）。每支在 `notes/re/wrap.md` 表格列一行 `| 位址 | 前 16 bytes | 行為 |`（格式同 draw.md，check-sigs.sh 會驗），並寫它在哪些畫面用、怎麼找到的（呼叫鏈位址）。
3. detour：讓這些函式在斷行時改用 `wrap_can_break`（可整支取代或在找斷點處改寫，由你判斷；簽章不符照慣例 log `sig-mismatch` 不裝）。build.sh 照現有 H／F／P 的作法把新簽章產進 hooks.h（讀 notes/re/wrap.md）。
   純英文字串的斷行結果必須與原版相同（規則對純 ASCII 只在空白斷，與原版等價——若原版還有別的斷點規則，如連字號，要保留並在筆記寫明）。
4. proxy.c 每支換行 detour 第一次被呼叫時 log 一行 `wrap hit <位址>`（之後不再印），供實機判定有沒有走到。

## 驗收（CI 驗不到的部分）
執行期由 cc 實機驗：中文長說明與戰鬥訊息在框內斷行、`wrap hit` 有出現、遊戲活到時限、遊戲目錄 diff 乾淨。交付時列出：每支找到的函式、證據（呼叫鏈）、哪些畫面你**不確定**有涵蓋。

## 交付回覆
附「規格疑慮」段；沒有就寫「無」。

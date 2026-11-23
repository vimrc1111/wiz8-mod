# I12 掉落查表：物品從哪來、怪物掉什麼（goals/stage16.md I12）

驗收: { bash scripts/loot-where.test.sh 2>&1 | tail -n 20 | grep -q '^PASS loot-where' || { echo 'FAIL loot-where'; false; }; } && { python3 -I scripts/i9-party.py selftest >/dev/null || { echo 'FAIL i9-selftest'; false; }; }
規模: 小
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔；不放寬或刪既有測試；不寫入遊戲目錄；不執行遊戲或 exe；不把原版 bytes、從 DBS 讀出的表、名稱或查詢結果寫進 repo（只在執行時印到 stdout）；註解不貼組語或反編譯碼
白名單: scripts/loot-where.py, scripts/loot-where.test.sh, notes/calc.md:263
預期紅: loot-where
執行器: codex

## 背景
- MG 2026-10-13：「只要能知道什麼武器在哪裏打或是什麼寶箱會出，就沒有什麼問題了，就是刷而已」。
- 格式全在 `notes/re/loot.md`（P5，已驗證）：ITEMS（0x10d／筆，名稱 UTF-16LE +0）、ITEMTABLES（表數 @0xf04，表 t 起點 0xf08+497t，名稱 +0 到第一個 NUL，40 槽 @+0x104+5j：u16 類別／非空、u16 item、u8 w；+0x1cc 等級篩選旗標、+0x1cd 抽取數骰、+0x1d5 金錢骰）、MONSTERS（0x297／筆，名稱 UTF-16LE +0，8 槽 @+0x1f3+10j：u8 kind、u16 類別／非空、u16 ref、u8 c、4-byte 數量骰；kind 0＝item，kind 1＝0-based 表）、NPC.DBS（可變長；+0x64 u32 d＝死亡池 1-based，d=0 或 d≥表數不載；尾表條件與 6-byte 貨項見 loot.md:25）。
- 骰 `(i16 Z, u8 N, u8 S)`；怪物槽接受 `R(100)≤c` → 近似機率 min(c+1,100)/100（loot.md 怪物掉落段）。
- 資料位置：`build/data-x/DATABASES/`（使用者自有原版解出；CI 副本沒有就 SKIP）。
- cc 原型（scratch，可參考不必照抄）已確認：34 在表 34、50；86 在表 54、111，怪 283 kind 1 ref 54 c=10。

## 要做的
1. `scripts/loot-where.py`（Python 標準庫，`python3 -I`）：
   - `item <DATABASES> <id|名稱子字串>`：列出含此物品的每張表（表號、名稱、該槽 w、表內非空槽數與 w 總和、+0x1cc 旗標、抽取數骰），以及引用它的來源：怪物（kind 0 直接或 kind 1 經表；印怪物 id、名稱、槽 c、近似機率、數量骰）、NPC 死亡池（NPC id、名稱、表號）、NPC 商店貨單（NPC id、名稱、量、旗標）。名稱子字串比對不分大小寫，多筆命中逐一列。
   - `monster <DATABASES> <id|名稱子字串>`：列該怪 8 槽（kind 0 的物品名；kind 1 的表名＋表內物品與 w）。
   - 表格式輸出（TSV 或對齊文字皆可），表的「近似抽中機率」不算（+0x1cc 等級門檻表未取得），只印 w／總和並在表頭註明。
   - 所有讀取都做邊界檢查；格式不符（長度、表數、EOF）明確報錯、非零退出。
2. `scripts/loot-where.test.sh`：缺 `build/data-x/DATABASES` 印 `SKIP loot-where` 並 exit 0；否則斷言：
   (a) `item 74` 恰列 7 張表，表／槽 w 為 20/2、21/1、22/2、33/3、50/1、57/1、58/7（loot.md「74 Faust Halberd 有 7 張池引用」）；
   (b) `item 34` 的表集合恰為 {34, 50}；
   (c) `item 86` 含怪物 283、c=10、經表 54；
   (d) `monster 283` 的輸出含表 54 且其內含 item 86；
   (e) NPC 死亡池：至少一筆 NPC 的 +0x64 連到表 58（loot.md:29 Antone）；
   (f) 名稱查詢 `item excaliber` 與 `item 34` 輸出相同；
   (g) 壞檔（截短的 ITEMTABLES 副本放 mktemp 目錄）非零退出。
   全過印 `PASS loot-where`。
3. `notes/calc.md:263`：用法、欄位意義、機率近似的前提與限制（地上物／寶箱擺放 .STS 未納入；表抽取的等級篩選未建模）。不貼查詢結果。
4. 四段注入（只動 loot-where.py）：把 kind 1 的表索引改成 1-based → 驗收印 FAIL loot-where → 副本還原 → PASS；貼進回覆。

## 規格疑慮
（執行方填）

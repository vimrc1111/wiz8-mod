# R6 研究：DBS 與 MSG 格式（stage6 6.1）

驗收: bash scripts/check-research.sh notes/research/2026-10-07-R6-dbs-msg-format.md
規模: 中（研究，一份報告）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單以外的檔；不寫入遊戲目錄；不執行遊戲或任何下載物；報告不貼反編譯碼、不貼遊戲原文（以檔名＋位移、記錄號指稱；欄位名、單字級的物品／法術名可寫）；不送 email 或真實姓名
白名單: notes/research/2026-10-07-R6-dbs-msg-format.md
預期紅: research:2026-10-07-R6-dbs-msg-format
執行器: muse

## 背景
Wizardry 8（Wiz8.exe v1.2.4，i386）中文化。階段 6 要翻物品、法術、怪物的**名稱以外的文字**（名稱留英文：物品名、法術名 MG 已裁定）與場景訊息。下一步 codex 寫 decode／encode 工具，驗收是全部原檔往返 byte 一致，所以格式要精確到位元組。
- 資料：`build/data-x/`（DATA.SLF 全抽出），`build/patch-x/`（PATCH.001 抽出：覆蓋 DATABASES/ITEMS.DBS、SPELLTABLES.DBS、STRINGS/STRINGDATA.DAT）。遊戲以 PATCH.001 版本為準。
- 反組譯全文 `build/wiz8.asm`（objdump intel）。執行期字串表 0x0068c09c（`notes/re/misc.md:34`）。
- 已知（cc `local_data_check`）：STRINGDATA 字元編碼為每個 u16 存 0x9696−c（`tools/strings/`）；MESSAGES\*.MSG 文字同編碼，但字串起點不一定在偶數位移；內容是場景事件敘述。
- 已知：SPELLDESC.DBS（25 KB）是法術說明；`tools/strings` 解不開它。
- MSG 共 51 個（`find build/data-x -iname '*.msg'`），DBS 12 個（`build/data-x/DATABASES/`）。

## 問題（每條結論附來源：`build/...` 檔名＋十六進位位移、`build/wiz8.asm:<行>`、`tools/...:<行>` 或 URL，與三態標記）
1. 每個 DBS：檔頭、記錄數、記錄長度（定長／變長）、哪些欄位是文字、文字編碼（ASCII／UTF-16／0x9696−c／其他）、文字欄有無長度上限（定長欄位寬度）。表：檔名｜結構｜文字欄｜編碼｜上限｜證據。沒有可翻文字的 DBS 一行帶過即可。
2. ITEMDESC.DBS、SPELLDESC.DBS：說明文字如何對應到物品／法術記錄（索引、id 欄位）；是否含格式記號（`%`、換行、色碼）。
3. MONSTERS.DBS、ITEMS.DBS、NPC.DBS：除名稱外有無玩家看得到的文字。
4. MSG：檔案結構（檔頭、條目表、每條起訖），奇數位移起點的原因；條目以什麼 id 被遊戲引用（`build/wiz8.asm` 載入函式）。
5. 遊戲讀這些檔的函式（位址）與是否有固定大小緩衝（譯文變長時會不會溢位）。
6. 開放題：會讓「decode→encode 往返 byte 一致」做不到的地方（壓縮、校驗碼、填充雜訊）。

## 報告格式
`notes/research/2026-10-07-R6-dbs-msg-format.md`，含 `## 發現` 段，每條 `- ` 結論附來源與 `[已驗證]`／`[推測]`／`[未取得]`。

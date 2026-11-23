# R10 研究：現成的 Wizardry 8 存檔／角色編輯器（2026-10-08）

範圍：Steam 版 v1.2.4、`Saves\*.SAV`。只讀網頁與文件；未下載、未解壓、未執行任何編輯器。

## 發現

- 現成工具以 `anonidiot/Wizardry8Editor` 為首選：作者 anonidiot，Qt/C++ 開源（BSD-2-Clause），原始碼與 Windows 版見 https://github.com/anonidiot/Wizardry8Editor ，最新 Release 0.2.3 發布於 2026-03-11（0.2.x 起支援 1.28 存檔）。 [已驗證]
- `JS Wizardry Keeper`（作者 bitumin）是瀏覽器離線存檔編輯器，GPL-3.0、原始碼見 https://github.com/bitumin/jswizardry-source ，線上版 https://jswizardry.pages.dev ，README 明寫 TODO「尚不支援 1.2.8 存檔」，最後推送 2025-03。 [已驗證]
- `Wizardry 8 Status Effect Save Editor`（作者 primaryobjects）只改角色狀態效果，頁面稱全程本地運作、不上傳存檔，見 https://primaryobjects.github.io/wizardry8/ ，原始碼在 https://github.com/primaryobjects/primaryobjects.github.com （該 repo 未聲明授權，2026-09 仍有更新）。 [已驗證]
- tk421.net 託管兩個 2000 年代老編輯器：`Wizardry 8 Editor (98k, files/w8edit.zip)` 與 `Wizardry 8 Editor (20k, files/wizardry8editor.zip)`，見 https://www.tk421.net/wizardry/files.shtml ，頁面未標作者與授權、無原始碼。 [已驗證]
- 老牌 `wizedit.zip`（加物品，wizardryrealms.com 站長作）與 `wizedit2.zip`（技能拉滿，cpommer 作）均為閉源免費軟體、無原始碼，記載見 https://www.zimlab.com/wizardry/recovered/samjack/ ；同頁另有 7Heaven 的 `Wiz8RFV` 資料檢視器與 Finster 的肖像編輯器（皆非存檔編輯器）。 [已驗證]
- `Wiz8Edit` Java 版（alpha，Wiz8Edit.rar，需 JVM）是另一個開源向選擇，見 http://i45.narod.ru/wiz/ ，但頁面自稱 alpha 且年久未更新。 [已驗證]
- Mad God 的 `Cosmic Forge`（Sergey Pershin 作）是 Wiz 6/7/8 通用遊戲數據 mod 工具（改物品／怪物／法術數值與掉落），不是存檔編輯器，見 https://www.zimlab.com/wizardry/recovered/samjack/ ；同類介紹見 http://www.jeffludwig.com/wizardry8/ 。 [已驗證]
- `Zyvr` 的 hex-edit 指南（2013，非程式）は公開的位移表，見 https://gist.github.com/l1x/6160913 。 [已驗證]
- Wizardry8Editor 功能覆蓋最廣：改屬性（上限 125）、法術（含 1.28 新儲存格式）、物品（新增／改數量 charges）、招募 RPC（0.2.1 新增，用來繞過 RPC 無法重招的 bug）、角色匯出（自動補 .chr 後綴）、改肖像、改隊伍／傳送座標、exe 補丁（含舊作匯入劇情補丁），見 https://raw.githubusercontent.com/anonidiot/Wizardry8Editor/main/Changelog 與 https://api.github.com/repos/anonidiot/Wizardry8Editor/contents/src 的對話框原始碼（DialogAddItem、ScreenSkills、DialogSpellInfo 等）。 [已驗證]
- Wizardry8Editor 的總等級／經驗值欄位在所讀頁面未直接確認（Changelog 只明確提到 level 欄位標籤調整與單職業時重置原始職業），等級相關編輯能力視為推定，見 https://raw.githubusercontent.com/anonidiot/Wizardry8Editor/main/Changelog 。 [推測]
- Wizardry8Editor 是否支援直接切換種族／職業、整隊成員替換（把現成角色換成另一個人），在所讀文件與原始碼檔名中未見明確證據，見 https://github.com/anonidiot/Wizardry8Editor 。 [未取得]
- JS Keeper 可改：XP、總等級＋15 種職業等級、HP/Stamina、六圍屬性、幾十種技能、法術抗性、物品表；nickname/name 為唯讀；原始碼中未見種族／職業切換欄位，見 https://raw.githubusercontent.com/bitumin/jswizardry-source/master/src/CharacterTab.svelte 。 [已驗證]
- primaryobjects 編輯器只能加減狀態（Disease、Poison、Paralysis 等），不碰等級屬性物品，見 https://primaryobjects.github.io/wizardry8/ 。 [已驗證]
- Steam 版即 v1.2.4，是最後一個官方補丁；1.26/1.28 皆社群 fan patch，見 https://steamcommunity.com/app/245450/discussions/0/762932162500603230 。 [已驗證]
- Fan patch 1.28 改變存檔結構：1.24 存檔可在 1.28 讀取，但轉成 1.28 後不能拿回 1.24 用；舊存檔編輯器只適用 1.24，新版編輯器才支援新格式，見 https://steamcommunity.com/app/245450/discussions/0/762932533852705030 。 [已驗證]
- 已知損壞操作一：0.1.4 之前版本建的「新遊戲」檔是壞的（官方 Changelog 要求重建；打過補丁的 exe 不用重打），見 https://raw.githubusercontent.com/anonidiot/Wizardry8Editor/main/Changelog 。 [已驗證]
- 已知損壞操作二：編輯帶死亡隊員的隊伍、1.28 法術讀寫在 0.2.2/0.2.3 才修好，務必用最新版，見 https://github.com/anonidiot/Wizardry8Editor/releases 。 [已驗證]
- 已知損壞操作三（舊編輯器）：存檔含 mod 非原版物品會讓舊編輯器當掉，可反過來當 mod 物品檢測法，見 https://www.gog.com/forum/wizardry_series/game_breaking_bug_in_wizardry_8/post5 。 [推測]
- 除版本結構差異外，所讀討論中未見 Steam／GOG 版專屬的「可用／壞掉」回報，見 https://steamcommunity.com/app/245450/discussions/0/762932533852705030 。 [推測]
- 安全訊號正向：Wizardry8Editor 官方 README 公布各 exe 的 sha256 可供核對，且在 GitHub 公開發行，見 https://raw.githubusercontent.com/anonidiot/Wizardry8Editor/master/README.md 。 [已驗證]
- 所讀頁面與討論中沒有看到針對上述任一工具的惡意程式回報，見 https://github.com/anonidiot/Wizardry8Editor 與 https://steamcommunity.com/app/245450/discussions/0/762932533852705030 。 [推測]
- 老編輯器（wizedit/wizedit2）風險較高：只有二進位 zip、無原始碼、需把 tabctl32.ocx 放入 System32，且託管於個人回收站；按本單禁令未下載驗證其內容，見 https://www.zimlab.com/wizardry/recovered/samjack/ 。 [未取得]
- 遊戲內建匯入：開新遊戲的 Gather Party 畫面可匯入舊作角色（Wiz6 經 Wiz7 轉、Wiz7、Wiz Gold 存檔），見 https://www.pcgamingwiki.com/wiki/Wizardry_8 。 [推測]
- 遊戲內建 .chr 途徑：把 `.chr` 角色檔放入 `Wizardry 8\Saves\Characters`，開新遊戲時可直接選用（GOG 玩家實測含改名換肖像），見 https://www.gog.com/forum/wizardry_series/wizardry_8_uber_character_memory_edited_on_character_creation_i_can_create_for_you/page1 。 [已驗證]
- 未發現開發者主控台或作弊指令：GameFAQs 問答稱 Wiz8 沒有作弊模式、變強只能靠存檔編輯器，但該頁本文回傳 403 未能直接讀取，僅見搜尋摘要，見 https://Gamefaqs.gamespot.com/pc/374906-wizardry-8/answers/270254-how-do-i-go-to-cheat-mode 。 [推測]
- 存檔結構公開文件：Zyvr 指南給出以角色暱稱首字母為錨點的位移表（隊伍順序 L1-R1-L2-R2-L3-R3、每人 1866 bytes、總等級 +84、各職業等級與屬性位移），見 https://gist.github.com/l1x/6160913 。 [已驗證]
- checksum／壓縮方式查無公開說明文件；僅知社群反組譯專案正在還原存檔序列化路徑（含 BitArray/Huffman 相關程式），可作後續自寫解析器的起點而非現成文件，見 https://github.com/agluszak/wizardry-8-decomp 。 [推測]
- i45 站另有各職業技能成長與轉職最低屬性表，可作改值合法性參考，見 http://i45.narod.ru/wiz/ 。 [已驗證]
- 推薦直接用 Wizardry8Editor 0.2.3：唯一仍在維護、開源可審、有 1.24/1.28 雙支援，且一次覆蓋升級、屬性、技能、法術、物品與 RPC 招募等需求，不必自寫解析器，見 https://github.com/anonidiot/Wizardry8Editor/releases 。 [推測]
- 推薦附帶條件：它是 Windows exe（MG 的 Steam 環境即 Windows，直接可用）；先備份 `Saves\` 再改；種族直切與整隊換人若證實不支援，缺口用「匯出 .chr＋新開遊戲選人」或 Zyvr 位移表手改補上，仍不必自寫完整編輯器，見 https://gist.github.com/l1x/6160913 。 [推測]
- 不推薦只靠 JS Keeper（不支援 1.28 且無 RPC／匯出）或 primaryobjects（只改狀態）滿足本單四項需求，見 https://github.com/bitumin/jswizardry-source 與 https://primaryobjects.github.io/wizardry8/ 。 [推測]

## 規格疑慮

無

## cc 抽查（2026-10-08，GitHub API／raw，main 分支）

- 授權 BSD-2-Clause、最新 release `VERSION_0_2_3` 2026-03-11、未封存：對上。
- 「README 公布各 exe 的 sha256」：README（main）grep `sha|hash` 零命中 → **不成立**；release 只有三個 exe，沒有雜湊檔。
- 「種族／職業切換未取得」→ **更正**：`src/character.h:278` `setProfession`、`:321` `setRace`、`:330` `setProfessionLevel`、`:384` `setXp` 存在；有 `src/ScreenLevels.cpp`（XP 欄位）。資料層支援，UI 是否開放改種族未看。

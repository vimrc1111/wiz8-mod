# A3 研究：文字資料與字型格式

驗收: bash scripts/check-research.sh notes/research/2026-10-07-A3-text-font-format.md
規模: 中（研究，一份報告）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單以外的檔；不寫入遊戲目錄；不執行任何下載物；不送 email 或真實姓名（User-Agent 用「wiz8-mod personal research (non-commercial)」）
白名單: notes/research/2026-10-07-A3-text-font-format.md
預期紅: research:2026-10-07-A3-text-font-format
執行器: muse

## 背景
專案：Wizardry 8（Steam 版，Wiz8.exe 3580513 B，i386）中文化＋現代相容。原版資料與美術一個 byte 不動，改動靠疊加（Patches/PATCH.00N 覆蓋檔、遊戲目錄放代理 ddraw.dll 做 hook）。你做研究、不寫碼；報告會被下一張實作票當規格用。
已知（不用再查）：遊戲目錄放的 ddraw.dll 已實測會被 Wiz8.exe 載入（notes/zh-patch-log.md:162）；德文翻譯以 PATCH.002 疊加文字；日文版換成日版 1.2J exe（英文 exe 無雙位元組文字支援）；FanPatch 1.28 有 Localization/<LANG>/{Info.ini,Lang.SLF,Text.txt}。見 notes/research/ 既有報告。

## 待答問題
1. Wiz8 的文字放在哪：STRINGDATA.DAT、*.MSG、*.DBS（物品／怪物／法術名）、對話檔等——各自在哪個 SLF 或目錄、格式（長度前綴？null 結尾？編碼）。用本機樣本驗（先 `strings` 找英文字串所在的檔與 offset）。
2. 字型：遊戲字型檔（STI 或其他）的格式公開說明；字元表大小（單位元組 256？）、字寬表、點陣大小。
3. 英文 exe 繪字的限制：單位元組字元、字串長度上限、緩衝區大小的公開討論（德文／法文／波蘭文翻譯遇到的長度問題、特殊字元怎麼塞進字型）。
4. 波蘭文 FanPatch（w8u 的 Localization/Fan-Patch1.28/POL/）：Text.txt 的格式、Lang.SLF 裡放了哪些檔（字型？），它怎麼處理非 ASCII 字元。
5. 日文 1.2J 版的雙位元組做法有沒有公開說明（字型格式、編碼 Shift-JIS？）。

## 本機樣本（唯讀；`w8`＝Steam 的 Wizardry8 目錄、`w8u`＝Wizardry8_Uninstall，經無空白的 symlink 掛入沙箱，可用 xxd／od／strings／objdump 讀，不可寫）
- `/tmp/claude-1000/-home-mg-Projects-games-wiz8-mod/d06f7575-86be-46cb-bd4f-47f006a4e13e/scratchpad/w8/`（Wiz8.exe、Data/、Patches/PATCH.001）
- `/tmp/claude-1000/-home-mg-Projects-games-wiz8-mod/d06f7575-86be-46cb-bd4f-47f006a4e13e/scratchpad/w8u/`（含 FanPatch 1.28 殘留：Localization/Fan-Patch1.28/POL/）
公開文件說的要拿本機樣本驗；驗過標 `[已驗證]`，只有文件說標 `[推測]`。

## 證據規格（機檢，驗收命令會檢查）
- 結論全部寫在 `## 發現` 段內，每條一行、以 `- ` 開頭。
- 每條必須含來源：網址（`https://…`）或本機 `path:line` 形狀（例 `Wiz8.exe:0x1a2b3c`）。
- 每條必須標 `[已驗證]`、`[推測]`、`[未取得]` 之一。本機驗證的條目把命令與關鍵輸出放 `## 命令紀錄` 段（該段不檢查）。
- 不要把「沒找到」寫成「不存在」。

## 不做
- 不開 Ghidra／IDA／反組譯器（objdump -p／-h 看檔頭與匯入表可以，不要 -d 反組譯）。不寫程式。
- 不下載、不執行第三方工具；只讀網頁與原始碼。
- 不改其他檔、不 commit。

## 交付回覆
最後回覆裡附「規格疑慮」段：這張單哪裡寫得不清楚、哪題你判斷問錯了；沒有就寫「無」。

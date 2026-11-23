# A2 研究：ddraw.dll 代理載入與串接

驗收: bash scripts/check-research.sh notes/research/2026-10-07-A2-ddraw-proxy.md
規模: 中（研究，一份報告）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單以外的檔；不寫入遊戲目錄；不執行任何下載物；不送 email 或真實姓名（User-Agent 用「wiz8-mod personal research (non-commercial)」）
白名單: notes/research/2026-10-07-A2-ddraw-proxy.md
預期紅: research:2026-10-07-A2-ddraw-proxy
執行器: muse

## 背景
專案：Wizardry 8（Steam 版，Wiz8.exe 3580513 B，i386）中文化＋現代相容。原版資料與美術一個 byte 不動，改動靠疊加（Patches/PATCH.00N 覆蓋檔、遊戲目錄放代理 ddraw.dll 做 hook）。你做研究、不寫碼；報告會被下一張實作票當規格用。
已知（不用再查）：遊戲目錄放的 ddraw.dll 已實測會被 Wiz8.exe 載入（notes/zh-patch-log.md:162）；德文翻譯以 PATCH.002 疊加文字；日文版換成日版 1.2J exe（英文 exe 無雙位元組文字支援）；FanPatch 1.28 有 Localization/<LANG>/{Info.ini,Lang.SLF,Text.txt}。見 notes/research/ 既有報告。

## 待答問題
1. 有沒有人對 Wizardry 8 做過 ddraw.dll 代理／包裝（例：dgVoodoo2、DDrawCompat、cnc-ddraw、各種 widescreen／windowed 修正）？做法、是否開源、網址。
2. 代理 ddraw 如何與 dgVoodoo2 串接（dgVoodoo2 也是 ddraw.dll）：既有工具的慣例（改名 chain-load、ini 指定下一層），dgVoodoo2 本身支不支援被包或包別人。
3. ddraw 代理要攔文字繪製時可掛的點：Wiz8 是否用 GDI（GetDC/TextOut/ExtTextOut）在 DirectDraw surface 上畫字，還是自己 blit 點陣字（公開討論、modder 文件）。用 `objdump -p Wiz8.exe` 的匯入表佐證有無 GDI 文字函數。
4. Steam 版的 RUNASADMIN／HIGHDPIAWARE 相容旗標對 DLL 載入與 %TEMP% 路徑的影響；以系統管理員執行時 %TEMP% 是不是同一個。
5. Windows 10/11 上 32 位元行程載入 ddraw 的已知相容墊片（DDrawCompat／Windows 內建 shim）會不會蓋掉或干擾代理。

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

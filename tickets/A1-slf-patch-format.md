# A1 研究：Wizardry 8 的 SLF 與 Patches/PATCH.00N 格式

驗收: bash scripts/check-research.sh notes/research/2026-10-07-A1-slf-patch-format.md
規模: 中（研究，一份報告）
預算: 重做 3 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單以外的檔；不寫入遊戲目錄；不執行任何下載物；不送 email 或真實姓名（User-Agent 用「wiz8-mod personal research (non-commercial)」）
白名單: notes/research/2026-10-07-A1-slf-patch-format.md
預期紅: research:2026-10-07-A1-slf-patch-format
執行器: muse

## 你是誰、要做什麼
你是研究執行方，做研究不寫碼（原定 agy＝派工方分身，額度用完改派你；要求不變）。專案：Wizardry 8 中文化，原版資料一個 byte 不動，改動靠疊加（PATCH 覆蓋檔或 DLL hook）。下一張票會依你的報告寫 Go 的 SLF 解包工具 `slf list|extract|pack`，所以報告要能直接當格式規格用。

## 待答問題
1. SLF 檔頭：長度、欄位、位元組序；檔名區（固定長度？零填充？）；entry 表位置、每筆欄位（名字長度、offset、size、時間戳、其他）；是否有壓縮或加密。
2. `Patches/PATCH.00N`：是否同 SLF 格式；遊戲如何載入（掃哪些編號、順序、同名檔誰贏、對 Data/ 下哪個 SLF 生效）。
3. 既有工具：Cosmic Forge／Wiz8 編輯器、SLF 解包器（名稱、作者、原始碼網址、授權）。
4. 已知的語系疊加先例：德文 PATCH.002、FanPatch 的 `Localization/<LANG>/Lang.SLF`——它們放了哪些檔名。
5. 未解：上面任何一項找不到就寫 `[未取得]` 加原因。

## 本機樣本（唯讀；`w8`＝Steam 的 Wizardry8 目錄、`w8u`＝Wizardry8_Uninstall，經無空白的 symlink 掛入沙箱，可用 xxd／od／head／strings 讀，不可寫）
- `/tmp/claude-1000/-home-mg-Projects-games-wiz8-mod/d06f7575-86be-46cb-bd4f-47f006a4e13e/scratchpad/w8/Patches/PATCH.001`
- `/tmp/claude-1000/-home-mg-Projects-games-wiz8-mod/d06f7575-86be-46cb-bd4f-47f006a4e13e/scratchpad/w8/Data/DATA.SLF`（另有 Levels/LEVELS.SLF、Data/MONSTERS/MONSTERS.SLF、Data/Sound/SOUND.SLF、Data/Sound/Monsters/MONSTERSOUND.SLF）
- `/tmp/claude-1000/-home-mg-Projects-games-wiz8-mod/d06f7575-86be-46cb-bd4f-47f006a4e13e/scratchpad/w8u/Localization/Fan-Patch1.28/POL/Lang.SLF`
公開文件說的格式要拿本機樣本驗：例「檔頭第 N byte 起為 entry 數，DATA.SLF 讀到 X，實際 entry 表長度吻合」。驗過標 `[已驗證]`，只有文件說標 `[推測]`。

## 證據規格（機檢，驗收命令會檢查）
- 結論全部寫在 `## 發現` 段內，每條一行、以 `- ` 開頭。
- 每條必須含來源：網址（`https://…`）或本機 `路徑:行號`／`檔名:offset` 寫成 `path:line` 形狀（例 `PATCH.001:0x100`）。
- 每條必須標 `[已驗證]`、`[推測]`、`[未取得]` 之一。本機驗證的條目附你跑的命令與關鍵輸出（可另開 `## 命令紀錄` 段，該段不檢查）。
- 不要把「沒找到」寫成「不存在」。

## 不做
- 不寫解包程式（下一張票的事）；可以寫一次性的 shell 指令讀 bytes。
- 不下載、不執行第三方工具；只讀網頁與原始碼。
- 不改其他檔、不 commit。

## 交付回覆
最後回覆裡附「規格疑慮」段：這張單哪裡寫得不清楚、哪題你判斷問錯了；沒有就寫「無」。

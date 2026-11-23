# wiz8-mod

Wizardry 8 繁體中文化。原版資料一個 byte 都不改：安裝時只往遊戲目錄加兩個檔，刪掉就回到原版。
必須搭配你自己的原版遊戲；本 repo 與發布包都不含任何原版檔案（仿 ScummVM）。

## 需要

- Wizardry 8 原版 v1.2.4（Steam 版已驗；`Patches\PATCH.001` 要在）。GOG 版、其他版本未驗。
- Windows（遊戲本體）。打包在 WSL／Linux 上做。

## 打包（開發機）

需要 Go、`i686-w64-mingw32` 工具鏈（建 ddraw.dll）、`/mnt/c/Windows/SysWOW64/ddraw.dll`（算匯出表；可用 `SYS_DDRAW=` 指定）。

```bash
bash scripts/release.sh /path/to/wiz8-mod.zip
```

包內：`ddraw.dll`、`zhpatch.exe`、`translations/`、`install.bat`、`uninstall.bat`、`INSTALL.txt`、`LICENSES/`（字型 OFL）。
不放 `PATCH.002`；它在安裝時用玩家自己的原版現場產生。

## 安裝（Windows）

1. 把 zip 解到遊戲資料夾**以外**的地方。
2. 在解出來的資料夾開命令提示字元（遊戲在 Program Files 底下要用系統管理員身分）：
   ```bat
   install.bat "C:\Program Files (x86)\Steam\steamapps\common\Wizardry8"
   ```
   遊戲在 Steam 預設位置可只打 `install.bat`。
3. 看到 `[wiz8-mod] Installed` 即完成。遊戲目錄多出：
   - `ddraw.dll`：中文繪字
   - `Patches\PATCH.002`：中文文字（遊戲本來就會依序疊讀 PATCH 檔）

安裝前會檢查：路徑是不是遊戲目錄、`PATCH.001` 在不在、原文 sha256 對不對得上。任一不符就什麼都不裝。
目錄裡已有舊版 wiz8-mod 時會先自動移除再裝；`ddraw.dll` 不是本 mod 的（別的模組），或有認不出來源的 `PATCH.002`，就拒絕、不覆蓋。

## 遊戲內使用

安裝後照常啟動遊戲（Steam 或直接執行 `Wiz8.exe`），選單、對話、物品、怪物、法術、日誌就是中文。

**F10：速度設定選單**（遊戲中按 F10 開關）

| 按鍵 | 作用 |
|---|---|
| 上／下 | 選一列 |
| 左／右 | 每次調 25%（範圍 10%～5000%） |
| Home | 這一列回到原版（100%） |
| End | 全部回到原版 |
| Esc 或 F10 | 關閉選單，設定存進遊戲資料夾的 `wiz8mod.ini` |

四列與預設值：怪物移動 400%、事件停頓 300%、攻擊動畫 200%、隊伍移動 100%（100%＝原版速度）。

- 截圖：遊戲內建的截圖鍵照用，存成 BMP，放在「使用者資料夾\Pictures\Wiz8\」。
- 熱鍵可在 `wiz8mod.ini` 改：`[menu] hotkey_vk=<虛擬鍵碼>`（預設 121＝F10）。
- 要用存檔修改器前先跑 `uninstall.bat`（中文化裝著時修改器顯示亂碼），改完再 `install.bat` 裝回。

## 更新到新版

解壓新版，直接跑新版的 `install.bat`：會先移除舊版再裝，存檔不受影響。

## 移除

```bat
uninstall.bat "C:\...\Wizardry8"
```

刪本 mod 的兩個檔與 F10 選單存的 `wiz8mod.ini`；`ddraw.dll` 不是本包那份（或不在）就兩個都不動（`PATCH.002` 本身認不出是誰做的）。存檔不受影響。

## 已知限制

- NPC 對話的關鍵字要點或打英文，譯文寫成「銀行（ bank ）」。
- 關鍵字清單在開新遊戲時寫進存檔，舊存檔的清單仍是英文；舊存檔裡的訊息歷史也是英文。
- `ddraw.dll` 會在 `%TEMP%` 寫 `wiz8-mod-ddraw.log`、`wiz8-mod-text.log`（除錯用，可刪）。
- zhpatch 的錯誤訊息是 UTF-8，cp950 主控台會顯示亂碼。

## 開發

### 目錄

| 路徑 | 內容 |
|---|---|
| `translations/` | 譯文（只有我們的文字；`stringdata.json`、`lib.tsv`、`monsters.json`…） |
| `tools/` | Go 工具：`slf`（解／打 SLF、PATCH）、`strings`、`strlib`、`nsf`、`dbsname`、`zhpatch`（玩家端產 PATCH.002）、`atlas`（字形圖集）、`calc`（傷害試算）、`trcheck` 等 |
| `hook/ddraw/` | 代理 `ddraw.dll`（中文繪字、F10 速度選單、截圖） |
| `release/` | 玩家用 `install.bat`／`uninstall.bat`／`INSTALL.txt` |
| `scripts/` | 建置、檢查、CI、實機跑測（`wiz8-run.sh`） |
| `notes/`、`notes/re/` | 研究筆記、反組譯推得的規則（只寫規則與位址，不含反組譯碼） |
| `decisions/`、`goals/`、`tickets/` | 裁決、階段目標、工單與收貨帳本 |
| `re/` | 靜態分析／模擬器輔助腳本（Ghidra、unicorn；環境在 `~/.local/state/wiz8-mod-*`） |

### 開發機需要

- Linux 或 WSL；Go（版本見 `go.mod`）、`i686-w64-mingw32` 工具鏈、`python3`、`cpio`。
- 原版遊戲目錄，預設 `/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8`，別處用 `WIZ8_DIR=` 指定。只讀，不寫入。
- 系統 32 位元 `ddraw.dll`（算匯出表），預設 `/mnt/c/Windows/SysWOW64/ddraw.dll`，別處用 `SYS_DDRAW=`。
- 字型兩檔，不入版（來源 URL 與版本見 `assets/LICENSES`）：
  - `~/.local/share/wiz8-mod/fonts/noto-sans-tc-2.004/NotoSansTC-Regular.otf`（或 `WIZ8_FONT=`）
  - `~/.local/share/wiz8-mod/fonts/fusion-pixel-12px-2026.09.25/fusion-pixel-12px-proportional-zh_tw.ttf`（或 `WIZ8_PXFONT=`）

### 常用指令

```bash
bash hook/ddraw/build.sh                         # → hook/ddraw/out/ddraw.dll
bash scripts/build-zh-patch.sh build/zh/PATCH.002 # 開發機直接產 PATCH（原文衍生物，放 build/，不入版）
bash scripts/release.sh /path/to/wiz8-mod.zip    # 發布包（見上「打包」）
go test ./...                                    # Go 單元測試
bash scripts/ci-run.sh wiz8-mod bash scripts/ci-check.sh   # 本機 CI（五道：go／hook／tr／lib／zhpatch）
```

- CI 證據寫在 `~/.local/state/wiz8-mod-ci/<SHA>/`；pre-push 閘要求 tip 有 exit 0 的證據。
- go／tr／lib／zhpatch 四道讀原版：原版不在時必紅。
- 原版衍生物（解出的檔、表、截圖、試算中間檔）一律放 `build/`（gitignored）或 `~/.local/state/wiz8-mod-*`，不入版。

### 配裝試算（`tools/calc`）

近戰與單體直接傷害法術的期望傷害；公式條目在 `notes/re/formulas.md`。結論與用法見 `notes/party.md`（最終隊伍建議）。

### 文件導覽

| 文件 | 內容 |
|---|---|
| `notes/party.md` | **最佳隊伍試算**：用從原版反組譯推得的戰鬥公式，對全部職業×種族×武器組合算期望擊殺值，排出 6 人隊與每人配裝（武器、取得處）。現行結論在檔頭 TL;DR；舊版定稿留在後段作沿革 |
| `notes/calc.md` | 試算工具 `tools/calc` 的用法與限制：`fight`（單角色一回合近戰）、`rank`（開局列舉排名）、`party`（創角 Pareto 前緣）、掉落查表 |
| `notes/calc-qa.md` | calc 題庫：「某情境該算出多少」的固定問答，CI 機檢讀取 |
| `notes/re/` | 反組譯推得的遊戲規則，每條附 exe 位址：`formulas.md`（近戰公式）、`combat-*.md`（戰鬥流程與時序）、`weapon-effects.md`（武器特效）、`loot.md`（掉落）、`growth.md`（升級成長）、`draw.md`（繪字，中文化用）等 |
| `notes/zh-patch-log.md` | 中文化技術紀錄：PATCH 疊讀規則、ddraw 代理、CJK 繪字路線、驗收規則、安裝流程實測 |
| `notes/research/` | 外部研究：檔案格式（SLF／DBS／NSF）、社群翻譯（德／法／日）、存檔修改器、社群攻略與隊伍說法對照，每條附來源 |
| `decisions/` | 重要裁決，例如反組譯解禁與「必須搭配原版」原則、試算只做數值不碰地圖幾何 |
| `goals/stage1.md`～`stage16.md` | 各階段目標與完成條件：1–7 中文化、8–16 戰鬥加速、遊戲內選單、配裝試算 |
| `tickets/` | 工單與收貨帳本（`tickets/ledger/`）：每張單的規格、驗收、派工紀錄 |
| `resume.md` | 進度總表與結案狀態（2026-11-01 結案） |

### 交給 AI 接手（ChatGPT／Claude 訂閱）

在 WSL 裡 `git clone https://github.com/vimrc1111/wiz8-mod.git` 後 `cd wiz8-mod`，依訂閱啟動：

- ChatGPT 訂閱：執行 `codex`（Codex CLI）
- Claude 訂閱：執行 `claude`（Claude Code，會自動載入 `CLAUDE.md`）

然後貼下面這段，最後一行換成要做的事：

```
你要接手這個 repo：Wizardry 8 中文化＋現代機器相容 mod，必須搭配 Steam 版 Wizardry 8 v1.2.4 原版使用。

先做這三件事再動手：
1. 讀完 README.md，再讀 CLAUDE.md 的「CI」段。CLAUDE.md 的「派工」「常設核准」是原作者自己的工作流程，不適用，忽略。
2. 照 README「開發機需要」逐項檢查環境：Go（版本見 go.mod）、i686-w64-mingw32、python3、cpio、原版遊戲目錄（WIZ8_DIR）、系統 32 位元 ddraw.dll（SYS_DDRAW）、兩個字型檔（來源見 assets/LICENSES）。缺什麼列給我，不要自己下載或複製任何原版遊戲檔。
3. 跑 go test ./... 和 bash scripts/ci-run.sh wiz8-mod bash scripts/ci-check.sh，回報五道（go／hook／tr／lib／zhpatch）各自的結果。

一律遵守：
- 原版遊戲目錄只讀，一個 byte 都不改；不碰 Saves 存檔。
- repo 不收原版檔案、從原版抽出的表或反組譯碼；這類產物放 build/（已 gitignore）。
- 改動前先說要改哪些檔；改完跑 CI，把結果給我看。

我要做的事：<例如「在新電腦裝好中文化」「修某段譯文」>
```

只想玩、不開發：照上面「安裝」段跑 `install.bat` 即可，不需要 AI。

### 工作紀錄

- 工作規則、派工、常設核准：`CLAUDE.md`；進度與結案狀態：`resume.md`（2026-11-01 結案）。
- 安裝流程實測紀錄：`notes/zh-patch-log.md:257`。
- 本 repo 從 `rpg-mod`（子目錄 `wiz8-mod/`）的最新進度拆出，只有一個 first commit；完整歷史（含 2026-10-19 併入前的原 repo 歷史）留在 `rpg-mod`。文件裡引用的 commit sha 要到 `rpg-mod` 查（`git -C rpg-mod show <sha>`）。

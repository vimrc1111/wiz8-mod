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
目錄裡已有 `PATCH.002` 或 `ddraw.dll`（裝過，或有別的模組）會拒絕，不覆蓋。

## 更新到新版

先移除再裝新包；install 不會覆蓋舊檔。

## 移除

```bat
uninstall.bat "C:\...\Wizardry8"
```

只刪本 mod 的兩個檔；`ddraw.dll` 不是本包那份（或不在）就兩個都不動（`PATCH.002` 本身認不出是誰做的）。存檔不受影響。

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
- go／tr／lib／zhpatch 四道讀原版：原版不在時必紅（2026-11-01 起 Steam 原版已解除安裝，重開本案先裝回）。
- 原版衍生物（解出的檔、表、截圖、試算中間檔）一律放 `build/`（gitignored）或 `~/.local/state/wiz8-mod-*`，不入版。

### 配裝試算（`tools/calc`）

近戰與單體直接傷害法術的期望傷害；公式條目在 `notes/re/formulas.md`。結論與用法見 `notes/party.md`（最終隊伍建議）。

### 工作紀錄

- 工作規則、派工、常設核准：`CLAUDE.md`；進度與結案狀態：`resume.md`（2026-11-01 結案）。
- 安裝流程實測紀錄：`notes/zh-patch-log.md:257`。
- 本 repo 從 `rpg-mod`（子目錄 `wiz8-mod/`）的最新進度拆出，只有一個 first commit；完整歷史（含 2026-10-19 併入前的原 repo 歷史）留在 `rpg-mod`。文件裡引用的 commit sha 要到 `rpg-mod` 查（`git -C rpg-mod show <sha>`）。

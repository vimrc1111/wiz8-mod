# 中文化與疊層紀錄（2026-10-07..08）

> 安裝基準、疊層規則、ddraw 代理、驗收規則、實機清單、發布包、CI 提速。stack.md 是活文件，其餘為當日紀錄。
> 2026-11-01 由 12 份原檔逐行併入（標題降一級、內文不改）；原檔見 tag `wiz8-notes-pre-merge`（85393087）。

目錄：
- 第 23 行起：Patch 疊層與症狀紀錄（原 `notes/stack.md`）
- 第 77 行起：2026-10-07 安裝目錄現況（as-found）（原 `notes/2026-10-07-as-found.md`）
- 第 102 行起：2026-10-07 Steam 乾淨重裝（取代 as-found 作為疊層起點）（原 `notes/2026-10-07-clean-install.md`）
- 第 127 行起：覆蓋規則決策表（stage2 §2.2，2026-10-07 實測）（原 `notes/2026-10-07-overlay-rules.md`）
- 第 162 行起：C2：只寫 log 的 ddraw.dll 實機載入（2026-10-07）（原 `notes/2026-10-07-c2-ddraw-proxy.md`）
- 第 193 行起：4.3 判讀規則（2026-10-07，先寫規則、後跑資料）（原 `notes/2026-10-07-cjk-acceptance-rules.md`）
- 第 212 行起：實機檢查清單：行距 +1（664e66b）＋NPC 對白（94167ce）（原 `notes/2026-10-07-pitch-checklist.md`）
- 第 257 行起：發布包安裝實測（2026-10-07，goal 第 5 步）（原 `notes/2026-10-07-release-install-test.md`）
- 第 299 行起：收尾：MG 實機清單＋待裁清單（2026-10-07，/goal 第 6 步）（原 `notes/2026-10-07-final-checklist.md`）
- 第 327 行起：2026-10-07 沙箱網路探針（A0）（原 `notes/2026-10-07-sandbox-net-probe.md`）
- 第 334 行起：2026-10-07 既有玩家翻譯盤點（摘要）（原 `notes/2026-10-07-fan-translations.md`）
- 第 349 行起：CI 提速與發布包實裝（2026-10-08）（原 `notes/2026-10-08-ci-speedup.md`）


---
<!-- 原檔 notes/stack.md -->
## Patch 疊層與症狀紀錄

每層：套用前跑 `hashdir scan` 存 `baseline/<日期>-<層>-before.json`，套用後再 scan，
`hashdir diff before after` 的輸出貼在該層「變動檔」，只准出現預期的檔案。

### 疊層（MG 2026-10-07 裁定，取代原「官方 1.24 → NTCore 4GB → FanPatch 1.28 → dgVoodoo2」）
MG 原話：「同意從原版開始，因爲這就是實際 steam 拿到的樣子，也是最合法的版本」；起因「很難要求別人先 patch 別人的東西」。
NTCore 4GB 會改 exe 檔頭（違反原版不動）；FanPatch 改為玩家可選、相容另驗。
起點：`baseline/2026-10-07-steam-clean.json`（見 `notes/zh-patch-log.md:102`）。

| # | 層 | 版本／來源 | 套用前症狀 | 套用後症狀 | 變動檔（hashdir diff） |
|---|---|---|---|---|---|
| 0 | Steam 原版（depot 本身，含 nGlide、`Patches/PATCH.001`） | 版本未取得，看到再記 |  |  | 基準本身：189 檔 |
| 1 | 本專案 `ddraw.dll`（hook 載入點；C1 只寫 log 版，commit 1715e3f） | 未取得（同次未做無 dll 對照啟動） | 2026-10-07 C2：Wiz8.exe 載入本 dll，log `attach … missing=0`；60 秒後仍在跑、視窗標題「Wizardry 8」；畫面未取得（`notes/zh-patch-log.md:162`） | 放入時只有 `A ddraw.dll`；移除後與放入前 0/0/0（log 寫在 %TEMP%，不進遊戲目錄） |
| 2 | dgVoodoo2（必要時；同為 `ddraw.dll`，需由第 1 層串接） |  |  |  |  |

#### 玩家可選層（不在主線，相容另驗）
| 層 | 備註 | 套用前症狀 | 套用後症狀 | 變動檔 |
|---|---|---|---|---|
| NTCore 4GB | 改 exe 檔頭 LAA 旗標；只能套在另存的 exe 副本 |  |  |  |
| FanPatch 1.28 | 不改 `Wiz8.exe`，另放 `Wiz8_v128.exe` 等；對照組 `baseline/2026-10-07-steam-uninstall-leftovers.json` |  |  |  |

症狀欄寫：啟動是否成功、解析度／全螢幕、崩潰點（哪個畫面、做了什麼）、`cfagent.log` 有無新錯誤。

### 實際渲染路徑（stage2 §2.1，2026-10-07 實測）
- 證據：`notes/runs/20261007-023242-noise.json` 的 `modules`（32 位元 PowerShell 列 WOW64 行程模組）。
- 鏈：`Wiz8.exe` → `srDD_Glide2x.dll`（引擎渲染外掛，同目錄另有 `srVP_PentiumII.dll`、`srEXT_*.dll`）→ `glide2x.dll`（nGlide）→ `d3d9.dll` → AMD 驅動（`atidx9loader32.dll`、`AMDXN32.DLL`）。
- 清單內無 `vulkan-1.dll`、`d3d11.dll`：nGlide 走 d3d9 後端。`DDRAW.DLL` 有載入（靜態匯入，C2），但不是畫圖路徑。
- 畫面：nGlide 全螢幕；主選單 640×480 拉伸填滿 3840×2160（x×6、y×4.5，無黑邊；視窗 rect 先 0,0,960,720，後 0,0,3840,2160）。
- 擷取：桌面 `CopyFromScreen`（DPI aware）可用；`PrintWindow` 全黑（d3d9 全螢幕）。
  `Screen.Bounds` 要在每次截圖當下讀，不能早讀（模式切換期間會讀到 960×720，截到左上角一小塊）。
- 雜訊底：主選單 t40，3 次 run 的 sha256 全同（`20261007-023242/023408/023535-noise`），逐像素差異 0。

### CJK 繪製路線（stage4 §4.2，2026-10-07）
- 選 **Unicode code point 直通**：階段 3.3 的 detour log 顯示繪字入口 A／B 收的是 `wchar_t*`（UTF-16），STRINGDATA 本身也是 UTF-16（`notes/re/misc.md:34`）；不需要雙位元組合成。
- 機制選**資料擴充**而非逐次繪製：繪字、量寬、換色三條路徑都只讀「字元表（`[0x006eb704]`）＋字型物件的 ETRLE 記錄陣列」，線性搜字元表得索引再取記錄（`notes/re/draw.md`）。
  dll 在字型載入函式 0x00406e00 回傳後把該字型的記錄陣列擴成 252＋N 筆（CJK 字形 ETRLE 編碼後附在像素資料尾），字元表擴成 252＋N 字。
  理由：引擎原路徑照畫（同緩衝區、同裁切、同換色），不必自己寫 blit；代價是每個字型多 N 筆記錄的記憶體（574 字 × 18 字型，量級 MB，可接受）。
- 字色：取該字型 'A' 字形的調色盤索引，所以換色路徑照常生效。≤4 色：剔除亮度 <最亮者 1/4 的索引（陰影）後取最多的單一索引；漸層字型逐列取眾數。
  - 字型調色盤在物件 +0xe0（16bpp，555，最高位另作他用）；blit 0x00411de0 以像素索引查它。
  - Wiz_Text_Font（slot 13）的 'A' 是亮色 fbda＋較多的近黑 8001（陰影），不是兩色逐列交錯；只取最多的會畫成黑字（run 033741 字色 FAIL）。
  - 標題字型（slot 17）是 4 階漸層，英文墨色中位數＝最多的 dab2；取最亮的會偏亮（run 040151 FAIL）。
  - 各字型用到的索引：ddraw log `font colors`（≤8 色才列明細）。現行規則 run 040805 字色 5 條全過。
  - 中文沒有陰影：英文陰影像素不在中文字形裡。
- 字高：遊戲字型字高夾到圖集範圍 7–24 取同高字形，基線 0.88 em。
- 已知上限：
  - 無陰影、無反鋸齒（1-bit）；漸層字型只近似逐列色。
  - 換行與寬度未處理：CJK 字寬＝字高（全形），長譯文可能溢出。
  - 7–8 px 字型（WIZFONTD*）的中文難以辨識。
  - 字表手列 574 字，表外字元照原路徑（找不到→索引 0，畫成 'A'）。
  - 舊字元表不 free（一次性，數百 bytes）；舊字型像素與記錄以 msvcrt free 釋放（與遊戲同一個 CRT，`call *0x005eb224`）。

---
<!-- 原檔 notes/2026-10-07-as-found.md -->
## 2026-10-07 安裝目錄現況（as-found）

基準：`baseline/2026-10-07-steam-as-found.json`（541 檔；同日重掃 diff＝0 added/0 removed/0 changed）。
來源：`/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8`，Steam appid 245450。

### 已不是乾淨 1.24
- `Wiz8.exe` 3580513 B（mtime 2026-10-06 19:04）≠ `Wiz8_v128.exe` 3299840 B（2020-11-14）。
- FanPatch 痕跡：`Wiz8.dll`、`cfagent1.28.dll`、`Lua5.1.dll`、`Data/FanPatch.Dat`、`Patch128_README/`、`ParallelWorld/`、`Localization/`。
- 目錄內有 `Wizardry 8 Fan-Patch 1.28 build 6735 Mini-Update.exe` 與 `... Beta build 6719 ...`（mtime 2026-09-27）。
- nGlide 已在：`glide*.dll`、`nglide_config.exe`；Steam installscript 寫 nGlide 登錄值與 `RUNASADMIN HIGHDPIAWARE`。
- `Patches/PATCH.001`＋`Wiz8Patch2001_12_23_README.txt`（官方 patch 檔）。
- LAA 旗標：`Wiz8.exe` Characteristics=0x10e、`Wiz8_v128.exe`=0x10f，**都沒有** 0x20（NTCore 4GB 未套）。[已驗證：PE header 讀值]
- 誰在何時套了 1.28（Steam depot 自帶或手動）：未取得。mtime 只是線索。

### 執行期會變的檔（之後 diff 的雜訊來源，未實測）
`Saves/`、`wiz8.ini`、`Wiz8.CFG`、`cfagent.log`、截圖 `Wiz8*.JPG`。[推測：無實測支撐]

### 波蘭文在地化的做法（參考脈絡，未深入）
- FanPatch 1.28 自帶語言包機制：`Localization/Fan-Patch1.28/<LANG>/` 放 `Info.ini`（`Lang=POL`、`LangInheritance`）、`Lang.SLF`（5313021 B）、`Text.txt`（FanPatch 新增 UI 字串，波蘭文 cp1250）。
- 對照：`Localization/Fan-Patch1.28/1.28 default eng text.txt`（英文模板）；`Data/RUS.SLF` 4963865 B vs `Data/ENG.SLF` 62522 B。
- `Lang.SLF` 開頭是 `LANG.SLF` 名稱欄位＋零填充，內含關卡名字串（Arnika2、Bayjin1…）——SLF 是原版封存格式。
- 意涵（未驗證）：中文化可能走「新增一個語言目錄」而不改原版檔，符合不動原版的原則；瓶頸預期在字型／雙位元組字元渲染，不在文字抽換。[推測：無實測支撐]

---
<!-- 原檔 notes/2026-10-07-clean-install.md -->
## 2026-10-07 Steam 乾淨重裝（取代 as-found 作為疊層起點）

MG 2026-10-07：`Wizardry8` 為乾淨安裝；`Wizardry8_Uninstall` 為 Steam 解除安裝後留下的檔案。
基準：`baseline/2026-10-07-steam-clean.json`（189 檔）、`baseline/2026-10-07-steam-uninstall-leftovers.json`（379 檔）。

### hashdir diff 結果
- clean → as-found：`354 added, 2 removed, 2 changed`
  - removed：`Saves/steam_autocloud.vdf`、`Saves/Characters/steam_autocloud.vdf`
  - changed：`3DVideo.CFG`、`Wiz8.CFG`（設定檔，執行期寫入）
  - added 分布：`User/` 244、`Patch128_README/` 42、`Saves/` 25、`Data/` 17、`Localization/` 4、`Dll/` 2，其餘為根目錄單檔（`Wiz8_v128.exe`、`Wiz8.dll`、`cfagent1.28.dll`、`Lua5.1.dll`、`d3dim700.dll`、`wiz8.ini`、Mini-Update 安裝檔…）
- as-found → leftovers：`0 added, 162 removed, 0 changed`；leftovers ＝ 上述 354 個新增檔 ＋ 25 個 `Saves/*.SAV`（與 clean 中同名檔 hash 相同）。

### 事實
- **`Wiz8.exe` 在 clean 與 as-found 完全相同**（3580513 B，sha256 `18a74ff6…`）：FanPatch 1.28 不改原版 exe，另放 `Wiz8_v128.exe`＋`Wiz8.dll` 等。[已驗證：hashdir diff 未列 Wiz8.exe]
- clean 的 `Wiz8.exe` 無 LAA（Characteristics 0x10e）。
- Steam depot 本身就帶 nGlide（`glide*.dll`、`nglide_config.exe`）與官方 patch 檔（`Patches/PATCH.001`、`Wiz8Patch2001_12_23_README.txt`）。
- clean 的 `Saves/` 有 25 個 .SAV：Steam Cloud 回灌，不是 depot 內容 [推測：無實測支撐]。
- clean 的 exe 是不是 1.24：未取得（exe 沒有 FileVersion 字串；README 只寫 2001-12-23 patch）。

### 對疊層的意涵
- 起點＝clean。`Wizardry8_Uninstall` 是現成的「FanPatch＋使用者資料」差集，可當第 3 層的對照組。
- `Saves/`、`*.CFG` 會隨執行變動，之後 diff 先把它們歸為預期雜訊。

---
<!-- 原檔 notes/2026-10-07-overlay-rules.md -->
## 覆蓋規則決策表（stage2 §2.2，2026-10-07 實測）

TL;DR：`Patches/PATCH.00N` 覆蓋 `DATA.SLF`；同名時編號大者贏；entry 名稱不分大小寫、斜線不正規化（`/` 查不到）。
四格與 Wiz8.exe 靜態分析（`notes/re/misc.md:17`）的預測一致，沒有衝突。鬆散 `Data/` 檔未取得。

### 方法
- X＝`OPTIONS\INTRO_BG_TEXT.STI`（主選單底圖；唯一存在於 `DATA.SLF`，`PATCH.001` 沒有）。
- 標記 patch 由 `scripts/mk-overlay-patches.sh` 產生（原文衍生，`build/overlay/`，不入版控）：X′＝(40,40) 白塊→區域 A，X″＝(200,40)→區域 B。
- 每個情境用 `scripts/wiz8-run.sh scenarios/overlay-*.json` 跑一次，t40 截主選單；放置檔的 sha256 記在各 run JSON 的 `placed`。
- 判定：`bash scripts/overlay-table.sh <雜訊 run…> <run…>`，區域改變比例 ≥0.5 算有標記。
- 雜訊底：3 次 noise run 兩兩比較，四區皆 0。倍數欄的分母取 max(雜訊底, 1 像素／區域像素)，所以 ×86400＝區域內 86400 像素全變（雜訊底為 0 時倍數就是像素數）。

### 決策表
| 問題 | 情境 | run-id | A | B | 判定 | 靜態預測 | 一致 |
|---|---|---|---|---|---|---|---|
| PATCH.002 vs DATA.SLF | p2：PATCH.002＝X′ | 20261007-025357-overlay-p2 | 1（×86400） | 0 | PATCH.002 贏 | patch 在 base SLF 之後 append，同前綴後者覆蓋 | ✓ |
| PATCH.002 vs PATCH.003 | p23：PATCH.002＝X′、PATCH.003＝X″ | 20261007-025524-overlay-p23 | 0 | 1（×86400） | PATCH.003 贏 | `Patch.%3.3d` n＝0..49 依序 append，後者覆蓋 | ✓ |
| 大小寫 | lower：entry 名全小寫 | 20261007-025650-overlay-lower | 1（×86400） | 0 | 不分大小寫 | 比較用 `_stricmp` | ✓ |
| 斜線方向 | slash：entry 名用 `/` | 20261007-025817-overlay-slash | 0 | 0 | 未覆蓋（`/` 不等於 `\`） | 不正規化斜線 | ✓ |
| PATCH.001 vs PATCH.002 | — | — | — | — | 未直接實測（X 不在 PATCH.001）；由 p23 的編號規則與靜態分析推得 PATCH.002 贏 | 同上 | [推測：p23 同機制外推] |
| 鬆散 `Data/` 檔 | — | — | — | — | 未取得：未核可暫放鬆散檔 | 靜態上 0x004131b0 只查 SLF 表，未見鬆散檔路徑 [推測：未追完所有開檔呼叫] | — |

雜訊 run：20261007-023242-noise、20261007-023408-noise、20261007-023535-noise。

### 對出貨的意義
- 我方中文 patch 用 `PATCH.002` 以上即可覆蓋原版；跟其他 mod 衝突時，編號大的贏。
- entry 名一律用 `\`；`tools/slf pack` 不排序，entry 必須自己照 stricmp 排好（遊戲用 bsearch 查）。

### 沿革
- 第一次挑 X＝`OPTIONS\INTRO_BG.STI`：p2／p23／lower／slash 全部零（023729、023856、024024、024150 及重跑 024608、024732、024900 等）。
  probe（四張 INTRO_BG* 全標）A 區 100%（024441）→ patch 有被讀，是挑錯資源；which（四張分標 A–D）只有 D（025211）→ 主選單用 `INTRO_BG_TEXT`。這是 §2.2 允許的那一次換 X。
- 產生器曾在 `pipefail` 下讓 `tail | head` 吃到 SIGPIPE、`set -e` 靜默退出，什麼都沒產生；改用 `dd skip_bytes,count_bytes`。早期 run 沒記放置檔雜湊，之後補上 `placed` 並重跑。

---
<!-- 原檔 notes/2026-10-07-c2-ddraw-proxy.md -->
## C2：只寫 log 的 ddraw.dll 實機載入（2026-10-07）

DLL：`hook/ddraw/out/ddraw.dll`，由 commit 1715e3f 的 `hook/ddraw/build.sh` 建出，`check.sh` PASS（pei-i386，22 個匯出與系統 SysWOW64\ddraw.dll 一致，匯入 KERNEL32／USER32）。

### 結果
- **載入：[已驗證：runtime_reproduction]** `%TEMP%\wiz8-mod-ddraw.log`（`C:\Users\User\AppData\Local\Temp\`）內容：
  `2026-10-07 01:50:24.219 pid=26148 exe=C:\Program Files (x86)\Steam\steamapps\common\Wizardry8\Wiz8.exe attach real=C:\WINDOWS\SysWOW64\ddraw.dll missing=0`
  → 遊戲目錄的 `ddraw.dll` 先於系統版被載入（不在 KnownDLLs 擋路），系統 ddraw 的 22 個匯出全部解析到。
- **執行：** `Start-Process -PassThru` → pid=26148；`WaitForExit(60000)` 逾時＝60 秒後仍在跑；`Get-Process`：`26148 Wiz8  MainWindowTitle=Wizardry 8`。tasklist：`"Wiz8.exe","26148","Console","1","244,268 K"`。
- **畫面／錯誤視窗：未取得**（看不到桌面；無頭）。Application 事件記錄同時段無 Wiz8 崩潰事件（只有 Security-SPP／SecurityCenter）。
- **UAC：** 未出現（`__COMPAT_LAYER=RunAsInvoker` 生效；tasklist 無 consent.exe）。Steam 未要求互動（就算有也看不到——未取得）。
- 收尾：`taskkill /IM Wiz8.exe /F` → 「成功: 處理程序 "Wiz8.exe" (PID 26148) 已經終止了。」

### hashdir
| 步驟 | 比對 | 結果 |
|---|---|---|
| 1 | clean → `baseline/2026-10-07-c2-before.json` | `A steam_appid.txt`、`M 3DVideo.CFG`、`M Wiz8.CFG`（見下「偏離」） |
| 2 | before → 放入 dll 後 | `A ddraw.dll`，1/0/0（兩次啟動各一次） |
| 4 | before → 移除 dll 後 `baseline/2026-10-07-c2-after.json` | `summary: 0 added, 0 removed, 0 changed`，rc=0（強殺未寫 CFG） |

### 偏離
1. **步驟 1 多出 `steam_appid.txt`（8 B，內容 `245450`＝Wiz8 的 Steam appid）與兩個 CFG 變動**：mtime 01:27–01:32，晚於 clean 基準（01:06）、早於本次 C2，即 C2 之前遊戲被啟動過一次。as-found 安裝也有這個檔。判為啟動期雜訊、繼續 C2；goal 字面只列 `Saves/`、`*.CFG`。
2. **goal 的啟動命令不能用**：`cmd.exe /c "set __COMPAT_LAYER=RunAsInvoker&& cd /d <dir> && start Wiz8.exe"` 第一次：cmd 的 AutoRun 先印字碼頁、帶引號的 `cd /d` 報「檔案名稱…語法錯誤」，遊戲沒起來；改 `cmd.exe /d /c 'set …&& start "" Wiz8.exe'`（cwd 在遊戲目錄）：cmd 卡住、無 Wiz8.exe、無 log，120 秒後收尾（taskkill 回「找不到處理程序」、移除 dll、diff 0/0/0）。原因未取得。
   第二次改用 PowerShell：`$env:__COMPAT_LAYER="RunAsInvoker"; Start-Process -FilePath <Wiz8.exe> -WorkingDirectory <dir> -PassThru`，成功（上面的結果）。所以實際啟動了「一次」遊戲，不是兩次。
3. 未做「無 dll」的對照啟動，所以第 1 層「套用前症狀」未取得。

### 順帶發現
- Windows 在 01:35 左右重開過（Security-SPP「上次開機後硬體已經變更」）。同時段 wiz8-mod 的 `.git` 有 5 個物件變成 0 byte（commit e886169 遺失；已用工作區重建為 05bc942，備份 `../wiz8-mod.git-backup-1791308055`）。

---
<!-- 原檔 notes/2026-10-07-cjk-acceptance-rules.md -->
## 4.3 判讀規則（2026-10-07，先寫規則、後跑資料）

起因：stop hook 指出 eb60b5c 的 4.3 有兩個缺口：門檻沒有明寫是從雜訊底推導的；字色沒驗。本檔在下面兩項資料取得前 commit。

### 1. 雜訊底（本情境重量）
- 資料：`cjk-empty` 再跑一次，與 run 033930 的 party.screen.png 整張逐像素比對（`tools/shot diff`）。
- 規則：差異像素數＝雜訊底 N0。遮罩 xor 上限＝N0 落在比對區域內的部分；N0＝0 → 上限 0（即 eb60b5c 用的值）。
- N0 > 0 時，eb60b5c 的上限 0 失去依據：改以 N0 在區域內的像素數當上限並重跑 cjk-check。

### 2. 字色
- 量法：遮罩像素（同 cmp.go 的局部門檻）在「有字幀」上的各通道中位數＝墨色。
  - 中文：中文幀 vs 空字串幀，在中文字串的位置。
  - 英文：無 patch 幀 vs 空字串幀，在英文字串的位置（vanilla text log 的 x,y；寬取 字數×gh，高 gh，外加 3 格）。
- 規則：同一條字串，中文墨色與英文墨色各通道差 ≤ T。
- T：遮罩內的像素至少有 0.815 權重是墨（雙線性上限 0.185，cmp.go），所以濾波造成的中位數偏移 ≤ 0.185×（墨色與背景的通道差）。T＝ceil(0.185×英文墨色與空字串幀同位置背景中位數的最大通道差)。
- 已知可能 FAIL 的情況：Wiz_Text_Font 的英文字是兩個調色盤索引逐列交錯（notes/zh-patch-log.md:23），中文只取一個索引。這時英文中位數是兩色混合，若超過 T 就照實記 FAIL 並寫進「待 MG」，不放寬 T。

---
<!-- 原檔 notes/2026-10-07-pitch-checklist.md -->
## 實機檢查清單：行距 +1（664e66b）＋NPC 對白（94167ce）

給 MG 實機用。每項看完記「正常／異常＋PrintScreen」（代理存 `%USERPROFILE%\Pictures\Wiz8\`）。
建置：`bash hook/ddraw/build.sh`；`bash scripts/build-zh-patch.sh build/zh/PATCH.002`；跑 `wiz8-run.sh play-free`。

### 行距（Wiz_Text_Font 系 slot 5／13／14／15 行高 11→12；其餘字型不變）
來源：`hook/ddraw/proxy.c` `line_height`（整個取代 0x004071f0，88 處呼叫都經它）。log 有 `pitch slot=N +1` 四行＝生效。
行距變大後，**固定高度的框可能少一行、捲動或溢出**，下面各項都要看：

1. 訊息區（畫面下方隊員台詞／事件訊息）：首行頂、末行底都不被裁；一屏能顯示幾行（跟原版比少了幾行）。
2. 物品說明框（右鍵物品）：首行、末行完整；長描述會不會被截掉或出現捲動。
3. 法術說明框（法術書點法術）：同上。
4. 技能說明框（角色頁點技能）：同上。
5. 戰鬥訊息 log：行有沒有互相疊、一回合的訊息有沒有被擠掉。
6. 戰鬥敵人列表（戰鬥中左上／敵群清單）：列有沒有超出框、最後一群看不看得到。
7. 交易列表（商店買／賣清單）：每頁筆數有沒有變少、最後一筆有沒有被框裁、捲動是否正常。
8. 魔法列表（施法時的法術清單、法術書清單）：同交易列表。
9. 組隊／角色名冊列（建角畫面隊伍列）：英文名與中文行貼在一起。同 build 對照（run 164054 +0／164309 +1）兩邊都有、名冊列 0 像素差 → 原本就有，不是行距造成；行距 +1 不影響名冊。要不要另修請裁示。
10. 按鈕字（ButtonFont，slot 10，**沒改**）：位置應與前一版相同；若變了＝行距 hook 波及，請回報。

### NPC 對白（Myles、Ferro、Antone；開新遊戲到 Arnika）
11. 三人對話視窗的台詞是中文；長台詞有沒有溢出對話框。
12. 台詞裡的關鍵字（如 Arnika、Dark Savant、bank）是英文、兩側有空白；點／打這些關鍵字，NPC 有回應。
13. 關鍵字清單有沒有把台詞裡出現的關鍵字加進去（關鍵字清單寫進存檔，要開新遊戲）。
14. Myles 交易畫面（7.4）：買賣流程正常。
15. 一般單字型關鍵字寫成「銀行（ bank ）」這種格式，讀起來可不可以接受（不行就改成只留英文或只留中文＋另列清單）。

### 階段 7 修正（2026-10-07，MG run 170652 回饋）
16. NSF 對白截斷：Myles 問關鍵字（Marten's Bluff、Trynton），回覆是完整中文，不是只剩前段英文；對話框上方橫幅與下方訊息區的回顯都不是亂碼。ddraw log 有 `hooked nsf` 與 `nsf widen utf8`。自動驗收：run 175048（text log 回覆與 `translations/nsf/NPC_MYLES.json` 逐字相同；截圖初驗橫幅中文無亂碼）。
17. 訊息區行距：訊息區 y 間距 12（原 11，寫死在 0x0058c749，不經 0x004071f0）。**行數 7→6**：行距 12 時框只容 6 行（run 180720 第 7 行被框底裁），滿 7 行時改畫最新 6 行（0x0058c62c）。看：末行底不被下方狀態列（如「5 - 吟遊詩人」）或框裁、首行頂不被裁；最新一則訊息在最底行；往上捲再捲回底部，內容連續不跳行（捲到頂時最舊一行看不到是已知上限）。ddraw log 有 `patched msg pitch 0x0058c749`、`hooked msg first 0x0058c62c`。自動驗收：run 181402（y＝366／378／…／426，子 agent 判讀 6 行、首末行完整）；捲動未自動驗（點擊不可靠）。**MG 實機 run 185712（2026-10-07）：「測試起來感覺沒什麼問題」**；log 訊息區 y 只有 366…426（無 438），Myles 問 Mt Gigas 回覆多行中文完整。
   - 補（2026-10-08）：行距 12 與「滿 7 行畫 6 行」只限窗格 0（探索時訊息紀錄）；戰鬥時訊息與清單同為窗格 2，維持 11（清單點擊判定寫死 ÷11，套用會讓游標垂直漂移、首項被藏）。戰鬥訊息／施法清單第 7 行底被裁 → 文字區下緣 +2（0x0058c5c9）。log 多 `hooked msg clip 0x0058c5c9`，pitch 改為 `hooked msg pitch 0x0058c73e`。**MG 實機 run 013125：「兩處都完整了，沒壓到框」**；清單對齊 run 011741：「清單對齊了」。
18. NSF 對白缺字：Myles「你懂的，介紹費。」的「紹」不再畫成「A」（MG run 185712 截圖 Wiz8-20261007-185937-323）。原因是圖集字表漏 translations/nsf，35600af 修（2321→2355 字，atlas-cover PASS）。待下次實機一起看。

### 收尾 goal 第 2、3 步（2026-10-07 晚）
19. 日記（FACT.DBS 337 句，translations/fact.json）：按 `j` 開日記，事件欄是中文。看：長句（例「摧毀 Savant 的船。用 Gigas 山頂的大砲。」）不超出羊皮紙右緣；跨多筆接續的句子（「……」開頭，例 Antone 配方、開場三行）讀起來順。自動驗收：run 212736（`j` 開得到；text log slot 1 19 行中文；子 agent 判讀無裁切、最長行止於 x≈1825／頁緣 x≈1960）。
20. 日記字形：子 agent 判讀「。」像小圓圈、「！」像小點塊（日記字型 2× 放大的 12px 點陣）；標題「地點」「事件」淡、無陰影但可讀（MG 原本回報「點狀破碎」）。要不要處理由 MG 看。
21. 戰鬥 log 中文行距：/LOAD 存檔訊息區只有英文（存檔訊息、吟遊詩人治療），中文戰鬥訊息與 12px 字框的相疊**未取得**；戰鬥中看訊息區每行上下不相疊、不被裁。
22. 底部狀態列：run 212736 主畫面子 agent 判讀「5 - 吟遊詩人」、頭像名牌、HP 數字皆無擠壓重疊；MG 原本回報「擠」的情境（哪個畫面）未取得，請實機遇到時截圖。
23. NPC 對白「你們」抽樣（81a2455）：任找幾位 NPC，對全隊說話用「你們」，單人指名用「你」。

### 不在本輪
- ButtonFont 等其餘字型的行距不動。

---
<!-- 原檔 notes/2026-10-07-release-install-test.md -->
## 發布包安裝實測（2026-10-07，goal 第 5 步）

證據方式：isolated_simulation（Windows 暫存目錄的原版最小複本，不是 MG 的遊戲目錄；遊戲本體未在此複本上啟動）。

- 包：`bash scripts/release.sh <zip>` @ 7f9a0f5＋本 commit 的 release 檔；233 檔（ddraw.dll、zhpatch.exe、translations 229 檔、bat×2、INSTALL.txt、OFL×2）。包內白名單機檢在 release.sh。
- 複本：`%TEMP%\wiz8-inst-test\game\`＝Steam 原版的 Data\DATA.SLF、Patches\PATCH.001、Wiz8.exe（147M），裝前 sha256 清單存 scratchpad。
- 呼叫方式：wrapper bat `call "<pkg>\%1" "<game>"`（WSL 直接傳帶引號參數給 cmd.exe 會被跳脫成 `\"…\"`，那是測試端的問題，不是 bat 的）。

| 步驟 | 結果 |
|---|---|
| install.bat | exit 0，「Installed」；只多 ddraw.dll、Patches\PATCH.002 |
| PATCH.002 對照 | 與同 HEAD `build-zh-patch.sh` 新產物 `cmp` 相同 |
| ddraw.dll 對照 | 與 hook/ddraw/out/ddraw.dll `cmp` 相同 |
| 原版三檔 | 裝後 `sha256sum -c` 全 OK |
| 再裝一次 | exit 1，「PATCH.002 already exists」 |
| uninstall.bat | exit 0；目錄 sha256 清單與裝前 `diff` 無差 |
| 別人的 ddraw.dll ＋ uninstall | exit 1，「not ours - left alone」，檔案原樣 |
| 譯文 sha 竄改（fact.json）＋ install | zhpatch 拒絕、exit 非 0，未產 PATCH.002、未複製 ddraw.dll；目錄與裝前相同 |

已知上限：
- zhpatch 錯誤訊息是 UTF-8 中文，cp950 主控台會顯示亂碼（英文部分與檔名、id 仍可讀）。
- 未在複本上啟動遊戲（實機效果已由 MG 遊戲目錄的 runs 驗過，同一份 PATCH.002／ddraw.dll bytes）。
- GOG 版、非 1.2.4 版未取得。

### 2026-10-09 跨版本移除／升級（修 uninstall 拒移舊版）

證據方式：isolated_simulation（同上 `%TEMP%\wiz8-inst-test\game` 原版複本）。缺陷：ddraw.dll 每次建置不同（atlas 隨字集變；新 2431678c、舊 c4e1aff1），uninstall 用 `fc /b` 比對本包 dll → 新包拒移舊安裝。修法：認 dll 內嵌字串 `wiz8-mod-ddraw.log`（findstr）；install 遇 ddraw.dll 先 call uninstall；release.sh 機檢 dll 含該字串。

| 情境 | 結果 |
|---|---|
| 乾淨安裝 | exit 0 |
| 舊版 dll＋PATCH.002 上跑新 install | 先移除、再裝，exit 0，dll＝新包 |
| 新包 uninstall 對舊版 dll | exit 0，兩檔移除 |
| 外來 dll（非 wiz8-mod） | install、uninstall 都 exit 1，dll 原樣、未產 PATCH.002 |
| 只有 PATCH.002 | install、uninstall 都 exit 1，原樣 |
| 什麼都沒有 | uninstall exit 0「Nothing to remove」 |
| 注入：包內換回舊 fc 版 uninstall，對舊版 dll | exit 1「not ours」→ 還原 → exit 0 |

複本原版三檔 sha256 全程前後相同。上限：識別字串可被別人的 dll 碰巧含有（機率可忽略）；PATCH.002 仍無自身識別。

---
<!-- 原檔 notes/2026-10-07-final-checklist.md -->
## 收尾：MG 實機清單＋待裁清單（2026-10-07，/goal 第 6 步）

細項與判讀標準在 `notes/zh-patch-log.md:212`（下稱「清單」，編號照它）。這份只排順序。
每項記「正常／異常＋PrintScreen」（代理存 `%USERPROFILE%\Pictures\Wiz8\`）。
建置：`bash hook/ddraw/build.sh`；`bash scripts/build-zh-patch.sh build/zh/PATCH.002`；`wiz8-run.sh play-free`。
或用發布包（`bash scripts/release.sh <zip>`，`INSTALL.txt`）；安裝實測見 `notes/zh-patch-log.md:257`。

### 實機（建議一場走完，開新遊戲到 Arnika）

1. **Myles 完整對話＋交易（7.4）**：清單 11–16、14（買賣流程）。Myles 之後 Ferro、Antone 各一次完整對話。
2. **「紹」**：Myles「你懂的，介紹費。」不再畫成 A（清單 18）。
3. **文字框末行（cbd2e08）**：物品／法術／技能說明框首行、末行完整不被裁（清單 2–4；cbd2e08 是末行底列被裁的修正，6.4 實機時發現，CI 不驗行為）。
4. **NPC「你們」抽樣**：對全隊用「你們」、單人指名用「你」（清單 23）。
5. **日記**：按 `j`，長句不出羊皮紙、接續句通順（清單 19）；字形「。」「！」標題淡（清單 20）。
6. **戰鬥**：訊息區中文行不相疊、不被裁（清單 21，未取得）；戰鬥 log、敵人列表（清單 5、6）。
7. **底部狀態列**：原本回報「擠」的畫面截圖（清單 22，情境未取得）。
8. 其餘清單 1、7、8、10（訊息區、交易／魔法列表、按鈕字）順路看。

### 待 MG 裁（都可逆，不擋發布）

- 清單 9：名冊列英文名貼中文行（原版就有，非行距造成）要不要另修。
- 清單 15：「銀行（ bank ）」括注格式可不可以接受。
- 清單 20：日記字形（。！、淡標題）要不要處理。
- **7.3 點中文關鍵字**（`notes/re/misc.md:47`）：遊戲有 translated→English 查表，放兩個 UTF-16 鬆散檔即可啟用，不用 hook；但點字只用空白切詞，點中文會拿到整段而對不上。要生效得在全部 888 處括注的中文兩側加空白（改排版）。目前沒開碼票；點括號裡的英文應可用（推測，未實測）。看完清單 12、15 再決定。
- 發布：zhpatch 錯誤訊息是 UTF-8，cp950 主控台會亂碼（英文與檔名可讀）；GOG 版未取得。

---
<!-- 原檔 notes/2026-10-07-sandbox-net-probe.md -->
## 2026-10-07 沙箱網路探針（A0）
命令：`sandbox-run.sh agy . curl -sS -o /dev/null -w '%{http_code}' -A 'wiz8-mod personal research (non-commercial)' https://www.pcgamingwiki.com/w/api.php`
輸出：`200`，rc=0。`sandbox-run.sh --selftest .`：`selftest PASS`。
結論：沙箱有網路，研究走 agy／muse，不需退回 Claude 子 agent。[已驗證：上列命令輸出]

---
<!-- 原檔 notes/2026-10-07-fan-translations.md -->
## 2026-10-07 既有玩家翻譯盤點（摘要）

全文：`notes/research/2026-10-07-fan-translation-{ja,fr,de}.md`（Claude 子 agent 研究，網路來源未逐條人工複核）。

| 語系 | 機制 | 雙位元組／特殊字元 | 對本專案 |
|---|---|---|---|
| 德 v0.97（2003） | 遊戲內建 Patches 系統：寫入 `Patches\PATCH.002`（Sir-Tech 格式），內含 DBS／STRINGDATA.DAT／MSG／NSF／字型 STI；解除安裝即還原 | 換 STI 字型處理變音字母；parser 不吃變音；名稱有長度上限 | **最貼合「不動原版」**：`PATCH.00N` 疊層，clean 已有 `Patches/PATCH.001` |
| 日 v0.4（2013） | 拿日本零售 1.2J 的 `Wiz8.exe`＋Data 蓋英文版，再加 STI 點陣字型與部分 DBS／MSG | 英文 exe 不支援 Shift-JIS；作者試過放棄（日文處理散在多處客製碼） | 證實瓶頸：英文 exe 的 DBCS 渲染要靠 hook 補 |
| 法（2020） | 自解壓 patcher 覆寫遊戲檔（含改過的 `Wiz8.exe`），主體加密 | 未取得 | 機制與原則相反，僅參考範圍 |

另：FanPatch 1.28 自帶 `Localization/Fan-Patch1.28/<LANG>/`（見 `notes/zh-patch-log.md:77`）。
三份都未提與 FanPatch 1.28 的相容性。

---
<!-- 原檔 notes/2026-10-08-ci-speedup.md -->
## CI 提速與發布包實裝（2026-10-08）

TL;DR：CI 394 → 72 s（501e594），靠的是先逐步計時再動手，而不是整體平行。發布包首次裝進 MG 的 Steam 目錄（3df5b4a）。
下面是可重用的做法與踩到的坑，數字都出自當天實跑。

### CI 提速

逐步計時 @2ef9e1ec（git archive 出來的樹，逐行包 `$EPOCHREALTIME`）：

| 步驟 | 前 | 後 | 做法 |
|---|---|---|---|
| `scripts/check-lib.test.sh` | 252 s | 41 s | 10 次 check-lib.sh（各約 25 s，逐檔起 jq／go／python）背景平行，結果寫 `$S/res.*` |
| `go test ./...`（`tools/slf` TestRealArchives） | 52 s | 29 s | 子測試 `t.Parallel()`，下限是最大檔 SOUND.SLF（600 MB） |
| `scripts/ci-check.sh` 整體 | 394 s | 72 s | 5 道平行（go／hook／tr／lib／zhpatch），各道 log 依序印 |

- **先量再平行**：單一步佔 64%，只做步驟間平行最快也是 252 s。瓶頸在步驟內部（N 次注入 × 完整檢查）。
- **共用輸出的步驟同一道、照順序**：`hook/ddraw/build.sh` 寫 `hook/ddraw/out/`，`check.sh`、`atlas-cover.sh` 讀它 → 同在 hook 道。其餘測試各自 `mktemp`，可分道。
- **平行後的計數由彙總方算**：check-lib.test 不再在背景累加 `fails`（子殼改不到父殼變數），改成數結果檔裡的 PASS 行必須剛好 10。背景工作半路死掉、沒寫結果，也會因為數目不對而紅。
- **bash 坑**：子殼放在 `if`／`&&`／`||` 裡，子殼內的 `set -e` 失效（失敗後照跑下一行）。`lane()` 改成先 `set +e`、子殼單獨一行、再取 `$?`。
- **步驟清單比對**：改寫前後用 `grep -oE 'bash [^ ]+|go test…'` 抽步驟排序後 diff，確認沒掉檢查。

注入驗證（地板 §7.6）：
- check-lib.sh 改 `exit 0` → check-lib.test 紅（全部「仍綠」）。
- space.test.sh 中途 `exit 1` → ci-check hook 道 rc=1、整體 rc=1。
- 第一次注入放在 `set -euo pipefail` **之前**（`false`），CI 照綠：是注入沒生效，不是平行化壞掉。注入要用 `exit 1`，且放在 `set -e` 之後。
- 用 `mv` 換檔做注入會掉執行位元（100755→100644），`cp` 還原副本不會改回 mode。注入用 `sed -i` 原地改，還原後 `git status` 確認連 mode 都乾淨。

### 發布包實裝到 Steam 目錄

- `release.sh` 打包 → 解到 Windows 端 `%TEMP%\wiz8-mod-install\` → 在那個目錄 `cmd.exe /c install.bat`（不帶參數用 Steam 預設路徑），避開 WSL 傳帶引號參數給 cmd.exe 會被跳脫的問題（`notes/zh-patch-log.md:257`）。
- **`cmd.exe /c 'echo %TEMP%'` 的輸出第一行是「使用中的字碼頁: 950」**，直接拿去 `wslpath` 會得到多行亂碼路徑，`mkdir -p` 就在 repo 裡建出亂碼目錄。Windows 端路徑用已知的 `/mnt/c/Users/User/AppData/Local/Temp`，或只取輸出最後一行再驗證存在。
- wiz8-run 每場收尾都會刪掉自己放的 ddraw.dll／PATCH.002，平時 Steam 目錄是乾淨原版；手動裝了發布包之後，wiz8-run 會拒跑（「已存在於遊戲目錄，拒絕覆蓋」），要先 uninstall.bat。
- 裝著 PATCH.002 時，slf 的 TestRealArchives 會把它當封存檔一起驗（目前綠）。

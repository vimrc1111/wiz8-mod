# 階段 2：眼睛與覆蓋規則（不改 exe 行為）

> 執行者：cc。開場照地板固定動作，讀 `CLAUDE.md`、`resume.md`、`decisions/2026-10-07-re-unlock-and-scummvm-principle.md`、本檔。
> 前置：階段 1 的 B 稽核收尾（`~/Projects/games/wiz8-mod-audit`；發現先轉成會紅的測試；收件在案目錄 commit）。
> 設計目標：MG 不在場也能跑完。需要 MG 的事 → `resume.md`「待 MG」，換下一項。

## 已知事實（2026-10-07 實測）
- 遊戲目錄有 nGlide（`glide2x.dll`、`nglide_config.exe`），`3DVideo.CFG` 首行 Glide2x 1280×1024：3D 畫面走 nGlide，不是 ddraw。ddraw 代理仍會被靜態匯入載入（C2）。
- WSL 端用 PowerShell `CopyFromScreen` 可以截桌面（2560×1440，已實測）。
- `HKCU\Software\Zeus Software\nGlide` 查無（沒存過設定）。

## 邊界
- 原版遊戲目錄只讀，例外是本階段要暫放的我方檔案（`ddraw.dll`、`Patches/PATCH.00N` N≥2）。每次執行前後都 hashdir，結束必須回到 clean（只准執行期雜訊）。
- 截圖含原作美術：**不入版控**，存 `~/.local/state/wiz8-mod-runs/<run-id>/`。repo 只收量測結果（JSON：雜湊、像素統計、區域差異）。
- 從原版資料改出來的標記檔（測試用 PATCH）同樣不入版控；repo 只收「產生它的腳本」。
- 鍵盤輸入只在前景視窗屬於 Wiz8 的 PID 時送出，否則不送、記錄、跳過。凌晨無人看螢幕，送錯視窗會打進別的程式。

## 2.1 執行器 `scripts/wiz8-run.sh <情境檔>`（cc 寫）
- 情境檔（JSON）宣告四件事：
  - 要放入的我方檔案；
  - 時間表，例如 `{"t":20,"shot":"menu"}`、`{"t":8,"keys":"{ESC}"}`；
  - 總時長；
  - run-id。
- 流程沿用 C2 已驗證的做法：
  1. hashdir 掃描，與 clean diff；
  2. 放入檔案；
  3. 用 PowerShell `Start-Process` 啟動，帶 `__COMPAT_LAYER=RunAsInvoker`；
  4. 依時間表截 Wiz8 主視窗的矩形，視窗找不到時截整個螢幕；
  5. 不論成敗都 `taskkill`；
  6. 移除放入的檔案；
  7. 再掃描、再 diff。
- 每次執行還要記：
  - `Get-Process -Module` 的模組清單，用來判定實際渲染路徑，例如 nGlide → d3d9／vulkan；
  - 每張截圖的 sha256、標準差，以及它是不是全黑或單色。
- 全黑或單色時，先懷疑擷取方式。依序試：
  1. 用 `PrintWindow` 擷取視窗；
  2. 用 nGlide 的視窗模式。設定位置用 `nglide_config.exe` 的公開說明查，改登錄或設定前先備份，跑完還原，前後 diff 都入記錄。
  - 都不行 → 記「未取得：畫面擷取全黑」以及原因，2.2 跳過，直接做階段 3 的非畫面項目。
- 測試：
  - 情境檔解析與 diff 判定抽成可離線測的部分，進 CI；
  - 實機部分不進 CI，證據寫入 `notes/runs/<run-id>.json`。
- 雜訊底：同一情境連跑 3 次，算主選單畫面的逐像素差異分布。之後所有「畫面變了」的判定，門檻都從這個分布推導，不寫字面數字（地板 §7.3）。

## 2.2 覆蓋規則實測
問題：遊戲對同名資料的取用順序。
- 鬆散 `Data/` 檔、`*.SLF`、`PATCH.001`、`PATCH.002`、`PATCH.003`，誰贏？
- entry 名稱的大小寫、斜線方向有沒有被正規化？

做法：
- 挑一個開機後 30 秒內一定上畫面的資源 X。用截圖對照找，A3 的字型／選單 STI 是候選。
- 做標記版本：只改 X 的一小塊，讓畫面上的指定區域可以機檢。
- 用 `tools/slf pack` 包出 PATCH.002／003。

情境：
1. 無 patch，作為基準；
2. PATCH.002 含 X′；
3. PATCH.002 含 X′、PATCH.003 含 X″；
4. 名稱只差大小寫；
5. 名稱只差斜線方向。

產出：`notes/2026-xx-xx-overlay-rules.md` 的決策表。
- 每一格附 run-id，以及區域差異相對雜訊底的倍數。
- 判定由腳本從 JSON 計算，不靠目測。
- 若 X 的改動在任何情境下都沒有上畫面：可能是 patch 沒被讀，也可能是挑錯了資源。換 X 一次，仍不行就記「未取得」，交給階段 3 用靜態分析回答。

## 完成定義
1. `wiz8-run.sh` 入版、離線測試進 CI，四段注入（§7.6）貼進 commit 訊息。
2. 雜訊底已量，渲染路徑已記入 `notes/zh-patch-log.md:23`。
3. 覆蓋決策表完成，或每一格標「未取得」並附原因。
4. 每次實機執行後，遊戲目錄 diff 都回到 clean，證據在 `notes/runs/`。
5. `resume.md` 已更新，工作區乾淨，最後一個 commit 有 CI exit 0 證據。

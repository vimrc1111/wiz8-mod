# 階段 1：拆格式、證明注入點（不卡 MG 版）

> 執行者：cc（Opus 5.5）。開場照地板固定動作，讀 `CLAUDE.md`、`resume.md`、本檔。
> 設計目標：全程只消耗額度，**不需要 MG 當場動作**。遇到需要 MG 的事 → 記進 `resume.md`「待 MG」，換做下一項，不停下等。
> 依據：MG 2026-10-07「其他項目我接受你的提案」「同意從原版開始」；派工分工見 `CLAUDE.md`。

## 全域問題找 claude-memory session（MG 2026-10-07）
MG 原話：「我開着 c-m 如果你遇到派工還是什麼全域 cc 的問題可以直接跟它溝通/請它修好」
- 派工腳本（`impl-run.sh`／`audit-run.sh`／`sandbox-run.sh`）、hook、地板層級的問題 → `ListAgents` 找 claude-memory 的 session，`SendMessage` 請它修；找不到就投它的 inbox（§6.9）。不自己改 claude-memory。
- 研究票型若要變成正式流程（A 的套用法跑通後），同樣交給它。
- 等它修的期間不空等：記進 `resume.md`，先做不依賴該修補的項目。

## 不變的邊界
- 原版遊戲目錄 `/mnt/c/Program Files (x86)/Steam/steamapps/common/Wizardry8` **只讀**，唯一例外是 C2 的暫時放入 `ddraw.dll`（放入前後都 hashdir，結束必須移除並 diff 回 clean，見 C2）。
- 不開 Ghidra／反組譯器，不做任何 hook（C 只寫 log）。不 push（沒有 remote）。不下載或執行第三方 patch 安裝程式。
- 下載物一律放 scratchpad 各自的空目錄，不執行。
- 每個做完驗證的單元就 commit；規格與實作分 commit。CI：`bash scripts/ci-run.sh wiz8-mod bash scripts/ci-check.sh`，每個實作 commit 後跑，exit 0 才算完成。

## A. 研究（agy → muse；grok 不做研究）
派法：`<claude-memory>/scripts/impl-run.sh <派工單> <專案根>`，`執行器: agy`（`AGY_MODEL=claude-opus-5-5-high`）；agy 週額度或 5 小時窗用完 → 同一張單改 `執行器: muse`。
派工單寫給「就是你自己」的讀者：目標、待答問題、證據規格、不做什麼，全寫進單裡，不靠對話脈絡。

A0（先做，不耗模型）：
- 寫 `scripts/check-research.sh <報告>`：每條結論行（`- ` 開頭、在「發現」段內）須含 `http(s)://` 或 `path:line`，且標 `[已驗證]`／`[推測]`／`[未取得]` 之一；違反就列行號 exit 1。四段注入驗證（壞報告 FAIL → 好報告 PASS）貼進 commit 訊息。
- 沙箱網路探針：`sandbox-run.sh` 包一條 `curl -sI https://www.pcgamingwiki.com` 之類的命令跑一次（不叫模型），記結果於 `notes/`。
  - 沙箱無網路 → A1–A3 改由 cc 的 Claude 子 agent 做（只花 Claude 額度），在 `resume.md` 記偏離與原因；**不得**繞過沙箱直接跑 agy／muse。

研究票（每張一份報告 `notes/research/<日期>-<題>.md`，驗收＝`check-research.sh` exit 0；白名單只有該報告檔）：
- A1 SLF 與 `Patches/PATCH.00N` 格式：公開文件、既有工具（含 Cosmic Forge 編輯器類）、遊戲讀 PATCH 的覆蓋規則（同名檔誰贏、編號順序）。本機樣本：`Patches/PATCH.001`、`Data/*.SLF`、`Localization/Fan-Patch1.28/POL/Lang.SLF`（在 `Wizardry8_Uninstall`）。
- A2 `ddraw.dll` 代理載入：Wiz8 是否有人這樣做過、Windows DLL 搜尋順序對 `DDRAW.DLL` 的實際行為（非 KnownDLLs？WOW64 下系統 ddraw 路徑）、與 dgVoodoo2 串接的既有做法、Steam 版 `Wiz8.exe` 的 `RUNASADMIN` 相容旗標影響。
- A3 文字與字型資料：`STRINGDATA.DAT`、`*.MSG`、`*.DBS` 文字欄位、STI 字型格式的公開說明；英文版文字繪製限制（單位元組、長度上限）的公開討論。

cc 抽查每份報告至少 3 條引文（實際開 URL 或檔案核對），抽查結果附在報告末尾。

## B. SLF 工具（codex 實作，cc 切單與複核；稽核 grok）
等 A1 回來再切單。目標 `tools/slf`：`slf list <檔>`、`slf extract <檔> <輸出目錄>`；格式允許的話加 `slf pack`。
驗收（全部機械）：
- 對 clean 安裝裡所有 SLF／PATCH 檔、以及 `Wizardry8_Uninstall` 的 `Lang.SLF` 跑 `list`：零錯誤；每個 entry 的 offset＋size 在檔案範圍內、彼此不重疊（由測試計算，不採信工具自報）。
- `extract` 後逐 entry 重讀比對 sha256。
- 若做 `pack`：`pack(extract(PATCH.001))` 與原檔 byte 相同。
- 單元測試用自製小 fixture；真實檔案的測試在檔案不存在時 `t.Skip`（CI 機器未必有遊戲）。
- 規則測試四段注入（§7.6）。
- 派 codex 走 `impl-run.sh`，`執行器: codex`；codex 不可用時 cc 自寫。稽核 `audit-run.sh` 給 grok，發現要先轉成會紅的測試才算數。

## C. `ddraw.dll` 只寫 log 的代理（cc 或 codex）
C1 建置：
- 需要 32 位元 Windows DLL 工具鏈。本機沒有 mingw；**允許** `brew install mingw-w64`（瓶頸：產出 i686 PE DLL；替代方案 clang＋lld 手寫 import lib 較脆）。安裝失敗 → 試 clang `--target=i686-w64-windows-gnu`；都不行 → 記「待 MG」，跳過 C2。
- `ddraw.dll`：`DLL_PROCESS_ATTACH` 時寫一行 log（時間、宿主 exe 路徑、PID）到 `%TEMP%\wiz8-mod-ddraw.log`（不寫進遊戲目錄），然後載入 `%SystemRoot%\SysWOW64\ddraw.dll`（64 位元 Windows）並轉送所有 ddraw 匯出。原始碼放 `hook/ddraw/`，建置腳本進版控。
- 機檢：`i686-w64-mingw32-objdump -p` 確認是 i386 PE、匯出表涵蓋系統 ddraw 的全部匯出名（從系統 dll 匯出表算，不手寫清單）。

C2 實機（不需要 MG 點擊）：
1. `hashdir scan` 遊戲目錄 → `baseline/<日期>-c2-before.json`；與 clean diff，只允許 `Saves/`、`*.CFG` 類執行期雜訊，否則停止 C2、記錄。
2. 複製 `ddraw.dll` 進遊戲目錄；再 scan、diff：只准 `A ddraw.dll`。
3. 從 WSL 啟動：`cmd.exe /c "set __COMPAT_LAYER=RunAsInvoker&& cd /d <遊戲目錄> && start Wiz8.exe"`（避開 UAC 提示）。輪詢 log 檔最多 60 秒。
4. 不論成敗：`taskkill.exe /IM Wiz8.exe /F`，移除 `ddraw.dll`，再 scan，與步驟 1 的 scan diff，只准執行期雜訊。
5. 結果（log 內容或「未取得」與原因、`tasklist` 輸出、遊戲是否顯示錯誤視窗——看不到畫面就寫未取得）記進 `notes/<日期>-c2-ddraw-proxy.md` 與 `notes/zh-patch-log.md:23` 第 1 層症狀欄。
- 若出現 UAC、Steam 要求、或任何需要點擊的視窗：不等，照步驟 4 收尾，記「待 MG」。

## 完成定義（/goal 的判準）
以下全部成立，或剩下的每一項都在 `resume.md`「待 MG」／「額度用完」有記錄與原因：
1. A1–A3 報告存在且 `check-research.sh` exit 0，各附抽查紀錄。
2. `tools/slf` 合併，CI 證據 exit 0，B 的驗收全過。
3. C1 產出 DLL 與機檢；C2 的 before／after diff 與 log 結果已記錄；遊戲目錄最終與 C2 步驟 1 只差執行期雜訊。
4. `resume.md` 已寫：完成項（附 commit）、偏離與原因、待 MG、下一步建議（含是否需要 Ghidra 解禁）。
5. 工作區乾淨，最後一個 commit 有 CI exit 0 證據。

額度規則：Claude／agy／muse／codex 任一家用完就照上面替代路走；全部可用路線都用完 → 寫 `resume.md` 收工。

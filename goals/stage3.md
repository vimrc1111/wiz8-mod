# 階段 3：定位繪字與字串（Ghidra）＋行程內 hook 證明

> 執行者：cc。前置：階段 2 完成，或其未完成項已記入 `resume.md`。
> 本階段不需要畫面：截圖若在階段 2 未取得，本階段照做，證據全走 log。

## 邊界
- 允許 `brew install ghidra`，會連帶裝 openjdk，屬已知套件。
  - Ghidra 專案檔放 `~/.local/state/wiz8-mod-ghidra/`，不進 repo。
  - 分析腳本由我們自己撰寫，放 `re/`，以 headless 執行 `analyzeHeadless`。
- 反編譯結果**不入版控**，也不貼進筆記。筆記只寫三樣：位址、前 16 bytes 簽章、行為描述（自己的話）。
- 公開的 agluszak/wizardry-8-decomp 只拿來當定位地圖：讀它來找位置，不從裡面複製程式碼。
- 研究派 muse（agy 額度用完）；Ghidra 由 cc 自己跑。

## 3.1 定位（筆記：`notes/re/`）
- **繪字**：gprintf／mprintf 一類的入口；字元 → 字形的查表方式（256 槽？字寬表？）；字串在這一層是 8 位元還是 UTF-16。
- **字串庫**：STRINGDATA.DAT 的讀取，以及 A3:8 所說的 0x96 高位元組變換規則。
- **patch 查找**：從 `%s\Patch.%3.3d` 的交叉引用找出編號範圍、順序、檔名正規化方式。與階段 2 的實測決策表互相驗證，不一致就停下回報。
- **機檢**：`scripts/check-sigs.sh` 讀筆記裡的「位址＋簽章」表，對 Wiz8.exe 逐條比對 bytes。
  - 進 CI；找不到遊戲時 skip，並印出 skip 留痕。
  - 四段注入：改壞一條簽章 → FAIL → 還原 → PASS。

## 3.2 字串工具 `tools/strings`（codex 實作，cc 切單；grok 稽核）
- `decode`：STRINGDATA.DAT → 逐筆 UTF-8 文字（JSON/TSV）。`encode`：反向。
- 驗收全機械：
  - `encode(decode(PATCH.001 的 STRINGDATA.DAT))` 與原檔 byte 相同；
  - decode 結果包含階段 2 截圖上出現的英文 UI 字串。字串清單由 cc 從截圖抄下，寫成測試期望值；沒有截圖就改用 exe 內已知字串。
  - 真實檔案測試在找不到遊戲時 `t.Skip`。
  - 四段注入。

## 3.3 行程內 hook 證明（cc 或 codex）
- ddraw 代理在 `DLL_PROCESS_ATTACH` 時對 3.1 找到的繪字入口裝 inline detour：
  - 只記錄前 N 次呼叫的字串（轉 UTF-8）與座標，寫到 `%TEMP%\wiz8-mod-text.log`；
  - 然後呼叫原函式，畫面行為不變。
- 簽章不符時，不裝 hook，log 寫 `sig-mismatch`，遊戲照常跑。
- 實機以 `wiz8-run.sh` 跑 60 秒。驗收：
  - log 裡出現 3.2 decode 得到的字串，至少 3 條；
  - 行程 60 秒仍在；
  - 遊戲目錄 diff 回到 clean。

## 完成定義
1. 三個定位主題各有筆記，`check-sigs.sh` 進 CI 且綠，附四段注入。
2. `tools/strings` 合併，往返測試全過，grok 稽核發現已轉成測試。
3. 3.3 的 text log 驗收過，證據在 `notes/runs/`。
4. `resume.md` 已更新，工作區乾淨，CI exit 0。

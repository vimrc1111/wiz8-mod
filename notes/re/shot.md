# 遊戲內截圖（PrintScreen）

靜態分析（Ghidra，2026-10-07）。反編譯輸出不入版控，以下是我方描述。

| VA | 前 16 bytes | 說明 |
|---|---|---|
| 0x004229e0 | 8b0d0cb75e0083ec2c5657ff1508b75e | 截圖函式，`void f(void)`，無參數、無回傳 |

- 唯一呼叫點：0x00426d00，位於 0x00426790 那個函式內（按鍵處理）。
- 流程：取 surface IO 管理器 → 載入 JPEGImporter 擴充 → 鎖住畫面緩衝（srGERD::lockBuffer，物件指標在 0x00659634）
  → 鎖成功才做：以全域計數器 0x00659724 產生檔名 `Wiz8%5.5d.JPG`（字串在 0x00603ed4）、計數器 +1、exportSurface 寫到工作目錄
  （0x006596f4 非零時改走 `QUALITY_0_35` 參數，並在 export 前後各呼叫 0x00439bc0／0x00439ca0）→ unlockBuffer。
- 不論鎖成功與否，最後把 0x00659728 的最低位元翻轉（用途未查）。
- 崩潰（run 042256，`runtime_reproduction` 2 次，無代理也崩）：JPG 寫出後崩在系統 ddraw.dll 10.0.26100.8115 +0x33159，0xC000041D。
  崩在 lock／export／unlock 哪一步：未取得。

## 修法與實機結果（2026-10-07）
- 代理以 jmp 完全取代 0x004229e0：擷取前景視窗（核對 pid）的 client 區，存 24 位元 BMP 到 `%USERPROFILE%\Pictures\Wiz8\`，不寫遊戲目錄；照原函式翻轉 0x00659728 最低位元。
- 擷取來源用螢幕 DC：PrintWindow(PW_RENDERFULLCONTENT) 回成功但給均勻暗圖（run 062022，平均亮度 14.2、最大 15）。
- Win11 預設按 PrintScreen 會開剪取工具，它搶走焦點，全螢幕遊戲被最小化。
  - 設定開著：run 062350 存 2 張後崩在 MSS32.DLL +0x144ac（0xC000041D）。
  - 對照：只用 Alt+Tab 切 3 次不崩（run 062707）。
  - MG 關掉「設定 → 協助工具 → 鍵盤 → 使用 Print Screen 鍵開啟螢幕擷取」後：run 063150 按 5 次，5 張 `shot ok`，活到 120 秒，after_clean，亮度 23–34／最大 136–187（非全黑）。
  - 推論（各 1 場，`runtime_reproduction`）：崩潰跟剪取工具有關；設定開著時是否必崩、原版崩潰是否同因，未取得。
- 已知上限：每張 BMP 在 4K 約 24 MB，不壓縮。

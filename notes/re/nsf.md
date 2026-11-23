# NSF 對白的窄字轉換（Wiz8.exe v1.2.4；2026-10-07 cc 靜態分析）

症狀：MG play-free run 170652 text log 第 143／147 行，Myles 的回覆只剩 ` Marten's Bluff `／` Trynton `（中文譯文第一個字之前的英文關鍵字）。

| 位址 | 前 16 bytes | 行為 |
|---|---|---|
| 0x0055a23c | 68b0c4610051ff151cb25e0083c41c33 | NSF 節點讀取（0x0055a140，由 0x0055a480 對 `Data\NPC Scripts\%s.nsf` 呼叫）裡，每個文本 chunk：配置 u16 長度＋1 **byte**，讀 長度×2 bytes 的 UTF-16，再 `sprintf(dst, "%S", 寬字串)`（格式 0x0061c4b0，IAT 0x005eb21c）轉成 char。msvcrt 在 C locale 遇 >0xFF 的字就停＝截斷點。call 在本列 +6（6 bytes `ff15 1cb25e00`）。 |
| 0x0052539b | 68b4c4610051ff1588b15e008a842444 | 轉回寬字的一處（NPC 回覆）：`swprintf(buf, L"%S", 窄字串)`（格式 0x0061c4b4，IAT 0x005eb188）。 |

- 用 L"%S"（0x0061c4b4）轉回寬字的呼叫點共 20 處：直接 call swprintf IAT（6）、`call ebp/esi`（暫存器由 swprintf IAT 載入，11）、經 0x0058aad0（2）或繪字 0x00407650（1）把格式傳給 vswprintf（IAT 0x005eb1f0）。所以 hook 放在 IAT（swprintf、vswprintf 各一格），比對格式指標＝0x0061c4b4，不逐點改。
- 訊息區回顯 NPC 台詞用另一個格式 L` "%S"`（0x0061c43c，呼叫點 0x005255d5、0x0052982f，皆 `call esi`＝swprintf IAT）。只處理 L"%S" 時，這裡把 UTF-8 bytes 一對一轉成寬字＝亂碼（MG 實機 run 174540，text log `"åä½ï¼…"`）。hook 對這兩個格式指標都解回，安裝時先比對兩處字串內容，不符就整組不裝。
- 另一處窄化 0x0052543f（`sprintf("%S")` 讀 `[0x0068c4ac]+6+4`）用途未查；不在 0x0055a140 的 NSF 讀取裡，沒改。
- 修法（`hook/ddraw/nsf.h`、`proxy.c` `install_nsf`）：0x0055a242 的 call 改成 call 到自己的函式：字全部 ≤0xFF 時照原版 sprintf；有 >0xFF 時寫 UTF-8，用 IAT realloc（0x005eb1f8）把緩衝放大。swprintf／vswprintf 的格式是 0x0061c4b4、而且字串含合法 UTF-8 序列時才解回，其餘呼叫原樣轉給 msvcrt。原版 NSF 文本只有 0x09／0x0A／0x20–0x7E／0x92（R7），不會出現 UTF-8 序列 → 未翻譯的檔案行為不變。
- 上限：原版其他窄字串若有 0xC4–0xEF 後接 0x80–0xBF，會被誤解成一個字（`nsf.h` ponytail 註記）。

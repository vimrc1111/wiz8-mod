# O 實作＋RE：日誌（FACT.DBS 事件文字）譯文寬度檢查

驗收: bash scripts/journal-fit.test.sh && bash scripts/check-fit.test.sh && bash scripts/check-fit.sh && go test -count=1 ./tools/fitcheck
規模: 中（codex gpt-6-astra medium；靜態 RE 找日誌字型與欄寬＋在 fitcheck 加一種檢查；Go 標準庫）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改 scripts/journal-fit.test.sh、scripts/check-fit.test.sh、既有 *_test.go 的既有案例、translations/ 下 journal-overflow.tsv 以外的檔與白名單外任何檔；不加第三方依賴；不寫入遊戲目錄；不執行遊戲；不放寬或跳過測試；不貼遊戲原文或反編譯碼（位址、bytes、自己的話描述可寫）
白名單: tools/fitcheck/fitcheck.go、tools/fitcheck/fitcheck_test.go（只加新案例）、scripts/check-fit.sh、translations/journal-overflow.tsv、notes/re/journal.md
預期紅: journal-fit
執行器: codex

## 背景
MG 實機截圖：日誌（Journal）事件欄的中文譯文有些在右緣被切掉（不換行，直接截）。MG 2026-10-08 裁定走「量寬＋縮短譯文」，不做換行 hook。
現有 `scripts/check-fit.sh`／`tools/fitcheck` 只量 STRINGDATA，用 Wiz_Text_Font、CJK 每字 max(字型高,12)。拿這個模型量日誌，切掉的句子（如 1289，245px）反而比完整顯示的 1232（293px）窄 → 日誌用的字型、CJK 字寬或欄寬跟現有模型不同。

## 要做
1. **RE（靜態）**：找日誌事件文字的繪製路徑，確定：用哪個字型槽／STI、裁切右緣（或可用寬度）多少 px、起始 x。
   - 工具：`build/wiz8.asm`（objdump）、ghidra 腳本 `re/run.sh`（反編譯輸出在 `~/.local/state/wiz8-mod-ghidra/decomp/`，不入版）。已知繪字入口、字型槽、0x00407220 設文字區（x1,y1,x2,y2）見 `notes/re/draw.md`。
   - 中文字是 hook 畫的：CJK 字寬怎麼算看 `hook/ddraw/proxy.c`（`wrap_char_width`、字形 blit）與 `hook/ddraw/wrap.h`；檢查要用實際繪製時的寬度。
   - 結論寫 `notes/re/journal.md`：位址＋前 16 bytes＋行為（自己的話），每條標 [已驗證]（讀過 asm／decomp）或 [推測]。
2. **檢查**：`check-fit.sh` 另外對 `$T/fact.json`（經 `trcheck expand` 展開 `{{代碼}}`）量寬，超過日誌可用寬度 → 印 `FAIL journal id=<id> zh=<px> max=<px>`，整體非零。原文 FACT.DBS 從玩家自有遊戲檔現取（`tools/fact decode`，參考 `scripts/check-lib.sh` 的用法）。
   - 登記簿 `translations/journal-overflow.tsv`（格式同 `overflow.tsv`：`id<TAB>理由`，`#` 註解；有理由才算登記；登記了但已不超寬 → `FAIL journal-stale id=<id>`）。
   - 把目前真譯文超寬的條目全列進登記簿，理由寫「待縮短（票 O）」，讓 `check-fit.sh` 對真譯文綠。cc 之後縮短譯文、刪登記。
3. `fitcheck_test.go` 加新案例：超寬紅、剛好等於上限綠、登記過期紅。

## 校準（驗收的依據）
`scripts/journal-fit.test.sh`（cc 寫，不准改）：截圖裡被切的 9 條（734 1224 1226 1230 1274 1285 1289 1382 1544）必須紅，完整的 1232 必須不紅，1289 縮短後不紅。
1382、1544 只切掉句尾 1–2 個字，是最緊的邊界。若找到的字型＋欄寬無法同時滿足這 11 項，不要硬湊常數：在回報寫出各條算出的寬度與上限，寫進 `## 規格疑慮`。

# Y4 case 依 index 尋址：拿掉 `also` 的固定記錄複製（grok 稽核 zoWN 發現 1–3）

驗收: bash scripts/case-index.test.sh
規模: 中（codex gpt-6-astra medium）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔；不改 scripts/case-index.test.sh、scripts/node-accept.sh、scripts/emu.test.sh；不刪 boundary 既有列（含稽核加的兩行）；不縮 random 範圍閃差異；不寫入遊戲目錄；不執行遊戲；不下載任何東西；筆記不貼組語或反編譯碼；不把原版 bytes、從 exe 讀出的表寫進 repo
白名單: re/emu/emu.py, notes/re/emu.md, re/emu/cases/0051cce0.json, re/emu/cases/0051e920.json, re/emu/cases/0055cc00.json, re/emu/cases/004ebc80.json, re/emu/cases/0052b4f0.json, internal/calc/n_51cce0.go, internal/calc/n_51e920.go, internal/calc/n_55cc00.go, internal/calc/n_4ebc80.go, internal/calc/n_52b4f0.go, notes/re/iface/51cce0.tsv, notes/re/iface/51e920.tsv, notes/re/iface/55cc00.tsv, notes/re/iface/4ebc80.tsv, notes/re/iface/52b4f0.tsv
預期紅: case-index
執行器: codex

## 背景
稽核全文：`~/Projects/games/wiz8-mod-audit/cx-20261009-051303-zoWN/audit/AUDIT.md`（點名 1、3 與「發現清單」1–3）。
case 的 `also` 把同一輸入值寫進固定幾筆記錄（物品 0..3、八名角色），原版依 index 讀「那一筆」；index 超出被寫的範圍（例：物品 4）時原版讀到別的值，Go 卻用輸入值 → diff。稽核的兩行（51cce0 `[4,11]`、51e920 `[4,10,0,5,0,0]`）已加進 boundary，目前 TestEmu 紅在這兩行。

## 要做的
1. `re/emu/emu.py`：`at` 支援 `{"object":"<名>","offset":<n>,"index":"<輸入名>","stride":<n>}`，位址＝物件基址＋offset＋(該輸入本列值)×stride；index 值 <0 時這個欄位不寫（讀則報錯）。index 輸入要先寫（或在寫入階段先解析所有輸入值）。物件要配足夠空間給 index 範圍，與其他物件不重疊（越界報錯，不靜默）。
2. `--selftest-stubs` 多印一行 `index ok`：合成 case 以 index 寫兩筆不同記錄、讀回各自的值才印。
3. 五個節點的 case 改用 index 尋址，拿掉全部 `also`：
   - 51cce0、51e920：物品／模板依 itemIndex 寫進那一筆；itemIndex 的 random 上限 ≥100（物品表在遊戲中遠大於 3）。稽核兩行要變綠：輸入值寫進 index 4 那一筆後，Go 用輸入值與原版一致。若 Go 節點確實要改（例如 index<0 的語意），照原版改。
   - 55cc00、4ebc80、52b4f0：「所選角色」的欄位依角色 index 寫；其他角色的對應欄位要與所選角色不同（另給輸入或固定不同值），讓「讀錯人」會產生 diff。52b4f0 輸出八人 SP 時，八人的初值要彼此不同。
4. iface 依新輸入更新。五節點 `bash scripts/node-accept.sh tickets/nodes/<addr>.list` 全綠。
5. `notes/re/emu.md` 寫 index 尋址的格式與限制。

## 規格疑慮
寫在 `notes/re/emu.md` 末段 `## 規格疑慮`，並在最終回覆列出。

# Y4a 51cce0／51e920 case 依 itemIndex 尋址（取代 Y4 第 3 項前半；grok zoWN 發現 1–2）

驗收: bash scripts/case-index.test.sh 51cce0 51e920
規模: 中（codex gpt-6-astra medium）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔（含 re/emu/emu.py、scripts/）；不刪 boundary 既有列（含稽核加的 `[4,11]`、`[4,10,0,5,0,0]`）；不縮 random 範圍閃差異；不寫入遊戲目錄；不執行遊戲；不下載任何東西；筆記不貼組語或反編譯碼；不把原版 bytes、從 exe 讀出的表寫進 repo
白名單: re/emu/cases/0051cce0.json, re/emu/cases/0051e920.json, internal/calc/n_51cce0.go, internal/calc/n_51e920.go, notes/re/iface/51cce0.tsv, notes/re/iface/51e920.tsv
預期紅: case-index
執行器: codex

## 背景
規格本體：`tickets/Y4-case-index.md`（第 1、2 項 emu 已由 cc 做完，3e9058a；本票只做第 3 項的 51cce0、51e920 與第 4 項）。
index 尋址格式：`notes/re/emu.md`「case 格式」的 `index` 條（`{"object":…,"offset":…,"index":"<輸入名>","stride":…,"count":…}`，index<0 不寫）。
稽核全文：`~/Projects/games/wiz8-mod-audit/cx-20261009-051303-zoWN/audit/AUDIT.md` 點名 1、發現 1–2。
現況：兩個 case 用 `also` 把同值寫進記錄 0..3；稽核兩行 itemIndex=4 讀到未寫的記錄 → TestEmu diff=1（main 上現在就紅在這兩個）。

## 要做的
1. 兩個 case 的物品／模板欄位改用 index 尋址（依 itemIndex），拿掉 `also`；物件留足 count×stride 空間、不與其他物件重疊。
2. itemIndex random 上限 ≥100；稽核兩行變綠。若 Go 節點語意（例 index<0）與原版不符，照原版改。
3. iface 依新輸入更新。`go test -count=1 -run 'TestEmu$' ./internal/calc` 全綠（其他節點不得變紅）。

## 規格疑慮
寫在最終回覆。

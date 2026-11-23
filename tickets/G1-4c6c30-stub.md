# G1 550c10／54fa30／545090：4c6c30 改位址樁，放開 +0x1dc 位元 0x400（goals/stage14.md，後續票 3）

驗收: bash scripts/nodes/g1-check.sh && bash scripts/node-accept.sh tickets/nodes/g1.list
規模: 中
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔（含 re/emu/emu.py、scripts/node-accept.sh、scripts/nodes/ 既有檔，除非白名單列出）；不刪 boundary 既有列；不縮 random 範圍或輸入域閃差異；不放寬任何驗證；不寫入遊戲目錄；不執行遊戲或 exe；筆記與程式註解不貼組語或反編譯碼；不把原版 bytes、從 exe／DBS 讀出的表寫進 repo
白名單: re/emu/cases/00550c10.json, re/emu/cases/0054fa30.json, re/emu/cases/00545090.json, internal/calc/n_550c10.go, internal/calc/n_54fa30.go, internal/calc/n_545090.go, notes/re/iface/550c10.tsv, notes/re/iface/54fa30.tsv, notes/re/iface/545090.tsv, notes/re/cov-exempt.d/550c10.tsv, notes/re/cov-exempt.d/54fa30.tsv, notes/re/cov-exempt.d/545090.tsv
預期紅: g1-check
執行器: codex

## 背景
- 0x4c6c30（零傷害表現）已登記「排除」（c359052），排除類准位址樁（`notes/re/emu.md` 位址樁段）。
- 三個 case 現在用 setup 把 actor +0x1dc（offset 476）固定成含位元 0x400，讓 4c6c30 原生跑時不動作；calc 端 `n_550c10.go:437`–`:443`、`n_545090.go:322`–`:328` 有對應判斷。這把 +0x1dc 其他位元的真實取值也一起綁死了。
- 前例：54e950 case 的 `"0x4c6c30"` 樁（`re/emu/cases/0054e950.json:9448`）。54fa30 與 550c10 共用 srdStubs；545090 經 52bb60 原生子樹呼叫 4c6c30。

## 要做的
1. 三個 case 的 stubs 加 `"0x4c6c30"`（參數／次數照 54e950 前例記進輸出），拿掉固定 0x400 的 setup。
2. +0x1dc 改成輸入（或保留原本其他位元的固定值但不含 0x400）；若 +0x1dc 其他位元會改變三節點行為，boundary 加覆蓋列。
3. calc（n_*.go）拿掉「0x400 才不動作」的假設，改照樁記錄輸出 4c6c30 的參數與次數。
4. 三節點 node-accept 綠。四段注入：把一個 case 的樁拿掉 → g1-check FAIL → 副本還原 → PASS。

## 規格疑慮
寫在最終回覆；沒有就寫「無」。

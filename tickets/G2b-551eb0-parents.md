# G2b 551eb0 父節點 arity 回歸（G2 314bbd8 的後續；goals/stage14.md）

驗收: { bash scripts/node-accept.sh tickets/nodes/g2b.list >/dev/null 2>&1 || { echo 'FAIL g2b-parents'; false; }; } && { bash scripts/calc-qa.test.sh >/dev/null 2>&1 || { echo 'FAIL g2b-calc-qa'; false; }; }
規模: 中
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔（含 re/emu/emu.py、scripts/、tests 既有檔）；不改任何 re/emu/cases/*.json；不刪 boundary 既有列；不縮 random 範圍或輸入域閃差異；不放寬任何驗證；不寫入遊戲目錄；不執行遊戲或 exe；筆記與程式註解不貼組語或反編譯碼；不把原版 bytes、從 exe／DBS 讀出的表寫進 repo
白名單: internal/calc/n_551eb0.go, internal/calc/n_543270.go, internal/calc/n_54dbe0.go, internal/calc/n_551ba0.go, internal/calc/n_5560a0.go, notes/re/iface/551eb0.tsv
預期紅: g2b-parents
執行器: codex

## 背景
- G2（314bbd8）在 551eb0 的輸入加了三個人工夾具欄 `noticeFmt0`、`noticeFmt1`、`suffix0`（`n_551eb0.go` sdrSpec／sdrIn，sdrFx 177→180），用來讓 551eb0 自己的 case 走到「格式文字只有單一 `'`」。
- 父節點 543270、54dbe0、551ba0、5560a0 的 F 世界按名稱整串帶上 551eb0 的輸入（例 `n_551ba0.go:201` 迭代 `sdrIn[sirSdrSkip:]`、`f.sdr.*`），於是父節點 Go 端 arity 多 3 欄、父 case 沒變 → `TestEmu` 這四個節點全列 `input/output arity`，calc-qa Q4（`calc f 00543270 …`）非零退出。G2 的白名單漏列父節點（cc 規格缺陷）。

## 要做的
1. 父節點的輸入定義與 case 都**不變**（arity 回到 G2 前）：三個欄只屬於 551eb0 自己的 case。父節點呼叫 551eb0 時，這三欄填「與 G2 前行為相同」的值（原版格式與後綴；值的出處寫在 iface，從 G2 前的 `n_551eb0.go` 行為推導，不從 exe 讀表入 repo）。
   做法自選（例：551eb0 匯出一個不含這三欄的父節點用名稱清單，或父節點迭代時略過），一處改好讓所有父節點走同一條路，不各自複製。
2. 551eb0 自己的 case 與 g2-probe 照綠（`bash scripts/nodes/g2-probe.sh`）。
3. `bash scripts/check-formulas.sh` 不新增不符（worktree 內 build/emu 是主 repo 完整副本，TestEmu 會跑全部節點）。
4. 四段注入：在父節點用的預設值注入錯值 → 驗收 FAIL → 副本還原 → PASS；貼進回覆。

## 規格疑慮
寫在最終回覆；沒有就寫「無」。

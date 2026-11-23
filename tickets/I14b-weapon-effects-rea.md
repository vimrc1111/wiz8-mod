# I14b 研究：武器特效＋高難度怪抗性（rea）

MG 2026-10-31：「武器特效可以開 REA 研究單嗎？需要計算高難度敵人的抗性」。取代 I14（agy 撞 429 擱置）的執行器；I14 的待答 1–6 照收，另加怪物抗性。
驗收: bash scripts/check-research.sh notes/re/weapon-effects.md
執行器: rea 子 agent（一次批完，追問用 SendMessage）
白名單: notes/re/weapon-effects.md
預期紅: research:weapon-effects
禁止項: 不 commit；不寫入遊戲目錄；不執行遊戲或 exe；不下載；報告不放反組譯碼、不轉錄 DBS 表（id 與欄位偏移可寫）。
預估讀入: build/Wiz8.exe（REA 載入）、build/wiz8.asm 指定區段、build/data-x/DATABASES/{ITEMS,MONSTERS}.DBS、下列筆記；約 1 次 Ghidra 分析。
停止條件: 待答 1、3、7 有 [已驗證] 或明列 [未取得]＋下一步就交；不做實機。

## 已知（引用，不重查）
- I14 已知與待答 1–6：`tickets/I14-weapon-effects-inventory.md`（槽號 1 麻痺 2 中毒 3 Hex 5 即死 6 KO；槽分派跳表 Wiz8.exe:0x543d74；各槽件數）。
- KO／麻痺／即死抵抗 `0x5520d0`：a=clamp(Difficulty(3(Lm−S)+Rj+bonus[k]),5,95)；持續、醒來、加倍函式 `0x5420b0`：`notes/re/ko-paralyze.md`。
- 難度：Expert（難度 2）怪物端 ×1.4、玩家命中 ×0.6：`notes/research/ranking-history.md:134`。
- 模型讀的怪物欄位：`internal/calc/load.go:71`–`73`（Resist＝r[0x176:0x17c] 六格、AC、Level、Absorb）。

## 待答（I14 的 1–6 之外）
7. **抗性**：怪物 6 格抗性各對應哪個領域（火水氣土心神）；每個特效槽查哪一格抗性（I13 已知 KO 3、麻痺 1、即死 5，補其餘槽）；抗性對元素傷害的作用式。另確認 MONSTERS.DBS 有無狀態免疫欄（I13 類別豁免表 0x6171a8 的「類別」是怪物哪個欄位、類別 17／20／22 是哪些怪）。
8. **目標怪表**：I9 的 8 隻（564、114、184、171、129、158、120、192），Expert、隊伍 L28（S＝43），每隻列：等級、6 抗性、類別豁免、對槽 1／2／3／5／6 及其他已識別槽的抵抗率 a（照 `0x5520d0` 式與難度倍率算出數字）。
9. **KO 加倍連結與每回合遞減量**：I13 兩個 [未取得]（`ko-paralyze.md` 第 11、19 條附近），能補就補。

## 證據規格
同 I14：`## 發現` 每條一行、來源 `build/wiz8.asm:<行>`／`Wiz8.exe:0x<VA>`／`internal/calc/<檔>:<行>`、標 [已驗證]／[推測]／[未取得]；另附 `## 總表`（I14 欄位）與 `## 目標怪抗性表`（待答 8）。

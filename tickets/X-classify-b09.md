# X-b09 RE：公式候選分類（stage10 B0，第 b09 批）

驗收: ONLY=build/re/batch/b09 bash scripts/check-formulas.sh --progress
規模: 中（60 個函式，靜態閱讀；不寫碼）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔；不改 scripts/check-formulas.sh；不寫入遊戲目錄；不執行遊戲；不貼反編譯碼或組語（只寫位址、常數、自己的話）；不下載任何東西；不憑社群資料下結論
白名單: notes/re/formula-graph.d/b09.tsv
預期紅: unclassified
執行器: codex

## 背景
Wizardry 8（Wiz8.exe v1.2.4）。目標是把全遊戲規則公式解析到可計算（goals/stage10.md B 段）。第一步是把機械掃出的 719 個候選函式逐一歸類。
候選＝呼叫亂數 0x40efa0（或其包裝）或引用記錄基址（角色 0x685174、角色副表 0x685178、物品 0x68516c、隊伍 0x6836a8）的函式。
材料（都在 worktree 的 build/）：
- `build/re/batch/b09`：本批 60 個入口位址；
- `build/re/funcs.tsv`：入口、大小、前 16 bytes、子呼叫、記錄基址引用、資料引用、基本塊；
- `build/decomp/cand-b09.c`：本批的 Ghidra 反編譯（只讀）；
- `build/wiz8.asm`（objdump）；
- 既有 RE 筆記 `notes/re/*.md`。

## 要做
在 `notes/re/formula-graph.d/b09.tsv` 為本批**每個**位址寫一列（可加表頭 `addr	class	depth	node	calc	evidence`），以 TAB 分隔：
- addr：小寫 hex，無 0x，與 batch 檔相同。
- class 三選一：
  - `公式`：函式的輸出會進入遊戲規則數值或機率。例如命中、傷害、擊殺、抗性、狀態、屬性／技能成長、經驗、HP／SP、法術效果、價格、掉落、遭遇、陷阱與開鎖、AI 的機率選擇、回復、負重。單純把這類數值搬來搬去（抄欄位、存讀檔）不算。
  - `非公式`：UI、繪圖、文字、音效、存讀檔 I/O、資源載入、腳本流程、純資料拷貝。也包含亂數只用在外觀的情形，例如音效變化、動畫變體、粒子。
  - `排除`：遠程彈道的碰撞、射線、遮擋判定。
- depth：`公式` 寫 `D0`；其他寫 `-`。
- node：`公式` 寫短的 snake_case 名稱，說它算什麼（例 `melee_to_hit`）。既有 notes/re 筆記已命名或描述過的，沿用它的說法。其他類可寫簡短名或留空。
- calc：留空。
- evidence：一句自己的話，說判定理由，必附 `build/wiz8.asm:<行號>`，指到決定性的指令（例如呼叫繪字、寫入某規則欄位、亂數結果用在哪裡）。
  - `公式` 加寫它讀哪些輸入、寫哪些輸出（欄位偏移即可）。
  - 若某位址在既有筆記出現過，附筆記位置（例 `notes/re/formulas.md:17`）。
- 拿不準時，判 `公式` 並在 evidence 寫「不確定：<原因>」。誤判成公式之後會被 D1 審查修正；誤判成非公式則會漏掉公式。

## 交付回覆
三類各幾個；「不確定」的位址清單；規格疑慮，沒有就寫「無」。

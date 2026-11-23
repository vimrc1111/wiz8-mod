# P RE：從 Wiz8.exe 找出近戰戰鬥公式（給之後的計算腳本用）

驗收: bash scripts/check-research.sh notes/re/formulas.md
規模: 中（codex gpt-6-astra medium；靜態 RE，一份筆記；不寫碼）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔；不寫入遊戲目錄；不執行遊戲；不貼反編譯碼或組語區塊（位址、前 16 bytes、常數、自己的話與數學式可寫）；不下載任何東西
白名單: notes/re/formulas.md
預期紅: research:formulas
執行器: codex

## 背景
MG 想不靠修改器就能比較角色配置（例：28 級各職業滿技能拿最強武器，誰的近戰輸出最高；武士 Lightning Strikes 的發動機率跟什麼有關）。
社群公式整理在 `notes/research/2026-10-08-R11-formulas.md`（主要來源 peddroelm 指南，反組譯＋實測，未附位址；Lightning Strikes 機率社群沒有公開）。
本票從 exe 直接找，產出可據以寫計算腳本的公式；下一票（cc 開）依本筆記寫計算工具與黃金測試。

## 工具（唯讀）
- 組語：`build/wiz8.asm`（worktree 內，gitignored；objdump，Wiz8.exe v1.2.4，sha256 開頭 18a74ff6）。
- 既有 Ghidra 反編譯輸出：`build/decomp/`（worktree 內，只讀）。沙箱內沒有 Ghidra 專案也沒有 exe，不跑 `re/run.sh`；以 asm 為準，需要的函式反編譯寫進 `## 規格疑慮` 請 cc 補跑。
- 已知位址與寫法範例：`notes/re/draw.md`、`notes/re/misc.md:47`。
- 入口線索：STRINGDATA 的戰鬥訊息（如 Lightning Strikes、Berserk、Kill 等字串）被誰引用；字串位址可從 `.rdata`／STRINGDATA id 反查。

## 要找的公式（依優先序；找不到的標 [未取得] 並寫查到哪裡）
1. 武士 Lightning Strikes：發動判定（機率式、用到哪些屬性／技能／等級／裝備／負重）、額外攻擊次數、是否只限主手。
2. 近戰每回合攻擊次數：攻擊速度式、揮擊數門檻、雙持與副手。
3. 命中：命中值（BAR）與對方 AC 的判定、擲骰範圍。
4. 傷害：武器骰、力量加成、狂暴、背刺、致命一擊（Kill／Critical）機率、對方吸收。
5. 負重對速度／命中的影響。
6. 職業特殊：武僧多段攻擊、忍者／武士的其他被動（有找到就記）。

## 報告格式
`notes/re/formulas.md`：
- `## 發現`：每條 `- ` 一個公式或判定，寫函式位址＋前 16 bytes、輸入（角色資料結構的欄位位移與意義）、式子（數學式或自己的話；擲骰寫明分布）、與 R11 對應條目比較（相同／不同／R11 沒有）。來源寫 `build/wiz8.asm:<行號>`，每條附 `[已驗證]`（讀過 asm／decomp）或 `[推測]`／`[未取得]`。
- `## 角色結構`：用到的欄位位移表（位移、型別、意義、怎麼確認的）。
- `## 規格疑慮`：沒有就寫「無」。

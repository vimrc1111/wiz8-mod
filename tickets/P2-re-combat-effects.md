# P2 RE：從 Wiz8.exe 找法術、抗性、遠程、特效（即死、打暈等）公式

驗收: bash scripts/check-research.sh notes/re/formulas-2.md
規模: 中（codex gpt-6-astra medium；靜態 RE，一份筆記；不寫碼）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔；不寫入遊戲目錄；不執行遊戲；不貼反編譯碼或組語區塊（位址、前 16 bytes、常數、自己的話與數學式可寫）；不下載任何東西
白名單: notes/re/formulas-2.md
預期紅: research:formulas-2
執行器: codex

## 背景
同票 P（`tickets/P-re-combat-formulas.md`，近戰：攻擊次數、命中、傷害、Lightning Strikes、負重；另一個 worktree 平行做）。本票做其餘戰鬥公式，之後一起寫成計算工具。
社群公式：`notes/research/2026-10-08-R11-formulas.md`。材料與寫法同票 P：`build/wiz8.asm`、`build/decomp/`（worktree 內，gitignored，只讀；沙箱沒有 exe 與 Ghidra 專案，不跑 `re/run.sh`，需要補反編譯寫進 `## 規格疑慮`）、範例 `notes/re/draw.md`。
跟票 P 重疊的函式（例如近戰與遠程共用的命中或傷害函式）只記位址並註明「同票 P」，不必重寫。

## 要找的公式（依優先序；找不到的標 [未取得] 並寫查到哪裡）
1. 攻擊特效發動：即死（Kill／Death）、打暈（武僧 Stun／Knock Out）、麻痺、中毒、睡眠等。發動機率式、用到哪些技能／屬性／等級，以及目標如何抵抗（等級差、抗性）。
2. 法術：施法成功／失敗、法術強度（Power Level、施法者等級、技能、Power Cast）、傷害或效果量、範圍。
3. 抗性：角色與怪物的抗性合成、抗性對傷害或效果的作用（減半、全免、機率）。
4. 遠程：攻擊次數、命中、傷害、彈藥、投擲武器、多發。
5. 狀態效果的持續時間與每回合的判定。

## 報告格式
`notes/re/formulas-2.md`，格式同票 P：
- `## 發現`：每條 `- ` 一個公式或判定，寫函式位址＋前 16 bytes、輸入（資料結構欄位位移與意義）、式子（擲骰寫明分布）、與 R11 對應條目比較（相同／不同／R11 沒有）。來源寫 `build/wiz8.asm:<行號>`，每條附 `[已驗證]`／`[推測]`／`[未取得]`。
- `## 資料結構`：用到的欄位位移表（位移、型別、意義、怎麼確認的）；法術表、怪物表欄位若有用到也列。
- `## 規格疑慮`：沒有就寫「無」。

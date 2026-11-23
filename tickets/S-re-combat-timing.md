# S RE：Wiz8.exe 戰鬥的等待從哪來（加速 hook 的候選點）

驗收: bash scripts/check-research.sh notes/re/combat-timing.md
規模: 中（codex gpt-6-astra medium；靜態 RE，一份筆記；不寫碼）
預算: 重做 2 次 / 高模插手 0 次 / 人類 無
禁止項: 不 commit；不改白名單外任何檔；不寫入遊戲目錄；不執行遊戲；不貼反編譯碼或組語區塊（位址、前 16 bytes、常數、自己的話與數學式可寫）；不下載任何東西
白名單: notes/re/combat-timing.md
預期紅: research:combat-timing
執行器: codex

## 背景
要在我們的 `ddraw.dll` 代理（`hook/ddraw/`，簽章比對後 detour，見 `notes/re/draw.md`、`hook/ddraw/build.sh` 的簽章列）加「戰鬥加速」：倍率可調、設 1＝原版行為；只加速戰鬥，不動探索與遊戲時間流。
已知：一般時間更新讀 `GetTickCount`（IAT 0x5eb02c）乘 f32 12 推進；戰鬥結算每回合固定送 120000 內部單位（`notes/re/combat-2.md:83`）。所以全域改 `GetTickCount` 會連探索一起加速，不是我們要的。
材料：worktree 內 `build/wiz8.asm`（objdump intel）、`build/decomp/`（Ghidra 反編譯）、`build/Wiz8.exe`（玩家自有副本，前 16 bytes 用 `xxd` 從這裡讀；檔案偏移＝VA−0x400000），全部只讀、gitignored。沙箱沒有 Ghidra 專案，不跑 `re/run.sh`；需要補反編譯就寫進 `## 規格疑慮`。範例寫法 `notes/re/draw.md`、`notes/re/combat-2.md`。

## 要找的（依優先序；找不到標 [未取得] 並寫查到哪裡）
1. 計時 API 全清單：exe 匯入的 GetTickCount、timeGetTime、QueryPerformanceCounter、Sleep、WaitForSingleObject 等，各自的呼叫點；哪些呼叫點在戰鬥路徑上（怎麼判斷的）。
2. 戰鬥訊息停頓：戰鬥 log 每條訊息顯示後等多久、由哪個常數或設定決定（遊戲選項若有訊息速度，找讀它的地方）。
3. 戰鬥動畫：攻擊／施法／怪物動作的動畫播放怎麼計時（以影格、以毫秒、等待動畫結束才進下一個行動？）。
4. 回合流程：戰鬥中「下一個行動」何時開始，是等計時器、等動畫結束旗標、還是等訊息隊列清空。
5. 「現在在戰鬥中」的旗標：全域變數位址與怎麼確認的（讓 hook 只在戰鬥中生效）。
6. 結論：建議的 hook 點 1–3 個（位址＋前 16 bytes 供簽章、hook 後改什麼值、預期副作用），按「改動最小、最不會碰到探索」排序。

## 報告格式
`notes/re/combat-timing.md`：
- `## 發現`：每條 `- ` 一個結論，寫函式位址＋前 16 bytes、做了什麼、來源 `build/wiz8.asm:<行號>`，附 `[已驗證]`／`[推測]`／`[未取得]`。
- `## 規格疑慮`：沒有就寫「無」。

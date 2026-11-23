# wiz8-mod

Wizardry 8 中文化＋現代機器相容。原版資料與美術一個 byte 不動，改動以 patch／hook 疊上；不重製引擎、不還原原始碼。
設計原則仿 ScummVM：必須搭配使用者自有的原版遊戲；repo 不含任何原版檔案（MG 2026-10-07）。反組譯與 hook 自第二階段解禁（`decisions/2026-10-07-re-unlock-and-scummvm-principle.md`）。

## 派工（MG 2026-10-07，後者取代前者）
1. 「待會如果開始正式做事，寫代碼的時候由你（Opus 5.5）來切單，可以用 agy（參數用的就是 Opus 5.5），codex（GPT-6.1 Sol），接下來是 Muse。清楚邊界的事情可以用 worktree 平行做」
2. 「4 很重要，這個是核心，做反而是其次。所以我建議直接派 agy 去（你要跟它定義清楚，其實它就是你），用完了查資料全部都叫 muse 去，grok 留給稽核，代碼就由 codex 跟 cc 寫，如何？」（「4」＝研究派工）
3. 「由於還有額度，如果比較難的票，你可以派 codex astra medium 去做，這個大概是 sol 6.1 的 5-7 倍消耗，但還行，不會比你差」（MG 2026-10-07；codex 10/10 重置，當時餘 87%）

4. 「agy 本週額度沒了而且它的背後選的模型就是 opus 5.5 就是你。codex 或許還夠而且模型也有力，但畢竟買的額度沒有 cc 那麼多，用完了要等三天後，cc 目前額度最多，所以 cc 翻最安全」「有鑑於 codex 剩下三天額度就重置了，非翻譯（這個給 cc 自己做）的事情，儘量找它做，硬一點的票可以開 6 astra 不然就預設 6.1 sol」（MG 2026-10-07）

5. 「如果 codex 用完了，小票可以派 Muse 它現在可閒了 muse 週額度 已用 11% 剩 89%」（MG 2026-10-07）


6. 「可以豪華一點，codex 一個小時前重置了」（MG 2026-10-07 晚，codex 額度重置）

7. 「之後大單拆小一點派給 gpt-6.1 sol，astra 等顯式要求再派」（MG 2026-10-12，取代第 3、4 條的 astra 部分）

現行分工：
- 切單、裁決：cc（Opus 5.5）。
- 研究（查資料）：agy（`claude-opus-5-5-*`）；agy 額度用完 → muse。研究產出每條附來源（URL 或 `file:line@commit`），cc 抽查。
- 代碼：codex（`gpt-6.1-sol`）與 cc。agy 不寫碼（取代第 1 條的實作優先序）。
  - 大單拆小派 sol；astra 只在 MG 明說時派（第 7 條，取代第 3 條）。
- 翻譯：cc 自己翻（第 4 條）。非翻譯的代碼票儘量給 codex：一律 6.1 sol（第 7 條）。
- codex 額度用完 → 小票派 muse（第 5 條），大票 cc 接手。2026-10-07 晚 codex 已重置，可放手派（第 6 條）。
- agy 本週額度用完（2026-10-07）→ 研究改派 muse。
- 稽核：grok（`audit-run.sh`），只稽核。
- 邊界清楚的票：各開 worktree 平行。
- 與地板 §10.8 的差異（專案規格優先）：cc 也寫碼；agy 從選填實作改為研究。

## 常設核准（MG 在本專案直接給，地板 §5.2）
- push：「給你常設 push 的授權」（MG 2026-10-08）。2026-10-19～11-23 曾併入 `rpg-mod`（子目錄 `wiz8-mod/`）；2026-11-23 拆回獨立 repo（單一 first commit，舊歷史留在 rpg-mod）。origin＝`https://github.com/vimrc1111/wiz8-mod.git`（MG 2026-11-23：「這是獨立 repo」）。照舊要 tip 有 CI exit 0 證據（pre-push 閘）。不含 force push、改寫已推歷史。
- 數值規劃範圍：「只要碰到要實際跑遊戲路線或投射障礙物的都不做。你可以理解爲我們現在單純跑數值規劃而不是實體世界」（MG 2026-10-11）。地圖幾何、路徑、移動、射線／視線、投射物、碰撞、放置一律排除，結果當 case 輸入（`decisions/2026-10-11-numeric-only-scope.md`）。
- 排除登記：「未來這些排除跟登記你自己判斷就好」（MG 2026-10-10）。範圍：`notes/re/formula-graph.d` 的 `排除` 歸類由 cc 判斷，不逐列上報；一列一 commit 附證據照舊（`decisions/2026-10-10-exclusion-delegated.md`）。

## CI
本機 CI：`bash scripts/ci-run.sh wiz8-mod bash scripts/ci-check.sh`（`.ci-evidence` 開閘，推送要 tip 有 exit 0 證據）。
- MG 2026-11-01 結案後解除安裝 Steam 原版（「wiz8 是我解除安裝的」）：go／tr／lib／zhpatch 四 lane 讀原版，之後必紅。b7276cd5 在 CI exit 1 下推出（cc 指令未以 CI 結果擋 push，pre-push 閘亦放行）。重開本案先裝回原版補跑 CI。

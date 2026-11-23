# 階段 13：選單加「隊伍移動速度」

> MG 2026-10-12：「之後追加一個『隊伍移動速度增加』的需求」。前置：階段 11 定義 1–4（B1，256c935）。
> 範圍是 hook 改速度，不是 calc 數值規劃；`decisions/2026-10-11-numeric-only-scope.md` 排除移動只管 calc，不管這裡。

## 已知與未知
- 已知：怪物移動倍率 hook（`notes/re/speed.md`）只碰怪物分支；社群 Wiz8Fast／Kaucukovnik 都沒動隊伍探索移動（`notes/research/2026-10-09-R13-combat-speed-community.md:14`、`:16`）。
- 未知：隊伍探索移動（前進／後退／平移／轉向）的速度在 exe 哪裡算、是常數還是讀時間差、和遊戲時鐘（怪物巡邏、回復）是否同一時基。社群無位址（同檔 `:29`）。

## 步驟
| 步 | 內容 | 完成判準 |
|---|---|---|
| 13.1 研究 | 找隊伍移動速度的讀取點：位址、原 bytes、語意、和遊戲時鐘的關係；只加速隊伍位移，不加速遊戲時間。寫進 `notes/re/party-move.md`（有位址，不含反組譯碼）。 | 每個位址附 `build/wiz8.asm` 行號；check-sigs 能驗 |
| 13.2 hook | 照 speed 慣例：比對原 bytes，對不上不裝、log sig-mismatch。倍率純函數（夾限、整數）。 | 純函數會紅的測試、四段注入 |
| 13.3 選單 | `menu.h` 項目表加一列 `party_move_pct`（預設 100＝原版，上下限、步長同表），即時生效、寫回 ini。 | menu.test 綠；項目表加一列，沒有別的地方要改 |
| 13.4 自動 run | 新情境：進遊戲、開選單調 party_move、關閉、在原地按前進數次。 | log 有 hook 安裝列、無 sig-mismatch、活到時限、after_clean；原版三檔不變 |
| 13.5 發布 | 發布包 install／uninstall 照舊（uninstall 一併移除 ini）。 | cmd 實測 rc 0、原版三檔雜湊不變 |
| 待 MG | 實際走路手感、轉向是否也要快、會不會穿牆或卡住 | cc 不判手感 |

## 派工
- 13.1 研究：muse（agy 額度用完）或 cc；位址由 cc 在 `build/wiz8.asm` 抽查。
- 13.2–13.3 代碼：codex（gpt-6.1-sol），worktree，材料放 worktree 內實體檔（B1b 教訓：沙箱讀不到 worktree 外的 symlink）。
- 跑遊戲與 CI 不同時；自動 run 期間請 MG 不動遊戲視窗。

## 停損
- 13.1 找不到只動隊伍的讀取點（只能拉整個遊戲時鐘）→ 停，回報 MG 選：放棄，或接受連帶加速（怪物巡邏、回復、遊戲時間一起變快）。

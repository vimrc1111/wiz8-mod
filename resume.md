# resume（2026-10-09 更新；中文化已重新安裝，見「待 MG」第一條）

## 結案（2026-11-01）
- MG：「wiz8 我沒有遺憾了，標為結案」「即時地圖給你做出來，最佳隊伍也算完了」。即時地圖見 `GOAL-wiz8map.md`；配裝結論見 `notes/party.md`（NPC 第 7、8 人建議 Umpani 領主＋T'rang 忍者）。
- 未了項不再追，重開再議：「先手／防禦」指標未做（未裁決）；NPC 屬性／等級未建模（借種族算，只當排序參考）。

## 重建（2026-10-20，rpg-mod 管理 session；結案保存）
- MG 2026-10-20：「mm3 及 wiz8 都算結案了，所以只要確保我們的心血（修改歷程跟結果）未來可以從 git 重建即可」；Steam 原地安裝不搬。
- 歷程：原 repo 全部 refs 都是 origin/main 的祖先；唯一例外 B1 分支 10311d4，patch 與 main 上 5a46f6d 逐字相同（rebase 前的重複）。
- 結果：從 origin 新 clone 跑 `bash hook/ddraw/build.sh` rc 0，atlas.h／hooks.h／exports.h／ddraw.def 與部署那次逐 byte 相同。ddraw.dll 與 Steam 部署版（sha 1ea89d3e…，10-14 13:55，原始樹＝99bcd27＝HEAD）差在 ImageBase：ld 依輸出路徑自動選基址，位址正規化後反組譯 0 行差。
- repo 外的建置輸入：字型兩檔（來源 URL 與版本在 `assets/LICENSES`）、系統 ddraw.dll、使用者自有原版。`~/.local/state/wiz8-mod-*` 都是原版衍生或量測紀錄，不需要；`wiz8-mod-savebak` 是 MG 存檔備份，不入 git。發布包由 `scripts/release.sh` 產生。


## 狀態
- 階段 0–1：見 git log（b3e555e … 7909440）。B 的 grok 稽核已收件：抓對 2 誤報 0（案目錄 7909440，補測進 slf_test.go）。
- 階段 2（`goals/stage2.md`）：完成，只剩鬆散 `Data/` 一格未取得。
  - wiz8-run.sh＋離線測試＋四段注入：835bb0d。渲染路徑與雜訊底（0）：`notes/zh-patch-log.md:23`。
  - 覆蓋決策表：`notes/zh-patch-log.md:127`（b4f60c9）：PATCH 編號大者贏、不分大小寫、斜線不正規化；與靜態分析一致。
- 階段 3（`goals/stage3.md`）：完成。
  - 3.1 筆記 `notes/re/{draw,strings,patch}.md`、check-sigs 進 CI＋四段注入：343b9ba。
  - 3.2 `tools/strings`（codex）：收貨 83f0e2c、合併 2c3f98c。C 的 grok 稽核 8yPB：抓對 1 誤報 0 編譯紅 0，補測 e861984；收件 wiz8-mod-audit f2c1481。
  - 3.3 繪字 detour：32e28a9（text-match 命中 19 筆、負對照 FAIL）。
- 階段 4（`goals/stage4.md`）：完成。MG 目視（2026-10-07，原話）：「我看到中文字了，稍微比較破碎，但確實是中文」。
  - 4.1 `tools/atlas`（Noto Sans TC 2.004，OFL，`assets/LICENSES`）＋雜湊測試＋四段注入：1577798。
  - 4.2 路線＝code point 直通＋資料擴充（字型載入後擴字形記錄與字元表），理由與已知上限：`notes/zh-patch-log.md:23`「CJK 繪製路線」。b9ec001；筆數欄位位移修正 8c75c34。
  - 4.3 判讀規則先寫後跑：`notes/zh-patch-log.md:193`（770dbba）。雜訊底 N0＝0（run 035119 與 033930 整張逐像素相同）→ xor 上限 0。
  - 4.3 現行結果：`scripts/cjk-check.sh 20261007-040805-cjk-zh 20261007-033930-cjk-empty 20261007-034112-cjk-vanilla` → 5 條中文 xor=0、墨色差 0–2（T＝32／40）、負對照 151–666、三 run alive＋after_clean（結果在 `notes/runs/20261007-040805-cjk-zh.json` `.cjk_match`）。
  - 字色機檢抓到真缺陷：Wiz_Text_Font 的中文畫成黑字（陰影索引）。修法與依據：`notes/zh-patch-log.md:23`「字色」。
  - 建置：`bash hook/ddraw/build.sh`；`bash scripts/build-zh-patch.sh build/zh/PATCH.002`。

## 完成定義對照（2026-10-07，HEAD 331239a 起算；指令與輸出在本 session 跑過）
| 項 | 證據 |
|---|---|
| S2.1 wiz8-run 入版、離線測試進 CI、四段注入 | `git ls-files scripts/wiz8-run.sh`；ci-check.sh 跑 `wiz8-run.test.sh`；注入在 835bb0d 訊息 |
| S2.2 雜訊底、渲染路徑 | `notes/zh-patch-log.md:23`「實際渲染路徑」與雜訊底（3 次 run 逐像素差 0） |
| S2.3 覆蓋決策表 | `notes/zh-patch-log.md:127`：四格已測；鬆散 `Data/` 標未取得＋原因 |
| S2.4 每次實機 run 後回到 clean | notes/runs 下 after_clean≠true 的只有 035916（MG 截圖存了 Wiz800000.JPG，已搬走）；下一個 run 040151 的 before_diff＝0 non-noise。035616 中斷前沒寫 json，下一個 run 035916 的 before_diff＝0 non-noise |
| S3.1 三主題筆記、check-sigs 進 CI＋注入 | `notes/re/{draw,strings,patch}.md`；ci-check.sh 跑 check-sigs；注入在 343b9ba 訊息 |
| S3.2 tools/strings 合併、往返測試、稽核轉測試 | 2c3f98c 合併；`go test ./tools/strings` ok；grok 8yPB 補測 e861984 |
| S3.3 text log 驗收 | `notes/runs/20261007-030938-text-hook-keys.json` stringdata_match.count＝19；負對照 030316 為 0 筆 |
| S4.1 圖集、detour 入版；測試進 CI＋注入 | tools/atlas（注入 1577798、a35c947）；ddraw 代理建置＋check.sh 進 CI（注入 331239a）。detour 的執行期行為只有實機 cjk-check，CI 驗不到 |
| S4.2 4.3 機檢 | `notes/runs/20261007-040805-cjk-zh.json` .cjk_match 5 條 ok |
| S4.3 resume 各段 | 本檔：狀態、偏離、待 MG（含目視）、下一步（翻譯管線、換行、字表） |
| S2.5／S3.4／S4.4 工作區乾淨、CI exit 0 | `~/.local/state/wiz8-mod-ci/<HEAD>/exit-code.txt`＝0；`git status --short` 空 |

## 偏離與原因
- 2.2 換 X 一次：INTRO_BG → INTRO_BG_TEXT（INTRO_BG 不是主選單用的底圖，probe／which run 證明 patch 有被讀）。
- 2.2 鬆散 `Data/` 檔：未取得：未核可暫放鬆散檔（goal 只核可 ddraw.dll 與 Patches/PATCH.00N）。
- C 第一次派工 r0 拒派：派工單「預期紅:」行帶括號註記，被解析成第 5 個測試名；改單（00756a3）重派。
- 3.3 與 4.3 都用組隊畫面不用主選單：主選單只有 `v1.2.4` 走繪字（選項字是圖）；送鍵 ↓＋Enter（前景視窗屬 Wiz8 才送）。
- wiz8-run.ps1 的 Ev 把 `{ENTER}` 當格式字串而崩（run 030541 留著當紀錄），32e28a9 修。
- 835bb0d 當時 CI 紅：slf_test 讀到 batch 暫放的 PATCH.003（競態，非碼缺陷）；實機 run 與 CI 不同時跑。
- impl-run／audit-run 空格路徑 → scratchpad symlink `$S/w8`（換 session 要重建）。
- C 稽核補測由 cc 寫（三筆輸入形狀 grok 已給），未另開 codex 票。
- 4.1 字表 `assets/charset-common.txt` 為 cc 手列 571 字，不是頻率表；新依賴帶進間接 x/sys、x/text。
- 4.2 字型物件版面先照 JA2 假設（筆數 +0x1c）→ 實機 run 033418 全部 n=0；反編譯 0x00406180 改 +0xf6。
- 4.3 期望/實測遮罩規則看過資料後改兩次（§7.5 偏離）：濾波暈、低反差漏判；現行規則由 nGlide 雙線性幾何推得，細節與上限在 eb60b5c 訊息與 `tools/atlas/cmp.go`。
- 4.3 「負對照」以無 patch 幀對空字串幀在同位置的 xor 計；背景幀＝5 筆譯文改空字串的 PATCH.002。
- 4.3 判讀規則是在 stop hook 指出缺口之後才補寫的（eb60b5c 已先用 xor 上限 0 與遮罩規則）；先寫後跑只涵蓋 N0 與字色兩項。
- 4.3 字色：規則檔預期 Wiz_Text_Font 會 FAIL 的原因（兩色逐列交錯）猜錯，實際是 dll 選到陰影索引。我沒照規則檔記 FAIL，而是修了 dll；T 沒有放寬。
  - 嘗試 1（取最多）：033741 FAIL（按鈕黑字）。
  - 嘗試 2（取最亮）：040151 FAIL（標題 67>32）。
  - 嘗試 3：先量資料（040543 `font colors` log），再定規則（剔除陰影後取最多），040805 過。
- 4.3 作廢的 run（json 留作紀錄）：
  - 035616：收尾刪 ddraw.dll 時行程還握著，rm 失敗中斷，state 改名 `.aborted`；wiz8-run.sh cleanup 改成等行程消失並重試。
  - 035916：MG 切視窗，組隊截圖時遊戲最小化（rect=-32000）；遊戲存了 Wiz800000.JPG，已搬到該 run 的 state 目錄，遊戲目錄複核 0 non-noise。
  - 040406：我加的 log 溢出 log_line 緩衝，遊戲 AV（0xC0000005）。

- 自由試玩 run 041922 作廢：MG 另從 Steam 開了一個 Wiz8，我開的 pid 16304 在 30 秒時 exit 0（推測被 Steam 那一個關掉，MG 證實有從 Steam 開）；
  腳本以為遊戲已結束，收尾刪了 ddraw.dll，但 PATCH.002 被 Steam 那一個佔著刪不掉。
  Steam 那一個只讀到 PATCH.002、沒載我們的 dll，中文全部顯示成 A（MG：「start new game 裏頭的中文目前都是 AAAAA」）。
  事後已刪 PATCH.002、搬走 Wiz800000.JPG，遊戲目錄 0 non-noise。

- 自由試玩 run 042256：MG 在遊戲內按 PrintScreen，遊戲寫出 Wiz800000.JPG（已搬到 state）後崩潰（0xC000041D）。
  事件記錄 1000：崩在系統 ddraw.dll（10.0.26100.8115）位移 0x33159。04:21:15 那一次是沒載我們代理的 Steam 啟動，同模組、同位移 → 原版加現代 Windows 的相容問題，不是 mod 造成的（`runtime_reproduction`，2 次）。

## 階段 5（`goals/stage5.md`，2026-10-07，HEAD 221090d 起算）
**結案**（MG 2026-10-07「A」，`decisions/2026-10-07-stage5-close.md`）：5.5 延後，其餘完成；下一步階段 6（`goals/stage6.md`），開始前先跟 MG 確認。
| 項 | 狀態 | 證據 |
|---|---|---|
| 5.1 分類 | 完成 | `translations/stringdata-classes.tsv`（2412 條＋表頭） |
| 5.2 譯文＋機檢 | 完成 | `scripts/check-translations.test.sh` 進 CI，5 種注入皆紅 |
| 5.3 斷行 | hook 完成；真機長句未驗 | F 合併 2b709da；E 入口拿掉 6a2d1cf（run 070513 崩 0xC000041D → 071209 alive）；fitcheck 進 CI 221090d，溢出 110 條登記 `translations/overflow.tsv` |
| 5.4 字形 | 完成：Fusion Pixel 12px，中文字框不小於 12（MG 裁定，取代 E）；MG 實機：「平常的文字幾乎可以說沒有大的問題，這個解法是對的」 | 51e38e0；run 075346 截圖 26 張子 agent 判讀（不入版控）。殘留：戰鬥／訊息 log 行距與 12px 字框上下相疊、底部兩行狀態列擠、日記羊皮紙上的標題（淡色無陰影）點狀破碎 |
| 5.5 關鍵字點擊 | 未取得 | 要走到 NPC 對話；盲送鍵走不到。MG 的 Saves 有存檔，讀檔會寫 CurrentGame.SAV，未經 MG 同意不碰 |
| 5.6 MG 截圖 | MG 已拍（run 081820、085203、090440）並回報；處置見下表，全部已修或列下階段。溢出表「待 5.6」65 列逐列處置（12 列實機有畫、MG 未提溢框→接受；53 列未上畫面→暫接受） | run 081820 截圖子 agent 判讀（不入版控） |

5.6 MG 回報處置（2026-10-07）：
| 回報 | 處置 |
|---|---|
| 魔法有說明畫面 | 名稱留英文定案（b7bdd59）；說明下階段 |
| 日記內容未翻 | 日記字型畫的 59 句 0 句在 STRINGDATA → 依 journal 裁定屬下階段（與 NPC 對白同批） |
| NPC 選單「離開」沒對齊 | 未修。text log：右欄 招募／扒竊 x=112、賣出 113、離開 114（同寬 24）；左欄三個都 46。量寬 0x00407010 對 CJK 一律同寬，差異疑來自按鈕框中心本身（英文標籤寬度不一而看不出）；未證實。2px 外觀，列下階段 |
| Kill 譯「即死」 | 1335 殺死→即死（05e843b）；2357 Kills 仍「擊殺」。run 085203 仍舊字＝PATCH.002 過期（07:31 建），wiz8-run 加「產物比來源舊就拒跑」；run 090440 MG：正常 |
| 「獲得 4 kills」 | exe .data `kill`／`kills`（0x006174c0／b4，0x004ea008 依數量選）→「次擊殺」；run 090440 MG：正常 |
| 戰鬥 log 中英間空格不一 | 票 H（aa4e17d 規格、5fdc411 實作）；run 085203 `space hit`，MG：「戰鬥文字有分開了」 |
| （run 081820 判讀）離開遊戲確認框字幾乎看不見 | 9f2339f：多色亮字取「亮度 ≥ 最亮一半」像素的中位數（離線 STI：dialog_font 66,66,57→222,222,198；Opt_title 44→214 原也偏暗），journal_font（slot 1）照舊。MG run 085203：「離開遊戲的字現在很清楚」、選項標題「比較清楚」 |
| 對話框「隊伍金幣：546685g」g 沒譯（商店的 g＝STRINGDATA 1943 已譯「金」）| 9f2339f：g 來自 exe .data UTF-16 `%s %dg`（0x0064fc8a）與 `%dg`（0x0064a0a0），dll 逐字比對後換成「金」。run 085203 第一處位址錯（指到字串中段，sig-mismatch 沒寫入），起點改 0x0064fc84；run 090440 patched ×2，MG：正常 |
| 商店「隊伍物品」偏右 | 未修。與「離開」同形：都是右欄按鈕的中文偏右 → 同一根因的第二例，下階段一起查（量寬或按鈕中心）|

偏離：
- 5.3 機檢比「總像素寬＋明示 LF 數」，不比「框內行數」（框寬沒有來源）；只用 Wiz_Text_Font 量。
- F 的簽章（codex 沙箱無 exe）錯 3 列，cc 改 wrap.md 後 check-sigs 29/29。
- 圖集原本不吃術語表 → 術語專用字顯示成 A（遊俠），d9e5263 修。
- 1222 Light Sword 依 MG 改「輕劍」；有星號的 *LIGHT* *SWORD*／*Light* *Shield* 是光劍／光盾（af69aa4）。
- 票 H 派 codex 時 impl-run exit 7：白名單外改動＝cc 同時 commit 的 Kill 譯文，與 cc 誤用 Decomp.java 參數順序在 repo 根留的空檔 `0058af60`（已刪）。
- R5 研究派單的 ledger 遺失，muse 重派因報告檔不存在被 r0 拒派（未重做）。
- stop hook 版的 5.6 寫「錄影 10 分鐘」，檔案已改 PrintScreen，依檔案。
- df1c318（日記裁定 commit）混入 BoutiqueBitmap OFL 刪除（字形實作的一部分）；已 commit 不改寫。
- 字形實作 cc 自寫（未派 codex）：與 dll／圖集緊耦合，實機回合短。
- run 073156 崩（疑 alt-tab／焦點，原版也有，未證實）；073421、075346 皆 600 s alive。
- run 075346 log 的英文（casts…regains、Game Quick-Saved!）出現在讀檔當下（text log ms 167921，早於「進入 地牢」）＝存檔裡的舊訊息歷史；同場之後的新訊息是中文。唯一新出的英文是「獲得 4 kills」→ 來源已找到（exe UTF-16 字面，當初只搜了 ASCII），已修。
- run 085203、090440 都是 MG 驗完自行離開（MG：「我是自己離開的」），未滿 600 s 不是崩潰；after_clean 皆 true。

## 階段 6（`goals/stage6.md`，2026-10-07，MG /goal；HEAD a218417 起算）
- 6.1 研究 R6（muse）：收貨 1991cd3。**報告有錯，以票 I 規格為準**：MSG 的「段後腳本」其實是偏移表（0xffffffff＋offsets＋N＋E）；MSG 是完整句子（告示、地點訊息、隊員台詞），不是碎片。
- 6.2 `tools/strlib`（6.2a＋6.2b 合票 I，codex sol）：規格 a102f9e、收貨 c8ff329、帳本 780977c。DBS 與 MSG 51 檔同一格式，54 檔逐 byte 往返。codex 沙箱沒有 DATA.SLF，真實檔測試在那邊是 skip；cc 本機跑 `go test -v` 六項 PASS。
- trcheck `-pronoun-forms=false`（字串庫不檢 STRINGDATA 724–732）：ca62bd5。
- 6.3a 物品描述 326 筆、法術描述 101 筆：1a510b5。`scripts/check-lib.sh` 依 `translations/lib.tsv` 從 DATA.SLF 現解原文跑 trcheck；check-lib.test.sh 四種注入皆紅，已進 CI。
- 6.3b PATCH.002 帶字串庫＋打包後讀回斷言（注入→FAIL）：a55e061，CI exit 0。遊戲吃不吃 PATCH.002 對 DATA.SLF 成員的覆蓋：**未驗，留 6.4**。
- SPELLEFFECT.DBS 160 筆全空，不翻。
- 6.3c MSG 51 檔 4308 筆（cc 子 agent 11 批）：89d36ca，CI exit 0。PATCH.002＝STRINGDATA＋53 個字串庫（405939 bytes），打包讀回全一致。
- **斷行／寬度風險（6.4 要看）**：MSG 尾端是偏移表，不是腳本 → 改長度不會壞資料（讀回已驗）。畫面上的風險還在：
  - 隊員台詞與告示走哪條繪字路徑、框寬多少：未取得。5.3 的 wrap.h 與票 H 的中英間距只在掛了 hook 的入口生效，MSG 的顯示入口有沒有經過它們，未驗。
  - 最長的 MONASTERY:0（日記）、MARTEN:20 原文就很長，中文字寬＝字高，可能溢出或被截斷。check-fit（5.3）沒有涵蓋字串庫：缺框寬資料。
  - 描述（ITEMDESC／SPELLDESC）同理：物品／法術資訊框的寬度與捲動未驗。
- 6.4 實機要驗（MG）：① 遊戲吃不吃 PATCH.002 對 DATA.SLF 成員的覆蓋（Patches/ 對 Data/DATA.SLF）；② 物品／法術描述框；③ 隊員台詞（訊息區）與告示；④ 長文是否溢出。
- 6.4 實機（MG play-free 三場，run 151320／152412／153832，2026-10-07）：
  - ① 覆蓋有效：描述、法術、隊員台詞皆中文（MG 目視）。
  - 缺字畫成「A」：圖集字表只收 stringdata＋術語表，缺 862 字 → 4510f0b 改收全部譯文，`hook/ddraw/atlas-cover.sh` 進 CI（舊 atlas FAIL 862／新 PASS）。MG run 153832：「缺字沒了」。
  - 末行底列被裁（訊息區、描述框；戰鬥訊息原本就有）：Wiz_Text_Font gh=11 cell=12 墨到第 10 列，框裁在第 10 列。cbd2e08 把墨底提到 gh-2（oy 0→-1）。**待實機驗**。
  - 描述框行距緊、子 agent 疑似見行首「、」「（」：MG「描述框還算可以看到」，未處理。
  - NPC 在對話視窗外講的話仍是英文 → 階段 7。
- 譯者存疑但未改（列給 MG 參考，不擋）：專名一律留英文（Savant、Cosmic Lords、Helm of Serenity、Destinae Dominus、Trynton 的 Bough 名）；TRYNNIE:22「TRYNTON——七枝之城」是意譯，與 Bough 留英文不一致；PC 台詞的口音（M_CUN20、M_ECC20、F_CHAOS20 原文是外國腔破英文）用破碎中文模擬。

## 階段 7（`goals/stage7.md`；MG 2026-10-07「開始做下階段的規劃」）
- 7.1 R7 研究 NSF 格式：規格 1c3de81，派 muse（impl-run，背景）。收貨後 cc 本機抽查（R6 報告有錯的教訓：結構要 cc 自己解一次）。
- 接著：票 J（codex）NSF decode／encode，165 檔往返 byte 一致＋改長度後偏移重算；lib 管線（PATCH.002）帶 NSF。
- 7.2 cc 翻對白；關鍵字照 goals 第 1 步留英文、前後空格；新機檢：原文關鍵字在譯文中仍是獨立英文詞。
- 7.4 實機：至少三位 NPC 完整對話＋交易。關鍵字清單寫進存檔 → 測試要開新遊戲（德文翻譯說明，推測）。
- **/goal（MG 2026-10-07，離開期間）進度**：
  - R7 收貨 6345485（muse）。cc 獨立掃描（scratchpad nsfscan2.py，不入版）：文本 10320、關鍵字 4590（n≥2）；R7 的 4594 多 4 筆 n＝1 誤判（CONSTABLE 8432、MADRAS 24090、SCREGHOLO 9551、DRAZIC 19604，DRAZIC 那筆吞掉 0x4cad 文本前導）；R7「31 段無前導第二文本」未證實（前導規則已涵蓋）。R7 報告不改。
  - R8 收貨 16dcf2b（muse）。**R8 給的 0x004071f0 簽章是錯的**（推導版 `a06e6b00`）；draw.md 用實際 exe bytes `8b4424048b0c85a0b66e008b4918668b`，R8 報告不改。
  - 行距 hook 664e66b：0x004071f0 整個取代（行高＝Sub[0].h＋oy＋pitch_extra[slot]），Wiz_Text_Font 系 slot 5/13/14/15 +1，其餘 0。check-sigs 30/30、CI 綠。
    同 build 對照 run 164054（+0）／164309（+1），子 agent 判讀：menu 0 差；party／party2 只有說明框標題行與頭像下名牌下移約 1 px，無裁切；名冊列英文名貼中文行**兩邊都有**（原本就有，非行距造成）。說明框內文、訊息區多行**沒被 scenario 走到**（盲送鍵到不了，未取得）→ 交 MG 實機：`notes/zh-patch-log.md:212`。
    第四場（oy−1 末行）結果：首行被裁，已 revert ad1bc4f。
  - 票 J（codex sol，worktree wiz8-mod-J）：規格 5fde200、收貨 e4dbf20（重做 1/2；本機 go test 五項 PASS，TestRealAll 165 檔實跑）。grok 稽核請求 wiz8-mod-audit c9fa04b（來源＝worktree detach 在 e4dbf20，主 repo 繼續 commit 不影響「受審 repo 沒被動」閘）。
    - 稽核 m0ft：抓對 1 誤報 0 編譯紅 0。F1（kw > 65535 未測、拿掉上限的突變全綠）→ 補測 aaa7134，突變 FAIL／還原 PASS；收件 wiz8-mod-audit 6894568。F2（改 chunk 數後節點前導 MM 不重算，重解會掉文本）＝已知限制：管線保持 chunk 切分不變，譯文不可增減 chunk。worktree wiz8-mod-J 已移除、分支 j-nsf 已刪。
  - 7.2 起步：NPC_MYLES／NPC_FERRO／NPC_ANTONE 全部對白（94167ce；cc 子 agent 三批）。選這三位：MG play-free text log 出現最多（FERRO 16、MYLES 12、ANTONE 4），開場台詞是隊伍剛到 Arnika；Myles 是商人可做 7.4 交易。RPC_MYLES（入隊版 Myles）不在此批。Ferro、Antone 是 Rapax 工匠。
    - 關鍵字機檢：trcheck `-keywords`（規格 69b452f、實作 9654e1f，注入→FAIL）；check-lib 對 .NSF 用該檔關鍵字定義（排除 YES／NO、NPC_／RPC_）；check-lib.test 第 5 種注入（關鍵字譯成中文）紅（924c6a0）。
    - 寫法：專名只留英文（` Arnika `）；一般詞寫「銀行（ bank ）」。可讀性待 MG 裁（清單第 15 項）。
    - PATCH.002 帶 3 個 NSF（924c6a0）；產物讀回逐 byte 一致，另驗解出的 chunk 序列＝原文代入譯文。CI exit 0 @94167ce。
- **階段 7 實機回饋修正（/goal，MG run 170652 回饋）**：
  - NSF 對白截斷（goal 1，完成）：349992f（0x0055a242 改呼叫 nsf_store 存 UTF-8；swprintf／vswprintf IAT 解回）、dcc30d4＋01ebd41（訊息區回顯格式 L` "%S"` 0x0061c43c 也解回；run 174540 亂碼來源）。驗收 run 175048：text log Myles 回覆與 `translations/nsf/NPC_MYLES.json` 逐字相同、問候完整；子 agent 判讀橫幅中文無亂碼。細節 `notes/re/nsf.md`。
  - 訊息區行距（goal 2，完成）：6b86462＋be56255（notes）、8dd5af9（hook）。行距寫死在 0x0058c749 imm8（不經 0x004071f0）→ 12。行距 12 時框只容 6 行（run 180720 第 7 行被框底裁）→ 滿 7 行時從第 1 行起畫最新 6 行（0x0058c62c stub）。**行數 7→6**；上限：捲到頂時最舊一行看不到（捲動處約 45 個 7 未改）。驗收 run 181402：y 366…426，子 agent 判讀 6 行、首末行完整。捲動未自動驗。MG 實機 run 185712：「測試起來感覺沒什麼問題」（第 16、17 項；log y 366…426、Mt Gigas 回覆完整中文）。
  - 圖集缺字（MG run 185712：「介紹費」的「紹」畫成 A）：字表與 atlas-cover 漏 translations/nsf → 35600af 改收全部子目錄，補 34 字；CI exit 0。實機待下次一起看（清單第 18 項）。
  - 實機清單第 16、17 項（`notes/zh-patch-log.md:212`）。
  - 自動情境的輸入：SetCursorPos／mouse_event 點擊不可靠（180015、180231 兩次失敗）。MG 2026-10-07 確認：自動 run 期間「我剛好操作好幾次因爲我做一定比你截圖 + ocr + 移動要快」→ 先前的點擊「成功」與 175048 的關鍵字詢問是 MG 操作；驗收證據仍成立（text log 逐字比對），但不能算自動情境走到。需要互動的實機驗證（點關鍵字、捲動）以 MG 操作＋cc 讀 log 為主。改用命令列 `/LOAD`（直接讀 CurrentGame.SAV，a2b7b37 情境 args）。每次讀檔會改寫 CurrentGame.SAV → 跑完從 `~/.local/state/wiz8-mod-savebak/` 還原並 `sha256sum -c`。
  - text log 第 10 欄＝呼叫端返回位址（找寫死常數的呼叫鏈用）。
  - 偏離：代碼票原應優先派 codex；實際由 cc 寫（hook 與反組譯緊耦合、沙箱無 exe 不能驗）。
- **NSF 全量翻譯（MG 2026-10-07「平行派工把全部的 NPC 一起派出去給 CC 翻譯」）**：
  - 83e8f70：162 檔（NPC／RPC／VOC 全部；連同先前 MYLES／FERRO／ANTONE 共 165），lib.tsv 218 行。check-lib 218 ok、0 FAIL；atlas 2691 字，cover PASS；CI exit 0 @83e8f70。
  - 工具：`scripts/nsf-pkg.sh` 產工作包（輸出在 repo 外）。14 批子 agent 翻譯，各自用 check 自驗。
  - 重審（MG：「確保前後文『翻譯的方式』一致，但不要改成『翻譯的語氣一致』」）：
    - 已統一：Cosmic {{Lord}}（28 處）改回英文 Cosmic Lord(s)；─ 改成 —；〈〉 改成「」。
    - 「你（ you ）」：MG 裁 A 拿掉 → 73bbcdf（YOU 關鍵字豁免）、b547f84（138 處）。
    - 稱呼：MG 裁「統一用你們」→ 81a2455（110 檔，只插或刪「們」，pguard 逐檔驗；CI exit 0）。單數「你」只留給指名單人、泛指、引述、NPC 對 NPC；「你好」不動。
      MG 記得的「Rapax 女王選人洞房」：NPC_QUEEN 裡沒有這段（女王是被救的囚犯）；實際是 Al-Sedexus 獻祭（VOC_AL-LURE／AL-MADEUS／AL-SEDEXUS，問話對全隊，只有 AL-SEDEXUS 16「我那麼想要你」對被選中的人）與 NPC_CHILD（對那位「父親」，保留「你」，id 4 對全隊）。
    - 留著不改：術語表 raw-term 規則逼出的同義詞（力量→威能、氣力等），一般語意不套術語。
  - 實機：MG 沒有每個 NPC 談話前的存檔 → 抽樣看，其餘在正常遊玩中驗。

- **收尾 /goal（MG 2026-10-07 晚，不卡住原則）**：
  1. 譯文重審、2. 日記、3. 殘留版面、4. 7.3、5. 發布 完成；6. 收尾見下。
  - 日記 FACT.DBS：票 K tools/fact（codex，8f2510c、帳本 862de04）；337 句 `translations/fact.json`（id＝2×record＋field，術語用碼）；check-lib／build 加 FACT 分支。實機 run 212736：`j` 開日記，text log 中文、子 agent 判讀無裁切（清單 19、20）。
  - 殘留版面：run 212736（`scenarios/layout-probe.json`）主畫面狀態列無擠壓；`m` 開不了地圖；戰鬥中文行距未取得（清單 21、22）。
  - 7.3：R9（muse）＋cc 補查 `notes/re/misc.md:47`（89f9abf）。兩個 UTF-16 鬆散檔即可啟用 translated→English 查表，但點字只用空白切詞 → **不開碼票**（偏離：原 goal 預期要做，改交 MG 裁）。
  - 發布：票 L tools/zhpatch（codex astra medium，r0 一輪過；891d50f、帳本 7f9a0f5）；**ALLOW_BIG_TICKET**（白名單 20 檔，codec 抽到 internal/ 是一次重構，不拆）。release（beae40d）：`scripts/release.sh <zip>`＝ddraw.dll＋zhpatch.exe＋translations＋bat＋INSTALL.txt＋OFL，不含原版衍生物。暫存原版複本安裝／移除實測 7 項全過（1e66f52，`notes/zh-patch-log.md:257`）。CI 加 zhpatch.test.sh，exit 0 @10ab3e3。
  - L 的 grok 稽核 C2qO：抓對 1 誤報 0 編譯紅 0（收件 wiz8-mod-audit 5ec4c71）。F3＝zhpatch.test 未進 CI，收件前已補 10ab3e3；F1 負 id、F2 CRLF lib.tsv 是與 oracle 的分歧，非缺陷（oracle 有 check-translations 擋／失敗退出）。worktree wiz8-mod-Laudit 已移除。
  - 實機清單＋待裁：`notes/zh-patch-log.md:299`。

- **2026-10-08 名稱 DBS 與法術名統一**：
  - 怪物名 2335 欄（7d29843，按 73 種族定尾詞；機檢 f93208a）、物品名 790 筆（09cf378）、法術名 120 筆（106697f）。MG 實機：「怪物名稱正常」、物品名正常。
  - dbsname 改名清零整欄 → 物品全變匕首（欄尾是記錄資料）。ec0869d 改為只寫新字＋NUL；R6 更正 fde53bb。MG：「物品類別跟圖示都正常了」。怪物欄尾同修，**未另行實機驗**。
  - 取得物品訊息 488／489＝「取得%s」（MG 裁定，b7f5302）。
  - 法術名統一 8005bbe（法術書描述 62、STRINGDATA 11、其他 4）；機檢 `scripts/spell-names.py`＋例外 `translations/spell-names-exempt.tsv`（d650daa），CI exit 0 @d650daa。**新 PATCH.002 未部署、未實機看**；法術名嵌句不加「」（MG 2026-10-08：「不用加「」」）。

- **MG 2026-10-08 裁定（「清單照建議即可」）**：清單 9 名冊英文名貼中文行 → 不修（原版即如此）；清單 15「銀行（ bank ）」→ 保留；清單 20 日記字形 → 先不做（MG 再確認「A」＝字形結案）；7.3 點中文關鍵字 → 不做（MG 實機點括號內英文可用即結案，不可用再回頭）；字距加寬清單 → 刪除。
- **實測 run 20261008-031828（play-free 600 s，PATCH.002 c8dd329a…，MG：「好了，看 log 都沒問題」）**：法術描述「擬態」、交易／賣出、Myles、He'Li／Antone 對白都是中文。關鍵字點選：對白提到的 `Fuzzfas Potion Palace`、`drinks` 之後各出現在輸入列，接著有回應（274781→277171 ms）→ 點括號內英文可用 → **7.3 結案**（`local_data_check`；點選動作是從 log 時序推的，沒看畫面）。
  - log 裡的英文系統訊息（Game SAVED!、Survivors earned…、skill increased…、healed for avg…）是**存檔裡的歷史訊息**：CurrentGame.SAV 存著 UTF-16 的「Game SAVED!」4 筆、「Survivors earned」13 筆，都是 mod 之前留下的。本場新產生的訊息是中文。這些系統訊息的譯文（stringdata 1780／1781／561／475）本場沒有新觸發，未實測。
  - 未見：「紹」（那一行沒走到）、戰鬥中的怪物名（log 裡沒找到）。
- 底部狀態列（清單 22）：MG 2026-10-08 實機 run 033445「底部狀態列也是好的」→ 結案。
- **run 20261008-033445（MG 提早關，存檔已還原 0 non-OK）**：MG「戰鬥也沒有問題」→ 清單 5、6、21 結案。log 看到：怪物名「煤魔像」、攻擊／命中／回合／經驗值／技能提升都是中文；「紹」（介紹費）有畫出來；讀心術「Myles 的腦中一片空白。」。
  - 英文行（The party receives 49 gold、A specific item is required here、casts Rest All…、Game Quick-Saved!）都在舊 .SAV 的 UTF-16 訊息歷史裡（Northern Wilderness Retro 等），是載入存檔帶出來的，不是漏翻。
  - 找到並修了：「死亡！！!」全半形混排（DIES!!／DIE!! 後面引擎會再補一個 !）→ 913／914 改「死亡!!」（c2beaf5），overflow 登記移除 913（c86c18f），CI exit 0 @c86c18f。新 PATCH.002 4d42084e **未部署實機**。

- **2026-10-08 第二輪 grok 稽核（未稽核區四道平行，受審 1596260，各開 detached worktree）**：
  - Acheck F0EO（譯文機檢，防線）：抓對 6 誤報 0；MG 定位譯文檢查只當參考，全未收（F4 Stockade Guard／Brute Hogar 複數是原版資料錯，我們的譯名是刻意更正）。
  - Ahook SUs8：抓對 3 → 9078dd6（build.sh 簽章列文法＝check-sigs；Y 組整組驗後才裝；文字 log 開不了仍裝字型）。F2／F3 只過 build＋check.sh，**實機未驗**。
  - Arel Pchm：抓對 4 → 31e2aed（白名單拒 symlink；uninstall 沒有本包 ddraw 不刪 PATCH.002；install 回滾刪不掉照實說）。bat 以真 cmd.exe 對假遊戲目錄驗過。F3（Saves/ 算雜訊）屬設計，未修。
  - Acodec 0dyK：抓對 3 → 90beb32（dbsname.test 步驟 4 假紅修好、接進 CI）。F2 負 id 由 check-lib coverage 先擋、F3 CRLF lib.tsv 為參考，未修。
  - verdict 在 `~/Projects/llm/grok/grok-audit/wiz8-mod-A*/2026-10-08-1596260-*/verdict.txt`；請求與 log 收件 wiz8-mod-audit。

## 階段 8（`goals/stage8.md`，2026-10-08，MG 離席前要新 goal；HEAD 6aaf0ad 起算）
- 8.1 cc：`calc.LoadExe`（2c9fa1c）讀種族初值、職業門檻、主技能、feature、技能控制屬性、職業×技能矩陣、F06 權重；TestExeTables 需 WIZ8_EXE。注入（PMult 寫死 4）FAIL／還原 PASS。
- 8.2 codex 票 R：規格 364fa44、golden ae3c93f（cc 獨立 Python 參考＋G2 手算）、收貨 c14dc84（重做 1/2，第 0 輪過；main 以真資料重跑 calc 全測 PASS）。CLI `calc build -exe -dbs <scenario.json>`。
- 8.3 note `notes/research/ranking-history.md:20`（47ac19a）：L28 時 11 族 STR/DEX/SPD 都到 100，種族對近戰幾乎無差（對 AC 0 完全相同，對 Dark Savant id 564 差 ≤4%）；Mook SPD 只到 97、v=99 差第三揮；雙持對 Dark Savant 113–120、低於所有單持（784d957 依 F1 重算；初版 47ac19a 雙持欄偏高 20–25%）；Excaliber 單持最高。3 格手算與 CLI 相同。
  - 下限／推測：技能只算手動 258 點；主手「一般」模式沿用 B15 遮罩抽法、戰士狂暴對所有武器可用（推測）；Ivory Blade／Fang／Excaliber 若單手可再加副手（單／雙手欄位未定位）；DefPen≈DefHit；負重、裝備屬性、特效未建模。
- 8.4 grok 稽核 R（cXan，受審 c14dc84）：抓對 3 誤報 0 編譯紅 0，收件 wiz8-mod-audit 4cd1571（f07.objdump 不入版控）。
  - F1 雙持 K 漏 F07 +Dual Weapons、d=35（0x4ee44e）：cc 規格／golden 漏項，formulas.md:29 本有寫。F2 真資料測試只比部分欄。F3 build 技能值未檢 0..125。
  - 測試 91eadd1（舊碼 3 支純斷言紅）→ 修 9eff044 全綠；F3 補 tools/calc/main_test.go（修前紅、修後綠）。worktree ../wiz8-mod-Raudit 已移除。
- 階段 8 完成；待 MG：無新增。
- 8 之後（2026-10-08 研究，均在 `notes/research/ranking-history.md:104`）：擊殺前十、8 職業種族×武器、裝備唯一性前十（貪婪）與每職業一人最佳分法（精確 B&B，scratchpad opt99.py）；鞭鏈類不雙持（MG 實玩）；Power Strike 解鎖時間差未建模（MG 提，未做）。HEAD 0f17241 已推。

## 階段 9 戰鬥加速（`goals/stage9.md`，2026-10-09 MG：「開工，先派研究票，你來執行到需要我測試爲止」）
- 9.1 已派（背景）：R13 社群做法 → muse（worktree `../wiz8-mod-R13`）；S exe 計時 RE → codex gpt-6-astra medium（`../wiz8-mod-S`，build/ 放 asm、decomp、Wiz8.exe 副本）。兩張 r0 都只紅在宣告的測試上。
- 9.1 收貨：R13（5ee1700）、S（16839bb）。關鍵：行動等待 0x6850dd 由遊戲內 Combat Speed 控制，下限 0 ms，不做 hook；怪物移動速度＝0x6850ff（Wiz8Fast 改 CFG 偏移 0x4f，cc 靜態確認）。
- 9.2 規格：`tickets/T-combat-speed-hook.md`＋`hook/ddraw/speed.test.sh`。戰鬥中把三處讀取改指我方 float（×`wiz8mod.ini [combat] monster_move_pct`，預設 400），不寫 CFG。
- 9.3 收貨 c6e9e74（codex sol）；CI exit 0 @c6e9e74；四段注入（speed.h 拿掉戰鬥判斷 → FAIL 探索不動 → 還原 PASS）。
- 9.4：發布包 @c6e9e74（zip 0e435ecc，解在 `%TEMP%\wiz8-mod-release-c6e9e74\wiz8-mod\`）。舊版用它的 uninstall.bat 移除 → 自動 run 20261009-030507-cjk-zh（60 s 活著、`speed monster_move_pct=400`、無 sig-mismatch、after_clean=true）→ install.bat 裝回。ddraw.dll 2ed92817、PATCH.002 19feeaf3；原版三檔不變（Wiz8.exe 18a74ff6、Data/DATA.SLF 12ce5055、PATCH.001 74bbc59a）。進戰鬥那行 `speed combat` 自動 run 沒走到，待 MG 實玩後讀 `%TEMP%\wiz8-mod-ddraw.log`。
- MG 試玩（2026-10-09）：「走位快很多了，不會太快」→ 預設 400 定案。log：`speed combat base=2500 eff=10000`（03:22:29，無 ini）。事件延遲（800／1500 ms）與動畫加速待試玩後再議。
- stage10 A1 U 收貨 fc681fd；A2 V 收貨 d571334、W 收貨 c2c103d（CI exit 0 @bfd5b44，四段注入 anim-rule）。
- A3（2026-10-09）：發布包 @5db11fb（zip bc325214，解在 `%TEMP%\wiz8-mod-release-5db11fb\wiz8-mod\`）。13c29f1 包 uninstall → 自動 run 20261009-042703-cjk-zh（`speed event_speed_pct=300`、`anim_speed_pct=200`、delay 五列、`hooked anim 0x004c2b5d`、無 sig-mismatch、活到 60 s、after_clean=true）→ 新包 install.bat。原版三檔前後 sha256 相同（Wiz8.exe 18a74ff6、DATA.SLF 12ce5055、PATCH.001 74bbc59a）。

## 階段 10 B 公式計算器（`goals/stage10.md`，2026-10-09）
- 盤點：候選 719 全數歸類（X b00–b11，codex sol；evchk 每列證據行落在自身函式內）。公式 384、其餘非公式／排除。
- 機制：模擬器差分 Y（92c179a）；D1 欄位 Z（383af25）；`scripts/node-accept.sh <清單>`（清單 `位址<TAB>calc 函數`）＝節點票驗收，只看清單節點的紅；覆蓋豁免各票寫 `notes/re/cov-exempt.d/<addr>.tsv`。
- 節點票：`tickets/N-<addr>.md`＋`tickets/nodes/<addr>.list`，一票一節點、白名單 4 檔（case json、iface、`internal/calc/n_<addr>.go`、cov-exempt.d）。就緒集＝D0 且公式類子呼叫全 D4。塊數 >60 派 astra，其餘 sol。
  - 派工要帶 `SANDBOX_EXTRA_RO=$HOME/.local/state/wiz8-mod-emu`（否則沙箱裡 build/venv 是斷鏈，執行方跑不了模擬器）。
  - 收貨：cc 在 worktree 重跑 node-accept，綠才搬進 main、登記簿標 D4＋calc 函數名。
- 進度（2026-10-09 14:00，HEAD 見 git log）：D4 148／公式 384（到第二十二波 9e1eb05）。D0 236。
  - 第一～十一波大多收貨。emu 樁陸續補（heap、getMsTime、printf 族、fopen、srHeap::allocate、CreateFileA、GetLastError、srTexture 建構子、getNewFrameHandle、_CIacos、FormatMessageA、swprintf）；SR.DLL 的 void thiscall 方法表 `SR_VOID_SETTERS`（emu.py 檔頭）。依情境的 import 回傳值寫 case `stubs`。
  - 更正票：舊票停手條件擋住而 case stubs 能解時，開 `N-<a>b.md`／`c.md`（複製＋「更正」段）重派。5242b0 在跑 N-5242b0c（SR.DLL import 一律 case stubs）。
  - `WAVE_PRIO="<addr…>" bash scripts/nodes/wave.sh K`：就緒集先取指定節點。calc-qa 待節點的公式閉包（非 D4）116 個，從第十二波起優先。
  - 收貨 commit 訊息要列實際收的節點：harvest 會連帶收跑完的其他 worktree（3a19142 漏列，下一個 commit 補註）。
  - impl-run 拒派也不印 `結算`：等待條件要同時認 `拒派`。
  - 重派前：worktree 的 TestEmu 若被前次半成品弄紅，先把 case／iface／cov-exempt／build/emu 證據挪到 `.claude/pm-diffs/prev/`（r0 不能有沒宣告的紅，否則 impl-run 拒派）。
  - 節奏約每小時 6–10 節點。
  - 2026-10-09 午：codex 用量上限（重置 2026-10-14 14:02）、muse 429（重置 2026-10-12T00:00Z）。改由 cc 子 agent（general-purpose、各自 worktree、照原票＋白名單、不 commit）做節點；cc 收貨：worktree node-accept → `scripts/nodes/merge-node.sh <a>` → main node-accept。
  - `scripts/nodes/redispatch.sh <a> <票>`：worktree 跟上 main、挪半成品到 prev/、重派。編譯不過的 `n_<a>.go` 也要挪（4edd20）。`NODE_EXEC=muse` 讓 mkticket 產 muse 票。
  - 新樁（本段）：srNode(srNode*) 建構子、MSVCRT 字串族（wcscat wcslen wcscmp strchr atoi _stricmp strncpy _strnicmp）、case stub `write`／`st0`、floor、_CIasin、memmove。未樁：_CIpow rand srand wcscmp 以外的寬字串比較。
  - grok 稽核 zoWN 結案（verdict 抓對 5 誤報 0；inbox 0fe6e12）：F1–F2 修畢（Y4a 01b8d04），F4 修畢（da65cfa），F5 根因 bb92e90，F3 修畢（Y4b–d 7e0b80c、Y4e cc 自做 2af1ef3；驗收 `scripts/case-index.test.sh`）。node-accept 跑全部 TestEmu，main 紅時所有節點票連坐紅。
  - CI 注意：10 個 codex 同跑時 SLF 實檔測試讀 /mnt/c 報 `cannot allocate memory`（單跑綠）；CI 等派工跑完再跑。
  - 4e99c0 撤票（7df1baf，依賴 553f10 仍 D0）。推送時這次刪票要 `ALLOW_TEST_SHRINK=1`，理由在 7df1baf。
  - cc 收貨腳本：`scripts/nodes/`（wave.sh <K> 派、harvest.sh 收、ready.sh 就緒集、regen.sh 重生 build/emu 證據；log 在 build/nodes-log/）。awk 位址一律字串比（`$1""==a""`，51e920 之類會被當科學記號）。
  - CI 競態已修（bf07e76）：emu.test.sh 改在 go 道 go test 之後跑。
  - 2026-10-09 晚：`scripts/nodes/take.sh <a> [備註]` 收 cc 子 agent 單一節點（worktree 跟 main→node-accept→merge-node→main node-accept＋vet＋calc test→commit→移 worktree）；`NODE_EXEC=cc wave.sh` 只開票＋worktree 不派。check-formulas ARG_MAX 假綠已修（f7b37cb，全登記簿重驗只 520750 補 iface）。新樁 `_CIpow`（ca2a99b）。
  - 子 agent 暫存一律 `scratchpad/N<a>/`（共用暫存曾互相覆蓋腳本、寫錯 worktree）。WAVE_PRIO 讀 `scratchpad/qa-addr`（閉包位址一欄）。
  - 已知覆蓋缺口（記結構不記判決）：4e65d0 模板快取未命中 4 塊豁免為「模擬器限制」（54a8a0 讀資料庫檔，無檔案 I/O 樁）；544b60 只驗提早返回（飛彈生成進實體系統、未樁 exit）；4f0990 UI 模式與 +0x90c 合一輸入；5237e0 不組 4edd20（gameStarted=0）。
  - 續（第二批）：4e4500 不驗 50e8c0 重算的怪物衍生屬性（rec+0x1db 起），效果清單固定空；50f1a0 templateId≠0 按案例環境回全 0（模板檔載入在 emu 恆失敗）；53fea0 不含怪物受害者（目標種類 3），519c00 其他怪物掃描為空操作；534690 模板 id 限 0..3 與 0x224；5012e0 who 固定 0。
  - 續（第三批）：4c1d20 不模擬 NaN／Inf 與 f32 溢位；4e6370 模板快取未命中 8 塊豁免為「模擬器限制」（同 4e65d0，54a8a0 讀物種模板檔），calc 前提＝模板已快取，cc 裁決甲（可逆；要補就給 emu 加「檔案內容為輸入」的 I/O 樁，可一次收 4e65d0／4e6370／50f1a0）；53c490 random 列 target kind 1 只在邊界列。 4e6370 另：每清單成員上限 2（arena 容量，calc 超過即 panic）、vtable 動畫虛函式以假碼代替。 56d030：wcscspn 固定 2（換行不覆蓋）、字型 0..1、面板子元件空、0x686a70 固定 1、認領 NPC 固定 partyNPC0。 4f0c80：怪物 fixture 8 筆（calc 超過 panic）；3 個 assert 塊合法輸入可達（模式≠0 且門檻 0），原版即 srAssertFail，calc panic 對應。 537380：情境 7＋類別 8＋旗標 0 原版 NULL 崩潰（calc panic）；法術 0..255、目標欄 0..10。5454c0：|座標|>2^31／NaN／Inf 不模擬；fsin／fcos 以 binary64（x87 末位未驗）。543dc0：法術／道具法術 ID 0..9、道具索引 0..3、怪物 ≤2 筆且 ID 123／124、攻擊 context 固定 0。 536d60：法術 0、1..6、88，物品槽 0..3，成員 ≤2；情境 7＋行動 8＋存活原版崩潰。
  - 續（第四批）：4fe740 怪 100–102、群 200–202 fixture（越界 panic＝原版 assert），target kind 2 語意推測。4ed550 怪 ≤8（panic）、battle／0x685174／0x685178 固定非 null。51b0a0 法術 1..3、物品 0..3、隊形 0..7／0xff（0x80..0xfe 負索引不涵蓋）。50c870 持有記錄清單建模 16 筆（原版約 499，超過 panic）、0x688290 除錯開關固定 0。53d870 who 0..3、tchar 0..7、ttype 1／3、mainAction 0..5、怪物目標只做 tmon=mid=0（v1≥2 模擬器崩）；既有 536f60／5199f0 calc 值域窄於此用法（本檔另寫，未改既有）。5545f0 0x68517c=0、介面全零；隨機批前一隊形固定 c→c，非有效成員區位 ≥5 原版讀堆疊外（calc panic）。
  - emu 位址樁（c28e782）：只准換登記簿「排除」類，`record`／`count` 把被換呼叫的參數與次數變輸出。4c75c0 以此換 544630 收貨（ae11958）。
  - 續（第五批）：4ecf50 怪 ≤8（panic）、58ac00 log 不在輸出。437980 NaN／Inf 與 scene+0x180 null 未測，437f30 閉包複製。55b290 已收。4c75c0 攻擊者 handle 固定、classKey -1、模板 0..2、random 只目標種類 3；cov-exempt 4c76fd（assert）。5592d0 補點游標越過 7 槽，建模 0..63（≥64 panic；搜尋最遠 41，非證明）。5871a0 585ed0 非公式不可樁、原樣執行（文字輸出不觀測，wcscspn import 樁只影響折行）；鎖銷 n≤8、領域索引≤5。518e30 怪 ≤3、物品 ≤4、法術 id<150 互異。536ad0 怪 id 123/124、範本 1..3、群組 0..1、法術 {0..6,88}；情境 7＋動作 8＋存活原版空解參考（calc panic）；與 536d60 差異見 n_536ad0.go。
  - 5308c0（monster_movement_choice）退件未收：子呼叫 434220→4604b0 拉出與 461960 同一棵未登記路徑子樹（45a1b0／460950／460b80…，>5KB），回傳決定可達塊。worktree 已拆、票留著。收尾與 461960 一起裁：子樹開節點票，或寫合格「未取得」。
  - 續（第六批）：55b250 itemId 0..818。48b9a0 記錄 ≤8、配置失敗回 NULL 不涵蓋（模擬器堆耗盡即報錯）。559bc0 cov-exempt 559cbb 依賴本版 exe 表值。519c00 距離 x87 另算（不共用 interceptReach）；突變存活 3（距離下限門檻 ±1、平方和順序、float32 取整）＝缺分辨列，D4 照收、列 grok 稽核重點。
  - 續（第七批）：53d010 怪 ≤2、群組 ≤2＋備 1、成員前綴。522a30 物品 0..3、後寫者勝別名模型。53cdf0 群組成員數>怪物數的列 calc panic、case 排除（原版該列提早回真；fixture 不一致世界，非合法狀態，但屬 cap 寬於原版）。
  - 4ec1e0：51d3b0 節點沒輸出它改寫的手部 avail（C+0x114d／+0x11a8），4ec1e0 內聯 saoAvail 重算；收斂＝51d3b0 輸出 avail 後改呼叫（小票，非阻擋）。iface field 語意未逐條對反組譯。
  - calc-qa Q3 部分作答（4f53d32）：旗標（53d870）＋命中 +10（541c00）；額外骰待 5420b0（weapon_damage，D0，尚未 ready）。
  - 43ccf0（lock_code_load_init）退件：整個函式輸入來自 404ea0（存檔串流讀取，ReadFile 包裝 `(handle, buf, size, &讀到)`，不在公式候選、未登記），一次呼叫約 20 次。要做需 emu 位址樁加「依序讀 case 位元組串流」並准換 404ea0；cc 試加白名單例外被 auto mode 擋（判為放寬守衛），列待 MG。worktree 留著、票留著。
  - ready.sh（b6c055b）改看穿過至多 2 層非公式函式的公式依賴；53fea0 在此前派出，其 519c00／518150 仍 D0。
  - 4ef7e0 退件未收：依賴 558640→557430→51c020（公式 D0，物品次數初始化）與 51c300 等放置鏈；worktree 已拆，票 tickets/N-4ef7e0.md 留著。wave.sh 會跳過已有票的節點：51c020 到 D4 後手動重開（mkticket／redispatch 或刪票重產）。物品定義來自載入資料（0x68516c），case 應對全部 id 生成定義，不寫 exe 起始裝備表 id。
  - 461960（ai_waypoint_path）退件未收：子樹 45b120／460950→460b80／461d10／45a1b0 約 7KB 無 calc，遠超票規模；分類本身標「不確定是否構成 AI 規則」。worktree 已拆、票留著。收尾時二選一：子函式各開節點票，或照 check-formulas 寫合格「未取得」（嘗試:／影響:）。
  - 循環群（呼叫圖強連通分量）永不就緒：162 D0 中 520d10 後面擋 149、547a50 106、533320 67。改一群一張票：清單多行（merge-node 1092d99、take 多成員），產票 `scripts/nodes/scc-ticket.sh <key> <成員…>`，worktree 手開（wave.sh 的開 worktree 段，decomp 每個成員的批次都要拷）。已派：520d10（4e8000 536570 53c990）、533320（5327e0 534ef0）。下一批 547a50 群（522fe0，依賴另 2 群）、43d940 等 21 節點大群（依賴 28 群）。
  - 533320 群退件（cc 子 agent 提早停手，零檔案）：被呼叫端 519f80 子樹（約 3.5KB）、51a2f0、51a510、5330e0、534dd0、53c630、532550→4d9080 子樹、4e5550／4e5620／4e5720／4e5840 等都**沒登記**（不是排除→不能樁）、沒 calc，合計約 8–9KB＋群本身 6KB＝整套怪物施法 AI。worktree 已拆、票留著。與待 MG「共同根因」同型（未登記 helper），一併裁：未登記 helper 由 cc 逐一分類入登記簿（公式→開節點票；引擎流程→排除可樁），或寫合格未取得。
  - 501660 已 D4（8c464aa，重派：重用 537220／522a30／51d800）。覆蓋缺口已補（a03a945：537220 輸入 136→187，模式／階段／選中角色／戰鬥記錄／隊形象限納入，3 組注入四段 FAIL→PASS；220 邊界列排除＝原版 assert／非法位址 202 列＋emu 未樁 `_finite`（atan2 隊形象限，隊伍與怪 Z 不同）18 列——後者是 emu 限制，5194e0 隊形也只驗 Z 差 0，未開票）。grok 8hvg 稽核：抓對 1（53c720 同值旗標假綠，96bc869 補列＋四段）。注意：take.sh 只收該節點白名單，跨節點 case 改動會隨 worktree 被刪（本次 53c720 改動遺失、cc 重做）。以下為首輪退件紀錄。
  - 501660 退件（零檔案）：靠 537220(who,spell,ctx)＝5010f0／53b7f0／537160／519180 組合（未登記），501530 的 calc 只是 ctx=6 特例，ctx=4 走物品法術射程、kind 3／4／6 走 5194e0／537780／5199f0。子 agent 探針證實 537220 ctx=3 對 501530 1357 列 0 差異（暫存區，未入庫）。建議：537220 開節點票（另 5 個呼叫者 5012e0／501400／501590／501790／501880 可複用）→ 501660 薄組合。同屬共同根因。
  - 520d10 群退件（零檔案）：536570／53c990 經目標搜尋 53c720（未登記，內含 qsort＋浮點比較器）、519180、537160、53b920、5010f0（未登記）與 5387f0／4e77b0／53b7f0（非公式，非排除→不能樁）；且 53c990 回頭呼叫 520d10 改寫 +0x3d／+0x6d，520d10／4e8000 不能單獨做對。worktree 已拆、票留著。**節點管線現況（2026-10-09）：ready.sh 只剩 501660 在跑，其餘就緒節點全有票且全卡同一根因**（被呼叫端未登記或非公式而不能樁），等待 MG 共同根因裁決。cc 建議：未登記 helper 由 cc 依語意入登記簿（規則→公式開節點票、引擎流程→排除），每批一 commit 附理由；量大（目標搜尋／施法 AI 整套）。
  - 2026-10-10：520d10 循環群 4 節點 D4 收貨（523888c；組合樁＋58aad0 排除樁；519180 rec0==6 幾何路徑未接、案例未達，panic）。check-formulas：D4 232／未取得 3／D0 151。新就緒 9 節點開票（71369c7）派 cc 子 agent：4e7590 536400 53bf80 51fb40 51e760 538e00 51c020 528ff0 51c5a0；533320 群仍在跑。prompt 在 scratchpad p-<addr>.txt（p-437980 模板＋fix-d 修正＋呈現 helper 停下回報）。
  - 待補（自己的帳）：536570／53c990 收貨時接受了 519180 種類 6（座標目標）幾何路徑 panic，但 536400 子 agent 查出合法輸入可達（a=7／8 且目標類別 3／4）。等 434f20 裁定（若屬「射線／遮擋」可登記排除→位址樁）後回補這兩節點並重驗；在那之前這兩個 D4 的定義域不含種類 6 座標目標。登記 427ab0／427ff0／4c5eb0 排除（403eb1c、1d26233、0d04d90）；4e82e0（存動作還原，寫遊戲資料）與 55f1e0（清游標手持物）不符條件①，原生跑。
  - 2026-10-10 第二波收貨：51fb40 7ddc36d、51c5a0 cbb5602、51e760 b2c4ae8、51c020 af65584、4e7590 重做 a9d552e、528ff0 35b1e46、538e00 ef34873、536400 31baa63（**上一條種類 6 缺口已補**：536570 種類 6 原生 coordReach，434f20 登記排除後樁，520d10 群重驗綠）、53bf80 e0de130。emu 新增 `ret:{at,n}` 依呼叫序樁（6d2c991）。check-formulas：D4 240／未取得 3／D0 143。
  - 已知缺口（53bf80 收貨時）：520d10 群 phase 的 icon-cursor 分支（55ee70→55ce80 快取物件 0x68ec68）原寫游標全域未經 emu 驗證，改夾具 panic；56aab0 draw 分支（phase mode 7）未入 case，阻塞者為未登記顯示 helper 420b40／4f21b0／4f23d0／482990／4c61f0／41f160／5a1950／4263f0／422d50（下層 4531a0／4f21e0／4f0800／40f100／4146e0）。待逐支查條件①、各一 commit 登記排除後回補。538c00 對 4c5eb0 的呼叫點（0x538cd3）calc 未建模（fixture 內未達）。
  - 第三波（7e5aa58）派 cc 子 agent：51e9f0 51ed30 50baf0 55a930 51f900 55a7b0 523330 4f88f0；533320 群仍在跑。跨節點改動 take.sh 會紅，由 cc 手搬＋重跑相關 list。
  - 2026-10-10 晚收貨：5248d0 10d2111、52b1c0 102a52d、533320 群 a69dac2、54b100 a665de3、52abf0 575b477、51d960 d0107d6、4f88f0 a87adc7、548e60 96148e9、5326f0 35b4d4c、520d10 群＋536400／53bf80 case 擴充 5f1be46、52b160 31ea03c、54e3f0 3044b37、5223a0 4507864、55a630 55f4cf8。check-formulas：D4 267／未取得 3／D0 116。
  - cc 依 MG 授權（「未來這些排除跟登記你自己判斷就好」）登記排除，一 commit 一列：5a19a0 54cf9e9、4f2380 cbffdb9、4f2040 301078f、51b5c0 非公式→排除 37f060f（前批 52f790／52e690／4f7610／4f93d0）。判準：不算規則數字、回傳不是規則輸入。522640／5227d0 **不排除**（經 506670 script_attribute_skill_gain 公式 D0），照 5223a0 原生跑＋calc 建模（cerSt.story／voice）。
  - emu 擴充：stub `write` 值可取 `{"at"}`（f2f5a96）；record `"n"` 逐次呼叫各留一格（fa87fc8，selftest 注入紅）。D4 樁 ret 政策：取自 case 輸入，除非 ret 設 0x5a5a5a5a 仍 diff=0（證明呼叫端不用 eax）；548e60 的 51f900 即被此查出（diff=445）。
  - 缺口：482a20（時間推進，寫 0x6874f7 日內毫秒＋0x687595 日數，呼叫 502010 公式 D0 與 5 個未登記 callee）未登記；520d10 群 tick 固定 delta 0 未覆蓋，要另開節點票或分類。5223a0：F 真 callee 路徑 737 列中 15 列與 Emu 不同未查（F 無自動測試）；圖示快取 miss（55ce80→549090 讀檔）未涵蓋。
  - 跑中（cc 子 agent）：55b7e0（照 5223a0 原生 522640／5227d0）、第五波 445500 522ef0 534850 55ab80 54f520。剩就緒：4f8cb0 559650。calc-qa Q4–Q8 待 5420b0 551ba0 551eb0 540170 5208f0 50bc90 4fa4d0。8bba541 之後未 push。
  - 2026-10-10 深夜：收貨 4f8cb0 8d76cab（F 走 4f88f0 kind-1 路徑時 panic，356/387）、547a50 群 210811e、52af50 739b68e、5029e0 822817f（F panic 309 列，全是 RNG 腳本不符）、5a1eb0 9a6b160。排除：524ca0 55a0a0 4e3af0 4ef4a0 4ef610 4acd80 5b1af0 5b1b30 4e4db0；523c00 monster_condition_apply 登記公式 D0 新節點（fe99a3d）。
  - 政策（cc）：被樁的 D4 callee 的回傳或寫入被節點讀到時拆 Emu／F：Emu 重放樁，F 呼叫真 Go 函數，F 專用輸入放 scratch 區，報 F panic 數與 F≠Emu 列數。豁免為不可達的分支，calc 要 panic。夾具上限不得限制合法 UI／狀態值（如 0x68ec78 模式）。非 D4 須直譯的碼超過約 3KB 就停手、報每函式位元組表。
  - 54f8c0 停手（worktree ../wiz8-mod-N54f8c0 保留）：等 523c00 D4。523c00 擋在 4e46f0／5049c0／4eb7f0／531540，這幾個再擋在 5323f0、506670、43d940（D0）。重開時沿用：4acd80 樁形式、52af50 的 Emu／F 拆法、actor+0x1e4＝怪物 id。
  - 缺口（前批）：55afa0 F panic 1289 列；534850 門檻列只涵蓋 tpl0；50e650 的 callee 有沒有寫狀態欄位，未量。
  - 2026-10-10 深夜續：收貨 52b7e0 50b160 504730 559650 553f10 504670 52a890 52a820 586e40 5208f0 50c040 5520d0 586c60 553350 5209f0 50bc90 5508e0 5535d0 5420b0 550a00（commit 見 `git log --grep 收貨`）。排除：561a20 55f1e0 58ac00。emu stub write 加 `"nullskip": true`（fe84db4，見 `notes/re/emu.md`）。
  - 政策補（cc）：樁的 ret 取自 case 輸入，除非 scratch 副本設 0x5a5a5a5a 證明呼叫端不讀（diff=0）；未樁的 import 用 case `stubs`，不 panic。
  - 缺口（本批）：52a2f0（最大 HP 重算）未登記、內聯在 52b7e0；586c60 未登記呈現函式 439b80 439d80 4f30f0 58ac00 585ed0 408860 在固定前提下原生跑；586e40 F 未建模 522fe0 體力歸零條件；553f10 不輸出 4ed9d0 屬性重建；5208f0 cov-exempt 為 NULL 物品 assert（calc 無此輸入、無 panic）；559650 51d960 被樁未組合 F；52a890 523a80 F 缺口、50e650 +0x161 寫入；50bc90 物品 id 0..15、553f10 RNG 未組合；5508e0 realm 0..5；5535d0 特殊模式 kill 指標非 null 看不出；550a00 事件佇列前態固定空、550a1f／550b20 目標種類固定 3 無 calc panic。
  - 2026-10-10 深夜續 2：收貨 5044d0 4e99c0 4e89d0 58a060 500060 4ef7e0 542960 517560 577290 4e4ab0。排除：408860 55e7c0 575390 55e2c0 55e1e0 569a50。emu 加 wcscspn 內建、拒樁 selftest 例改 571660（a24ad01；原例 5420b0 升 D4 後 selftest 自 ac666ee 起 exit 1，node-accept 不跑 emu.test.sh → push 前必跑完整 CI）。
  - 缺口（本批）：5044d0 517970 原生跑、calc 呼 diceNode；4e99c0 swing 次數上限 16、該欄寫入者未查；4e89d0 怪物 ≤8、模板 0..3；58a060 F panic 102/735；500060 4ffe70 提示字未建模（ebx=0）、地面列表上限 8、547940 0x1f 分支未走到、F 用零世界 520d10；4ef7e0 超 3KB（約 5.1KB，Emu diff=0 收）、F panic 279/298（子節點物品夾具）、劇情 id 路徑僅夾具不可達；542960 tgt==1 F panic 1346（姿態選擇器原因屬推測）；517560 0x4a1710 動畫模型 bbox（未登記 1706B、mesh 檔 I/O）calc panic，未排除（bbox 是射程輸入）；577290 單字寬、0x689b34 假設空字串；4e4ab0 F≠Emu 0 但無可觀察差、夾具上限。
  - 503100 停在 3KB 上限（可達需建模 4241B，其中 4ed9d0 子樹約 2.7KB）。cc 裁：4ed9d0 登記公式 D0（36b52a0）另開節點（此節點上限放到 4KB），503100 worktree 停放，4ed9d0 D4 後 SendMessage 續做（ff-merge main；呈現函式 58b410 590950 517a90 562a50 照 n_4ef7e0／n_52add0 先例原生跑、不入 Out、不計上限；4d9700 57dbb0 5179f0 4f96a0 小，照建模；除數 b19＝0 原版 #DE、calc panic）。登記後 50e5c0 50e650 50e700 553f10 4ef7e0 因 d1 依賴 4ed9d0 在 progress 模式紅，4ed9d0 D4 前不 push。
  - 收貨 531ce0（0557f2f）、5323f0（f5035b0，F≠Emu 531/531：F 組合未對 Emu 驗）、4ed9d0（534471c，progress 回綠）。
  - 2026-10-11：Sonnet 週額度用完（重置 10/12 10:00 台北），codex／muse／agy 也用完（MG 經 claude-memory-74 轉達：大票 Opus、小票 Sonnet、重研究 Fable）。子 agent 派工單模板存 `build/nodes-log/cc-prompt.tpl`（`@A@`＝節點位址；scratchpad 重啟會清）。
  - 收貨 531540（f655f8c；F≠Emu 9032/11932，F 以停放世界跑真實子節點，缺口見 commit）。排除：5a1890 56a5a0 595570 56aa30 56aab0 58b410（53a320 不排除：寫規則狀態 683fe7、68407f..87）。
  - 4e8370 裁決（cc）：53a930 不排除、原生跑＋calc 建模（它寫的目標表 683790+j*0x118 被 5387f0／53ac30 讀，53ac30 由 D4 4e7590 呼叫＝規則輸入）；assert-only 區塊照 emu.md:135 cov-exempt＋calc panic。
  - 收貨 503100（e8a5fb3；F panic 2840/3344、F≠Emu 165，F 世界缺 formation 表 554e70／戰鬥物件 0x68edcc，cc 裁不另開票，同 531540 記缺口）。
  - 剩 D0 69：除 48ca20、4e8370 外全在一個 24 節點強連通塊（施法 4fb4c0／54ba00 效果群、狀態 523c00／551ba0／551eb0、擊殺獎勵 4e46f0、劇本事件 506670／5049c0／43d940），其餘 42 個依賴它。已派 Opus 研究拆法（只讀，報告 $S/scc24.md）。
  - 24 節點塊拆法（cc 依 §5.2 自定，可逆）：研究（scratchpad scc24.md，static_analysis）找到兩條只在劇情條件下才通的回邊 4e46f0>5090c0（8 個模板）、5049c0>50cf70（+0xe9 一次性旗標、地圖 0xd 陷阱）。登記 `scripts/nodes/ready-cut.tsv` 不穿過（只影響派工順序；d1 只看直接子呼叫）。節點仍原生跑跳點，calc 精確建模已達路徑；進到 D0 公式（506670、43d940）的劇情路徑 calc panic、記缺口（先例 517560）。沒有找到可歸「排除」的切點（三組可行切法的跳點都寫規則狀態），所以不用排除。
  - 拆後順序：L0 5049c0 4e46f0 → L1 {4eb7f0,523c00} → L2 54e950 551eb0 5500c0 54ec80 550c10 54f8c0 54df00 → L3 551ba0 5517c0 → L4 54fa30 54d5c0 54dbe0 → L5 54e610 → L6 施法核心 {4fa4d0,4fb4c0,54ba00,54c930}（12.7KB＋）→ L7 {43d940,4a4cf0}（巨型 switch 12.9KB）→ L8 506670（5.6KB switch）。L6–L8 量大，可能超出 3KB 上限要再拆。
  - 收貨 4e8370（d870ae4；原生非 D4 約 3.6KB，cc 接受、可回頭樁 53a930／4e77b0）、4e46f0（286a905）、48ad20（caf0cf7；48af4e 人數 0 assert 合法資料可達，cc 維持豁免）、5049c0（a0c8baf）。排除：50ff40 50fc20（群放置／連結）、4f01d0（戰鬥移動）、530150（距離門檻啟用）。emu：40efe0 百分比擲骰入亂數樁（a9f0aef）。
  - 2026-10-11：收貨 524540（9cfd7ac）、48ca20（9400cef）、循環群 {523c00,4eb7f0}（d8f0889；cond19 原版空指標當機、penalty≠0∧cond≠7 assert，收窄經 cc 裁定為非法狀態）。D4 324／D0 60。N54f8c0 殘留已拆。
  - 收貨 5244a0（7cd4670）、54f8c0（c6ff271）。新排除：50b2f0 54a760 4be5c0 510cc0 539e70 4e4390 4ecb40 5471d0 519f80 53c630 4efd30 522090（各一 commit）。
  - 跑中（Opus 子 agent）：551eb0 556b10 50b590 552ef0 54e950 54ec80 550c10。就緒未派：54df00 5500c0 5526f0 545e50。
  - **4eae20 擱置**：子 agent 印整段反組譯時回覆被 safety classifier 擋下收回，它停手交回（未寫檔；worktree wiz8-mod-N4eae20 乾淨，設計要點在其交回報告，scratch N4eae20/）。cc 不繞過、不改派規避，記待 MG。
  - 4eae20 停（零檔案）：A／B 組 13 個實體世界／呈現 callee 已登記排除；C 組（547440 51a800 51a840 5471d0 4eb980 4fb0a0 534290 545b20 523f40 535150）未登記 helper 原生跑，但 523f40、5471d0→547570→5477d0 會進 4eb7f0（循環群票）→ 等 {523c00,4eb7f0} 收貨後 SendMessage 續跑（樁 4eb7f0）。
  - 48ca20：510930、48c8e0 排除＋emu 樁 write global／依序取值（7d68775），已續跑。
- `calc party`：收貨 8e089e1（`go run ./tools/calc party -exe build/Wiz8.exe -metric attr:0[,…]`），驗收 `scripts/party.test.sh`（CI go 道）。
- `notes/calc-qa.md`＋`scripts/calc-qa.test.sh`（b69169c）：8 題，Q1–Q2 有答，其餘 `待節點:`；CI 跑 --progress，完成模式要全部換成答案或 `未取得:`。待節點題先試 `calc f <頂層節點> <輸入>` 直接作答；不夠才開組合查詢擴充票（背刺、即死、怪物攻擊、鑑定、扒竊、施法）。
- 已知差異：既有 `AttackRatings` 用寬整數算百分比，原版 32 位回繞（55cc00 節點依原版）。
- 官方攻略本／說明書（MG 2026-10-11 下載）：repo 外 `~/.local/state/wiz8-mod-ref/{hintbook/wiz8-hintbook.pdf sha256 fbd319ae…,manual/wiz8-manual.pdf 6bcfb489…}`。子 agent 抽規則（附頁碼）→ `~/.local/state/wiz8-mod-ref/extract/rules.md`；回來後 cc 對照已驗公式與 R14，重點 Valkyrie vs Lord（MG 實玩與社群共識不符；最終以 calc 為準）。同時在跑：503990、540170、4fa4d0 群、53e250 節點 agent。cc 用量 94%（10-11），重置後若 agent 中斷，看各 worktree 進度檔續派。
- 4fa4d0 群第一次被擋（2026-10-11）：子 agent 讀 4fb4c0 尾段反組譯時回覆被 safety classifier 攔下，停手交回；四成員零碼，worktree wiz8-mod-N4fa4d0 乾淨（HEAD 9ee7b3e），語意筆記在 scratch `N4fa4d0/progress.md`（壓縮 9 次，單 agent 脈絡裝不下）。cc 裁定（可逆）：照 53e250 先例**只再試一次**、拆單：先一張 fixture 票（gen.py 加 spcWorld），再依 54c930 → 4fb4c0 → 4fa4d0 → 54ba00 各派一個 agent，各自讀 progress.md；被讀回的 D4 callee 在 F 側呼叫既有 Go 節點（組合式，不 panic），panic 只給 cov-exempt 的 assert 區塊（54c930:54ca76、54ba00:54bb41/54c057/54c10b/54c1cb、4fa4d0:4fa99d/4fabaf、4fb4c0 的兩個非法輸入分支）。再被擋一次就停、待 MG。等 cc 用量重置後再派。
- 之後：其餘節點 → calc 組合查詢擴充票 → calc-qa 補答 → L28 重算 → grok 稽核落地 → check-formulas 完成模式 exit 0 → CI → push。

## 階段 12（`goals/stage12.md`，2026-10-12 MG /goal：A1＋B1）
- B1 規格 45dfc3b（menu.test 預期紅）→ codex astra worktree → 收貨 5a46f6d（menu.test／speed.test／check-sigs／build／check 綠；四段注入：拿掉夾限 → FAIL menu-rule 上限 5000 → scratch 還原 → PASS）。codex 在 sandbox 用 Unifont 驗收；主 repo 以正式字型重建（dll 460ff1eb）。
- B1 實機：自動 run 20261012-050839-menu-b1 活到 70 s、after_clean=true，但**代理 ddraw 沒載入**（無 log、譯文畫成「A」、F10 前後同圖）。對照 run 20261012-051236-ctrl-old-ddraw 用 10-09 可用的 5db11fb dll 也沒載入 → 環境變了，非 B1 代碼（841a40b）。stage11 定義 1–3 的實機部分未驗。
- 發布包 @5a46f6d（zip f7d9fe76，解在 `%TEMP%\wiz8-mod-release-5a46f6d\wiz8-mod\`）：uninstall／install rc 0；現已安裝（ddraw.dll 460ff1eb、PATCH.002 19feeaf3）；原版三檔不變（18a74ff6／12ce5055／74bbc59a）。

## 待 MG
- ~~⚠ 代理 ddraw 不再被載入~~ 已解（c0c56b8）：根因是 wslhost.exe 的 ImageLoad.PreferSystem32=ON 傳給從 WSL 啟動的行程，DDRAW.DLL 匯入跳過遊戲目錄。wiz8-run 改 WMI 啟動。相容性 layer 不是原因（無 layer 路徑重現仍壞）；Defender 無關。wslhost 何時開始帶這個旗標未取得（10-09 的 run 正常）。HKCU 的 RUNASADMIN layer 是 MG 今天設的，未動。
- **B1 選單試玩（stage11 定義 5）**：環境修好後，進遊戲按 F10，上下選項、左右調 ±25%、Home／End 回 100%、Esc 關；看疊加框（左上）字與手感。log：`%TEMP%\wiz8-mod-ddraw.log` 應有 `menu hotkey_vk=121`、`hooked menu wndproc`、`hooked menu overlay`、`menu open`、改值後 `delay live pct=`、關閉 `menu saved`。
- ~~B1 疊加框畫壞~~ 已解（baa2507，B1b codex）：列讀寫 ABI 是 (buf,y,x0,x1)，舊碼當 (x,y,count) 全寫到 y=16。run 063420 代理載入、F10 開關；MG 2026-10-12：「視窗可以用了，但還沒實際進戰鬥」。
  - **待 MG**：進戰鬥實測三項倍率即時生效（stage11 定義 5）。遊戲目錄留有 `wiz8mod.ini`（run 期間寫回 575/375/300，非原版檔；要移除給 MG `Move-Item`）。
- **B1 stage11 完成定義 1–4 證據（2026-10-12 盤點）**：
  1. 純函數四段注入：5a46f6d 訊息（拿掉 menu_key 夾限 → FAIL menu-rule → scratch 還原 → PASS）；`menu.h` 項目表（key、def、label、min、max、step、apply）。
  2. 驗簽：check-sigs 綠（B1b 驗收 55 條）；run 062233／063420 sig-mismatch 0。
  3. 自動 run：062233（@002edc6）menu hook 安裝列 2、sig-mismatch 0、活到時限、after_clean true。063420（@baa2507，同 hook、只改列讀寫參數）同上，但 after_clean false＝MG 在 run 中調倍率寫出 `wiz8mod.ini`（log `menu saved`×2，情境本身只送 F10）。之後重跑被 wiz8-run 以該 ini 拒跑；刪 ini 歸 MG。
     - **最終版已補（2026-10-12）**：run 072352 @e4845de：menu hook 安裝列、sig-mismatch 0、活到時限、after_clean true；放入 ddraw.dll 6766dc61＝hook/ddraw/out＝發布包。072100 不乾淨＝MG 未看到「別動視窗」而按 F10 調倍率（已入檔）。
     - uninstall.bat 現在一併移除 wiz8mod.ini（2ed8cbd、e4845de）。現已裝發布包 @e4845de（`%TEMP%\wiz8-mod-release-e4845de\wiz8-mod\`），MG 的 ini（575/300/200）已放回，備份 `~/wiz8mod.ini.mg-0722.bak`；原版三檔 18a74ff6／12ce5055／74bbc59a。
  4. 發布包 @3a14ddc（zip 7d405c80，解在 `%TEMP%\wiz8-mod-release-3a14ddc\wiz8-mod\`）：cmd 跑 install／uninstall／install 皆 rc 0；現已安裝 ddraw.dll 1e0483ad（＝hook/ddraw/out）、PATCH.002 19feeaf3；原版三檔前後 18a74ff6／12ce5055／74bbc59a。release 腳本與 bat 自 5a46f6d 未改。
- **A1 完成（43b790e 起）**：`calc fight` 可查；L28 前十 calc 重算入 `notes/research/ranking-history.md:104`「更正」段（全低 11–21%，主因命中）。
  - **待 MG**：過目 A1 排名，決定 A2 是否開工（stage12 順序）。未建模：Lightning Strikes、combatSkill 0x11、無副手 item7 代值（`notes/calc.md:15`）。
- **新需求（MG 2026-10-12）**：「之後追加一個『隊伍移動速度增加』的需求」→ 選單項目表加一列（B 路線，未排程、未開票）。MG 同時回報：「實測有視窗出來可以用」。
- ~~4eae20 classifier 擋~~ MG 2026-10-11 裁 (a)：「(a) 續做，只看小片段」→ 子 agent 續跑，只讀小段位址範圍、不整段印反組譯。
- 4eae20 第二次被擋（續跑那輪只做了 git merge/status）→ MG 2026-10-11 裁 (1)：「開全新的 agent，grok 還有額度但我不確定它能做到什麼地步」→ 已開全新 opus agent（小片段讀法，被擋即停）。再被擋就擱置，不改派 grok（grok 只稽核）。
- 53e250 子 agent 一次回應被安全分類器中途擋（2026-10-11；被擋內容 agent 未重做、cc 不知內容）；另壓縮約 22 次無產物。照 4eae20 先例 (1) 開全新 agent，交接用 scratch N53e250/sum2878.txt；未重試被擋內容。若新 agent 再被擋，停下等 MG。
- 506670 子 agent 讀子函式本體時被安全分類器擋（2026-10-11；未寫檔，被擋內容未重做）。照 4eae20 先例 (1) 開全新 opus agent（小片段讀法，被擋即停），交接 scratch N506670/、p-506670-cont.txt。再被擋就擱置等 MG；506670 卡著 5061a0 下游約 12 個 D0（50ca80 502860 4de7d0 56c6d0 51dde0 → 502d00 502010 4e9b20 4ea310 …）。
- 後續票（cc 記）：551eb0 F 夾具收怪物目標 charIdx=-1（551ba0 的 F 220 列 panic）；F-only case 待補：556b10、54df00、5517c0、50b590、551ba0、550c10、54dbe0、543270、4eae20（嚴格亂數 748 列 extra RNG）、544d30、4ebfe0、545090、5560a0（F 餵 551ba0 目標的 -1 欄改 0，kind1 不讀 +8／kind3 不讀 +4 待核）、5526f0（後槽讀 stub 掉的 8 個 D4 callee 寫入，F 590/1459 panic）。
- 後續票（5500c0 發現）：590950 後綴是 wcscat 到另一個未初始化的兩字元區域（＝58aad0 arg4），長度判斷只看格式文字；n_551eb0.go 的 590950 模型把後綴接在格式文字後，格式文字只有單一 ' 時會差。補 551eb0 case 走到該情形＋修模型。
- 後續票：4c6c30 已登記排除（c359052）。550c10、54fa30（共用 srdStubs）、545090（52bb60 原生子樹）的 case 仍以 +0x1dc 位元 0x400 固定設定讓 4c6c30 不動作 → 改為位址樁並放開該位元（同 54e950 修法）。
- 後續票（法術 SCC c9e44d9）：(a) recast 深度夾具上限 3（408860 記錄 3、4fac40 NFA 4），正常遊玩深度受 SP/cost 限制可 >3，calc 超過即 panic → 擴大記錄上限或改 n 記錄；(b) 4fa5cc 豁免前提「法術 +0x12b 模式只有 0..3」未以實資料驗：SpellTables.dbs（載入 0x4acc..，stride 0x1bf，檔頭 150／1）非連續原始記錄，需解析格式後統計 +0x12b；(c) F 差異：4fa4d0 F≠Emu 377/596（school selection 輸入域 81 panic）、54ba00 91/762、4fb4c0 155/979，併入 F-only。
- 後續票（547cb0 49494d2）：夾具怪物筆數上限 8（超過 calc panic），正常遊玩一群可能 >8 → calc party／calc-qa 會走到時放寬記錄上限；原版 P∈(-N,0) 無限迴圈，calc panic（合法級數不到）。
- 後續票（540170 d9b4171）：(a) 544010 既有 D4 把 spellIndex（M+0x2e5）限 ≤2，540170 實傳 mode 0..8 → 544010 case／calc 輸入域缺陷（正常遊玩值會 panic），開修正票；(b) 540170 cov-exempt 54037e（配置失敗）、540844（M 已解參考後判零）calc 無對應分支、未 panic（違「每個豁免塊 calc panic」字面；calc 模型中條件無從成立）→ 補 panic 或登記理由；(c) 540170 F panic 1204/1526（986 列目標種類非 1/3＝原版亦斷言）。
- 後續票（5ba740 6f8ca98）：F 端 iicDsrItem＝0x20007000 寫死 005a6340.json 物件位移，該 case 搬物件 F 會悄悄餵錯，無自動檢測 → 改由 case 讀或加斷言。
- 後續票（5e3800 4b39e48）：類別 8 分支目標記錄 +4..+0xc、+0x1c 原版未初始化，calc 照模擬器 0 輸出（非遊戲值，檔頭註明）。
- 43d940 循環群票（771f96a）作廢：子 agent 靜態分析＋cc 抽查跳表，43d940＝觸發器總派發、4a4cf0＝投射物飛行，改登記排除（38701ea、b4cd140）。未涵蓋：陷阱全隊傷害的骰參數負值預設 1／6／2（43d940 內），calc 查陷阱傷害時要自己套；需要時開小公式節點。
- 修正票 F1（544010 spellIndex 域 0..8）、F2（540170 豁免塊 panic）派 cc 子 agent，worktree ../wiz8-mod-F1（f67d68b）。
- F1 7f979b6、F2 ba105d8 收貨；5e3ab0 276b71a、51ee70 640b7aa 收貨。
- 後續（51ee70）：共用 spcWorld.cast（n_4fb4c0.go）不建模 517a90 寫 0x689fd0 訊息文字，51ee70 iface 標無關；要驗文字輸出就在 4fb4c0 補。F panic 69 列（35 列多叫亂數＝比對產物、34 列 callee 夾具上限）。
- calc-qa 6ebba13：Q4 每回合未答（需 Samurai＋村正 build 情境，且 build 不含 Kill 抗性）；Q5 命中未答（缺架勢／疲勞／目標四維／AC10 defhit 的情境輸入）；Q8 用 4ff7f0 答。F3（551ba0 符號）、F4（4fa4d0 人物施法）派出 108084b，worktree ../wiz8-mod-F3。
- 排除登記 506670 一批（11 列＋4f6cf0 改類）、589a80 一批（3 列＋5a0110 改類）；4ddf40 改排除 fbb40a8（夾到 R(100)>35 擊殺）。
- F-only 測試（模擬器不樁子函式、整鏈原生跑，比 calc F；驗接線）：MG 2026-10-11「照你建議的，D4 完再說」→ 全部公式 D4 後，只補 calc party／calc-qa 會走到的呼叫鏈（近戰、法術、即死、成長），不全面鋪。待補清單加 550c10、54dbe0。
- ~~**4f7610 手持物丟到世界（2026-10-10，擋 5223a0 character_equipment_removal）**~~ **已決：MG 2026-10-10 授權排除由 cc 判斷（079572b），cc 登記排除**。原文：5223a0 卸下裝備後，物品 +0x41 bit1=0 時呼叫 4f7610(0)（build/wiz8.asm:333070；呼叫點 build/wiz8.asm:393659、393740 附近），以鏡頭位置三角函數算前方落點，經 434f20／433820／451800 查地形，成功則 4f6b90 把手持物 0x6874cb 建成世界物件，失敗播音效 408860。回傳不被讀。與 4f93d0 同屬世界幾何放置，但它決定「手持物是否離開隊伍」，不只是位置，cc 不自行類推 4f93d0 的核准。選一：**准**＝登記排除，5223a0 用位址樁記參數＋次數；**不准**＝5223a0 寫合格「未取得」。522640／5061a0／506280（劇情旗標）cc 判定原生跑，不需裁決。worktree 保留。
- ~~**4f93d0 地形貼合（2026-10-10，擋 4f88f0 item_pool_selection）**~~ **已決：MG 2026-10-10「可排除」，登記 667c2a6**。原文：4f88f0 每個被接受的槽都經 4f6b90（物品物件建構子）呼叫 4f93d0，回傳被忽略，只設新物品物件的世界位置與旗標（往下 433820 地形／碰撞查詢，需真實地形資料，emu 無法建合法夾具：世界物件為零在 0x43b492 崩、填指標則 _ftol 溢位）。它不是訊息／音效／UI（條件①的列舉），也不是遠程碰撞，cc 不自行擴解釋。選一：**准**＝登記排除（世界幾何放置，與 434f20 同類），4f88f0 用位址樁記參數＋次數；**不准**＝4f88f0 寫合格「未取得」（影響：物品池抽選；calc-qa 無題直接依賴）。worktree 保留。
- ~~43ccf0 IO 樁、呈現類 helper 歸排除~~ MG 2026-10-09 選單裁決：兩項都選「准（推薦）」（`decisions/2026-10-09-io-and-presentation-stubs.md`，規格 bcbcb1d，emu 404ea0 stream 樁 f00561e，三組注入 FAIL→PASS）。43ccf0 已重派（cc 子 agent）。下方 43ccf0、「一次裁」兩條由此結案。
- ~~組合裁決~~ MG 2026-10-09 裁**甲**（`decisions/2026-10-09-compositional-stubs.md`，規格 707a172，機制 9475d06：emu 准樁 D4 公式、`ret:{at}`、`Node.Emu`）。已重派 520d10 群、533320 群（cc 子 agent，NEW RULE 附在 `$S/add-stub.txt`）；547a50 群依賴這兩群，D4 後用 scc-ticket.sh 開票。Node.Emu 的會紅測試待第一個節點收貨時補。第四輪兩群又停手（零檔案）：派工單「子樹數 KB 就停」用靜態閉包判，但本 exe 靜態呼叫圖是約 490KB 單一強連通塊，永遠超標＝派工單缺陷（§4.8）。第五輪（2026-10-09）改為只看 emu 實際跑到的程式碼、准把內嵌在別節點檔的 helper 抽成函式（先例 537220 原生跑 519180 無樁），補述在 `$S/fix-d.txt`。43ccf0（404ea0 IO 樁）MG 未回，仍待。
- **⚠ 組合驗證的結構瓶頸（2026-10-09，擋 149 節點）**：520d10 群第三輪停手（零檔案）。上層節點原生跑 13 個已 D4 公式 callee（5201b0、4ee220、50e5c0、5458a0、4e79a0、536f60、4ee9d0、4f0480、536a20、53c720、51d180、51d3b0、5194e0），但各 callee 的 case 世界互相衝突（0x683f78、0x685178 指向不同物件版面等），沒有統一世界；輸出不縮窄＝要含 50e5c0／51d3b0 數百欄寫入。現行規則「公式類不能樁」下，上層節點等於要重建整個遊戲記憶體模型。選一：
  - 同型：533320 群第二輪停手（零檔案）：循環群實為 5 成員（加 532550／534dd0），helper 閉包新寫量 >5KB；519f80／51a2f0／51a510 只內嵌在 n_53fea0／n_544010／n_519c00，未抽成可呼叫函式；4c5a80／4c4660 虛擬呼叫目標未取得。甲若准，已 D4 的部分可樁，剩餘量大減；不准則要 B→C→D helper 票＋改既有檔抽函式（白名單外）。worktree 已拆、票留。
  - 甲（cc 建議）**組合式驗證**：已 D4 的公式 callee 在上層 case 可位址樁（記參數＋次數、回傳值當輸入），上層 calc 對樁記錄與自身寫入負責；callee 本身已由自己的 D4 證過。代價：callee 對共享記憶體的寫入如何被上層讀回不再端到端驗，這是放寬現行 D4 定義，所以交 MG。
  - 乙 先開「統一世界 fixture」基礎建設票再重派：量大（數千行版面描述＋約 25 個未登記 helper 轉寫），時程不可估。
  - 丙 上層節點寫合格「未取得」：連帶 149 節點，等於階段 10 定義 2 只完成底層。
  不裁＝520d10 群以下停在 D0；底層與 533320 群等仍照常推。
- **共同根因（2026-10-09，5 節點同型；另 533320 循環群 3 節點同型：被呼叫端未登記不能樁，見階段 10；520d10 循環群 4 節點、501660 同型）**：【522b80 D4（0b0aa8f）、4e6ce0 D4（8fd8e3e；隊伍數 0 讀未初始化堆疊以 0 模擬、math.Pow 對 x87 80 位 pow 未驗）；3e451d2：4c7f10／58aad0 排除，8abf09f＋60d9e9a：533320 群閉包 41 個 helper 分類（37 非公式＝查表／謂詞／目標搜尋，含 519f80 叢集與 SCC 成員 532550／534dd0——登記公式會讓 10+ 個已 D4 節點轉紅；4 排除），533320 群第二輪已派。0fb959a：461960／5308c0／48ad20 寫合格未取得（可逆；ready 計算顯示各只擋 0–1 節點，520d10 群擋 149）。48ad20 二度退件：①40efe0（51B 百分比機率，直呼 CRT rand，emu 只攔 40efa0）要 emu 加攔截；②50fc20 子樹實為 >3.5KB 生成邏輯（547570／510590／5100b0…）＋506280→5080f0 2.5KB 遞迴；worktree 已拆、票留，與 5308c0／461960 同歸「大子樹」一併裁。a5f6040：48fc10／55f260 登記排除、52e5c0 非公式→排除，4e6ce0、522b80 已重派（位址樁）；8d30deb：595600／593330 排除，520d10 群第三輪。cc 已依 §5.2 自定 A（可逆，MG 可推翻）：未登記 helper 由 cc 入 `notes/re/formula-graph.d/b90-helpers.tsv`、開節點票；第一張 537220（行動目標資格，501660 等 6 處共用）已派。登記列不先 commit——登記後依賴它的 D4 節點（501400、5012e0）progress 模式即紅，列隨節點收貨一起入（take 會 add notes/re）。53c720 收貨（cc60ea7）時 537220 列一併入庫→progress 模式 501400／5012e0 紅，到 537220 D4 前不 push；537220 已 D4（9f9f757，描述槽返工＋注入四段），progress 回綠。已重派 501660（ctx=4 薄組合）與 520d10 群（第二輪）。5194e0 calc 距離缺陷由 53c720 差分抓到、a3cd614 修。】43ccf0（404ea0 存檔讀取）、4e6ce0（48fc10 訊息、52e5c0 語音）、48ad20（4c7f10 怪物音效載入、58aad0 MonGen 日誌，另有 50fc20／50f1a0 生成子樹數 KB）、522b80（55f260 拒絕訊息框，依 0x68ec78＝6/7 進 UI 子樹數 KB）都卡在「引擎流程 helper 未登記或登記非公式→不准位址樁」。一次裁：准不准 cc 把 I/O、訊息、音效、日誌類 helper 逐一登記為「排除」（登記一 commit、附理由行），之後用位址樁記參數＋次數？不准＝這些節點寫合格「未取得」。5308c0／461960 不同型（路徑檢查回傳值決定流程，是規則輸入），另裁。
- **4e6ce0 戰鬥難度評等（2026-10-09）**：【已結案：同上裁決，4e6ce0 D4】子 agent 已看清公式（怪強度×血%累加→556000 威脅 1..50→隊伍等級均值比→狀態 0/1/2），卡在兩個副作用呼叫：48fc10（訊息輸出，未登記，~8KB 文字／音訊子樹）、52e5c0（語音事件，登記「非公式」）。要不要准把兩者改登記「排除」（或補樁）？准＝位址樁記參數＋次數，4e6ce0 可到 D4；不准＝合格「未取得」。cc 未改登記簿（同 43ccf0：改守衛邊界交 MG）。worktree 已拆、票留著。
- **43ccf0 存檔串流（2026-10-09）**：【已結案：MG 2026-10-09 准，43ccf0 D4 71f3f74】要不要准 emu 位址樁換 404ea0（存檔串流讀取，非公式候選、登記簿未登記）？准＝emu.py 加 `IO_STUBS={0x404ea0}` 例外＋`stream` 欄位（case 位元組依序餵給 buf），43ccf0 可到 D4；不准＝43ccf0 照 check-formulas 寫合格「未取得」（影響：鎖碼初值查詢）。cc 自己加例外被 auto mode 擋，未改。
- **試玩戰鬥加速（stage10 A4，2026-10-09 已裝 @5db11fb）**：Steam 目錄已用 `%TEMP%\wiz8-mod-release-5db11fb\wiz8-mod\install.bat` 安裝（ddraw.dll ef34efe3、PATCH.002 19feeaf3）。請實玩一場戰鬥，回答：
  1. 攻擊／施法後的停頓有沒有變短、會不會太短（事件停頓）；
  2. 怪物攻擊／施法動畫有沒有變快、看不看得清（動畫倍率）；
  3. 音效有沒有被切斷或對不上動畫。
  - 調整：遊戲目錄 `wiz8mod.ini` 的 `[combat]` 段，`monster_move_pct=400`（怪物走位）、`event_speed_pct=300`（事件停頓：1500→500、800→266、600→200 ms）、`anim_speed_pct=200`（戰鬥中怪物攻擊／施法序列 9–0x11、0x19 動畫）。100＝原版。不寫 ini 就用這些預設。
  - **MG 已實玩（2026-10-11 17:53 起）**：log 所有 hook 裝上、無 sig-mismatch；`speed combat base=5000 eff=20000`（17:57:25）；遊戲跑到 18:58 無崩潰跡象。MG 回答（2026-10-11）：「停頓有變短，動畫看得清楚，音效沒問題」→ 三問通過，加速器 W 試玩結案，預設值維持。
  - **MG 第二次實玩（2026-10-12 04:08 起）**：log pid 21748 三個 speed 列＋`hooked anim 0x004c2b5d`＋`speed combat base=5000 eff=20000`（04:08:40），sig-mismatch 0。MG：「戰鬥差不多，不會看不到訊息，也比以前加速很多」→ 維持預設值。
  - 試玩後 cc 讀 `%TEMP%\wiz8-mod-ddraw.log`（`speed combat` 列）。已知限制：動畫倍率在序列中途切換不鎖邊界；近戰半格輪詢可能讓傷害晚一點觸發（`notes/re/speed.md` W 段）。
- **2026-10-09 現況（覆蓋下一條的安裝狀態）**：MG 為了用 save editor 在移除中文化。
  - **已重新安裝（2026-10-09，MG 指示）**：cc 用 `%TEMP%\wiz8-mod-release-13c29f1\wiz8-mod\install.bat`（cmd.exe）安裝，輸出「Installed」。ddraw.dll cdd5d609（＝發布包）、PATCH.002 19feeaf3；原版三檔前後相同（Wiz8.exe 18a74ff6、DATA.SLF 12ce5055、PATCH.001 74bbc59a）。13c29f1..9aef4a8 只改筆記，發布內容未變。下面幾條是移除期間的紀錄。
  - Steam 目錄 `ddraw.dll` 已不在（目錄 mtime 2026-10-08 23:36:15，不是 uninstall.bat 刪的：bash 跑 .bat 第 6 行語法錯誤即停、cmd 跑回「No ddraw.dll」；誰刪的未取得，待問 MG）。
  - `Patches\PATCH.002`（06:25:16 版，本 mod 產）：cc 刪除被 auto mode 擋，交 MG 用 PowerShell `Move-Item` 移到 `$HOME\PATCH.002.zh-0625`；移走與否未確認。
  - 新發布包：HEAD 13c29f1，解在 `%TEMP%\wiz8-mod-release-13c29f1\wiz8-mod\`。裝回中文用這份 `install.bat`（PowerShell：`cd $env:TEMP\wiz8-mod-release-13c29f1\wiz8-mod; cmd /c install.bat`）。舊包 `wiz8-mod-install`、`-0f17241` 已刪（2026-10-09 MG 指示）。
  - 跨版本缺陷已修（7b79f1e）：uninstall 認任何版本的 wiz8-mod dll（findstr 識別字串）、install 遇舊版先自動移除；實測表 `notes/zh-patch-log.md:257` 末段。
  - Wizardry8Editor（R10 推薦）在中文化下全畫面亂碼：它用遊戲 STI 字型＋原 252 字字表畫字，標籤也從 PATCH.002 讀到中文 → 查不到畫成怪字（MG 截圖 2026-10-09）。存檔不受影響。用法：移除 PATCH.002（＋ddraw.dll）再開 editor，改完裝回。可考慮寫進 INSTALL.txt 已知限制。
- **Steam 目錄已用發布包安裝（2026-10-08 更新，release.sh @2c2c229，zip sha256 9f26328f…）**：舊版（@94fb672）用舊包 uninstall 移除 rc 0 → 新包 install.bat rc 0；ddraw.dll c4e1aff1（＝hook/ddraw/out，含稽核修正 9078dd6）、PATCH.002 4d42084e（譯文未變，同前）；原版三檔前後 sha256 相同（Wiz8.exe 18a74ff6、DATA.SLF 12ce5055、PATCH.001 74bbc59a）。MG 自行慢慢實測。
  - 在 uninstall.bat 之前，wiz8-run 會拒跑（「已存在於遊戲目錄，拒絕覆蓋」）。要跑自動 run 先 `cmd.exe /c uninstall.bat`（用 `%TEMP%\wiz8-mod-release-13c29f1\wiz8-mod\`；任何版本都能移除）。
  - MG 手玩會改存檔（MG 2026-10-08：「我存檔也沒差吧，你可以 restore 回去」）。整個 Saves 目錄已備份到 `~/.local/state/wiz8-mod-savebak/Saves.20261008/`（51 檔，對 saves.sha256 全 OK）。
    還原：`rsync -a --delete ~/.local/state/wiz8-mod-savebak/Saves.20261008/ "$G/Saves/"`，再 `(cd "$G/Saves" && sha256sum -c ~/.local/state/wiz8-mod-savebak/saves.sha256)` 0 non-OK。MG 手測結束、或下一次自動 run 前做。
- 「稍微比較破碎」（階段 4 目視）已由 5.4 解掉（51e38e0 Fusion Pixel 12px，MG 實機認可）。按鈕字偏右（x=396，英文 370）：MG 2026-10-08「其他我覺得沒問題」→ 不修。
- 鬆散 `Data/` 檔覆蓋實測：已跑（b4f60c9，`notes/zh-patch-log.md:127`）。
- 3.3 的 B 入口（0x004077d0）在組隊畫面沒被呼叫，用途未取得（不擋驗收）。

## 下一步建議
- 2026-10-08 清掉已完成的過期項（5.1 抽取、字形候選、翻譯管線、文字來源盤點、換行、字表：皆已做，見上方各段）。譯文覆蓋：物品名 28 筆未譯＝09cf378 刻意留英文（關鍵字 26＋編號）；怪物 45、法術 24 筆＝佔位詞（Temp、Death01、<reserved>）。
- 相容支線 C（PrintScreen）：完成（run 063150），代理存 BMP 到 `%USERPROFILE%\Pictures\Wiz8\`。前提是 Win11 PrintScreen 開剪取工具的設定關閉（MG 已關）。細節與上限：`notes/re/shot.md`。MG 之後可直接拍圖給 cc。
- 開場動畫字幕：**不做**（MG 2026-10-08）。背景：Wizardry8.bik、Virgin.bik 無燒入字幕、無字幕檔，要做得聽寫＋翻譯＋播片疊字 hook。
- 戰鬥加速：中文化收尾後開研究票（muse 查：戰鬥動畫／訊息等待計時在 exe 何處、社群既有做法），再決定做不做；做法同現有簽章比對 hook。影音不做：畫面交給 nGlide／dgVoodoo 等包裝層，只需確保我們的 ddraw.dll 可與之串接；音效無更好來源。MG 2026-10-08：「中文化收尾後再開研究票，我覺得加個戰鬥加速就差不多了」
- EOB 中文版（MG 的智冠版是安裝後壓縮燒進光碟的檔案）：MG 說先處理好 wiz8，之後另開 repo。
- 翻譯範圍（MG 2026-10-07 實機：「關鍵字系統無法輸入中文，所以這表示有很多東西不用翻譯」）：關鍵字清單（English_Keywords.txt）與對白中須照打的關鍵字維持英文；對白本身照翻，關鍵字寫「中文（English）」。translated_Keywords.txt 機制見 `notes/re/misc.md:47`（7.3）。
- 試玩錄影：MG 用 Win+Shift+S 錄整場（PrintScreen 會崩），影格抽 5 s 一張交子 agent；text log 第 9 欄毫秒，log_ms ≈ 影片秒 +13～15（run 044004／045348，報告在各 state 目錄 frames-report.md）。
- 戰鬥與成長計算工具（MG 2026-10-08）：先由票 P（近戰）、P2（法術、抗性、遠程、特效）、P3（創角與升級）、P4（職業與種族天生能力、物理減傷；MG：「矮人武僧也是一個玩法，這部分應該也能算」）從 exe 找公式，四張都收貨後開計算工具票（codex，附 golden 測試）。工具要算：各種族×職業的可分配點數、各屬性到 100 要幾級（進階技能）、到 28 級累計的屬性點與技能點（MG：「遊戲玩到末期大概也很難到等級 28，即便是 expert 難度」）、每回合攻擊次數與期望傷害、法師與戰士的輸出對比（MG 問「巫術為什麼後期是近戰世界」）。
  - codex 沙箱看不到 repo 外路徑：RE 票的材料要先 `cp build/wiz8.asm`＋`~/.local/state/wiz8-mod-ghidra/decomp` 進 worktree 的 `build/`。worktree：`../wiz8-mod-P`、`-P2`（已收貨 9ea7772）、`-P3`、`-P4`。

## 額度用完
- agy：週額度，約 2026-10-13 重置。研究照派 muse。
- codex：2026-10-14 14:02 重置。muse：2026-10-12T00:00Z 重置。期間節點票由 cc 子 agent 做。
- cc：週額度 2026-10-11 晚已用 82%（MG：「今晚就做到額度用完吧，之後重置了再繼續」），2026-10-12 09:59 重置；5 小時窗 10-12 01:19 重置。

## 接續點（2026-10-12，HEAD bda44b0 已推）
- 階段 10 完成定義 2 達成：`bash scripts/check-formulas.sh` → `ok check-formulas：719 候選、939 列（final 模式）`，D4 381（4e8ea0 收貨 bda44b0 後跑）。
- 定義 5：CI exit 0（證據 `~/.local/state/wiz8-mod-ci/bda44b0a0e049e32866cfb408860db54b7c42387`），`ALLOW_TEST_SHRINK=1 git push origin main` 8bba541..bda44b0。
- 本段排除登記：56ee20／587a30（非公式→排除）、58a860、5b3100、596cf0、592e60（565740 UI 刷新，呼叫端不讀 eax）；4bf970（動畫總影格，回傳只當等待閘，結果當 case 輸入）＋vtable 入口 5ed20c（8c77801）。
- 階段 10 完成定義對照（2026-10-12，HEAD 8ee4895 起算，本機跑）：
  1. A1 fc681fd、A2 d571334＋W c2c103d、A3 發布 5db11fb＋自動 run 20261009-042703-cjk-zh（after_clean=true，三檔雜湊不變，見階段 9／10 段）；「待 MG」有試玩條目。
  2. check-formulas exit 0（見上）。
  3. `calc f`：公式類 381 個逐一 `calc f <8 位址>`，0 個「未知節點」；`bash scripts/calc-qa.test.sh`（final 模式）→ `ok calc-qa（16 題，待節點 0）`；`bash scripts/party.test.sh` → `ok party`。CI 副本無原版檔，這兩支印 SKIP，證據只在本機。Q4 每回合值、Q5 命中缺的是情境輸入（Samurai＋村正建檔、怪物架勢／目標四維），不是節點；題目內已寫明。
  4. grok 稽核 zoWN 結案：抓對 5、誤報 0，F1–F5 全轉成修正與測試（01b8d04、da65cfa、bb92e90、7e0b80c、2af1ef3；inbox 0fe6e12）。
  5. CI exit 0 並 push（bda44b0、8ee4895）。
- 下一步：「待 MG」各後續票與下方 F 模式待追；F-only 測試（MG：「D4 完再說」）現在可開，只補 calc party／calc-qa 會走到的路徑。

## F 模式待追（2026-10-11～12 收貨時 agent 回報；節點驗收只比 Emu，F 只有 scratch 統計）
- 502d00：F 651 列 panic 514，皆 callee fixture 上限（怪物法術目標索引、群組清單長度等）[推測：未逐列歸因]。
- 51dde0：F 1149 列 panic 167，158 來自 51d610 itemIndex 上限 0..3（特殊物品 0x29f）。
- 4e9b20：F 未模型化 5044d0 的體力改動、52b1c0 內 523330 對 b01 的改動。
- 4e9f90：F 361 列 panic 53（子節點值≠樁值時 517fb0 多擲骰）；48f9e0 音樂原生跑、前提 0x65ba7e=0（音樂開時消耗 RNG，calc 未建模）。
- 4e9490：F 626 列 panic 108（683fce≠0 且 918≠1，遊戲不出現）；F 呼叫 4e9f90 不接 4e9b20 效果（a56），改 F-only df_* 輸入。
- 4e8ea0：F 943 列 panic 17（extra RNG call）、F≠Emu 37（選人鏈 4e79a0／524a10→565740、547310、4e8370 次數）。
- 共通：F≠Emu 比例高是結構性（Emu 回放 case 輸入，F 跑真函式）；F 正確性無機械比對。`calc f` 走到上列路徑可能 panic 或偏離原版。

## 階段 13 隊伍移動速度（`goals/stage13.md`，2026-10-12 MG /goal）
- 13.1：研究改派 codex（muse／agy 額度用完）→ `notes/re/party-move.md`（dd5d5e9）。只動隊伍的點：0x0041f78f 隊伍 dt 副本 P+0x14；cc 抽查 exe bytes 與 wiz8.asm:41983、41987 相符。未觸停損。
- 13.2–13.3：P13 codex（d2c7d83）：5-byte jmp 到 naked stub，非戰鬥時 P+0x14 乘 `party_dt` 係數；選單第四列 `party_move_pct`（預設 100）。四段注入：speed.h 拿掉戰鬥判斷 → FAIL party-rule ×2 → 副本還原 → PASS。stub 內的戰鬥判斷是 asm，純函數測不到。
  - 規格缺陷（cc）：menu.test 寫死第 3 列，codex 指出後 cc 改測最後一列（獨立規格 commit）。
- 13.4：run 20261012-075451-party-move：`speed party_move_pct=200`、`hooked party 0x0041f78f`、`menu saved`、無 sig-mismatch、活到 75 s、after_clean=true；選單截圖四列、隊伍移動 225%（子 agent 判讀 screen.png）。
  - **缺口**：run 停在「召集你的隊伍」畫面，沒進探索，隊伍移動路徑**未實際跑到**（只證明安裝、選單、寫回）。window.png 全黑（擷取路徑壞，screen.png 正常）。
- 13.5：發布包 @73b5d86（zip 68b37afc，解在 `%TEMP%\wiz8-mod-release-73b5d86\wiz8-mod\`）：cmd install→uninstall→install 全 rc 0；ddraw.dll 10ad9242（＝包內）；原版三檔前後相同（18a74ff6／12ce5055／74bbc59a）。MG 的 ini（575/300/200）已放回，沒有 party 列＝100。
- **待 MG**：F10 調「隊伍移動」後實際走路：速度是否變快、轉向（本票未加速）要不要也快、垂直物理／負重頻率連帶變化可否接受、大倍率會不會穿牆／卡牆角／漏觸發陷阱與事件。log 在 `%TEMP%\wiz8-mod-ddraw.log`。
- **MG 試玩（2026-10-12）**：隊伍移動 400%（怪物 600／事件 300／動畫 200）：「我覺得還算正常」。移動路徑實機已跑到。轉向要不要加速、穿牆／漏觸發 MG 未特別回報，仍開著。
- **MG 裁定（2026-10-12）**：「加速的部分我判定結案」。階段 13（隊伍移動）結案；轉向加速、穿牆／漏觸發不再追，MG 回報再開。

## 階段 14 修補票（`goals/stage14.md`，2026-10-12 MG「先全面把修補票完成」）
- 規格 862dd56（calc-reach 148 檔、G1–G6）、73be308（驗收缺檔印 FAIL）、9365451（G2b、G6b）。後續票 7＝dd74262（cc）、8＝檔頭已載、6a/6b＝F1/F2。
- 收貨：G1 912cf61（4c6c30 樁，疑慮：53e250 夾具 srdHit 仍有 0x400 防護，白名單外）、G2 314bbd8、G4 cf95712（150 筆 +0x12b 全 0..3）、G5 98315a3（上限 8→9，原版上限〔未取得〕＝清單動態擴容）、G6 a0c9e89（原生比對基建；54df00 試點紅：523330 F 缺 HP／體力重建、52e690 記錄 8 次不夠）。
- **回歸**：G2 給 551eb0 加 3 輸入欄，父節點 543270／54dbe0／551ba0／5560a0 arity 跟著變 → TestEmu 紅、calc-qa Q4 紅（main 現況紅，未 push）。根因：G2 白名單漏父節點＋worktree 沒複製 build/emu（父節點沒被測）。G2b 修；之後 worktree 一律複製 build/emu。
- 進行中：G3（astra，../wiz8-mod-G3）、G2b（sol）、G6b（astra）。之後：G7（交集其餘節點原生 diff=0 或逐節點〔未取得〕）→ check-formulas／calc-qa／party → CI → push。
- 磁碟 / 剩 7G（251G 用 232G）：本 session 清了舊 scratch 與 go build cache。
- 2026-10-12 收尾：G2b／G3／G6b 收貨並於 main 重驗全綠，24be112 已推；G7 移出（stage15 不在範圍）。磁碟清後剩 40G。

## 階段 15 最佳化試算 v0（`goals/stage15.md`，2026-10-12 MG /goal）
- H1 PlanSkills c443266、H2 calc rank 528c224（codex sol）；cc 修 3aa9a62（hb 傷害% 符號延伸，class 7 L1 −5 撞 5420b0 域）、3264d3b（H1／H2 測試缺檔改 SKIP，CI 副本無原版）。
- H3 結果：`notes/research/ranking-history.md:610`（範本 `notes/calc-qa/rank-h3-{564,411}.json`，各 3960 組約 5 分鐘）。L28 兩怪皆 Rogue 第一（Felpurr／Faerie）；Pareto 全 Rogue（411 多 Dracon Fighter）。
- 後續候選：Pareto 同族多列是浮點噪音（≈1e-11），可加容差或四捨五入；法術 A2、生存 A3、六人隊 A5；Train 情境（技能練到 100）再排一次。
- 磁碟清理：MG 2026-10-12「應該不用擔心了」，剩 40G，結案；舊 codex-audit run cache 未刪。

## 階段 16 最佳化試算 v1（`goals/stage16.md`，2026-10-12 MG /goal：I3→I6→I4 中途不等 MG）
- I0–I3、I5 收貨見 git log（f305e1e … beb3269）。I6 收貨 1a12168。2026-10-12 重做 main 四段：無條件 SPD 125 注入原測試仍綠（缺「不開 Haste 時 SPD 原樣」斷言，原記錄的 FAIL 不成立），cc 補斷言後 PASS → FAIL「SPD 125 without Haste want 97」→ 副本還原 cmp 相同 → PASS。
- I3 main 四段注入（2026-10-12 補做，收貨訊息 13978ab 未記）：PASS → RankModeDistribution 改均分 1/k → FAIL「B15 k=2 mode=0 got 0.5 want 0.5556」→ 副本還原 cmp 相同 → PASS。第一次注入留 j 未用＝編譯紅，不算，重注。I6 派 codex sol：帳本 da3444c（r0 dispatch＋usage），worktree 收貨後已拆。
- cc 修 781152d：fightExpect 依 goroutine 完成順序加總，同輸入重跑末位不同；改前兩層分支分桶、排序後加總（TestFightDeterministic，舊 fight.go 注入 FAIL×3）。
- I4 結果：`notes/research/ranking-history.md:682`；重跑 `bash scripts/i4-rank.sh`（build/i4/）。副手池剪枝（每職業篩選前 8 名 ∪ MG 副手），依據是全副手版（build/i4-full，59m36s）的最佳副手名次全 ≤7。
- Rogue 慢（每檔約 21 分）：d77612c 修（5420b0 連擊門檻在 calc fight 展開 100 支→二分），I4 全跑 42m→18m。Train 不看 Avail（加過濾 TSV 逐 byte 相同，未改）。
- I7 空手（977696c，codex sol）：Main 省略＝空手、技能 14。I4 加 `unarmed` 組重跑：只有 Ninja、Monk 可用；Monk L28 169／170，低於 81 單持 234（0.72），不進 Pareto；既有列逐字不變。見 I4 筆記「空手」節。
- I8 收貨 79f64a0（rank Monsters＋Metric kill）。I9 驅動 50f56be（scripts/i9-party.py：範本＋分支定界），但全隊實跑未完成（build/i9/party-c*.tsv 空、i9-run.log 空）；甲隊配裝來自 I10 基準，不是 I9 結果。
- I10 收貨 adef9af 等：`notes/research/ranking-history.md:404`（基準、8 怪 Expert、女武神、領主 DW、遊俠）。840656b 修 rank Train 給不可用技能。
- I11（2026-10-13）：規格 2f4b0c4、驗收更正 5408137、實作 e846fa4（空手副手拳；注入兩處各紅）。修後 Monk 空手 .243 對 Zatoichi .268（8 怪 Expert L28），L28 起 Zatoichi 反超（揮擊數），見 baseline-align 第六部分；I4 筆記空手節已加更正。
- NPC 兩人（2026-10-13，baseline-align 第七部分）：前四組 .517–.526 打平。MG 裁定（2026-10-13）：光劍維持只一把（「聽說有第二把機率非常低，我沒有看過」「如果光劍不止一把就不用算最佳化了」）；T'rang 能否裝 SoD 要重玩才知道（未取得）；NPC 不跑 Normal（「export 的怪才夠打，不然 normal 可以 5 戰士 + 1 盜賊破關」，Expert 才是比較基準）。未做：I9 全隊實跑、hAttacks 等級公式、打暈／先攻。
- I9b（2026-10-13）：Expert 8 怪全隊實跑完成，`notes/research/ranking-history.md:827`。最佳 戰賊忍僧領侍 2.073，對甲隊 2.025 +.048（每人差 ≤ .02）；遊、領的 Ivory 落在池邊界（Normal 篩選池，Expert 重篩未做）。結果在 build/i9x（不入版）。I12 掉落查表 2a5b40c（`scripts/loot-where.py`）。F10 補血／解狀態／復活：問 MG 範圍與戰鬥中可否，未答。
- I9c（2026-10-13）：Expert 重篩武器池，`notes/research/ranking-history.md:921`。最佳 戰賊忍僧領侍 2.126，對甲隊 +.101，戰士 The Avenger berserk .399（五族並列）；和 I9b 共有格 89088 格 0 差；池邊界放寬 8/10 後結果不變。排除清單改成 MG 沒印象的 7 個 id，The Avenger 改為計算。結果在 build/i9e（不入版）。
  - 下一步：MG 要的感知優先序（0,4,5,6）重算數個最佳隊成員。
  - 特效槽：模型只算槽 5 即死。MG 說 Thieves Dagger 帶中毒、Zatoichi Bo 和徒手帶 KO（KO 後受擊傷害加倍），都沒建模，所以僧侶的 Zatoichi Bo 和徒手被低估。候選：逆向特效槽的意義，再對 KO 和中毒建模（等 MG 開）。
  - I13（2307bbc，muse）：KO／麻痺的抵抗式、難度 ×1.4、持續 1..12 取大、KO 累計受擊 ≥10 醒、戰士／武僧 KO 加成已取得，見 notes/re/ko-paralyze.md。未取得：KO 和受擊加倍的連結、每回合遞減量。下一步候選：實作票（codex sol），加倍取上下界，重算僧／賊／忍（等 MG 開）。SEN 優先序已完成（6ebb9b6，I9c 筆記「感知優先序」節）。
  - I14 武器特效盤點（tickets/I14-weapon-effects-inventory.md）：agy 跑 5 分鐘撞 429 未交件；MG 2026-10-13「擱置。我覺得做這件事需要的資源已經本末倒置讓我其他 sess 都沒辦法再做事」。擱置，不改派；worktree 已移除。I9c 數值最佳化線依 MG「sense 做完就停止」到此停。
  - 後續候選（不建議做，除非 MG 要）：依種族的 PS／升級路徑模型、Snake Speed 與先攻順序、地圖物件解析（寶箱位置、重生、表 57 是否唯一）、凍結候選表。
  - 小地圖疊層構想（未排程）：原作沒有小地圖（MG）；要逆向關卡幾何與隊伍位置，超出數值範圍，算 B3／新階段。
  - 待 MG 測：中文化 patch 下按 `m` 是否開得了大地圖。F10 範圍仍未答。
- 後續候選：遠程副手、法術 A2、生存 A3、六人隊 A5、Expert 重篩武器池。

## 攻略面板 G1–G5（2026-10-14，HEAD 25fc82a 已推、已裝）
- F7 開；上下換層（1–3，累加）、左右翻頁；Esc／F7 關。每層首行與頁首行右側黃色 (L/3)（G5）。內容 `~/.local/state/wiz8-mod-ref/guide/guide-zh.tsv` → 遊戲目錄 `wiz8mod-guide.tsv`（不入 repo、不入發布包；installer 不動它）。
- 地圖索引實機核對：MG 手邊所有存檔都走過一輪（MG 2026-10-14）。log 看到 0–10、13、14、18、19、24–26、29、31、32、38，名稱與 `u32 [0x686a70]`＋地圖表一致；13／14／38 tsv 無內容（顯示「這張圖沒有攻略」）。修道院 8＝Monastery1、9＝Monastery2，哪層對哪個內容 MG 未回。
- G3 稽核（grok 1eEY）已裁決：抓對 1（F1 wrap max 截斷，出貨路徑不可達，契約註解 2395b16）；F2 重入未證。
- **未結：跳回桌面當機**。Application Error：系統 ddraw.dll +0x33159、c000041d（user callback 例外），10-07 04:04 起 17 次（代理 1715e3f 10-07 01:41 之後才有）；10-14 三次都在 F7 操作後 2–10 秒（13:23:50 pid 14000、13:35:20 pid 36112、13:40:41 pid 10316）。嫌疑：代理的 wndproc／WH_KEYBOARD 路徑（與 F2 同區）。待 MG 回答當機前是否 Alt+Tab；之後開排查單。
- 緩解：99bcd27 游標鎖在遊戲視窗（`[window] clip_cursor`，預設 1）；MG 2026-10-14 實機三項（出不去、Alt+Tab 放開、切回再鎖）都過。根源未修；之後再出現 c000041d 才開排查單。
- 修道院 8／9：MG 回「有東西」，樓層對應未確認。
- 小瑕疵：按住 F7 時 auto-repeat 每次多記一行 log。

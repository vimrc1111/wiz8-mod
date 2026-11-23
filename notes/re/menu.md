# B1 戰鬥倍率選單

## 操作與設定

- `[window] clip_cursor`：預設 1。遊戲在前景且未最小化時，每幀把游標鎖在遊戲視窗客戶區（`ClipCursor`）；失去前景（`WM_ACTIVATEAPP` 0）或視窗銷毀時放開。0＝不鎖。啟動記 `window clip_cursor=<值>`。起因：雙螢幕誤點第二顆螢幕使遊戲最小化，失焦偶發當在系統 ddraw.dll +0x33159（c000041d，MG 2026-10-14）。
- `[menu] hotkey_vk`：Win32 virtual-key，預設 121（0x79，F10）；1–255 外退回 F10。啟動記 `menu hotkey_vk=`。
- 上下選項（不繞回）；左右每次 25%，範圍 10–5000%；Home 本項、End 全部回原版 100%；Esc／熱鍵關閉。熱鍵自動重複不反覆開關。
- `[combat] monster_move_pct / event_speed_pct / anim_speed_pct`：預設 400／300／200，唯一來源為 menu.h 項目表。開啟保留選項位置；只有實際改值才 dirty，關閉 dirty 選單寫回 exe 旁 wiz8mod.ini，成功記 `menu saved`，失敗記 ERROR。
- 移動及動畫倍率以對齊 32-bit Interlocked 寫入；選單及呈現快照以 critical section 保護。移動 worker 每 50ms 取值；動畫每次呼叫取值。
- delay 安裝先驗原有五處簽章、保存原始 imm32；即時更新只從保存值重算，五頁皆取得寫入權限才寫入，反序恢復權限並 FlushInstructionCache。輸入在遊戲訊息執行緒處理，不由背景 worker 改程式碼。

## 鍵盤靜態依據

| VA | 前 16 bytes | 說明 |
|---|---|---|
| 0x00401b30 | 8b44240433d23bc20f8c180100003815 | WH_KEYBOARD callback，stdcall 三參數；入口前六 bytes 無相對指令，可保留於跳板 |
| 0x00426ceb | 8b0d34966500ff1528b65e00a0119765 | 主畫面 UI 畫完後呼叫 srGERD::endFrame；只驗簽，不改此處指令 |
| 0x00425590 | 8b4424048b0d609665008b11568b7020 | surface 描述的寬／高在 +0x1c／+0x20；blit 呼叫依此取得來源尺寸，只驗簽 |
| 0x005ebdb4 | 26205e0020205e001a205e0014205e00 | srColorSurface vtable 的 getPixelRow／setPixelRow／raw row 槽，只驗簽 |

- 0x00401f4d：SetWindowsHookExA，種類 WH_KEYBOARD（2），callback 0x00401b30，執行緒為呼叫者。
- 0x00401b58：區分按下／放開；0x00401bce、0x00401c42 將按鍵送入 0x00402270；Ctrl／Shift／Alt 自行保存狀態。這在 WndProc 之前，單靠子類化不會阻止遊戲收到按鍵；不是 DirectInput／GetAsyncKeyState。
- 因此新增上述入口 6 bytes detour；先驗 16 bytes，不符記 `sig-mismatch menu 0x00401b30` 並跳過。簽章由 build.sh 從本表產出 M 列。
- HC_ACTION 才改選單；HC_NOREMOVE 只傳後續 Windows hook，避免 peek 重複套用。負 code 走原 callback。
- 背景執行緒 EnumWindows 尋找同 pid、無 owner、class WndProc 為 0x004011e0 的主視窗，再 SetWindowLongA(GWL_WNDPROC)；不改 WndProc 遊戲 bytes。
- WM_KEYDOWN／WM_SYSKEYDOWN 與 callback 共用狀態機；被吃掉的按鍵、放開及 scan code 對應的 CHAR／SYSCHAR／DEADCHAR 都攔截，包含關閉選單的那次按鍵。開啟前已交給遊戲的按鍵會經原 callback 送 release，避免卡鍵。
- 0x00401756–0x004017aa：遊戲主執行緒依序 Peek/GetMessage、TranslateMessage、DispatchMessage。

## 呈現路徑與依據

- 既有 `notes/research/2026-10-07-A2-ddraw-proxy.md` 記錄預設 Glide2x；因此不選 DirectDraw Flip／Blt，避免只在 DirectX renderer 生效。
- 0x00659634 是主 srGERD。主 UI 收尾於 0x00426ceb，接著 0x00426cf1 呼叫 IAT 0x005eb628 的 srGERD::endFrame；其他渲染處也走同一匯入（例如 0x0042719f）。0x004289c0 的 flipFrame 只有局部動畫呼叫，不能當作一般每幀入口。
- 將 IAT 0x005eb628 改成 thiscall 包裝；先驗上表 frame／surface／rows 三條，再核對 IAT 等於 SR.DLL 匯出 `?endFrame@srGERD@@QAEXXZ`，不符記 `sig-mismatch menu <位址>` 並不安裝。IAT 以 VirtualProtect／InterlockedExchangePointer 更新，再還原保護。
- 包裝只處理主 srGERD：flushRenderers（0x005eb654）完成排隊的幾何繪製，lockBuffer（0x005eba58）取得 surface，畫完 unlockBuffer（0x005eba60），最後呼叫原 endFrame 呈現。其他 srGERD 原樣通過；有重入保護。
- lock／unlock ABI 沿用截圖函式的 0x00422a09／0x00422acc，見 shot.md。與截圖不同，本次不使用 JPEG importer/exporter。
- **B1b 更正（2026-10-12）**：舊版只由 EXE 0x00486ba7 推定 `(x, y, count)`，參數語意錯誤。唯讀材料 `build/sr.dll` 的 preferred ImageBase 為 0x10000000；以下 SR 位址均為此基底下的 VA，執行時可重定位。`getPixelRow` 0x1005d940／`setPixelRow` 0x1005daa0 的 thiscall ABI 實為 `(ARGB buffer, long y, long x0, long x1)`，set 的來源 buffer 為 const；水平範圍是 `[x0,x1)`，轉換數量由 `x1-x0` 取得，空範圍不處理，callee 清理四個參數共 16 bytes。
- SR.DLL 匯出的 srColorSurface vtable 起於 0x100772d0；+0xa4／+0xa8（槽 41／42）分別指向上述 get／set 本體。EXE 0x005ebd10 的匯入 vtable 與原有 0x005ebdb4 簽章不變。`getAddress` 0x1005b5b0 用 `(x0,y)` 定位：資料指標 +0x58，加 y 乘 pitch（+0x24），再加 x0 乘每像素 bytes（+0x3c 的值加一）。`getWidth` 0x10059a60／`getHeight` 0x100598f0 確認寬／高欄位為 +0x1c／+0x20。
- lockBuffer 0x10020df0 成功後回傳的是包裝 surface：建構路徑 0x100205d0 設定 vtable 0x100767f8，槽 41／42 指向 0x10020990／0x10020840。兩者同樣接受 `(buffer,y,x0,x1)`，先依 clip rectangle 裁切，再向 renderer 讀／寫該列；非 ARGB8888 格式透過暫存 srColorSurface 的 row accessor 轉換。`convertToARGB8888` 0x1005b5d0／`convertFromARGB8888` 0x1005b610 依物件 PixelFormat（+0x30）和 read／write converter（+0x48／+0x44）處理，因此 16-bit 565／555 不需自行假設 raw masks、pitch 或通道順序。
- 時機由 SR 本體確認：lockBuffer 0x10020df0 先取得 renderer buffer 鎖再建立包裝 surface；unlockBuffer 0x10020f30 釋放包裝 surface 並解除 buffer 鎖；endFrame 0x1001a810 在 flushRenderers 0x10019a40 後交 renderer 呈現。沿用 flush → lock → row 讀寫 → unlock → 原 endFrame 的順序。
- 原呼叫 `(buffer,16,16+y,w)` 固定寫第 16 列，並把變動 y 當 x0、寬度當 x1，造成單條細線及短白段。B1b 改為 `(buffer,16+y,16,16+w)`，120 次讀寫分別對應 y=16–135、x=16–575（正常尺寸）；底框、三列文字、反白及提示列共用此矩形。
- 疊加區在 (16,16)，最多 560×120：ARGB 暫存區逐 channel 減半得到半透明底，選中列白底黑字，三列倍率及一列按鍵提示，最後寫回同一 surface；不持有引擎 surface 指標跨幀。
- 中文取 atlas.h 12 級 1-bit 字形；G3 起 ASCII 改由同一 Fusion Pixel 字型另產 ascii_bits／ascii_adv，F10 與 F7 共用。原 CJK 四個陣列保持不變。
- 安裝成功記 `hooked menu overlay srGERD endFrame/lockBuffer`，第一次畫完記 `menu overlay drawn srGERD`；第一次開啟記 `menu open`。Lock 失敗、配置失敗或不合理尺寸都有 ERROR。

## 驗證與已知限制

- B1b（2026-10-12）依序執行指定五項驗收皆通過：menu／speed（含 delay、anim）、55 條 EXE 簽章、建置、DLL 格式／22 個匯出／僅 Win32 匯入。使用 `SYS_DDRAW=build/sysddraw.dll`、`WIZ8_EXE=build/Wiz8.exe` 及下列替代字型設定；畫面仍由 cc 實機截圖驗收。
- `menu.test.sh`、`speed.test.sh`、`check-sigs.sh`、`build.sh`、`check.sh` 保持原測試內容。此環境缺少 Fusion Pixel／Noto 且無法連線下載，建置檢查使用既有 `/usr/share/fonts/opentype/unifont/unifont.otf` 設定 WIZ8_FONT／WIZ8_PXFONT，GOCACHE 指向 /tmp。未增加或提交字型檔；產物字形不是正式指定字型，正式發布須重建。
- 產出的 atlas_chars 核對：三個中文標籤與所有中文提示缺字皆為無；ASCII 原本不在 atlas。
- 未執行遊戲；全螢幕獨占、lockBuffer 的可寫呈現緩衝行為及實際畫面／手感待實機確認。靜態依據只能證明呼叫位置與 ABI，不能代替實機驗收。
- 非戰鬥改值只存值，進戰鬥才有效。開著選單仍正常推進遊戲幀，不是暫停功能。
- 本票新增遊戲記憶體變更：既有 delay 五處 +3 imm32（0x004ea0c1、0x0053eed6、0x00543737、0x00587594、0x0058a4f3，各 4 bytes）的即時重寫；WH_KEYBOARD 入口 0x00401b30 的 6 bytes；endFrame IAT 0x005eb628 的 4 bytes。T／U／W 既有位址與簽章不變；frame／surface／rows 三處僅驗簽。
- 未執行遊戲、未寫入遊戲目錄、未 commit。倍率持久化是 DLL 在實機執行時的功能。

## G3 攻略面板

- `[guide] hotkey_vk`：Win32 virtual-key，預設 118（0x76，F7），1–255 外退回 F7；啟動記 `guide hotkey_vk=<值>`。G4：按一次開第 1 層第 1 頁，再按關閉；Esc 僅開啟時攔截並關閉。上下換層（1–3），換層頁碼歸第一頁；左右翻頁，到邊界停住仍吃鍵。方向鍵支援自動重複，熱鍵與 Esc 的 repeat 吃掉但不作用；其他鍵交給遊戲。F10 選單開啟時攻略熱鍵不改層；開 F10 先關攻略。若兩個 hotkey_vk 相同，F10 優先。
- F7 靜態檢查：0x004f9b1c 是效果類型分派的上限；0x004fb6bc／0x004fc05c 是效果 ID 118 的特例；0x00528362 是效果清單的 ID 特例；0x00532883 是效果／狀態類型分派上限。均非 virtual-key 判斷。0x005d3fe5／0x005d3fea 的文字輸入路徑略過整段 F1–F12，並非 F7 動作；0x00402270 將鍵值送入事件佇列，0x00401b30 為鍵盤 callback，未發現 F7 專用分支；未發現內建固定 F7 衝突。遊戲自訂按鍵仍可能綁 F7，實機由 MG 確認，可用 ini 改攻略熱鍵。
- 路徑與 `speed_init` 相同，取主 EXE 目錄的 `wiz8mod-guide.tsv`（與 wiz8mod.ini 同目錄）。只在 0→1 以 CreateFileA／ReadFile 重讀，最多 1 MiB；讀不到、超限或讀取失敗當作沒檔，F7 不畫。2、3 層沿用同一份快取。無攻略內容、原版資料表進 repo 或發布包。
- TSV 為 UTF-8，每列 `地圖索引<TAB>層<TAB>文字`，可 CRLF、末列可無換行。只接受三欄、純十進位索引及層；忽略空列、# 開頭及壞列。索引需對上目前地圖，層為 1–3；依層小到大、同層照檔案順序累加。內容由使用者自行準備。

| VA | 前 16 bytes | 說明 |
|---|---|---|
| 0x0042a7b5 | 8935706a6800ffd58d94248401000052 | 地圖索引寫入點，只驗簽、不裝 detour、不寫此處；失配停用攻略，禁止讀取地圖索引 |

- build.sh 由本表產生 `M:map:0x0042a7b5`；不符記 `sig-mismatch guide 0x0042a7b5`，不影響 F10。簽章通過才唯讀 u32 0x00686a70；0xffffffff 或超過 47 視為不在地圖。地圖名唯讀表 0x00604478，每筆 107 bytes、+50，最多 50 ASCII 字元。
- 共用 menu_cs、menu_keyboard／menu_wndproc、menu_end_frame／menu_render 的 flush→lock→row→unlock 路徑；沒有新增 endFrame hook。框 (16,16)，寬 min(surface 寬−32,600)，高依列數且不超過 surface 高−32，行高 16；層間留 8px。標題含地圖名、索引、目前層、頁碼／總頁數、上下換層／左右翻頁提示及實際熱鍵；無地圖顯示「不在地圖上」，TSV 完全沒有該圖有效列時顯示「這張圖沒有攻略」。
- guide_select 挑列；MultiByteToWideChar 轉 UTF-16，guide_wrap 使用 wrap_can_break／wrap_space，advance 與 menu_text 同源。ASCII 0x20–0x7e 全部由 Fusion Pixel 12px 的獨立 ascii_bits／ascii_adv 繪製；既有 atlas_chars／atlas_bits／atlas_off／atlas_cell 不變。布局依地圖、層、頁及 surface 尺寸快取。
- 每次攻略面板吃掉的按下鍵（含 repeat）記一行 `guide map=<索引> name=<地圖表名> layer=<層> page=<P>/<N> lines=<TSV列數>`，P 從 1 起算；release 僅沿用按下時的抑制，不另記錄，未吃的按下鍵不記錄。關閉後 page=1/1、layer=0、lines=0。F10 開啟時不切攻略。無檔只記一次 `guide no tsv <路徑>`；只有超過 1026 個排版行才以最後一行「…」提示並記一次 `guide truncated`（重新開啟重設）；配置／轉碼失敗記 ERROR。
- 已知限制：同地圖不細分建築或區域；左右分頁，排版行上限 1026。中文限既有 atlas 字表，其他字保留字寬但無字形；補字需另票，不能改本票既有 CJK 陣列。過長標題依框寬裁切。攻略面板不暫停遊戲。未執行遊戲，實際 lockBuffer 畫面、熱鍵與地圖索引由 MG 實機驗收。

- G3 驗證（2026-10-14）：指定七項驗收全部通過；67 條簽章吻合，DLL 為 i386、22 個匯出吻合，僅 GDI32／KERNEL32／USER32 匯入。使用 build/fonts 的指定兩套字型，GOCACHE=/tmp/g3-go-cache（預設 cache 不可寫）。以改前生成器、相同字型與輸入生成基準，抽取 atlas_chars／atlas_bits／atlas_off／atlas_cell 與改後 out/atlas.h 比對，四者宣告及陣列內容逐 byte 相同。攻略固定中文提示皆在原字表中。

### G4

- `guide_key` 改為兩個純函數：`guide_nav` 管理 `{layer, page}` 與吃鍵規則；`guide_paginate` 依 step 貪婪分頁，空內容仍一頁。`guide_select`／`guide_wrap` 不變。
- 內容仍累加第 1 層到目前層，先算全部行的 step：一般 16px，換層首行 24px。可用高度為面板高度減 36px（標題下方 y=28 到底邊內縮 8px），只畫目前頁。頁首省略層間額外 8px，直接從 y=28 開始；分頁計算仍保留該行的 24px step。
- 按鍵前按目前 surface 尺寸重排（尚未取得 surface 時用 client 尺寸，無視窗時以 640×480 預估），確保左右使用目前層的總頁數；換層／翻頁後重排與更新標題。surface 尺寸改變也重排並夾住頁碼。
- 開啟仍重讀 TSV；開 F10 仍關攻略。面板開著時方向鍵不送遊戲，其他鍵仍正常操作遊戲；開啟前已送遊戲的方向鍵補 release，避免移動卡住。熱鍵與 Esc 不因 repeat 反覆切換，方向鍵可按住連續換層／翻頁。
- 未執行遊戲、未寫入遊戲目錄、未 commit；MG 實機驗收：F7 開、下下到第 3 層、左右翻完 Arnika、上上回第 1 層、Esc／F7 關，並確認其他鍵可操作遊戲。
- G4 驗證（2026-10-14）：指定七項驗收全部通過，67 條簽章吻合；DLL 為 i386、22 個匯出一致，僅 GDI32／KERNEL32／USER32 匯入。固定標題與訊息字形均在既有 atlas 字表；未改測試、scripts 或 atlas 生成器。首次驗收因 out/atlas.h 尚未生成而停於覆蓋檢查，先用 build/ 指定字型離線建置，再按指定順序完整驗收。GOCACHE=/tmp/g4-go-cache，GOPROXY=off、GOSUMDB=off，未下載。
- F11 x-ray 傾印（X1 執行期驗證，2026-10-31）：`[xray] hotkey_vk` 預設 122（F11）；F12 不用（Steam 截圖鍵）。原版 F11 用途：資料檔與 notes 所見範圍內未見，實機由 MG 確認。按一次在遊戲執行緒 end-frame 寫 `%TEMP%\wiz8-mod-xray-<pid>-<N>.txt`，log 一行 `xray dump N ok|fail <路徑> scene_m=`；成功 MessageBeep(MB_OK)、失敗 MB_ICONHAND。

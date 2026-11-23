# Wiz8.exe 戰鬥等待與加速 hook 候選點

靜態研究；未執行遊戲，未實作 hook。VA 以映像基址 `0x400000` 表示；本文所有前 16 bytes 都由 `xxd` 讀 `build/Wiz8.exe`，檔案偏移＝VA−`0x400000`，不是從 objdump 猜回機器碼。簽章只適用於本副本；簽章長度不等於可直接覆寫的指令長度。

副本 SHA-256：`18a74ff61c65b8a2d4cfa11ffce82ad7fef022a94eaf0c2f217e479e981420d2`。`[已驗證]` 僅表示本副本的靜態資料／控制流已核對，不代表實測效果。函式名稱與結構欄位名稱是本文的用途描述；未證實的遊戲語意另外標出。

## 發現

### 計時來源與 API 覆蓋

- [已驗證] exe 的 Windows 計時／等待匯入包含 `GetTickCount`（IAT `0x5eb02c`）、`Sleep`（`0x5eb080`）、`SetTimer`（`0x5eb238`）、`KillTimer`（`0x5eb234`）、`WaitMessage`（`0x5eb290`）；另有檔案／曆時 API、CRT `time/difftime`、Bink 等待與 SR 計時器，完整清單及可辨識引用見下表。由 PE import descriptor 的 FirstThunk 逐項讀名稱核對，不僅搜尋 API 字串。未匯入 `timeGetTime`、`timeBeginPeriod/timeEndPeriod`、`QueryPerformanceCounter/QueryPerformanceFrequency`、`SleepEx`、`WaitForSingleObject/Ex`、`WaitForMultipleObjects/Ex`、`MsgWaitForMultipleObjects/Ex`、`GetSystemTime/GetSystemTimeAsFileTime`。代表初始化函式 `0x406ba0`（前 16 bytes：`ff152cb05e00a308b76e00a30cb76e00`）；來源 build/wiz8.asm:8360、build/wiz8.asm:8368。

- [未取得] 「所有執行時呼叫及各點戰鬥可達性」尚未完整取得。附表列出所有以 IAT 位址明示的引用，並補上已辨識的暫存器間接呼叫；沒有把載入函式指標冒充呼叫。尚未建立全程式跨函式指標傳遞、動態 `GetProcAddress` 與 DLL 內部呼叫圖；尤其 SR 匯入主要由虛擬表到達，不能以 exe 沒匯入 QPC 推論 SR.DLL 不用 QPC。已追到 SR 毫秒 thunk `0x5e21ca`（前 16 bytes：`ff2528b75e00ff252cb75e00ff2530b7`） 與動畫讀值；來源 build/wiz8.asm:653221、build/wiz8.asm:242895。

- [已驗證] 共用 UI 毫秒時鐘由 `0x406b70`（前 16 bytes：`ff152cb05e008b0d08b76e003bc1730f`） 更新：讀 `GetTickCount` 減起點 `0x6eb708`，寫入 `0x6eb70c`。`0x406ba0`（前 16 bytes：`ff152cb05e00a308b76e00a30cb76e00`） 以 `SetTimer` 要求 10 ms 回呼間隔；這是要求值，不是保證的更新精度。`0x406be0`（前 16 bytes：`a10cb76e00c3ff15c8b25e0090909090`） 讀快取；`0x406bf0`（前 16 bytes：`8b4424048b0d0cb76e0003c1c3909090`） 回傳「快取＋duration」；`0x406c00`（前 16 bytes：`8b4424048b0d0cb76e003bc176032bc1`） 用 unsigned 比較回傳剩餘量或 0。初始化及跨 DWORD 回繞有原版特殊行為，倍率 1 應原樣通過，不能順手重寫它。戰鬥與訊息都讀這組 helper，因此不應全域倍率化。來源 build/wiz8.asm:8346、build/wiz8.asm:8360、build/wiz8.asm:8380、build/wiz8.asm:8388、build/wiz8.asm:8397。

- [已驗證] 唯一明示的 `Sleep` 呼叫是 `0x5e2fc2`，傳入 500 ms；所在函式 `0x5e2fa0`（前 16 bytes：`568bf18b0685c074286a0050ff15d0b2`） 前後操作 Bink handle，屬影片停止／清理序列，沒有發現它作為每個戰鬥行動的等待。影片更新函式 `0x5e3020`（前 16 bytes：`568bf18b0685c0743d50ff15e8b25e00`） 在 `0x5e302a` 呼叫 `BinkWait`。`WaitMessage` 位於訊息迴圈（附表），它等待 Windows 訊息，不是找到一個「戰鬥固定 Sleep」的證據。來源 build/wiz8.asm:654575、build/wiz8.asm:654588、build/wiz8.asm:654638、build/wiz8.asm:690、build/wiz8.asm:128451、build/wiz8.asm:202607。

- [已驗證] 遊戲時間與上述呈現等待不可混用：一般更新在 `0x4825b4` 讀 `GetTickCount`，以 f32 `12`（`0x60a390`）換算並送往 `0x482a20`（前 16 bytes：`558bec83e4f883ec54566830694d006a`）；戰鬥回合結算則在 `0x4e9c34` 附近固定送 `120000` 給同一時間推進函式。這兩者都不是本次候選修改值。來源 build/wiz8.asm:174458、build/wiz8.asm:313727；另見 notes/re/combat-2.md:83。

### 戰鬥訊息：顯示期限不等於行動停頓

- [已驗證] log 插入函式 `0x58ac00`（前 16 bytes：`b810200000e8967105005356e89f2fff`） 的自動 pane 選擇在 `0x58ace0` 讀戰鬥 byte `0x683f94`，一般分支將「非零」轉為 pane 1，探索為 pane 0；對話等前置分支可覆寫。不能把其他研究看到的 pane 2 一概當成戰鬥 log。來源 build/wiz8.asm:533378、build/wiz8.asm:533440。

- [已驗證] 每條訊息的保留期限由 `0x58b5f0`（前 16 bytes：`668b442404663dffff750666a1056968`） 設定。pane 參數 `0xffff` 代表取目前 pane `0x686905`；從 `0x689b17+4*pane` 取已發出索引，文字記錄基址 `0x68f2d8`，每 pane 350 筆、每筆 36 bytes，期限在記錄 `+8`。令 `L=wcslen(文字)`、`S=DWORD[0x6850d9]`，正常設定與無乘法溢位時，`D=min(60000, floor(S*(100+5L)/100))` 毫秒；`0x58b698` 經 `0x406bf0` 轉成絕對期限。預設 `S=2500`，例如 L=20 → 5000 ms。特殊 pane 3 分支在 `0x58b2b8` 附近也用同一設定與算法；尚不能只改一點就宣稱涵蓋所有訊息。來源 build/wiz8.asm:534155、build/wiz8.asm:534179、build/wiz8.asm:534198、build/wiz8.asm:533883。

- [已驗證] 設定初始化 `0x54b560`（前 16 bytes：`53bb010000005733c0b929000000bfc8`） 在 `0x54b5ca` 將訊息基準 `0x6850d9` 設為 2500，並將行動等待設定 `0x6850dd` 設為 1000。設定 UI 轉接函式 `0x5a6e20`（前 16 bytes：`83ec08568bf18b868c00000085c0750b`） 在 `0x5a7039` 一帶把前者與 UI 物件 `+0x48` 的浮點欄位互轉；在 `0x5a723a` 一帶另把後者與 `+0x40` 互轉。輸入處理局部 `0x591c6b/0x591dd2` 對訊息值以 250 階梯增減，範圍 0–5000；`0x591b1f` 有行動值增加分支、上限 5000。因此兩種速度可由設定分開控制。來源 build/wiz8.asm:450347、build/wiz8.asm:450371、build/wiz8.asm:570099、build/wiz8.asm:570247、build/wiz8.asm:570404、build/wiz8.asm:542557、build/wiz8.asm:542662、build/wiz8.asm:542462。

- [未取得] 尚未將這兩個 UI 欄位的本地化文字資源還原成確切選項名稱，不能保證介面文案就叫「訊息速度／戰鬥速度」。已確認設定值的讀寫與等待消費端；`0x5a6e20`（前 16 bytes：`83ec08568bf18b868c00000085c0750b`） 的 UI 欄位是後續補對文字的起點。來源 build/wiz8.asm:570247、build/wiz8.asm:570404。

- [已驗證] 已發出訊息的過期檢查函式 `0x58d760`（前 16 bytes：`0fbf05056968008b0485179b680085c0`） 在 `0x58d79e` 呼叫剩餘時間 helper，render 在 `0x58c442/0x58c5af` 使用期限相關結果。但 UI 更新 `0x562a80`（前 16 bytes：`5356e86928ffffe874fd0300a1cced68`） 在已發出數小於總數時，經 `0x5b1740`（前 16 bytes：`b001c390909090909090909090909090`）（本副本固定回 true）就呼叫 `0x58b5f0` 發出下一條，該路徑未等上一條期限歸零。故已找到的是訊息顯示期限，不是「每條 log 顯示 D 毫秒後才繼續戰鬥」；也未找到戰鬥主狀態機直接等待這組文字索引清空。來源 build/wiz8.asm:536835、build/wiz8.asm:536852、build/wiz8.asm:535293、build/wiz8.asm:535396、build/wiz8.asm:482425、build/wiz8.asm:482449、build/wiz8.asm:482458、build/wiz8.asm:585291。

### 回合／行動流程與戰鬥旗標

- [已驗證] 戰鬥旗標為 `BYTE[0x683f94]`，戰鬥資料指標為 `G=DWORD[0x6836a8]`。建立函式 `0x4e7090`（前 16 bytes：`83ec20535657e8257e000084c075075f`） 配置 `0xa64` bytes，初始化 `G+0x7ac` 時戳、`G+0x7b0` 狀態、`G+0x7b4=-1` 與 `G+0x7b8=0`，在 `0x4e731d` 設旗標 1；退出函式 `0x4ea310`（前 16 bytes：`a0ce3f680084c07405e812630000a0cd`） 在 `0x4ea481` 清為 0。主更新局部 `0x56032e` 依旗標分流，戰鬥且未受 UI 暫停旗標阻擋時在 `0x560362` 呼叫戰鬥更新。這是建立、退出、dispatch 三方證據；hook 保護可用旗標非零，需讀 G 時再確認 G 非空，但不能因此跳過原版的 UI／暫停判斷。來源 build/wiz8.asm:310267、build/wiz8.asm:310445、build/wiz8.asm:314377、build/wiz8.asm:479550、build/wiz8.asm:479565。

- [已驗證] 戰鬥更新 `0x4e8ea0`（前 16 bytes：`a0bc4068005333db3ac356570f85a705`） 不只等一種條件：先檢查 UI／模式旗標，再查 `0x420e10` 狀態、怪物 busy 函式 `0x4e8c30`，以及 `G+0x7c0` 指示的音效 handle（`0x408ef0`）、`0x500e50` 的事件就緒與 `0x4a2760/0x4a5790` 的效果物件狀態。任一條件不滿足即可返回，後續 update 再試。`G+0x918=1` 另有特殊分流，不能將一般分支概括為每一種戰鬥模式。來源 build/wiz8.asm:312685、build/wiz8.asm:312718、build/wiz8.asm:312731、build/wiz8.asm:312742、build/wiz8.asm:312757。

- [已驗證] `0x4e8ea0`（前 16 bytes：`a0bc4068005333db3ac356570f85a705`） 以 `G+0x7b0` 分派 0–3 狀態：0 經 `0x4ed710` 後呼叫 `0x4e9490` 選行動；1 在 `0x4e9135` 檢查 `G+0x7ac` 剩餘時間，非零即返回，歸零才到玩家準備 `0x4ea5c0`／怪物準備 `0x4eae20`；2 分派玩家攻擊 `0x53e250`、施法 `0x4ebfe0` 或怪物執行 `0x4eb5a0` 並接回狀態；3 收尾並重設供下個行動。這是「先等截止時間，再受動畫／效果狀態及行動函式返回值控制」的合作式更新，不是一次阻塞式 sleep。來源 build/wiz8.asm:312814、build/wiz8.asm:312847、build/wiz8.asm:312856、build/wiz8.asm:312858、build/wiz8.asm:312923。

- [已驗證] 最明確的戰鬥局部 duration 產生點在行動選擇 `0x4e9490`（前 16 bytes：`518b15a83668008b82b007000085c0c6`）：到達 `0x4e986a` 時讀 `DWORD[0x6850dd]`；當 `G+0xa60` 非零，且 `0x6850d2` 為零或局部條件為假，將大於等於 800 的值限為 800 ms。`0x4e9890` 建立期限並寫 `G+0x7ac`。前置條件可繞過這段，不能說每一行動必等預設 1000 ms。設定 `0x6850dd` 在 `0x4f02e0` 附近尚有其他消費端，因此局部修改比全域改設定更容易限制作用範圍。來源 build/wiz8.asm:313142、build/wiz8.asm:313421、build/wiz8.asm:313433、build/wiz8.asm:313436、build/wiz8.asm:322228。

- [已驗證] 還有一組直接 `GetTickCount` 的事件延遲：攻擊函式 `0x53e250`（前 16 bytes：`6aff689b735e0064a100000000506489`） 在 `0x53eeda`、另一攻擊結果路徑 `0x543110`（前 16 bytes：`8b54241053568b7424142bf28b542410`） 在 `0x54373b` 記錄事件物件 `+0x34` 的時戳，`+0x30` 設 800；戰鬥結束路徑 `0x4e9f90`（前 16 bytes：`5356e8294f00008ad8e8d2ce050084db`） 在 `0x4ea0c5` 記錄時戳並設 1500。共同消費局部 `0x52e01b` 比較經過時間與 `+0x30`，未超過時返回。這些是由戰鬥函式建立的定時事件，不能當成上一節文字保留期限；完整事件／語音用途及其與所有 action gate 的關係尚待補追。來源 build/wiz8.asm:432617、build/wiz8.asm:433506、build/wiz8.asm:439243、build/wiz8.asm:314074、build/wiz8.asm:410033。

### 動畫與施法

- [已驗證] 動畫使用另一個 SR 計時來源。計時物件建立函式 `0x439550`（前 16 bytes：`64a1000000006aff688b425e00506489`） 呼叫 SR.DLL 的 `srTimer` 建構子，vtable `0x5ec078` 的 `+0x14` 指向 `0x5e21ca`（前 16 bytes：`ff2528b75e00ff252cb75e00ff2530b7`），再轉到 `0x5eb728` 的 `srTimer::getMsTime`。動畫物件建立 `0x4b53d0`（前 16 bytes：`6aff6848635e0064a100000000506489`） 在 `0x4b544f` 取 `0x6598b8`，於 `0x4b547c` 呼叫該槽，把毫秒值存到動畫物件 `+0x68`；`0x4c5a00`（前 16 bytes：`8b44240485c0742753568bb0d8010000`） 的啟動／重設路徑也重存此時間。因而不能只搜尋 `GetTickCount` 就認為已涵蓋動畫時鐘。來源 build/wiz8.asm:76753、build/wiz8.asm:76789、build/wiz8.asm:653221、build/wiz8.asm:242862、build/wiz8.asm:242895、build/wiz8.asm:264477。

- [已驗證] 怪物動作由影格與狀態共同判斷。令 `B=DWORD[怪物+0xc]`、`A=DWORD[B+0x1d8]`，`0x4c5710`（前 16 bytes：`8b4424048b88d80100008a91a7000000`） 檢查 `A+0xa7` 是否非 `0xff`（待處理狀態），`0x4c59e0`（前 16 bytes：`8b44240485c0740a8b80d80100008a40`） 讀 `A+0x6d` 活動旗標。查詢函式 `0x4c4660`（前 16 bytes：`8b4424045683ceff83f809578bf90f87`） 經 `0x4c5b40` 包裝：selector 0 取目前序列總影格、4 讀 `A+0x64` 目前影格、6 讀 `A+0xa4` 序列 ID、7 綜合 `+0x70/+0x6d/+0x6e` 與首末影格判斷完成。這些欄位不等同單一毫秒 deadline。來源 build/wiz8.asm:264114、build/wiz8.asm:264461、build/wiz8.asm:262662、build/wiz8.asm:264636。

- [已驗證] 戰鬥 busy 檢查 `0x4e8c30`（前 16 bytes：`51a1ad3f6800535556575033ede82ea0`） 會檢查上述動畫狀態、`0x4c5ee0` 與轉向差值；活動序列 ID 1、2、4、`0x15` 有豁免分支。正在執行的怪物行動若 `0x5430c0(序列ID)==6`，在 `0x4e8cff/0x4e8d09` 比較「目前影格是否已到總影格的一半」，未到才繼續擋住；另一分類 7 則看怪物 `+0x2df`。因此不能一律稱為「播完整段動畫才進下一步」，更不能直接強制 busy=false，否則可能提前觸發傷害或效果。來源 build/wiz8.asm:312472、build/wiz8.asm:312513、build/wiz8.asm:312538、build/wiz8.asm:312579。

- [已驗證] 玩家攻擊 `0x53e250`（前 16 bytes：`6aff689b735e0064a100000000506489`） 也分階段：在 `0x53e3a1` 查尾部資料 `+0xa5`，初次呼叫 `0x549ef0` 建立呈現並設旗標，回傳狀態 2，之後更新再進入結算路徑。施法 `0x4ebfe0`（前 16 bytes：`6aff68286e5e0064a100000000506489`） 在 `0x4ec054` 查 `P=DWORD[G+0xa4c]` 的 byte `P+0x1e4`，未就緒回傳 2，就緒後設 `P+0x1e6` 並走效果／目標處理；`0x500d90` 內有寫入該就緒欄位的路徑。這是等待狀態，不是已找到固定「施法 X 毫秒」常數。來源 build/wiz8.asm:432617、build/wiz8.asm:432706、build/wiz8.asm:316579、build/wiz8.asm:345822。

- [未取得] 尚未完整取得每個序列的 FPS、SR 毫秒差如何換算成 `A+0x64` 影格、玩家攻擊／全部法術效果的就緒旗標生產鏈。已追到 `0x4c4660`（前 16 bytes：`8b4424045683ceff83f809578bf90f87`） 的影格查詢、`0x4bf6f7` 一帶保存毫秒起點、`0x4e8c30` 的戰鬥消費端，但缺動畫逐幀更新者的完整資料流；本次材料也沒有這些戰鬥函式的完整 Ghidra 輸出。因此沒有提出「加快 SR 共用 timer」或跳過動畫旗標的可用 detour，精確動畫倍速仍需補 RE。來源 build/wiz8.asm:262662、build/wiz8.asm:256931、build/wiz8.asm:312472。

### 候選 hook（由小到大；尚未實作／執行驗證）

- [推測] **首選：只縮短選行動後的 duration。** 所屬函式 `0x4e9490`（前 16 bytes：`518b15a83668008b82b007000085c0c6`）；局部 hook `0x4e988f`（前 16 bytes：`50e85bd3f1ff8b15a836680083c40489`），在原版限幅之後、送入 deadline helper 之前，僅當 `BYTE[0x683f94]!=0` 且 G 有效時將 EAX 中 D 改為 `floor(D/k)`（k 為有限正倍率；k=1 完全直通，無效值直通，結果限制在 DWORD 範圍）。原位置含 1-byte 參數準備及 5-byte 相對呼叫，若採局部 detour 需搬移完整 6 bytes 並重定位對 `0x406bf0` 的呼叫，回到 `0x4e9895`；16-byte 比對只是識別簽章；除指定 duration 外，須保留該位置原有暫存器、flags、堆疊與呼叫約定。預期只縮短本次行動前等待，不改探索、全域時鐘、120000 回合結算或動畫／效果 gate；動畫占大宗時加速有限。副作用：台詞與行動更容易重疊、可讀時間縮短；k 很大得到 0 仍受 update 頻率限制。倍率切換只影響新建期限，既有期限不重算。來源 build/wiz8.asm:313142、build/wiz8.asm:313421、build/wiz8.asm:313432、build/wiz8.asm:312858。

- [推測] **次選／純呈現選配：縮短戰鬥 log 保留期限。** 所屬函式 `0x58b5f0`（前 16 bytes：`668b442404663dffff750666a1056968`）；局部 hook `0x58b697`（前 16 bytes：`52e853b5e7ff8947088b3cb5179b6800`），在 60000 上限之後，僅當戰鬥旗標非零且 ESI 的有效 pane=1 時將 EDX 中 D 改為 `floor(D/k)`；倍率與邊界規則同首選，其他 pane 原樣通過。此處也是完整 6-byte 參數準備＋相對呼叫，搬移時需重定位 `0x406bf0`，回到 `0x58b69d`，保留非目標暫存器、flags 與堆疊。不改全域 `0x6850d9`；已顯示訊息保持原期限。副作用是文字更早消失、同 pane 的非攻擊提示也可能受影響；特殊 pane 路徑不涵蓋。**沒有證據此點能縮短下一行動的等待，不可把它當成完整戰鬥倍速。** 若產品只要實際行動加速，先只採首選。來源 build/wiz8.asm:534155、build/wiz8.asm:534197、build/wiz8.asm:533440、build/wiz8.asm:482458。

### API 與呼叫點附表

下表的位址是呼叫指令位置；標為「取址」者只是載入 IAT 指標。SR 的「thunk」是匯入跳轉入口，不是上層呼叫函式。未列出完整戰鬥可達性證據的項目均保留未知，不能據此認定探索專用或可以刪掉。

| DLL／匯入名稱 | IAT VA | exe 明示引用／已追到呼叫 |
|---|---|---|
| KERNEL32.DLL `GetFileTime` | `0x5eb010` | 呼叫 `0x40569c`（build/wiz8.asm:6209） |
| KERNEL32.DLL `CompareFileTime` | `0x5eb014` | 呼叫 `0x40572a`（build/wiz8.asm:6265）；呼叫 `0x4058e5`（build/wiz8.asm:6432） |
| KERNEL32.DLL `GetTickCount` | `0x5eb02c` | 見下方直接及間接完整盤點 |
| KERNEL32.DLL `FileTimeToLocalFileTime` | `0x5eb07c` | 取址 `0x4056a7`（build/wiz8.asm:6212）；呼叫 `0x512132`（build/wiz8.asm:370672）；取址後呼叫 `0x4056ae`（build/wiz8.asm:6214）、`0x4056b6`（build/wiz8.asm:6218）、`0x4056be`（build/wiz8.asm:6222） |
| KERNEL32.DLL `Sleep` | `0x5eb080` | 呼叫 `0x5e2fc2`（build/wiz8.asm:654588） |
| KERNEL32.DLL `FileTimeToSystemTime` | `0x5eb09c` | 呼叫 `0x512140`（build/wiz8.asm:370676） |
| KERNEL32.DLL `GetLocalTime` | `0x5eb0a4` | 呼叫 `0x511e2f`（build/wiz8.asm:370468） |
| MSVCRT.DLL `difftime` | `0x5eb194` | 呼叫 `0x431bf7`（build/wiz8.asm:66571） |
| MSVCRT.DLL `time` | `0x5eb1d4` | 呼叫 `0x40ef82`（build/wiz8.asm:19136）；呼叫 `0x4314f8`（build/wiz8.asm:66063）；呼叫 `0x431bed`（build/wiz8.asm:66568） |
| USER32.DLL `KillTimer` | `0x5eb234` | 呼叫 `0x406bd8`（build/wiz8.asm:8377） |
| USER32.DLL `SetTimer` | `0x5eb238` | 呼叫 `0x406bbf`（build/wiz8.asm:8368） |
| USER32.DLL `WaitMessage` | `0x5eb290` | 呼叫 `0x4017b7`（build/wiz8.asm:690）；取址 `0x45fc2b`（build/wiz8.asm:128451）；呼叫 `0x497118`（build/wiz8.asm:202607）；取址後呼叫 `0x45fc47`（build/wiz8.asm:128456） |
| BINKW32.DLL `_BinkWait@4` | `0x5eb2e8` | 呼叫 `0x5e302a`（build/wiz8.asm:654644） |
| MSS32.DLL `_AIL_sample_ms_position@12` | `0x5eb3cc` | 呼叫 `0x40989d`（build/wiz8.asm:12004） |
| MSS32.DLL `_AIL_stream_ms_position@12` | `0x5eb3d0` | 呼叫 `0x4098bb`（build/wiz8.asm:12015） |
| SR.DLL `?getRawTime@srTimer@@UAEKW4e_timerReadControl@1@@Z` | `0x5eb714` | thunk `0x5e21ac`（build/wiz8.asm:653216） |
| SR.DLL `?getRawTime@srTimer@@UAEKAAVsrQuadWord@@W4e_timerReadControl@1@@Z` | `0x5eb718` | thunk `0x5e21b2`（build/wiz8.asm:653217） |
| SR.DLL `?getUTime@srTimer@@UAEKAAVsrQuadWord@@W4e_timerReadControl@1@@Z` | `0x5eb720` | thunk `0x5e21be`（build/wiz8.asm:653219） |
| SR.DLL `?getTime@srTimer@@UAENW4e_timerReadControl@1@@Z` | `0x5eb724` | thunk `0x5e21c4`（build/wiz8.asm:653220） |
| SR.DLL `?getMsTime@srTimer@@UAEKW4e_timerReadControl@1@@Z` | `0x5eb728` | thunk `0x5e21ca`（build/wiz8.asm:653221） |
| SR.DLL `?reset@srTimer@@UAEHHHH@Z` | `0x5eb72c` | thunk `0x5e21d0`（build/wiz8.asm:653222） |
| SR.DLL `?resume@srTimer@@UAEKXZ` | `0x5eb730` | thunk `0x5e21d6`（build/wiz8.asm:653223） |
| SR.DLL `?pause@srTimer@@UAEHXZ` | `0x5eb734` | thunk `0x5e21dc`（build/wiz8.asm:653224） |
| SR.DLL `?getAscTime@srTimer@@UAEPADPADW4e_timerReadControl@1@@Z` | `0x5eb738` | thunk `0x5e21e2`（build/wiz8.asm:653225） |
| SR.DLL `??0srTimer@@QAE@HHH@Z` | `0x5eb73c` | 呼叫 `0x4395ce`（build/wiz8.asm:76789）、`0x43985d`（build/wiz8.asm:76983） |
| SR.DLL `??1srTimer@@UAE@XZ` | `0x5eb740` | 呼叫 `0x4397d3`（build/wiz8.asm:76939） |
| SR.DLL `?getUTime@srTimer@@UAEKW4e_timerReadControl@1@@Z` | `0x5ebad0` | thunk `0x5e21b8`（build/wiz8.asm:653218） |

#### GetTickCount 直接呼叫（IAT 0x5eb02c）

「共用」表示已看到會影響戰鬥以外的時間／UI；「戰鬥事件」有上文的建立端證據；「未取得」表示只完成呼叫點盤點，戰鬥可達性未判定。前 16 bytes 列的是呼叫點簽章，不是函式入口。

| 呼叫點 VA | 前 16 bytes | 戰鬥關係 | 來源 |
|---|---|---|---|
| `0x401f94` | `ff152cb05e00668b151c056f00668b35` | [未取得] | build/wiz8.asm:1336 |
| `0x402144` | `ff152cb05e008bf0a0ed046f0084c074` | [未取得] | build/wiz8.asm:1445 |
| `0x406b70` | `ff152cb05e008b0d08b76e003bc1730f` | [已驗證] 共用 UI 時鐘 | build/wiz8.asm:8346 |
| `0x406ba0` | `ff152cb05e00a308b76e00a30cb76e00` | [已驗證] 共用時鐘初始化 | build/wiz8.asm:8360 |
| `0x4091f6` | `ff152cb05e00898668416e005fb0015e` | [未取得] | build/wiz8.asm:11392 |
| `0x409519` | `ff152cb05e008b8e444b6e0003d803d9` | [未取得] | build/wiz8.asm:11678 |
| `0x409706` | `ff152cb05e008be88b471c3bf00f84a9` | [未取得] | build/wiz8.asm:11850 |
| `0x40981d` | `ff152cb05e008d0cf68d144e5e8b0c95` | [未取得] | build/wiz8.asm:11959 |
| `0x409f95` | `ff152cb05e00898650416e00889e5441` | [未取得] | build/wiz8.asm:12605 |
| `0x40a257` | `ff152cb05e00898650416e00a0500e65` | [未取得] | build/wiz8.asm:12809 |
| `0x40a527` | `ff152cb05e00898550416e00a0500e65` | [未取得] | build/wiz8.asm:13033 |
| `0x40b005` | `ff152cb05e008b8b3c4b6e0089865041` | [未取得] | build/wiz8.asm:13958 |
| `0x40b1df` | `ff152cb05e008b8f444b6e0003d803d9` | [未取得] | build/wiz8.asm:14108 |
| `0x421bea` | `ff152cb05e008b4c242c668b542430a3` | [未取得] | build/wiz8.asm:45328 |
| `0x426d15` | `ff152cb05e008b0d9c4065003bc1720c` | [未取得] | build/wiz8.asm:51838 |
| `0x428202` | `ff152cb05e00a39896650083c424c390` | [未取得] | build/wiz8.asm:53666 |
| `0x428220` | `ff152cb05e002b0598966500c3909090` | [未取得] | build/wiz8.asm:53685 |
| `0x43bbbf` | `ff152cb05e008b4c241003f889be5403` | [未取得] | build/wiz8.asm:80286 |
| `0x440734` | `ff152cb05e000fbe4c241333d2f7f18b` | [未取得] | build/wiz8.asm:86143 |
| `0x44b5a6` | `ff152cb05e008946688a46703c017404` | [未取得] | build/wiz8.asm:100855 |
| `0x470ae2` | `ff152cb05e008b4c2434898140040000` | [未取得] | build/wiz8.asm:150825 |
| `0x473ade` | `ff152cb05e00898640040000293de8a0` | [未取得] | build/wiz8.asm:154785 |
| `0x480d1a` | `ff152cb05e0033d2f7f733c089442410` | [未取得] | build/wiz8.asm:172335 |
| `0x482000` | `ff152cb05e00a354a16500c390909090` | [未取得] | build/wiz8.asm:173974 |
| `0x482027` | `ff152cb05e00a354a165008bc65ec390` | [未取得] | build/wiz8.asm:173986 |
| `0x482143` | `ff152cb05e008b1554a165008bc82bca` | [未取得] | build/wiz8.asm:174092 |
| `0x4825b4` | `ff152cb05e008b0da8b965003bc17305` | [已驗證] 共用遊戲時間 | build/wiz8.asm:174458 |
| `0x482729` | `ff152cb05e00a3a8b96500c390909090` | [已驗證] 共用時間基準重設 | build/wiz8.asm:174577 |
| `0x482949` | `ff152cb05e008b0da8b965003bc17305` | [未取得] | build/wiz8.asm:174757 |
| `0x482a77` | `ff152cb05e00a3a8b96500a1f7746800` | [未取得] | build/wiz8.asm:174867 |
| `0x482ebd` | `ff152cb05e008b0da8b965003bc17305` | [未取得] | build/wiz8.asm:175178 |
| `0x4834bc` | `ff152cb05e008b0da8b965003bc17305` | [未取得] | build/wiz8.asm:175692 |
| `0x48356c` | `ff152cb05e008b0da8b965003bc17305` | [未取得] | build/wiz8.asm:175746 |
| `0x4842a3` | `ff152cb05e00d944241cd81d34bb5e00` | [未取得] | build/wiz8.asm:176779 |
| `0x484304` | `ff152cb05e008b0dbcb965008be82bc1` | [未取得] | build/wiz8.asm:176820 |
| `0x484cd5` | `ff152cb05e006a1089466c895e70c746` | [未取得] | build/wiz8.asm:177706 |
| `0x484f72` | `ff152cb05e006a1089466c895e70c746` | [未取得] | build/wiz8.asm:177964 |
| `0x485139` | `ff152cb05e0089476c8b55708957708b` | [未取得] | build/wiz8.asm:178125 |
| `0x48540a` | `ff152cb05e0089466c5ec20400909090` | [未取得] | build/wiz8.asm:178370 |
| `0x485645` | `ff152cb05e002b466c89442410895c24` | [未取得] | build/wiz8.asm:178593 |
| `0x48f321` | `ff152cb05e008b4e188941288b5618c7` | [未取得] | build/wiz8.asm:192591 |
| `0x48f99c` | `ff152cb05e008b0d74ba6500a378ba65` | [未取得] | build/wiz8.asm:193139 |
| `0x48fc9d` | `ff152cb05e008b0d74ba650048a378ba` | [未取得] | build/wiz8.asm:193400 |
| `0x49c3d8` | `ff152cb05e008b4c24148944240c895c` | [未取得] | build/wiz8.asm:208807 |
| `0x49c751` | `ff152cb05e0089442408c744240c0000` | [未取得] | build/wiz8.asm:209062 |
| `0x49cbac` | `ff152cb05e00894424188b8544020000` | [未取得] | build/wiz8.asm:209456 |
| `0x49d0e3` | `ff152cb05e00897c240c89442408df6c` | [未取得] | build/wiz8.asm:209844 |
| `0x4a9c20` | `ff152cb05e008b4c2404894128c39090` | [未取得] | build/wiz8.asm:227512 |
| `0x4aa219` | `ff152cb05e008bf8385e1c757b385e3a` | [未取得] | build/wiz8.asm:228145 |
| `0x4bb07f` | `ff152cb05e008b47483bc3747e8b4f50` | [未取得] | build/wiz8.asm:251199 |
| `0x4c269d` | `ff152cb05e008b96080200008b8e0c02` | [未取得] | build/wiz8.asm:260173 |
| `0x4c2779` | `ff152cb05e0068d00700008986080200` | [未取得] | build/wiz8.asm:260226 |
| `0x4c3906` | `ff152cb05e0033d2f7f588542413eb1a` | [未取得] | build/wiz8.asm:261667 |
| `0x4ea0c5` | `ff152cb05e006a00894634e83b020000` | [已驗證] 戰鬥結束事件 | build/wiz8.asm:314074 |
| `0x50526b` | `ff152cb05e0089463480bd8a02000001` | [未取得] | build/wiz8.asm:351625 |
| `0x524da9` | `ff152cb05e008b15b8c468008b0db4c4` | [未取得] | build/wiz8.asm:397336 |
| `0x524de3` | `ff152cb05e008b15b8c468008b0db4c4` | [未取得] | build/wiz8.asm:397352 |
| `0x5251da` | `ff152cb05e00a3b8c46800c705b4c468` | [未取得] | build/wiz8.asm:397639 |
| `0x5253f8` | `ff152cb05e005f5e5da3b8c468005b81` | [未取得] | build/wiz8.asm:397817 |
| `0x525daa` | `ff152cb05e002b05f0c468003df40100` | [未取得] | build/wiz8.asm:398551 |
| `0x527379` | `ff152cb05e00a3b8c46800e9a70e0000` | [未取得] | build/wiz8.asm:400226 |
| `0x527478` | `ff152cb05e00a3b8c468008b4c24108b` | [未取得] | build/wiz8.asm:400305 |
| `0x52750f` | `ff152cb05e00a3b8c468008b431050e8` | [未取得] | build/wiz8.asm:400345 |
| `0x5275c4` | `ff152cb05e00a3b8c468008b4b1051e8` | [未取得] | build/wiz8.asm:400394 |
| `0x527642` | `ff152cb05e00a3b8c468008b531052e8` | [未取得] | build/wiz8.asm:400432 |
| `0x52a190` | `ff152cb05e00a3e39a6800c390909090` | [未取得] | build/wiz8.asm:404101 |
| `0x52e01b` | `ff152cb05e002bc33bc70f861f010000` | [已驗證] 上述事件消費端；非戰鬥專用性未證 | build/wiz8.asm:410033 |
| `0x52f1bb` | `ff152cb05e0089463483c7044d75b25f` | [未取得] | build/wiz8.asm:411663 |
| `0x52f3d0` | `ff152cb05e008947348b1544c56800a1` | [未取得] | build/wiz8.asm:411864 |
| `0x53eeda` | `ff152cb05e00894734eb438a4c242884` | [已驗證] 攻擊事件 | build/wiz8.asm:433506 |
| `0x54373b` | `ff152cb05e00894334eb0a6a09e88373` | [已驗證] 攻擊結果事件 | build/wiz8.asm:439243 |
| `0x55f577` | `ff152cb05e008b15cced680089827402` | [未取得] | build/wiz8.asm:478589 |
| `0x566207` | `ff152cb05e008b0dcced68008981fc02` | [未取得] | build/wiz8.asm:486559 |
| `0x566459` | `ff152cb05e008b15cced68008982fc02` | [未取得] | build/wiz8.asm:486706 |
| `0x56683c` | `ff152cb05e008b0dcced68008981fc02` | [未取得] | build/wiz8.asm:487013 |
| `0x566900` | `ff152cb05e008b0dcced68008981fc02` | [未取得] | build/wiz8.asm:487060 |
| `0x566b7b` | `ff152cb05e008b0dcced68008981fc02` | [未取得] | build/wiz8.asm:487235 |
| `0x566c47` | `ff152cb05e008b0dcced68008981fc02` | [未取得] | build/wiz8.asm:487285 |
| `0x566fb2` | `ff152cb05e008b0dcced68008981fc02` | [未取得] | build/wiz8.asm:487540 |
| `0x567019` | `ff152cb05e008b0dcced68008981fc02` | [未取得] | build/wiz8.asm:487567 |
| `0x567124` | `ff152cb05e008b0dcced68008981fc02` | [未取得] | build/wiz8.asm:487659 |
| `0x5671f2` | `ff152cb05e008b0dcced68008981fc02` | [未取得] | build/wiz8.asm:487701 |
| `0x569c88` | `ff152cb05e008b0dcced68008981fc02` | [未取得] | build/wiz8.asm:491202 |
| `0x569cd7` | `ff152cb05e008bc8a1cced68002b88fc` | [未取得] | build/wiz8.asm:491231 |
| `0x56b5cc` | `ff152cb05e00a3b8ed6800c605caed68` | [未取得] | build/wiz8.asm:493222 |
| `0x56b634` | `ff152cb05e008bf0a0caed680084c074` | [未取得] | build/wiz8.asm:493259 |
| `0x56b679` | `ff152cb05e00a3b8ed6800c605caed68` | [未取得] | build/wiz8.asm:493278 |
| `0x56b6fc` | `ff152cb05e008b0db8ed68008d91f401` | [未取得] | build/wiz8.asm:493318 |
| `0x56b887` | `ff152cb05e005f5ea3b8ed68008a4424` | [未取得] | build/wiz8.asm:493421 |
| `0x56e779` | `ff152cb05e002b0578ee68003df40100` | [未取得] | build/wiz8.asm:497115 |
| `0x56ec0e` | `ff152cb05e00894634e8947a0000a11c` | [未取得] | build/wiz8.asm:497437 |
| `0x573080` | `ff152cb05e00a378ee6800a11c9f6400` | [未取得] | build/wiz8.asm:502477 |
| `0x57e8a3` | `ff152cb05e008b1568f268008982f400` | [未取得] | build/wiz8.asm:517467 |
| `0x587598` | `ff152cb05e008946346a006888bb6400` | [未取得] | build/wiz8.asm:528644 |
| `0x587676` | `ff152cb05e008946346a006888bb6400` | [未取得] | build/wiz8.asm:528704 |
| `0x58a4f7` | `ff152cb05e008946346a0068f0bc6400` | [未取得] | build/wiz8.asm:532759 |
| `0x58a640` | `ff152cb05e00894634a1486a68006a00` | [未取得] | build/wiz8.asm:532850 |
| `0x590f6d` | `ff152cb05e008b15c8b7690089824001` | [未取得] | build/wiz8.asm:541541 |
| `0x5915ac` | `ff152cb05e008b0dc8b769008bb94001` | [未取得] | build/wiz8.asm:542025 |
| `0x591685` | `ff152cb05e008b15c8b7690089824001` | [未取得] | build/wiz8.asm:542085 |
| `0x5a6774` | `ff152cb05e005fa30cc169005e83c414` | [未取得] | build/wiz8.asm:569584 |
| `0x5a67a8` | `ff152cb05e002b050cc169008b0d18c1` | [未取得] | build/wiz8.asm:569609 |
| `0x5bc3e9` | `ff152cb05e008b8c24300100005ea39c` | [未取得] | build/wiz8.asm:599691 |
| `0x5bc593` | `ff152cb05e008bc82b0d9cc46900b8d5` | [未取得] | build/wiz8.asm:599848 |
| `0x5d56ed` | `ff152cb05e0033d2b9e8030000f7f181` | [未取得] | build/wiz8.asm:635138 |

#### GetTickCount 取址後的間接呼叫

以下分開列出取址與後續已追到的呼叫位置；這些呼叫的完整跨函式戰鬥可達性均 [未取得]。這不是全程式函式指標分析的替代品。

| IAT 取址 VA（來源） | 暫存器 | 已辨識的實際呼叫 VA（來源） |
|---|---|---|
| `0x408e9d`（build/wiz8.asm:11088） | EBX | `0x408ead`（build/wiz8.asm:11090）、`0x408ec8`（build/wiz8.asm:11098） |
| `0x409313`（build/wiz8.asm:11499） | EDI | `0x409325`（build/wiz8.asm:11504）、`0x40932c`（build/wiz8.asm:11507） |
| `0x409362`（build/wiz8.asm:11533） | EDI | `0x409368`（build/wiz8.asm:11534）、`0x409380`（build/wiz8.asm:11541） |
| `0x426eca`（build/wiz8.asm:51962） | EDI | `0x426ed0`（build/wiz8.asm:51963）、`0x426f07`（build/wiz8.asm:51977） |
| `0x4827b4`（build/wiz8.asm:174640） | EDI | `0x4827c0`（build/wiz8.asm:174642）、`0x48286c`（build/wiz8.asm:174690） |
| `0x48299c`（build/wiz8.asm:174786） | ESI | `0x4829a9`（build/wiz8.asm:174788）、`0x4829b9`（build/wiz8.asm:174793） |
| `0x485525`（build/wiz8.asm:178482） | EDI | `0x485559`（build/wiz8.asm:178497）、`0x485573`（build/wiz8.asm:178504） |
| `0x48fa1b`（build/wiz8.asm:193180） | ESI | `0x48fa21`（build/wiz8.asm:193181）、`0x48fa86`（build/wiz8.asm:193212） |
| `0x4c302d`（build/wiz8.asm:260941） | EDI | `0x4c3033`（build/wiz8.asm:260942）、`0x4c3040`（build/wiz8.asm:260946） |
| `0x4c7402`（build/wiz8.asm:266781） | EDI | `0x4c7415`（build/wiz8.asm:266784）、`0x4c742e`（build/wiz8.asm:266788） |
| `0x50d58c`（build/wiz8.asm:363726） | ESI | `0x50d594`（build/wiz8.asm:363728）、`0x50d5e1`（build/wiz8.asm:363750）、`0x50d5f0`（build/wiz8.asm:363754） |
| `0x525943`（build/wiz8.asm:398225） | ESI | `0x525949`（build/wiz8.asm:398226）、`0x525987`（build/wiz8.asm:398239） |
| `0x525b78`（build/wiz8.asm:398378） | ESI | `0x525b92`（build/wiz8.asm:398382）、`0x525b99`（build/wiz8.asm:398384） |
| `0x5268a1`（build/wiz8.asm:399430） | ESI | `0x5268ba`（build/wiz8.asm:399434）、`0x5268c1`（build/wiz8.asm:399436） |
| `0x526997`（build/wiz8.asm:399498） | EDI | `0x5269b0`（build/wiz8.asm:399502）、`0x5269b7`（build/wiz8.asm:399504） |
| `0x5272ef`（build/wiz8.asm:400192） | ESI | `0x527309`（build/wiz8.asm:400196）、`0x527310`（build/wiz8.asm:400198） |
| `0x52914a`（build/wiz8.asm:402699） | ESI | `0x529164`（build/wiz8.asm:402703）、`0x52916b`（build/wiz8.asm:402705） |
| `0x529231`（build/wiz8.asm:402767） | ESI | `0x52924b`（build/wiz8.asm:402771）、`0x529252`（build/wiz8.asm:402773） |
| `0x529589`（build/wiz8.asm:403106） | ESI | `0x5295a3`（build/wiz8.asm:403110）、`0x5295aa`（build/wiz8.asm:403112） |
| `0x562588`（build/wiz8.asm:482114） | ESI | `0x5625c2`（build/wiz8.asm:482131）、`0x562615`（build/wiz8.asm:482150） |
| `0x562cef`（build/wiz8.asm:482575） | ESI | `0x562cf7`（build/wiz8.asm:482578）、`0x562d4d`（build/wiz8.asm:482600） |
| `0x568c4e`（build/wiz8.asm:489923） | ESI | `0x568c5e`（build/wiz8.asm:489927）、`0x568c65`（build/wiz8.asm:489929） |
| `0x57f8c0`（build/wiz8.asm:518594） | ESI | `0x57f8c6`（build/wiz8.asm:518595）、`0x57f8f4`（build/wiz8.asm:518607） |
| `0x5910ac`（build/wiz8.asm:541645） | ESI | `0x5910b2`（build/wiz8.asm:541646）、`0x59111d`（build/wiz8.asm:541674） |
| `0x591428`（build/wiz8.asm:541900） | ESI | `0x59142e`（build/wiz8.asm:541901）、`0x591443`（build/wiz8.asm:541906） |

## 規格疑慮

- 指定機檢只驗證「發現」各條的來源格式與三態標記，不驗證位址、API 清單完整性、簽章正確性或加速效果；本報告另以 asm、PE 匯入表與 xxd 核對靜態證據，未執行遊戲。
- 「API 各自全部呼叫點／哪些在戰鬥路徑」只完整盤點本 exe 明示的 IAT 引用，另補已追到的暫存器與 SR 虛擬呼叫；未完成動態載入、跨函式指標傳遞與所有 DLL 內部呼叫的可達性，已標 [未取得]。
- 訊息顯示期限已找到，但未證實是下一行動的阻塞原因；第二候選僅為呈現選配。首選只縮短戰鬥行動前等待，尚不等於攻擊／施法／怪物動畫等比例加速。
- `build/decomp/` 缺本次關鍵戰鬥／動畫函式的完整反編譯；未跑 `re/run.sh`。需要補 `0x4e8c30`、`0x4e8ea0`、`0x4e9490`、`0x4ebfe0`、`0x53e250`、`0x4c4660` 及動畫逐幀更新鏈、`0x500d90` 相關效果就緒寫入鏈；設定文案需補 UI 字串資源對照。這些缺口不阻止交付已證實的局部 duration 候選。

# 原版函式 × calc 節點重放

`bash re/emu/run.sh 00517970`、`bash re/emu/run.sh 00552250` 從唯讀
`build/Wiz8.exe` 映射 PE32 headers 與各節區的 raw data 至 ImageBase，保留
.data 初值、零填 virtual tail。堆疊、arena、import 樁區另外映射。
不啟動 Windows 程序，不執行遊戲入口，不下載依賴。
啟動器以自身位置找 repo，使用 `build/venv/bin/python -I`，不存在時改用
`~/.local/state/wiz8-mod-emu/venv/bin/python -I`，不受 cwd 影響。

## case 格式

檔名為 `re/emu/cases/<8 位小寫 hex>.json`；位址由檔名決定。
新增公式時新增 case 與 `internal/calc/nodes.go` 的 Nodes 登記／adapter 即可；
Python 不包含種子公式或依節點位址分派的記憶體配置。

| 欄位 | 定義 |
| --- | --- |
| `cc` | `cdecl`、`stdcall`、`thiscall`；回傳時檢查 esp。後兩者由 callee 清掉 args。 |
| `objects` | 名稱 → arena 的 byte offset；arena 起點 0x20000000、大小 1 MiB，物件不得重疊。 |
| `args` | 依序放在入口 esp+4、esp+8… 的 32 位參數；整數或 `"@物件名"` 指標。 |
| `ecx` | 選用，初始 ecx（thiscall 的 this）；整數或物件指標。 |
| `setup` | 固定寫入欄位清單；每項使用 `at`、`size`、`signed`、`value`，value 可用物件指標。 |
| `in` | 輸入欄位清單，順序等於邊界列、random ranges 與 Go Node.In。 |
| `out` | 輸出欄位清單，順序等於 Go Node.Out。 |
| `boundary` | 明列的輸入陣列清單，含零、端點及控制流程邊界。 |
| `random` | `seed`、`count`、`ranges`；ranges 是各輸入的 `[含下限,含上限]`，用 Python 固定種子 Random 產生。 |
| `steps` | 每次從入口重放可執行的最大指令數，含子函式；到限未回傳就 exit 1。 |
| `path_limit` | 每組輸入允許全列舉的最大完整路徑數。 |
| `samples` | 超過 path_limit 後，改輸出多少條固定種子抽樣路徑。 |

輸入／輸出欄位有 `name`、`at`、`size`（byte，預設 4）、`signed`（預設 false）。
`at` 使用以下其中一種：

- `{"arg":0}`：第一個參數的 stack slot。
- `{"reg":"ecx"}` 或 `{"reg":"eax"}`：暫存器；輸出通常用 eax。
- `{"reg":"st0"}`：僅供 `out`，必須 `size:8`；將 x87 ST0 轉為 IEEE754
  double，再將 64 位元以有號 int64 輸出，忽略 `signed`，不彈出 ST0。
- `{"object":"dice","offset":2}`：arena 物件欄位；offset 預設 0。
- `{"object":"chars","offset":4,"index":"who","stride":"0x1234","count":8}`：
  依同一列輸入 `who` 的值尋址（基址＋offset＋who×stride），用來讓原版讀「那一筆」記錄。
  `index`、`count` 必填；index<0 時該欄位不寫、讀則報錯（無記錄）；index≥count 報錯。
  物件在 arena 內要留足 count×stride 的空間、不與其他物件重疊——emu 不檢查重疊，靠 case 作者。
  `also`（同值複寫到固定幾筆）讀不出「讀錯人／讀錯筆」，新 case 用 index（grok zoWN 發現 1–3）。
- `{"global":"0x685174"}`：絕對全域位址。

數值可用 JSON 整數或 `"0x…"` 字串。輸入可加 `also` 欄位清單，把同一值
寫入其他位置；本票用來同步角色與怪物模板的抗性／等級。
case 可加 `"stubs": {"<import 名>": {"ret": <物件名或整數>, "pop": <callee 清的 bytes>}}`：該 import 直接回傳指定值，
用在語意依情境而定、不宜寫成通用樁的查表（例 SR.DLL 的 srRegistry::getClassNode）。要寫入參數指的結構（輸出參數）時加 `"write": [{"arg": <第幾個堆疊參數，1 起，thiscall 不算 ecx>, "offset": <bytes>, "size": <bytes>, "value": <整數、"@物件" 或 {"at": <位置>}>}]`，依序寫入、超出 size 的高位截掉；同一樁有呼叫者傳 NULL 輸出指標（原函式自己判空）時該項加 `"nullskip": true`，指標為 0 即不寫（2026-10-10，4ef7e0 經 51c300 呼叫 51f900 第 3 參數為 0）；`{"at"}` 從該位置讀 size bytes，位置可由 case 輸入寫，讓差分涵蓋樁寫出的值（2026-10-10，4f88f0 的 51c020 記錄 id）。回 double 的 import（mangled 回傳型別 `N`，例 srNode::getLocationX `QBENXZ`）加 `"st0": <數值字串>` 或 `"st0": {"at": <位置>}`（從該位置讀 8 bytes double，位置可由 case 輸入寫，讓差分涵蓋），像 fld 一樣推上 x87 堆疊（例 GlobalMemoryStatus 的 MEMORYSTATUS、srPixelConvert::mapPixelFormat 的 PixelFormat&）。內建樁：malloc／operator new（bump heap，
每路徑歸零）、free／delete no-op、fopen 一律 NULL（不模擬檔案 I/O）、sprintf／vsprintf（共用 vswprintf 格式器，窄字 Latin-1）、
srTimer::getMsTime 回 case 的 tick。
位址樁：`stubs` 的鍵寫 `"0x<位址>"` 時換掉 exe 內部函式，欄位同 import 樁。**只准換公式登記簿分類為「排除」的函式**（公式邊界外的引擎流程，例 544630 建彈道），**或深度已 D4 的「公式」類**（MG 2026-10-09 裁甲，`decisions/2026-10-09-compositional-stubs.md`；回傳值取自 case 輸入位置）；
例外：404ea0（存檔串流讀取）准樁，case 以 `stream` 欄位依序供位元組（MG 2026-10-09，`decisions/2026-10-09-io-and-presentation-stubs.md`）。其餘公式類、非公式類或未登記的位址在 Machine 建構時拒絕（`--selftest-stubs` 驗）。被換掉的呼叫靠兩個欄位留下可比對的輸出：
`"record": [{"arg": <第幾個堆疊參數，1 起>, "size": <bytes>, "at": <位置>}]` 把參數值抄到該位置，`"count": {"at": <位置>}` 是 u32 呼叫次數；
兩者寫在 case 的輸出位置裡，calc 照同樣規則算出「會用什麼參數呼叫幾次」。
record 項加 `"n": <次數上限>` 時第 k 次呼叫寫 `at+k*size`，每次呼叫的參數都留下（不加 n 只留末次，非末次成員的參數看不到）；超過 n 次停下報錯（2026-10-10，52b160／54e3f0）。
同一函式在一次執行內要回不同值時，`"ret": {"at": <位置>, "n": <次數上限>}` 第 k 次呼叫讀 `at+4k`（依呼叫序的 case 輸入，重放從頭）；超過 n 次停下報錯，不回舊值。calc 照呼叫序取同一串值。
初始化順序為 args/ecx、setup、in；同位置後寫者覆蓋前寫者。
欄位寬度／signed 必須能容納輸入值，越界直接失敗。

00517970 的輸入為 Z（i16）、N/S（u8），使用 cdecl 的骰結構指標，輸出有號 eax。
00552250 的輸入為 value、Rj、Lt、S、targetType；原版第一參數是可修改的
value 指標，輸出讀取該欄位，**不是 eax**。targetType=3 走怪物，1/2 走
原版的非怪物分支。領域固定 0、角色索引固定 0、怪物額外抗性固定 0。
case 自建單一怪物列表與已載入模板，讓 0x4e5550、0x4e5620、0x4e5720
及列表子函式直接執行；沒有將它們換成公式樁。
資料結構依據：build/wiz8.asm:307807、307890、307977、378002、459320、653823、654256。

## 亂數、隔離與產物

每組輸入從空前綴做 DFS，遇到下一個 R(n) 就分支 0..n−1，R(0) 特例為唯一
回傳 0。每個前綴均從原版入口重新執行，還原 CPU context 與曾寫入的記憶體頁，
避免前一路徑留下全域或物件變動。原版子呼叫照常執行。
frontier 的最少葉數超過 path_limit 時丟棄該組 DFS 暫存結果，以
`random.seed + 輸入列索引` 重建固定種子，抽 samples 條完整路徑；每次 R(n)
在其支集中均勻抽樣。這些行全部標 sampled=true，不與先前 DFS 結果混合。
抽樣 trace 不是原版 RNG 機率分布的估計器。

`build/emu/<addr>.jsonl` 每條完整路徑一行：

```json
{"in":[2,1,6],"rng":[[6,3]],"out":[6],"sampled":false}
```

`rng` 包含所有呼叫的 `[n,回傳值]`，包含 R(0)，無呼叫時為空陣列。
`.hit` 僅記錄已輸出完整路徑命中的本函式基本塊；與 funcs.tsv 第 7 欄
交集，一行一個小寫、不帶 0x、不補零的 hex。子函式執行但其塊不列入此檔。
開始執行節點時先刪其舊 jsonl/hit/d3，避免失敗沿用舊證據。

`TestEmu` 掃描所有 jsonl；檢查輸入／輸出數目、每次 n、亂數值範圍、呼叫
總次數及 out。每檔寫 `cases=N diff=K` 至 .d3；格式錯誤、未知節點、空檔也失敗。
無檔案時印 SKIP。`EMU_WRONG=1` 啟用獨立 TestEmuWrong，對骰子結果加 1，
確認差分捕獲錯誤且不改 .d3。
`TestEmu` 的 JSON 直接解碼至 `emuRow.Out []int64`（In 亦同），沒有經過
float64 中介，因此超過 2^53 的 double 位元整數仍精確；Go 解碼不需修改。

## 樁清單與 selftest

- 所有正常 PE import descriptor 的 IAT 槽都改指向獨立攔截位址，保留槽位／DLL／名稱。
  不支援的 import 被呼叫立即非零退出，stderr 為
  `unstubbed import 0x<IAT位址> <dll>!<name>`；不使用靜默回傳零的預設樁。
- `_ftol`：讀 x87 的實體 TOP 暫存器（80 位尾數／指數），以整數移位朝零截斷，
  回傳 edx:eax，標空原 ST0 並推進 TOP。初始 x87 control word 為 0x037f、空堆疊。
  NaN／Infinity／超出 int64 明確報錯；不模擬 CRT 的無效轉換例外細節。
- `qsort`：Python 重現 MSVCRT qsort（CUTOFF 8 的 shortsort＋三點取中分割、顯式堆疊），
  每次比較把 (a, b) 推上堆疊、呼叫原版比較函數，返回點 QCONT 收 eax（有號）。不支援巢狀 qsort（報錯）。
  `--selftest-qsort` 只斷言排序結果；**等值鍵的相對順序與比較呼叫序列未對使用者的 msvcrt.dll 驗證**
  （注入「改切換門檻」「改 higuy 掃描條件」都不紅，因結果仍是排序好的）。節點若依賴等值鍵順序，差分會在那裡紅。
- `vswprintf`（舊 MSVCRT，無 count）：`%d %i %u %x %X %c %s %S %%`、旗標 `-` `0`、寬度、`h`/`l`；`%s` 寬、`%S` 窄。
  窄字串一律 Latin-1，不模擬 locale／碼頁；精度、`*`、其他轉換明確報錯；輸出上限 0x100000 UTF-16 單位。
- `wcscpy`（cdecl）：複製 UTF-16LE 字串及結尾零，eax 回傳 dst；使用相同的
  0x100000 code-unit 掃描上限（須在上限內找到結尾零），保留未配對 surrogate。
  呼叫端負責目的緩衝區容量；重疊來源／目的不在 CRT 定義的支援範圍。
  寫入經隔離頁追蹤，未映射存取使用既有錯誤診斷。
- `wcscat` `wcslen` `wcscmp` `strchr` `atoi` `_stricmp` `strncpy` `_strnicmp`（cdecl，`STR_FUNCS`）：C 語意；
  wide 依 UTF-16 code unit 比、narrow 一律 Latin-1；比較回 -1／0／1（C 只保證符號）；`_stricmp` 系照 C 區域只折 A-Z（exe 無 setlocale；0x80 以上不折）
  （非 ASCII 字母的大小寫折疊與 MSVCRT C locale 可能不同，未驗證）；`atoi` 溢位環繞不飽和。
- `_CIpow`（x87 內部呼叫，無堆疊參數）：ST1 = ST1 ** ST0，彈出一次，結果留 ST0；以 binary64 `math.pow` 計算
  （原版保留 80-bit 中間值，末位可能差 1 ulp，未驗證）；負底數配非整數指數等定義域錯誤回 NaN。
- `SR.DLL!srNode::srNode(srNode*)`（thiscall，1 個參數＝父節點，callee 清 4）：eax 回傳 ecx，
  不寫物件、不掛進父節點（節點樹不模擬）。用到節點樹內容的節點要另寫 case。
- `SR.DLL!srClass::srClass()`（protected 預設建構子，thiscall，無參數）：
  eax 回傳 ecx（this），不寫物件記憶體。五個呼叫點返回後皆由衍生端寫入
  vtable，依據 build/wiz8.asm:56188、56190、80207、80208、197162、197163、
  276995、276996、277073、277074；衍生欄位自行初始化的依據為
  build/wiz8.asm:80235、80236、80237、277023、277024、277033、277100、277101、277106。
  可見呼叫端未依賴基底建構子的欄位寫入；此樁限於這些呼叫點所需語意，
  不宣稱重現 SR.DLL 內部註冊或其他副作用。
- x87 ST0 輸出：依 FPSW.TOP 選實體 80 位暫存器，以整數運算直接捨入至
  double（最近值，平手取偶數），涵蓋次正規值、帶符號零、溢位至 Infinity。
  NaN 保留符號與高位 payload 並 quiet；空堆疊與非正規 unnormal 編碼報錯。
  不模擬浮點例外或依 control word 改變輸出捨入模式；僅提供 8 byte 輸出。
- `GetTickCount`：回 case 檔的 `"tick"`（u32，預設 0），同一次執行每次相同。目前不是輸入欄，Go 節點要的話得另議。
- FS 段：GDT 映射一塊 TIB，`fs:[0]` 初值 0xffffffff（SEH 鏈尾）、`fs:[0x18]` 指回 TIB，可寫。
- `SR.DLL!srAssertFail`：停止並報 `srAssertFail stop <檔名>:<行號>`；case 要避開 assert 路徑，assert 專屬塊登記 cov-exempt。
- 未映射讀寫／取指：錯誤訊息 `unmapped eip=0x… addr=0x…`。
- `--selftest-stubs`：合成呼叫端驗 Y2 項目及 Y3 的 wcscpy 內容／結尾零／dst
  回傳、srClass 的 this 回傳／記憶體保持、3.5 經 case `out` 讀 ST0 的位元輸出。
- 0x40efa0：以本路徑 RNG 前綴取代，記錄 n/value；其餘遊戲函式無樁。

`--selftest-import` 執行 **原版 0x40efa0，n=6，停用該 RNG 樁**。
它直接抵達未實作的 CRT rand import，預期 stderr：
`unstubbed import 0x5eb1d0 MSVCRT.DLL!rand`，exit 1。
這不是直接跳到合成 import stub 的測試。

## Go 節點與 calc f

Nodes 以八位位址為鍵，Node 有 Addr、In、Out 與 F；F 接收 `[]int64` 和
`Rand.R(n int) int`，回傳所有輸出。NodeDist 回傳 joint output distribution，
相同輸出合併，按輸出值排序。每個分支權重是所有 32768 個 CRT rand 整數
經 `floor(r*n/32767) mod n` 的實際計數除以 32768，R(0) 權重為 1。
因此 CLI 顯示的分布不是均勻骰近似。

骰子 adapter 使用 DiceDist 的 S=0 確定性結果給出 Z 與每骰固定的 +1，
再加入重放 R(S)。抗性 adapter 保留兩個亂數呼叫與 FR 上限，用既有 Resist
的 ER=1 確定性特化共享百分比取整：該特化恰削減輸入值的 floor(1%)，
所以把 v*FR 交給它即可取得本次削減量。乘積恰為 9999 時，使用 9998
取得相同的 floor(1%)，避開 Resist 的 sentinel。v=9999 或 ER≤0 直接使用
既有 Resist。不另維護一套傷害取整公式；測試將 adapter 的完整均勻路徑分布
與現有 Resist 對照，含 sentinel、FR=100、ER=199/200 與最大允許 value。

```text
calc f 00517970 2 3 6
calc f 00552250 101 100 1 1 3
```

輸出先印 dist，每行 `值: 機率`，再印各輸出平均及輸入／輸出名稱。
多輸出節點以逗號分隔 joint values。

## 驗證與已知上限

本票驗證：兩節點 702／2341 條路徑重放 diff=0，本函式基本塊 3/3、9/9。
兩個 case 都含完整列舉與固定種子抽樣，steps=20000、path_limit=512、samples=64，
各另產生 24 組固定種子隨機輸入。D4 代表這批輸入走遍本函式基本塊，不代表
所有值、所有子函式基本塊或所有遊戲狀態都已驗證。

验收命令為 `bash scripts/emu.test.sh`。此沙箱的預設 Go cache 唯讀，實際使用：

```sh
GOCACHE="$PWD/build/go-cache" bash scripts/emu.test.sh
```

另驗證 calc 套件／CLI 測試、_ftol 的 ±3.75、±4294967297.75、0 的截斷及
彈棧、一步指令上限失敗、從不同 cwd 啟動及輸出重現性。

已知限制：僅 PE32/x86、正常 import directory，沒有 Windows loader、TLS callback、
DLL 實作或 delay-import loader；映射區目前允許讀寫執行，不用 PE 節區權限抓非法存取。
stack 64 KiB、arena 1 MiB，case 負責物件大小與有效配置。呼叫約定不涵蓋 fastcall／varargs
thiscall。_ftol 不涵蓋無效浮點轉換。Python 每個 case 的產物暫存於記憶體，沒有總輸入數
或整體 wall-clock 限制；每次重放有硬性指令上限，超限不會靜默取樣。

NodeDist 最多一百萬次 DFS 節點重放、256 層 RNG、n=0..65535，超限回錯誤，
不回部分分布；所以巨大骰組可能無法用 calc f 列出分布。機率以 float64 累加，
計數源自精確 CRT 整數映射，顯示值仍有浮點捨入。
抗性節點目前接受 value=0..42949672、Rj/Lt/S=0..255、targetType=1..3，
避免原版 32 位乘法溢位並限定本票合成目標可表示的範圍；不宣稱支援任意 u32、
其他領域、負怪物額外抗性或尚未載入模板的遊戲狀態。

Y3 驗證：`--selftest-stubs` 印出三行指定結果；另以合成資料驗 ST0 正負值、
帶符號零、實體 TOP、重放、平手取偶數、次正規值（含避免雙重捨入的邊界）、
溢位與空堆疊拒絕。`GOCACHE=/tmp/wiz8-y3-go-cache go test -count=1 ./internal/calc`
通過。

## 規格疑慮

- Y3：派工時 worktree 漏複製 `build/re/funcs.tsv`（cc 派工缺陷），執行方照禁止項停手；cc 補檔後 `bash scripts/emu.test.sh` 為 ok emu。
- 字面要求 `Dist(node, in)` 與現有 `calc.Dist` 型別同名，Go 不允許同套件型別／函數
  同名，且 calc.go 不在白名單；實作命名為 `NodeDist(node, in)`。
- 「完整分布」受上述 NodeDist 資源上限限制，超限明確回錯誤；Python 大路徑數則依規格抽樣。
- 驗收腳本本身不檢查 D4 全塊覆蓋與 _ftol，故另行比對 funcs.tsv 並測試浮點樁；
  兩個種子目前均全塊命中。

## G6：F 原生鏈比對

`bash scripts/node-native.sh 54df00`（可列多個位址）呼叫
`bash re/emu/run.sh --native 0054df00`，再執行該位址的 `TestFNative`。
原生模式依登記簿移除所有「公式」類位址樁；排除、IO 與 import 樁照常。
RNG 仍由同一套路徑列舉／固定種子抽樣提供。輸出僅在 `build/emu-f/`，
開始前刪除該位址的舊 jsonl、hit、d3f，失敗不能沿用舊成功證據。

每列 jsonl 保留原 `in`、`out` 順序，另有 `compare` 布林陣列（與 out 等長）。
只有被移除公式樁的 record／count 儲存區不比較；按該列解析後的實際位址與
大小判斷，包含 n 個連續儲存格、物件別名、index 尋址。部分重疊直接報錯。
公式樁 write 的真正遊戲狀態目的地仍然比較，不列入 record 遮罩。
`unused_in` 列出完全落在移除樁 ret／st0／write.value 來源儲存區的輸入索引，
其 also 目的地也須落在來源區；這是樁夾具來源位置的標記，輸入值仍保留並寫入，
不拿它縮減 random 或 boundary。這些儲存區應由 case 分配為樁專用區域。

可選的 `re/emu/native/<addr>.json` 僅接受 objects、setup、stubs：
objects 與 stubs 追加新鍵（拒絕覆蓋既有鍵）；setup 追加，依原順序在 in 前寫入，
所以可覆寫原 setup 的固定世界狀態，但不能蓋過本列輸入。不允許修改輸入／輸出
定義、boundary、random、步數或抽樣設定。新增位址樁同樣受登記簿守衛約束。

`TestFNative` 直接呼叫 Nodes[addr].F，驗輸入、輸出、遮罩長度與至少一個比較欄；
panic、輸出差異、RNG 多叫／少叫／範圍或 n 不符、空或壞 jsonl 都失敗。
每檔寫 `cases=N diff=M` 至 `.d3f`，沒有 jsonl 才 Skip。

54df00 試點目前未收貨：原 207 個輸出中比較 157 欄，50 欄為兩個公式樁的
record/count；8 個 decayPost 輸入只作已移除樁的 write.value 來源。
原 case 的四個位址樁實際分類為兩個公式（5236a0、523330）與兩個排除
（52e690、590b40），並非四個公式。原生夾具另保留排除類 52f790。

完整原生鏈暴露 523330 的 Go callee 未傳遞 HP／體力重建副作用；原 boundary 3
的單列診斷重放得 `cases=8 diff=8`，第一個差異 c1_b11：F=52、原生=7。
完整原生輸入還會觸發死亡回應，超過既有 52e690 的 8 次記錄容量；保留 FAIL，
沒有把初始角色類別改成略過能力重建的哨兵值，也沒有減少輸入或忽略 RNG。
詳見 iface/54df00.tsv 的 native 列。因未修 callee，不能完成要求的
「F-only 注入→node-native FAIL→還原→PASS」；TestFNativeContract 僅證明
比較器能拒絕輸出錯誤與 RNG 違約，不充當該四段驗收。

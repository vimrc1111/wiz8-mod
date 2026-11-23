# 零星 RE 筆記

> patch／字串庫查找表、關鍵字表、所在地圖、法術 +0x12b、裝備加成。check-sigs 照樣掃本檔表格。
> 2026-11-01 由 6 份原檔逐行併入（標題降一級、內文不改）；原檔見 tag `wiz8-notes-pre-merge`（85393087）。

目錄：
- 第 17 行起：patch 查找（原 `notes/re/patch.md`）
- 第 34 行起：字串庫 STRINGDATA.DAT（原 `notes/re/strings.md`）
- 第 47 行起：關鍵字翻譯表 translated_Keywords.txt（7.3；cc 2026-10-07 補查，R9 未定位的部分）（原 `notes/re/keywords.md`）
- 第 65 行起：目前所在地圖（F7 攻略面板 G1，2026-10-14）（原 `notes/re/level-id.md`）
- 第 77 行起：法術記錄 +0x12b 模式欄位（G4／後續票 4b）（原 `notes/re/spell-12b.md`）
- 第 97 行起：裝備加成（技能／屬性）（原 `notes/re/equip-bonus.md`）


---
<!-- 原檔 notes/re/patch.md -->
## patch 查找

| 位址 | 前 16 bytes | 行為 |
|---|---|---|
| 0x004126f0 | 83ec08c605a00f6e002e66c70528b76e | 開 6 個基礎 SLF（Data\Data.slf 等），配 56 格（0x8c0/0x28）archive 表於 0x006eb724。 |
| 0x00412870 | 81ec0c010000535533db565733ed895c | `sprintf("%s\Patch.%3.3d", "Patches", n)`，n＝0..49 依序；存在才開，追加在基礎 SLF 之後。 |
| 0x004131b0 | 81ec08010000a128b76e0053555683ce | 依路徑選 archive：逐一比 archive 目錄前綴（`_strnicmp`，不分大小寫），候選再以 bsearch 找 entry；同前綴長度時**後面的 archive 蓋前面**。 |
| 0x00413360 | 8b44240481ec000200008d8c24000100 | bsearch 比較器：`_stricmp(要找的路徑, 目錄＋entry 名)`。 |

靜態結論（2026-10-07 資料前寫下，與 2.2 實測對照見 `notes/zh-patch-log.md:127`）：
- 編號：Patch.000–Patch.049，數字越大越晚載入、越優先。
- 名稱比對不分大小寫；**斜線不正規化**（比較器只做 stricmp，`/` ≠ `\`）。
- entry 必須依 stricmp 排序（bsearch）；載入時不排序 → `tools/slf pack` 產多 entry patch 時要先排好。
- 鬆散 `Data/` 檔：本輪未讀檔案層 0x00404c80 的「先找實體檔」分支，未定。

---
<!-- 原檔 notes/re/strings.md -->
## 字串庫 STRINGDATA.DAT

| 位址 | 前 16 bytes | 行為 |
|---|---|---|
| 0x00518360 | 8b4424045355566a006a4150e80fc9ee | 開檔（預設 `Data\Strings\StringData.DAT`，`/STRINGDATA` 可改路徑），讀 u32 筆數，逐筆讀 u32 位元組長＋內容，原地解碼，存入指標陣列 0x0068c09c。 |
| 0x005300e0 | 8b4c240885c97e1c8b44240433d2668b | 原地解碼：每個 u16 `u ← ~u + 0x9697`（＝`0x9696 − u`，mod 2^16）。 |

- 呼叫端直接索引 `0x0068c09c[i]`（數百處 inline，沒有共用 getter）→ 3.3 不 hook getter，hook 繪字入口。
- A3:8 的「0x96 高位元組」規則精確形式＝上式，與 `tools/strings` 的 oracle 一致。
- 路徑經檔案層（0x00404c80）開啟，所以 PATCH.00N 裡的 `Strings\StringData.DAT` 可以覆蓋（見 patch.md 查找規則）。

---
<!-- 原檔 notes/re/keywords.md -->
## 關鍵字翻譯表 translated_Keywords.txt（7.3；cc 2026-10-07 補查，R9 未定位的部分）

來源：`build/wiz8.asm`（objdump，Wiz8.exe v1.2.4）＋Wiz8.exe 匯入表解析（scratchpad，不入版）。[已驗證]＝本次讀 asm／匯入表；[推測]＝未實機。

- 路徑字串 `Data\Strings\translated_Keywords.txt` 在 VA 0x64a04c、`English_Keywords.txt` 在 0x64a074（.data）。[已驗證]
- 載入 0x56c200（由 0x56c57c 呼叫）：先清表 0x56c130，再對兩檔各呼叫 0x56bed0（0x56c264 English、0x56c34a translated），各自成一張表接到陣列 0x68ee8c（筆數 0x68ee84）。[已驗證]
- 0x56bed0 讀檔用 MSVCRT `fopen(path,"rb")`（0x5eb204，模式字串 0x64a048）＋`fgetws`（0x5eb134，緩衝 1000 字元）＋`wcslen`／`malloc`／`wcscpy`：**真實檔案系統、UTF-16 文字，不經 SLF 虛擬檔層**（PATCH 疊不上；要放鬆散檔 `Data\Strings\`）。[已驗證]
- 兩檔都載入成功才把旗標 0x68f0f8 設 1（0x56c3a1／0x56c406）；清表時設 0。英文版沒有這兩檔 → 旗標 0。[已驗證]
- 查表 0x56c440(in, out)：旗標 0 → `wcscpy(out, in)` 原樣；旗標 1 → 在 translated 表逐列逐詞以 0x402920 比對 in，命中則依同一位置抄出對應英文。[已驗證：控制流]；比對是否不分大小寫（0x402920）與每列的切詞規則（0x56be40）未讀 [未取得]。
- 唯一呼叫點 0x525ea6，在 0x525e80（目前 NPC 0x68c4ac 的關鍵字比對）開頭：**打字與點詞送出的詞都先過這張表，再比對 NSF 關鍵字**。[已驗證：呼叫點唯一；打字／點詞都走 0x525e80 為推測]
- 對中文化的意義：
  - 放兩檔即可讓「中文詞→英文關鍵字」生效，不必改 exe。[推測：未實機]
  - 但點中文時，取詞只以空白切（R5），我們的譯文中文不加空白，點下去取到的是整段到下一個空白為止的長串，查不到表；打字輸入中文可能被輸入框拒收（德文版前例）。
  - 「銀行（ bank ）」兩側空格寫法下，點括號內英文取到 `bank`，本來就能比對（實機清單第 12 項）。
- 決定（cc，可逆）：7.3 不開碼票。要讓點中文生效，得在譯文每個關鍵字中文兩側加空白，再出這兩檔；這會改動全部 888 處括注的排版，先交 MG 看實機第 12、15 項再說。

---
<!-- 原檔 notes/re/level-id.md -->
## 目前所在地圖（F7 攻略面板 G1，2026-10-14）

`static_analysis`；未實機讀值。

- **目前地圖索引**：`u32 [0x00686a70]`，無地圖時 `0xffffffff`。寫入點：載圖函式 `0x42a7b5`（`mov ds:0x686a70,esi`）、`0x42a889`；清成 -1：`0x42ae2a`、`0x4e31c6`、`0x54b225`（`build/wiz8.asm:57189`、`:57253`、`:57669`、`:304841`、`:450056`）。全檔 95 處引用。
- **地圖表**：`0x00604478` 起，每筆 107 bytes（索引 i：`lea eax,[i+i*2]; lea esi,[eax+eax*8]; shl esi,2; sub esi,i` ＝ 107i，`build/wiz8.asm:56853`–`56860`），上限 47（`cmp edi,0x2f`，`:56844`）。欄位：+0 目錄名[50]、+50 地圖名[50]、+100 三字代碼、+104 byte（用途未取得，`0xff` 時跳過 `0x605880` 表）。
- 表內容（索引 地圖名 代碼），由 `build/Wiz8.exe` 檔案偏移 `0x204478` 直接讀出（VA−0x400000）：
  0 Arnika2 ARN；1 Ascension ASC；2 Bayjin1 BA1；3 Bayjin2 BA2；4 Circle CIR；5 Marten1 MR1；6 Marten2 MR2；7 MountainPass MNT；8 Monastery1 MO1；9 Monastery2 MO2；10 MtGigasWaterCaves MGW；11 MtGigasBelowCaves MGB；12 MtGigas1 MG1；13 MtGigas2 MG2；14 MtGigasOuter MGO；15 MtGigasTop MGT；16 Camp CMP；17 RapaxCellar RAC；18 RapaxMainFloor RAM；19 RapaxUpperFloor RAU；20 RapaxExterior RAE；21 Rift1 RIF；22 SeaCave1 SC1；23 SeaCave2 SC2；24 Swamp SWM；25 Trynnie1 TY1；26 Trynnie2 TY2；27 Arnika_Trynton CT1；28 Trynton_Swamp CT2；29 SouthEastWilderness CT3；30 Rift_Peak CT4；31 NorthEastWilderness CT5；32 NorthWilderness CT6；33 Arnika_StarterDungeon CT7；34 Peak_RapaxCastle CT8；35 Footsteps FS1；36 SavantTower SAV；37 Ratkin RTK；38 CampNoRapax CNR；39 Dungeon DUN；40–46 Spare14–20（未用）。
- 未取得：實機讀值（G3 面板做好後在 ddraw log 印索引，MG 走幾區對照）；同一地圖內的子區（例如 Arnika 城內不同建築）要不要分，取決於攻略內容粒度。

---
<!-- 原檔 notes/re/spell-12b.md -->
## 法術記錄 +0x12b 模式欄位（G4／後續票 4b）

目的：驗 `notes/re/cov-exempt.d/4fa4d0.tsv` 的 4fa5cc 豁免前提（法術資料的模式只有 0..3）。

### 發現

- 法術載入函式（0x4acc 一帶）按筆數乘上每筆長度配置記憶體法術表，基址寫入 0x65be1c（build/wiz8.asm:231822、build/wiz8.asm:231827、build/wiz8.asm:231830）。
- 逐筆載入時先把檔指標向前跳過 0x101 bytes，再讀 0x1bf bytes 進記憶體當筆位置（build/wiz8.asm:231845、build/wiz8.asm:231847、build/wiz8.asm:231853、build/wiz8.asm:231857），筆間以 0x1bf 步進（build/wiz8.asm:231866）。
- 0x101＋0x1bf 等於磁碟 stride 704，故記憶體記錄是 DBS 記錄後段的連續拷貝：記憶體 +o ＝ DBS 記錄 +0x101+o（與 `internal/calc/calc_test.go:130` 的既有結論一致）。
- 0x4fa4d0 函式以法術索引乘步長 0x1bf 定位記錄（build/wiz8.asm:337227、build/wiz8.asm:337230），讀記憶體 +0x12b 為 4 bytes 整數（build/wiz8.asm:337236），與 3 比較、大於則跳往 0x4fa5cc，否則按值分派（build/wiz8.asm:337237、build/wiz8.asm:337242、build/wiz8.asm:337243、build/wiz8.asm:337281）。
- 旁證：同函式另讀 +0x133 與 +0x49（build/wiz8.asm:337234、build/wiz8.asm:337235），按同一個 0x101 平移正好對上 LoadSpells 已知的 DBS 欄位 +0x234（Realm）與 +0x14a（ManaPerPL）。
- 結論：記憶體 +0x12b ＝ DBS 記錄 +0x22c 的 4 bytes 整數；測試讀 SPELLTABLES.DBS（檔頭 8、stride 704）每筆 +0x22c 即可，不改 load.go。

### 結果

- `TestSpellMode12bDomain`（internal/calc/spell12b_test.go）：150 筆直方圖為 0:129、1:10、2:2、3:9，全在 0..3，豁免前提在現行資料成立，測試綠。
- 四段注入：允許集合縮為 0..2 時，9 筆值為 3 者逐筆指名 FAIL；副本還原後 PASS。

---
<!-- 原檔 notes/re/equip-bonus.md -->
## 裝備加成（技能／屬性）

rea 反編譯（2026-11-01），對象 `build/Wiz8.exe`（base 0x400000）。

- 0x50e540（`53 55 56 8b 74 24 10 57 8d ae 09 17 00 00 33 c0`）重算入口：清 A＝C+0x1709 → 0x50e980(C,A) → 清 T＝C+0x1770 → 0x50f090(T,A)、0x50f090(T,C+0x16a2 法術／buff)。
- 0x50e980（`83 ec 18 53 55 56 8b 74 24 2c 57 33 c0 b9 06 00`）逐 12 個裝備欄（C+0xf5d，stride 12）累加；欄 8、9 跳過，欄 6、7（雙手）不加 +0x48/+0x49。
  - 技能：item+0xb1≠-1 → A[0x13+i8(+0xb1)] += i8(+0xb2)；T+0x13＝C+0x1783，技能 0＝劍。
  - 屬性：item+0xb3≠-1 → A[0x0c+i8(+0xb3)] += i8(+0xb4)；對應 C+0x177c+idx，7 屬性。
  - 單件不夾；夾限在讀取端：技能 clamp(…,0,125)（0x553a96、0x553c5a、0x55407d），屬性夾 1..125（0x5539e0）。
- 0x50f090（`53 55 56 57 8b 44 24 14 8b 4c 24 18 8a 11 00 10`）合併：技能、屬性、抗性相加，所以物品加成與法術加成疊加。
- 0x53dac7 讀 ITEMS+0xad 加進每次攻擊的揮數／箭數，近戰同一條（見 `notes/party.md:14` 多箭段）。
- 屬性 index 0 力、1 智、2 信、3 體、4 敏、5 速、6 感 [推測：只以 Fang 力 +10 對上；calc 測試 TestFightEquipBonus 斷言 index 0]。
- [未取得] 其他寫 C+0x1783 的路徑；0x50e540 的觸發時機。

實作：`internal/calc/fight.go`（75975f93），測試 `TestFightEquipBonus`。

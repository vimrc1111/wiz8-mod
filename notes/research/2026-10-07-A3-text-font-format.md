# A3：文字資料與字型格式

## 發現

- EXE 內寫死 `Data\Strings\StringData.DAT`，即字串庫的虛擬檔路徑 [已驗證] (Wiz8.exe:2180832)
- PATCH.001 的 entry3 為 `STRINGS\STRINGDATA.DAT`，offset 0x5e651、size 131660，與 tiling 切出的第 4 個 blob 完全吻合 [已驗證] (PATCH.001:519141)
- STRINGDATA.DAT 的 blob 以 `greaterdemonback.tga` 開頭，後接固定模式的二進位記錄頭，屬通用字串庫而非純對白 [已驗證] (PATCH.001:386641)
- STRINGDATA.DAT 的 blob 尾部為高位元組固定 0x96、低位元組落在 ASCII 範圍的雙位元組序列（如 `96 76 96 25`），不是明文 UTF-16LE 英文 [已驗證] (PATCH.001:518160)
- 上條位元組形態與反組譯文件所載索引字串庫「header 編碼旗標為非零時逐 code unit 轉換」的編碼形式相容，但確切轉換規則未解 [推測] (https://raw.githubusercontent.com/agluszak/wizardry-8-decomp/HEAD/docs/wiz8-data-formats.md)
- EXE 內有 `Data\Messages\%s.msg` 格式字串，即 MSG 檔的虛擬檔載入點 [已驗證] (Wiz8.exe:2124816)
- POL Lang.SLF 的 entry 表內有 16 個 `MESSAGES\*.MSG`（ARNIKA、ASCENSION、BAYJIN、CONNECTIVETISSUE…），彼此相隔 0x118 [已驗證] (Lang.SLF:5250021)
- 上條 16 個 MSG 檔名與 A1 附錄所列 16 個僅 ARNIKA、BAYJIN 重疊，A1 名單疑似誤植 DATA.SLF 的而非 Lang.SLF 的 [已驗證] (notes/research/2026-10-07-A1-slf-patch-format.md:113)
- ARNIKA.MSG 的 entry 登記 offset 0x1e5a91、size 9965，可直接定位 blob [已驗證] (Lang.SLF:5250277)
- ARNIKA.MSG 的 blob 首部是二進位腳本結構、尾部夾雜同款 0x96 變換文字、`ffffffff` 哨兵與 `11 00 00 00` 計數字：MSG 是「二進位＋編碼文字」混合格式 [已驗證] (Lang.SLF:1987985)
- EXE 內有 `Data\Databases\EncounterTables.dbs` 等 DBS 虛擬路徑與 `Data\Databases` 目錄字串 [已驗證] (Wiz8.exe:2139972)
- Items 819 筆固定 stride 0x10d、Monsters 595 筆 stride 0x297（含四組 0x30 位元組 UTF-16 名欄位）等 DBS 版式見反組譯文件，本機未逐欄驗 [推測] (https://raw.githubusercontent.com/agluszak/wizardry-8-decomp/HEAD/docs/wiz8-data-formats.md)
- ItemDesc／SpellDesc／SpellEffect 共用索引字串庫格式：5 位元組 header（版本 1＋編碼旗標）、每筆兩個 u32＋UTF-16 code-unit 計數、檔尾逐筆絕對 offset 表＋筆數＋零 dword、reader 拒收超過 2000 code units 的字串 [推測] (https://raw.githubusercontent.com/agluszak/wizardry-8-decomp/HEAD/docs/wiz8-data-formats.md)
- EXE 內對話腳本載入點為 `Data\NPC Scripts\%s.nsf` 與 `Data\NPC Scripts\*.nsf` [已驗證] (Wiz8.exe:2212620)
- ENG.SLF 單一 blob（NPC_ZANT.NSF）內對白為 UTF-16LE，關鍵字為 u16 長度前綴的單位元組字串（如 `07 00`＋DOMINUS），即 NSF 混合雙格式 [已驗證] (ENG.SLF:4474)
- EXE 內另有 `Data\Strings\translated_Keywords.txt` 與 `Data\Strings\English_Keywords.txt`，即 NPC 關鍵字清單檔 [已驗證] (Wiz8.exe:2400332)
- NPC 對話關鍵字清單會寫入存檔，舊存檔混用英文關鍵字、需開新遊戲，見德文翻譯說明 [推測] (notes/research/2026-10-07-fan-translation-de.md:63)
- RUS.SLF 首個 blob 以 `Arnika2\0` 起頭，為單位元組 NUL 結尾關鍵字串列，俄文段的實際 codepage 未定 [已驗證] (RUS.SLF:536)
- STI 為 Sir-Tech STCI 系格式：64 位元組 header（魔數、尺寸、旗標、調色盤資訊）＋調色盤＋ETRLE 壓縮圖資，同廠 JA2 有公開說明，Wiz8 是否逐欄相同未驗 [推測] (https://raw.githubusercontent.com/tais/1.13/HEAD/FILEFORMAT.md)
- POL Lang.SLF 內 26 處 `STCI` 魔數，數量等於 FONTS 18＋JOURNAL 1＋LEVEL LOAD 1＋OPTIONS 6，全是字型／選單點陣圖的替換槽 [已驗證] (Lang.SLF:1602666)
- 鬆散 STI（LoadingPleaseWait.sti）64 位元組 header 後緊接調色盤資料，頭內尺寸欄與 640x480 8-bit 相容 [已驗證] (LoadingPleaseWait.sti:0)
- EXE 共列 34 個 `Data\Fonts\*.sti` 路徑，其中 `Palette*.sti` 屬換色用資源而非字形 [已驗證] (Wiz8.exe:2180144)
- 字元表是否 256 槽、字寬表與字形索引方式無公開說明；俄社群有 STI Image Editor 可作後續線索，未取得其文件 [未取得] (https://rpgcodex.net/forums/threads/wizardry-8-tool-1-0-7-sti-image-editor.136286/)
- 各翻譯皆以「整檔 STI 覆蓋」換字型（德文換 FONTS／INTROTEXT，日文換 11＋2 個 STI），無逐字形 patch 機制 [推測] (notes/research/2026-10-07-fan-translation-de.md:59)
- 德文 FAQ：物品名與敵人名有固定長度上限、被迫縮寫；戰鬥句 "X shoots at Y" 的 at 無法變格、一律用 nach；parser 不接受變音關鍵字 [推測] (notes/research/2026-10-07-fan-translation-de.md:71)
- 英文 EXE 內有 `MAX_TEXT_BOX_LINES`、`uiTextBoxLinesUsed[iTextBuffer]`、`Drawing text box while Text Buffer = %d` 等文字緩衝區符號，屬索引式文字框結構 [已驗證] (Wiz8.exe:2392332)
- 上條緩衝區的確切位元組上限與行數上限需反組譯，本票禁止反組譯故未取得 [未取得] (Wiz8.exe:2408012)
- 法文版把 journal 版面改成容納更長句子，但改法機制作者未公開 [推測] (notes/research/2026-10-07-fan-translation-fr.md:43)
- 英文 EXE 無日文顯示能力，jpmod 作者試圖 binary 改 GOG 版後放棄，改用日版 1.2J EXE [已驗證] (https://jpmod.blogspot.com/2013/09/wizardry-8.html)
- POL Text.txt 與英文預設檔皆 353 行、CRLF、無 BOM，為逐行對應的 FanPatch 新增選項字串覆蓋 [已驗證] (Text.txt:1)
- POL Text.txt 是單位元組 legacy codepage（0xC3 出現 0 行、高位元組 207 行，ó ż ł ś ę 以單位元組存放），即 cp1250，相容單位元組字形索引 [已驗證] (Text.txt:1)
- POL Info.ini 登記語言名 Polish／Lang=POL；Lang.SLF 九類 258 筆的分類見 A1 [已驗證] (Info.ini:1)
- 0x80–0xFF 如何對到字形槽的引擎機制無文件，FanPatch 文件只定義 Info.ini／Text.txt／Lang.slf 三件套 [未取得] (Patch128_11_README ENG translate Google.txt:352)
- FanPatch 讀 Text.txt 行數不足會放棄套用本地化並曾因此當機，後加檢查 [推測] (Patch128_11_README ENG translate Google.txt:321)
- FanPatch 當時記載「*.MSG 本地化即將實作」，即 Text.txt 機制最初不含 MSG [推測] (Patch128_11_README ENG translate Google.txt:668)
- 日版 1.2J 約當英文 1.2.2，但 STRINGDATA.DAT 差異大；英語版雖號稱 Unicode，日文顯示是散在各處的獨自轉換處理 [已驗證] (https://jpmod.blogspot.com/2013/09/wizardry-8.html)
- 日文版資料以 SJIS 存放（CF editor 日文亂碼 patch 跳過 MultiByteToWideChar 即為證），字型出處為 IPAex 明朝等現成字型 [推測] (notes/research/2026-10-07-fan-translation-ja.md:28)
- 波蘭曾有 CD Projekt 官方發行版，FanPatch POL 疑承襲其文字資產，但血緣未見文件 [推測] (https://www.gamespot.com/articles/wizardry-8-heads-to-poland/1100-2815644/)
- 日版 SJIS 雙位元組如何對到字型檔內位置、有無多頁，查無公開說明 [未取得] (https://jpmod.blogspot.com/2013/09/wizardry-8.html)

## 命令紀錄

樣本經無空白 symlink 掛入沙箱，只讀；`w8`=Steam Wizardry8 目錄，`w8u`=Wizardry8_Uninstall。未執行任何下載物；`strings`／`xxd`／`od`／`objdump` 只讀、`grep` 只做位元組定位；未寫程式、未反組譯。

    find w8u/Localization -maxdepth 3 | sort
    # Fan-Patch1.28/{1.28 default eng text.txt,POL/{Info.ini,Lang.SLF,Text.txt}}

    wc -l "w8u/Localization/Fan-Patch1.28/1.28 default eng text.txt" "w8u/Localization/Fan-Patch1.28/POL/Text.txt"
    # 兩檔皆 353 行

    xxd POL/Text.txt | head -20
    # Skr.ty klawiszow…：0xf3=ó、0xbf=ż、0xb3=ł、0x9c=ś、0xea=ę，CRLF，無 BOM

    grep -a -c $'\xc3' POL/Text.txt; grep -a -c $'\xc3' "1.28 default eng text.txt"
    # 皆 0 行：排除 UTF-8；LC_ALL=C grep '[\x80-\xff]' POL/Text.txt 得 207 行含高位元組

    strings -t d -a w8/Wiz8.exe | grep -c "Data\\\\Fonts\\\\"
    # 34；首見 Wiz8.exe:2180144 Data\Fonts\FONT12POINT1.sti

    strings -t d -a w8/Wiz8.exe | grep "Messages\|StringData\|Keywords\|NPC Scripts\|Databases"
    # 2124816 Data\Messages\%s.msg；2180832 Data\Strings\StringData.DAT
    # 2400332 Data\Strings\translated_Keywords.txt；2400372 Data\Strings\English_Keywords.txt
    # 2212620 Data\NPC Scripts\%s.nsf；2213880 Data\NPC Scripts\*.nsf
    # 2139972 Data\Databases\EncounterTables.dbs；2423484 Data\Databases\ItemDesc.dbs

    strings -t d -a w8/Wiz8.exe | grep -i "text buffer\|TEXT_BOX_LINES"
    # 2392332 WARNING: Drawing text box while Text Buffer = %d (AlexP)
    # 2408012 gStatus.uiTextBoxLinesShown[iTextBuffer] < MAX_TEXT_BOX_LINES

    grep -a -b -o "STCI" POL/Lang.SLF | head -25; grep -a -c "STCI" POL/Lang.SLF
    # 26 處，首處 1602666；= 18 FONTS＋JOURNAL＋LEVEL LOAD＋6 OPTIONS

    xxd -s 1602666 -l 64 POL/Lang.SLF；xxd w8u/Data/Options/LoadingPleaseWait.sti | head -6
    # 皆 STCI 開頭＋64 位元組 header，0x40 起調色盤；LoadingPleaseWait 頭內 e001 8002 與 640x480 相容

    strings -t d -a w8u/Data/ENG.SLF | head；xxd -s 4400 -l 256 w8u/Data/ENG.SLF
    # 4474 DOMINUS；`07 00`＋DOMINUS（u16 長度前綴單位元組串）混 UTF-16LE 對白 "Yessss..."

    strings -t d -a w8u/Data/RUS.SLF | head -3；xxd -s 536 -l 64 w8u/Data/RUS.SLF
    # 536 Arnika2：`4172 6e69 6b61 3200` 單位元組 NUL 結尾串列

    xxd -s 518301 -l 64 w8/Patches/PATCH.001；entry 逐筆 +0x100/+0x104
    # entry0 ITEMS.DBS @0x19e9c size 220315；entry1 SPELLTABLES @0x214 size 105608
    # entry2 GDEMON_DODGE.MON @0x4fb37 size 60186；entry3 STRINGDATA.DAT @0x5e651 size 131660

    xxd -s 386641 -l 64 w8/Patches/PATCH.001
    # STRINGDATA 首串 greaterdemonback.tga，記錄頭固定模式

    xxd -s 518160 -l 144 w8/Patches/PATCH.001
    # STRINGDATA 尾：0x96 高位元組雙位元組序列，entry 表 `...STRINGDATA.DAT` 緊接其後

    strings -t d -a POL/Lang.SLF | grep -i "messages"
    # 16 筆（5250021 起，步長 0x118），與 A1 附錄名單僅 2 個重疊

    xxd -s 5250277 -l 24 POL/Lang.SLF
    # ARNIKA.MSG @0x1e5a91 size 9965

    xxd -s 1987985 -l 128 POL/Lang.SLF；xxd -s 1997800 -l 150 POL/Lang.SLF
    # MSG 首二進位、尾 0x96 變換文字＋ffffffff 哨兵＋長度字

    web（User-Agent：wiz8-mod personal research (non-commercial)，只讀未下載）：
    # JA2 STCI 文件 https://raw.githubusercontent.com/tais/1.13/HEAD/FILEFORMAT.md（64B header、調色盤、ETRLE）
    # 反組譯 wiz8-data-formats.md（LIBHEADER／DIRENTRY、DBS 版式、索引字串庫、2000 上限）
    # jpmod https://jpmod.blogspot.com/2013/09/wizardry-8.html（1.2J、獨自轉換、放棄改 GOG 版）
    # rpgcodex STI Image Editor 討論串存在但回 403，內容未取得

## cc 抽查（2026-10-07，3 條，全數核實）
- 第 5 行（Wiz8.exe:2180832 `Data\Strings\StringData.DAT`）：`dd skip=2180832` 讀出該字串。核實。
- 第 6 行（PATCH.001 entry3 STRINGS\STRINGDATA.DAT，offset 0x5e651、size 131660）：`slf list PATCH.001` 第 4 列 `386641 131660 STRINGS\STRINGDATA.DAT`，0x5e651＝386641。核實（用 B 的工具，與 A1 格式互證）。
- 第 33／40 行（jpmod：英文 EXE 改不動、改用 1.2J EXE；1.2J≈英文 1.2.2）：curl HTTP 200，頁面 meta 為「GOG版と1.2Jのパッチを使って日本語表示。1.2JEXE(NOCD)…」，內文「1.2Jは、英語版で言う1.2.2相当」。核實。

# A1：SLF 與 Patches/PATCH.00N 格式

## 發現

- SLF 為小端序、無魔數：固定 0x214 bytes 檔頭，資料區緊接，entry 表在檔尾，entry 數存檔頭 +0x200；DATA.SLF 讀到 3314，dir 起點 0x8d4a245 等於 檔大小−3314×0x118 [已驗證] (DATA.SLF:0x200, https://github.com/agluszak/wizardry-8-decomp/blob/HEAD/docs/wiz8-data-formats.md)
- 檔頭欄位：檔名 @0x0、目錄名 @0x100（皆 NUL 結尾零填充），entry 數 @0x200（@0x204 存同一數），@0x208 恆 0x200FFFF、@0x20C 恆 1、@0x210 恆 0，其餘全零；9 個本機 SLF 全同 [已驗證] (DATA.SLF:0x208)
- 檔頭 @0x204/@0x208/@0x20C/@0x210 的語義無公開文件；SGP 原始碼 LibraryDataBase.h/.c 未找到 [未取得] (https://github.com/agluszak/wizardry-8-decomp/blob/HEAD/docs/wiz8-data-formats.md 只給偏移與行為，未貼結構定義)
- entry 表起點 = 檔大小 − entry數×0x118；9 檔逐筆驗（DATA.SLF 3314 筆起 0x8d4a245、PATCH.001 4 筆起 0x7e89d、Lang.SLF 258 筆起 0x4ff7cd，全部 OOB=0） [已驗證] (DATA.SLF:0x8d4a245)
- 每筆 0x118 bytes：檔名 0x100 bytes（NUL 結尾、零填充、`\` 分隔），+0x100 u32 資料偏移、+0x104 u32 大小、+0x108 u32 狀態、+0x10C u64 FILETIME、+0x114 u32 全零 [已驗證] (PATCH.001:0x7e89d)
- 偏移為絕對檔偏移；同檔 blob 互不重疊且資料區末端無縫接 entry 表（9 檔 overlaps=0、gap_end=0） [已驗證] (PATCH.001:0x214)
- 首 blob 起於 0x214（DATA.SLF、PATCH.001、LEVELS.SLF 的 rec0 偏移皆 532，內容各為 STCI、 spell表、OCT） [已驗證] (DATA.SLF:0x214)
- 無壓縮、無加密：blob 原地即為可解析格式（PATCH.001 內 ITEMS.DBS 首 u32=819 且 4+819×0x10d=220315=登錄大小；SPELLTABLES 首 u32=150；STI 首 4 bytes 為 STCI） [已驗證] (PATCH.001:0x19e9c)
- FILETIME 欄位轉出合理日期（DATA.SLF 首筆 2001-01-15；PATCH.001 三筆 2001-12-05/10，與 12_23 patch 時序吻合） [已驗證] (PATCH.001:0x7e89d)
- 9 檔全部 entry 的狀態欄與 +0x114 欄皆為零（含 DATA.SLF 3314 筆） [已驗證] (DATA.SLF:0x8d4a245)
- 「狀態低位 byte 為零才保留」僅見反組譯文件，本機無非零樣本可對照語義 [推測] (https://github.com/agluszak/wizardry-8-decomp/blob/HEAD/docs/wiz8-data-formats.md)
- 檔名全 ASCII，最長 68 bytes（SOUND.SLF）；大小寫混合保留（如 LEVEL LOAD 含空格；ENG.SLF 單筆為 DATA\NPC SCRIPTS\NPC_ZANT.NSF） [已驗證] (SOUND.SLF:68, ENG.SLF:0xf322)
- PATCH.00N 與 SLF 同格式：PATCH.001 檔頭 0x214、4 筆、表在 EOF、4 個 blob 無縫 tiling [0x214,0x19e9c,0x4fb37,0x5e651,0x7e89d) [已驗證] (PATCH.001:0x7e89d)
- EXE 內含格式字串 `%s\Patch.%3.3d` 與目錄名 `Patches`，即掃 Patches\ 下三位數編號 patch [已驗證] (Wiz8.exe:2097296)
- 起始編號（000 或 001）、缺號即停或上限 999，字串只給形狀，需反組譯確認 [未取得]（未做反組譯；字串見 Wiz8.exe:2097296，命令紀錄）
- 六個 SLF 路徑寫死 EXE：Data\Data.slf、Data\Sound\Sound.slf、Data\Sound\Monsters\MonsterSound.slf、Data\Music\Music.slf、Data\Monsters\Monsters.slf、Levels\Levels.slf [已驗證] (Wiz8.exe:2097352)
- Data\Music\Music.slf 磁碟無對應檔（本機 5 個 SLF 無 MUSIC.SLF），且 EXE 內 MonsterSound 與磁碟 MONSTERSOUND.SLF 大小寫不一 [已驗證] (Wiz8.exe:2098129)
- patch 同名檔優先於 base SLF [推測]（patch 機制目的＋德文 PATCH.002 生效實證，notes/research/2026-10-07-fan-translation-de.md:44；精確查找順序需反組譯）
- patch 之間編號大者贏 [推測]（PATCH.002 與 PATCH.001 共存且皆含 STRINGDATA.DAT/ITEMS.DBS，德文顯示生效，notes/research/2026-10-07-fan-translation-de.md:42）
- PATCH.001 單檔橫跨多 SLF 樹：DATABASES/STRINGS 見於 DATA.SLF 樹，MONSTERS\GDEMON_DODGE.MON 對應 MONSTERS.SLF 內裸名 GDEMON_DODGE.MON；故單一 patch 檔可覆蓋多個 SLF [已驗證] (PATCH.001:0x7e89d)
- 上條的檔名正規化規則（剝首段目錄？大小寫？後綴匹配？）無文件、需反組譯 [未取得]（字串與樣本只能證明字面不相等；PATCH.001:0x7e89d）
- 本機 Steam 版 PATCH.001 有 3 個 blob 與 base SLF sha256 相同（ITEMS.DBS、GDEMON_DODGE.MON、STRINGDATA.DAT；SPELLTABLES 未比） [已驗證]（見命令紀錄；PATCH.001:0x19e9c）
- 由上條推論此 build 的 base SLF 已含 patch 內容、PATCH.001 實質冗餘；是否所有發行版皆如此未知 [推測]（樣本僅此一份，PATCH.001:0x7e89d）
- Cosmic Forge：作者 mad god、freeware、主頁 http://www.angelfire.com/magic2/madgod/CosmicForge.html [已驗證] (https://appdb.winehq.org/objectManager.php?sClass=application&iId=5284)
- Cosmic Forge 授權全文與原始碼 [未取得]（freeware 僅為散布形式，無公開原始碼；主頁 angelfire 已不可考，僅 WineHQ 留檔；https://appdb.winehq.org/objectManager.php?sClass=application&iId=5284）
- wiz8vd：Go、MIT、https://codeberg.org/bsdcode/wiz8vd，只改 LEVELS.SLF 固定偏移視距、不是 SLF 解析器 [已驗證] (https://pkg.go.dev/codeberg.org/bsdcode/wiz8vd)
- wiz_strings（jpmod，.NET，UTF-8 CSV 双向轉 STRINGDATA.DBS/DBS/MSG，無授權條款） [已驗證] (notes/research/2026-10-07-fan-translation-ja.md:28)
- 德文 wiz8editor（Ulf Wohlers 私有工具，無公開下載） [已驗證] (notes/research/2026-10-07-fan-translation-de.md:66)
- STI-Edit 僅 zimlab 提名，無作者/下載/授權 [未取得] (https://www.zimlab.com/wizardry/mods/wiz8mods.htm 全頁無連結)
- 開源 SLF list/extract/pack 通用工具 [未取得]（多組關鍵字搜尋無結果，見命令紀錄；下一票 Go 工具需新寫；最接近的公開工具 https://codeberg.org/bsdcode/wiz8vd 只改固定偏移不解析 SLF）
- 德文 PATCH.002（約 4.7 MB）：含 DATABASES 6 個 DBS、STRINGS\STRINGDATA.DAT、MESSAGES 15、QUOTES 36、NPC SCRIPTS 159、字型/選單 STI [已驗證] (notes/research/2026-10-07-fan-translation-de.md:52)
- FanPatch POL Lang.SLF：258 筆九類（DATABASES 13、FONTS 18、JOURNAL 1、LEVEL LOAD 1、MESSAGES 16、NPC SCRIPTS 166、OPTIONS 6、QUOTES 36、STRINGS 1），檔名無 DATA\ 前綴 [已驗證] (Lang.SLF:0x4ff7cd)
- RUS.SLF 218 筆全帶 DATA\ 前綴、ENG.SLF 僅 1 筆（DATA\NPC SCRIPTS\NPC_ZANT.NSF） [已驗證] (RUS.SLF:0x4acfa9)
- RUS/ENG.SLF 的載入方式與前綴差異原因 [未取得]（FanPatch 文件只定義 Lang.slf，見下條；w8u 的 Data/*.SLF 無說明文件；僅 Wiz8.dll:306608 字串無文件）
- FanPatch Localization 機制：Localization\Fan-Patch1.28\<LANG>\ 下 Info.ini/Text.txt/Lang.slf（connecting archive），開關為 Wiz8.ini [Games] Fan_Patch_1_28_Localization [已驗證] (Patch128_11_README ENG translate Google.txt:358)
- Wiz8.dll 另有 Mods\*.SLF 與 Localization\%s\%s\rus.dat 字串，屬 FanPatch 擴充載入器 [已驗證] (Wiz8.dll:306608)
- 先前 de note 稱 entry 緊接檔頭是誤判：DATA.SLF 0x100–0x213 仍是檔頭（dirname＋count），真表在 EOF；舊推斷作廢 [已驗證] (DATA.SLF:0x200)

## 命令紀錄

（樣本經無空白 symlink 掛入沙箱，只讀；`w8`=Steam Wizardry8 目錄，`w8u`=Wizardry8_Uninstall）

    xxd DATA.SLF | head -20
    # 00000100 行起出現 Data\，此前 0x000–0x0ff 僅檔名 DATA.SLF 非零 → 檔頭 ≥0x200

    od -A d -t u4 DATA.SLF | head
    # 檔頭前 256 bytes 僅檔名非零，無 count

    python3 解析：檔頭 +0x200/+0x204/+0x208/+0x20c/+0x210
    # DATA.SLF:3314,3314,0x200ffff,1,0；PATCH.001:4,4,0x200ffff,1,0
    # 6 檔（+LEVELS/MONSTERS/SOUND/MONSTERSOUND）全同，dir 起點 = size−count×0x118
    # DATA.SLF dir=0x8d4a245（與反組譯文件值一致），PATCH.001 dir=0x7e89d，Lang.SLF dir=0x4ff7cd

    python3 逐筆驗 9 檔：OOB=0、overlaps=0、gap_end=0、status 全零、+0x114 全零
    # DATA.SLF n=3314 / MONSTERS.SLF n=3612 / SOUND.SLF n=25334 / MONSTERSOUND.SLF n=2039
    # LEVELS.SLF n=4956 / PATCH.001 n=4 / Lang.SLF n=258 / ENG.SLF n=1 / RUS.SLF n=218

    python3 tiling（PATCH.001）：[0x214,0x19e9c,0x4fb37,0x5e651,0x7e89d) 無縫
    # SPELLTABLES 105608 / ITEMS 220315=4+819×269 / GDEMON 60186 / STRINGDATA 131660

    python3 sha256 對比 patch blob vs base：ITEMS/GDEMON/STRINGDATA 三組相同
    # 如 DATABASES\ITEMS.DBS patch 220315 vs base 220315 identical True

    strings -a Wiz8.exe | grep -i patch|slf → `%s\Patch.%3.3d`、Patches、六 SLF 路徑
    strings -a Wiz8.dll | grep → Mods\*.SLF、Localization\%s\%s\rus.dat、Fan_Patch_1_28_Localization
    strings -a cfagent1.28.dll | grep lang/patch → 無命中（hook 由 EXE/DLL 側載入）

    web 搜尋（User-Agent：wiz8-mod personal research (non-commercial)，只讀未下載）：
    # "Wizardry 8 SLF file format" → 反組譯文件（LIBHEADER/DIRENTRY、0x214/0x118、InitializeLibrary）
    # "Cosmic Forge Mad God" → WineHQ（freeware、angelfire 主頁）
    # "wizardry 8 slf extractor unpacker"／site:github.com → 無通用開源 SLF 解包器
    # "PATCH.002" wizardry → 無公開技術文件（德文檔名清單引自本專案 de note）

## 附錄：Lang.SLF（POL）258 檔名全文

    DATABASES\ENCOUNTERTABLES.DBS
    DATABASES\FACT.DBS
    DATABASES\HITSOUNDS.TXT
    DATABASES\ITEMDESC.DBS
    DATABASES\ITEMS.DBS
    DATABASES\ITEMTABLES.DBS
    DATABASES\LEVELS.DBS
    DATABASES\MISSILETABLES.DBS
    DATABASES\MONSTERS.DBS
    DATABASES\NPC.DBS
    DATABASES\SPELLDESC.DBS
    DATABASES\SPELLEFFECT.DBS
    DATABASES\SPELLTABLES.DBS
    FONTS\DIALOG_FONT.STI
    FONTS\EMBOSSED.STI
    FONTS\ENGRAVED.STI
    FONTS\FONT10ARIAL.STI
    FONTS\FONT10ARIALBOLD.STI
    FONTS\FONT12POINT1.STI
    FONTS\FONT6ARIAL.STI
    FONTS\FONT8ARIAL.STI
    FONTS\FONT8ARIALBOLD.STI
    FONTS\HIGHDPI_FONT.STI
    FONTS\NORMAL_FONT.STI
    FONTS\OPT_DETAIL_FONT.STI
    FONTS\SMFNT.STI
    FONTS\WIZ_TEXT_FONT.STI
    FONTS\WIZ_TEXT_FONT_BOLD.STI
    FONTS\WIZ_TEXT_FONT_MONOPALETTE.STI
    FONTS\WIZ_TEXT_FONT_SUB23.STI
    FONTS\WIZ_TEXT_FONT_SUB81.STI
    JOURNAL\JOURNAL_FONT.STI
    LEVEL LOAD\LEVELLOAD_FONT.STI
    MESSAGES\ARNIKA.MSG
    MESSAGES\ARROWDAMAGE.MSG
    MESSAGES\BAYJIN.MSG
    MESSAGES\LOWERMONASTERY.MSG
    MESSAGES\MINE_TUNNELS.MSG
    MESSAGES\MOUNTAINWILDERNESS.MSG
    MESSAGES\RAPAXAWAYCAMP.MSG
    MESSAGES\RAPAXRIFT.MSG
    MESSAGES\SEACRAGS.MSG
    MESSAGES\SWAMPWILDERNESS.MSG
    MESSAGES\TRANG.MSG
    MESSAGES\TRYNTON.MSG
    MESSAGES\UMPANIBASECAMP.MSG
    MESSAGES\UPPERMONASTERY.MSG
    MESSAGES\WHIDDL.MSG
    MESSAGES\YAMIR.MSG
    NPC SCRIPTS\（166 筆，NPC_*/RPC_*/VOC_* .NSF）
    OPTIONS\INTROTEXT_BOTTOM.STI
    OPTIONS\INTROTEXT_MIDDLE.STI
    OPTIONS\INTROTEXT_TOP.STI
    OPTIONS\INTROTEXT_TOP_RUS.STI
    OPTIONS\OPTIONS_BUTTONS.STI
    OPTIONS\SAVE_LOAD_BUTTONS.STI
    QUOTES\PCS\（36 筆，F_*/M_*.MSG；如 F_AGGR10.MSG）
    STRINGS\STRINGDATA.DAT

註：NPC SCRIPTS 166 筆與 QUOTES 36 筆僅記模式（命名皆 NPC_*/RPC_*/VOC_*.NSF、F_*/M_*.MSG），全文可用「命令紀錄」段 python3 目錄解析重現；RUS.SLF 218 筆為同類文本子集（DATABASES 12、NPC SCRIPTS 162、OPTIONS 7、QUOTES 36、STRINGS 1）加 DATA\ 前綴。

## cc 抽查（2026-10-07，3 條，皆相符）
1. 第 5、8 條（DATA.SLF entry 數 @0x200、表在 EOF）：`od -t u4 -j 0x200 -N 4 DATA.SLF` → 3314；size 149080821 − 3314×0x118 = 0x8d4a245；該處首筆檔名 `AUTOMAP\AUTOMAPLARGE.STI`，+0x100 偏移 532（=0x214，與第 11 條一致）、+0x104 大小 311408。PATCH.001：n=4、表 0x7e89d。[已驗證：local_data_check]
2. 第 18 條（`%s\Patch.%3.3d` @Wiz8.exe:2097296）：`strings -a -t d Wiz8.exe` → `2097296 %s\Patch.%3.3d`。[已驗證：local_data_check]
3. 第 5 條引用 https://github.com/agluszak/wizardry-8-decomp/blob/HEAD/docs/wiz8-data-formats.md：HTTP 200；raw 檔第 12–14 行「reads a `0x214`-byte header」「seeks to `file_count * -0x118` relative to EOF」，第 19 行「directory begins at `0x08d4a245`」。[已驗證：static_analysis]
註：該文件是第三方反組譯結果；本專案未開反組譯器（邊界不變），引用的是其文字結論，並以本機樣本複驗。

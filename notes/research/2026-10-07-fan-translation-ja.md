# Wizardry 8 日文 fan translation (jpmod) 調查

## TL;DR
- PCGW 的 Japanese 列連到 jpmod 部落格「Wizardry 8 日本語化MOD」(2013)；最終版 v0.4 (2013-09-30)，MediaFire 仍可下載，已取得並列檔（未執行）。
- 機制**不是**我們要的 hook：它要求先把 **日本語零售版 1.2J 的 Wiz8.exe + Data 蓋到 GOG 英文版上**（EXE 本身已有 DBCS 處理），再疊上換成含日文字形的 .sti 點陣字型 + 少量翻譯檔，並以 WIZ8_JP_PATCH.EXE 把 EXE NOCD 化。作者明說「英文 EXE 沒做日文顯示，要改只能硬塞顯示常式進 EXE」，所以放棄。
- 只翻 UI/系統訊息/物品法術說明；NPC 對白、日誌未翻。無授權條款。與 FanPatch 1.28 的相容性：文件未提，推論不相容（用 1.2J EXE 取代 Wiz8.exe）。
- 對我們有用：確認 (a) 英文 EXE 無法直接顯示 SJIS（作者試過放棄）；(b) 字型是 Data/FONTS/*.sti 位圖整檔替換；(c) 作者另有 wiz_strings 工具可 pack/unpack STRINGDATA.DAT/FACT.DBS/ITEMDESC/SPELLDESC/*.MSG（UTF-8 CSV）。

## 1. 來源、版本、授權
- PCGW 列 (verified, 以 PCGW API 取 wikitext)：`Japanese | interface=hackable | subtitles=hackable | notes=Unknown if taken from the Japanese retail version... [https://jpmod.blogspot.com/2013/09/wizardry-8-mod.html download]`。來源：https://www.pcgamingwiki.com/w/api.php?action=parse&page=Wizardry_8&prop=wikitext&format=json（網頁本身被 Cloudflare 擋 403，API 可）。同表另有 Japanese 列「Local retail version only」。
- 作者：部落格 jpmod（readme 署名 `Jpmod(jpmod.blogspot.jp)`）(verified)。
- 該頁 https://jpmod.blogspot.com/2013/09/wizardry-8-mod.html (2013-09-13) 是 v0.1，連結已刪線，改指 label 頁。(verified)
- 版本串（全 verified，來源 https://jpmod.blogspot.com/search/label/Wizardry%208 與各文章）：
  - 2013-09-10 概念文章 https://jpmod.blogspot.com/2013/09/wizardry-8.html：GOG 版 + 1.2J 的 EXE(NOCD)/patch data + 自製日文字型，可正常執行。
  - v0.1 (09-12/13)、v0.2 (09-16, 附 wiz_strings；http://www.mediafire.com/download/1jrewgy6fo30gin/wiz8_jpmod_v0_2.zip) -> **MediaFire 回 error.php?errno=999，已失效** (verified, 2026-10-07)。
  - v0.3 (09-20, 附 Cosmic Forge Editor 日文亂碼 patch)。
  - **v0.4 (2013-09-30)**：http://www.mediafire.com/download/idppwdwsmy3ibzw/wiz8_jpmod_v0_4.zip ，檔名 wiz8_jpmod_v0_4.zip，3,067,492 bytes，sha256 `e41b5f7ab99580369a1115bc6188df7e4a2ef07d27b1182aca4e555ee00b205e` (verified，已下載)。
  - 翻譯作業表 (Google Sheet)：Google Sheet 連結（ID 省略，見 jpmod 部落格原文） （未開啟，not obtained）。
- 目標遊戲版本 (verified, readme)：作者只有 GOG 英文 1.2.4，其他版本「不知道能否正確運作」。需要 4Gamer 的 1.2J patch：http://www.4gamer.net/patch/data/wiz8.html（verified 存在，為日本語版 Version 1.2J 更新說明，列 Miles 更新等）與 Wizardry8 uploader「半日本語化のみ.zip」(ITEMDESC.DBS/SPELLDESC.DBS) http://ux.getuploader.com/wiz8ReforgedMod/ （HTTP 403，not obtained）。
- 授權/再散布：readme、部落格皆**未見**授權條款（所見範圍內未見）。字型出處在 readme：IPAex 明朝（大字）、ナガ10（11px 點陣）、美咲ゴシック（7px）(verified)。作者聲明「自己負責」。
- 附註：作者 1.2J 取代 EXE 的說法：1.2J 的 Changelog 約等於英文 1.2.2 級更新，但 STRINGSDATA.DAT 等資料差異大；「日本語版與英語版資料配置差很多，各自 EXE 針對各自資料 build」(verified, v0.3 文章 https://jpmod.blogspot.com/2013/09/... 同 label 頁)。

## 2. 機制與安裝（v0.4 zip 內容，verified，已列檔）
zip 內：
- `MOD/Data/FONTS/*.sti` ×11（dialog_font, embossed, engraved, font10arial, font10arialbold, font12point1, opt_detail_font, smfnt, wiz_text_font, wiz_text_font_bold, wiz_text_font_monopalette）、`MOD/Data/JOURNAL/journal_font.sti`、`MOD/Data/LEVEL LOAD/levelload_font.sti`：~0.26–1.6 MB 每個。
- `MOD/Data/DATABASES/FACT.DBS`、`MOD/Data/MESSAGES/*.MSG`×10（地名類）、`MOD/Data/QUOTES/PCS/F_AGGR10.MSG`/`F_AGGR20.MSG`。
- `MOD/WIZ8_JP_PATCH.EXE`（67,100 B，sha256 bdcd5b82...9dc9a46）：strings 可見 `Wiz8.EXE`、`Wiz8.exe NOCD`、`DIFFMAKE`、`UPDATE.TMP`，推論為 binary-diff patcher，作用依 readme 是「Wiz8.exe NOCD 化、ITEMDESC/SPELLDESC 原版化」。
- `wiz_strings/wiz_strings.exe`（.NET 4.0，UTF-8 CSV <-> STRINGDATA.DAT/FACT.DBS/ITEMDESC.DBS/SPELLDESC.DBS/*.MSG）+ bat；`cfe_patch/CFE_jp_patch.exe`（修 Cosmic Forge Editor 日文亂碼，作者說是 skip 掉 MultiByteToWideChar，因日文版資料以 SJIS 存）。
- 未執行任何 exe；未對 exe 做反組譯。
安裝（readme，verified）：①把 1.2J patch 的 Wiz8.exe 與 Data 複製到遊戲目錄（備份原 exe）；②半日本語化的 ITEMDESC/SPELLDESC 放 Data\DATABASES；③複製 MOD/ 全部到遊戲目錄並執行 WIZ8_JP_PATCH.exe。解除：還原 Wiz8.exe，刪 Data 下 DATABASES/FONTS/JOURNAL/LEVEL LOAD/STRINGS。
=> 是**直接覆蓋原始檔案**（Data 資料夾 + EXE），與我們「不改原檔、以 patch/hook 疊加」的約束衝突。未使用 Localization/Fan-Patch1.28/<LANG>/ 機制，也沒提到 SLF 重新打包（改放鬆散檔於 Data\ 之下覆蓋 DATA.SLF 內容）(verified from readme/zip)；該鬆散檔覆蓋 SLF 的行為本身是遊戲原生支援（inferred，非該 mod 文件所述）。

## 3. 雙位元組(SJIS)渲染（最重要）
- 作者概念文章 (verified 引文大意)：「遊戲本身雖英文版也支援 Unicode，但日文顯示部分是**自己的處理，各處都做轉換**，原本想用 GOG 版顯示日文，但改 binary 太麻煩而放棄」→ 改用**日本語零售 1.2J 的 EXE**（已內建 SJIS 處理）。
- v0.3 文章 (verified)：「要讓用到 CF 的 MOD 也日文化，只能把日文顯示常式硬塞進英文版 EXE」。→ 無 DLL hook、無 EXE 為 DBCS 修補（除 NOCD patch 外）。
- 字型：**位圖 .sti 整檔替換**。大字用 IPAex 明朝、11px 用ナガ10、7px 用美咲ゴシック (readme verified)。.sti 檔頭 `STCI`，font12point1.sti 前 40 byte 欄位顯示 640x480、256 色索引（xxd 讀頭, verified；檔長 371,456 > 307,200 像素資料，尾端結構（子圖/字元表）未解析 = not obtained）。字元如何由 SJIS 雙位元組對應到字型內位置、是否多頁：**not obtained**（未反編/未解析 STI，且屬 1.2J EXE 內建）。
- 作者警告 (verified, v0.1 文章)：使用不含日文字的字型去顯示日文會讓遊戲**崩潰**。
- 文字檔編碼：日文版資料以 SJIS 存（CF editor 文章 verified）；wiz_strings pack 時要 UTF-8，工具轉成遊戲格式（內部轉成 SJIS，inferred）。
- 斷行/截斷/UI 溢位：readme 與文章**未討論**（not obtained）；僅提「測試只做到確認動作程度，可能還需要追加字型」。影片字幕：Bik_mod 可行，但 Bink 庫舊，白色字幕變水色 (verified, v0.1 文章)。
- 已知限制 (verified)：只有 UI/系統/物品法術說明翻；NPC 對白（在 NSF 腳本內，解析中）、日誌未翻；英日資料配置不同，作者說 「與日本語版資料對照前不確定能否玩到最後」。

## 4. 與 FanPatch 1.28 相容性
- 文件完全未提 FanPatch（2013 作者基於 1.2J/1.2.4；FanPatch 1.28 日期較晚）— not stated。
- 推論 (inferred)：不相容。它要求以 1.2J 的 Wiz8.exe 取代英文 EXE，而 FanPatch 1.28 要使用其自己的 EXE/Localization 機制；兩者 EXE 衝突。若硬疊字型 .sti 到英文 + FanPatch EXE，作者實驗顯示英文 EXE 無日文處理（雙位元組不會被解讀）。

## 對我們專案的意涵
1. 日文社群走「換成日版 EXE」的路，等於放棄在英文 EXE 上做 DBCS；我們若不改 EXE，需另設 hook（DLL 注入/攔截文字繪製）— 此 mod 沒有提供可借鏡的 hook。
2. 可借鏡：.sti 為字型載體（檔案與路徑固定，可作為字型替換點）；wiz_strings 的 STRINGDATA.DAT/FACT.DBS/ITEMDESC/SPELLDESC/MSG pack/unpack 格式知識（若要研究可反編 .NET exe，未做）。
3. 日文零售 EXE 的 DBCS 繪製路徑是可逆向的現成參考（但 EXE 取得需日版）。

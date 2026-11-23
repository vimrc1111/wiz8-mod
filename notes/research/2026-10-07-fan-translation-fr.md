# Wizardry 8 French fan translation (JP Donald) - findings

TL;DR
- Link from PCGW (verified via API wikitext): https://jpdonald.organisationchrome.fr/html/Wizardry_8.htm ; author "JP Donald"; patch exe dated 22-06-2020, hosted file mtime 2025-05-30. No license/redistribution terms stated (not obtained).
- Mechanism: NOT a Localization/<LANG> drop-in. It is a Multimedia Builder self-extracting patcher EXE that overwrites game files in-place: replaces Wiz8.exe (modified), nglide_config.exe, itemdesc.dbs, npc.dbs, 3 NPC .nsf, 3 .STI, and an encrypted maj.zip (main payload, contents unseen). Requires FULL install + game version 1.26.
- Compatibility with FanPatch 1.28: not stated anywhere; it ships its own Wiz8.exe built for 1.26, so likely incompatible (inferred). Tooling/codepage/font/overflow details: not obtained (author publishes none).

## 1. Download / author / version / license
- PCGW French row: "Fan translation, download here." -> https://jpdonald.organisationchrome.fr/html/Wizardry_8.htm. Verified: https://www.pcgamingwiki.com/w/api.php?action=parse&page=Wizardry_8&prop=wikitext&format=json (the normal page returns 403 / Cloudflare; API worked). PCGW marks interface+subtitles "hackable", audio "unknown".
- Download page (verified, fetched): states game is Wizardry 8 by David W. Bradley; "La traduction est a effectuer sur la version 1.26". Links:
  - https://jpdonald.organisationchrome.fr/html/telechargement/Wizardry_8/Wizardry_8_Traduction_FR.rar (4,463,819 bytes, sha256 a69a1f6d0d82334cae3cf292c826147dbd7cf1cc456f82c26311b0e26d7767b0, Last-Modified Fri 30 May 2025 per HTTP header)
  - https://jpdonald.organisationchrome.fr/html/telechargement/Wizardry_8/Wizardry_8_videos.rar (249,002,335 bytes, optional French-subtitled videos, to replace originals in Data/Flics/Intro; HEAD only, not downloaded)
- Author: "JP Donald" (string in the EXE: "JP Donald", "Traduction FR de Wizardry 8", "(Version du 22-06-2020)"; file dates in rar 2020-06-23). Site index https://jpdonald.organisationchrome.fr/index.htm also points to forum hardware.fr thread and Threepwang's forum (Traductions & Fans patchs).
- Features claimed (Info.txt + page): full redo of translation; accentuation; keyboard entries adapted to AZERTY+accents; 450 items entirely redrawn (item icons/graphics -> .STI); journal display adapted for longer sentences; 3 added scrolls about hidden mazes.
- License/redistribution: none stated on the page, in Info.txt, or the EXE strings (verified absence in those three). Site says patches only patch installed originals and are not full copies of the game (index.htm). Page warns the patch changes program resources and may be flagged as a virus.
- rar contents (verified, extracted to scratchpad/dl-fr/ex, nothing executed): Info.txt (1499 B, sha256 f551dcb0...e59), "Wizardry Traduction FR.exe" (4,523,600 B, sha256 9543c2c3ef47444e53a1dcea4201422e41172d01002359f3f429ee1a07182831, PE32 GUI, no code run).

## 2. Mechanism
- Installer: EXE "Created with Multimedia Builder 4.9.8.13" (verified, string). Script strings (verified): sets/uses <SrcDir>\Saves\Sample Party.SAV (probable install-dir sanity check, inferred), message "Veuillez copier le patch dans le dossier du jeu", checks <SrcDir>\Data\DATA.SLF exists, MFilesAndFolders.dll Delete_File, then runs `FBZip.exe` with `WAIT -e -p** "<Embedded>maj.zip" "<SrcDir>"` i.e. extracts a password-protected embedded maj.zip over the game dir (password masked as ** in the string; inferred from the -p flag). Ends "Traduction terminee".
- Plain (unencrypted) embedded zip entries seen via local headers (verified names/sizes, content not recovered - my raw-deflate extraction failed CRC, so unseen):
  nglide_config.exe (57,344), Wiz8.exe (3,299,840 -> replaced/patched EXE), data/databases/itemdesc.dbs (124,773), data/databases/npc.dbs (118,692), data/npc scripts/NPC_BALBRAK.nsf, NPC_FUZZFAS.nsf, NPC_ZANT.nsf, data/options/INTROTEXT_HIGHLIGHT.STI, INTROTEXT_NORMAL.STI (93,328 each), OPTIONS_BUTTONS.STI (199,694).
  These are loose files in Data/ (game loads loose files over SLF), plausibly overriding SLF contents. The bulk (text, 450 item icons) is presumably in maj.zip - NOT obtained (encrypted, would require running FBZip/installer; not done).
- Does NOT use Localization/Fan-Patch1.28/<LANG>/ (Info.ini/Lang.SLF/Text.txt): no such names in the EXE strings (verified by grep of strings; only DATA.SLF checked).
- Install procedure (verified from page+Info.txt): full install of v1.26; copy patcher into game dir; run it. Also: delete glide.dll, glide2x.dll, glide3x.dll, run nglide_config.exe (4:3, vertical res close to screen, GPU scaling) for resolution. Optional videos replace originals in Data/Flics/Intro.

## 3. Tooling / accents / overflow
- Tools used to produce text: not obtained. Author gives no tool list ("sans la source originale du jeu" - translating without source; page text). Installer tool is Multimedia Builder + FBZip + MFilesAndFolders (verified), not a text tool.
- Codepage/font bitmap handling: not obtained. Info.txt only says "Accentuation" and AZERTY keyboard entries. The modified Wiz8.exe suggests EXE binary edits for input/charset (inferred, unverified).
- Overflow: only claim is "journal display adapted for longer sentences" (verified text); mechanism not obtained (likely in Wiz8.exe patch, inferred).

## 4. FanPatch 1.28 compatibility
- Not stated. Target is 1.26 (verified). Because it overwrites Wiz8.exe with its own build, applying over a 1.28 FanPatch install would revert the EXE to the 1.26-based one (inferred) and likely drop 1.28 features (widescreen/hotkeys per Steam discussion search hits; not verified). Steam version differences not checked.
- General background (search result, unverified): FanPatch 1.28 build 6320 at http://itmax99.000webhostapp.com/Wiz8/ (from web search summary only).

## Not obtained
- maj.zip contents (encrypted; needs executing untrusted installer), license terms, fonts/codepage approach, author's tool list. Possible follow-ups: contact form on the author site, hardware.fr thread https://forum.hardware.fr/hfr/JeuxVideo/PC/wizardry-superbe-meconnu-sujet_66421_1.htm (linked from page, not read).
- Scratch files: /tmp/claude-1000/-home-mg-Projects-games-wiz8-mod/d06f7575-86be-46cb-bd4f-47f006a4e13e/scratchpad/dl-fr/

## 補充：patcher 附帶的 info.txt（MG 2026-10-07 貼入）
原檔是 cp1252，在 MG 的 Windows（非 Unicode 程式語系＝cp950）下顯示成亂碼；以下依上下文還原。[已驗證：MG 貼入原文；還原字詞為推測]
- 翻譯整套重做：補上重音字母；輸入改為適配法文 AZERTY 鍵盤（含重音）。
- 450 個道具圖重畫（＝改美術資源）。
- 日誌顯示改成能放更長的句子（版面有動）。
- 新增 3 張隱藏迷宮的卷軸；劇情沿用原作。
- 安裝：完整安裝、遊戲版本 1.26、把 patcher 複製到遊戲目錄後執行；測試環境 Windows 7 64 位元。
- 解析度：建議刪除 `glide.dll`／`glide2x.dll`／`glide3x.dll`，用 `nglide_config.exe` 設 4:3、垂直解析度接近螢幕。
- 警告：「patch 會改動程式的部分資源，某些系統上可能被判成病毒」→ 證實它直接改 exe（資源區）。
- 有字幕的過場影片另外下載（約 240 MB）。

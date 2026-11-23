# loc-de: German fan translation of Wizardry 8 (wizardry-8.de)

TL;DR
- Link on PCGW row "German": http://www.wizardry-8.de/download.htm. Team Ulf Wohlers (coder/translator), Andreas Valley, Sebastian Bombera, Spawn, Manfred (fonts). Latest v0.97 (28.06.03). Free, but copyrighted; central redistribution only with their permission; no fees.
- Mechanism: NSIS installer drops ONE Sir-Tech-format patch file `Patches\PATCH.002` (about 4.7 MB) into the game dir. It uses the game's built-in Patches system, not Data/*.SLF and not an EXE patch. It carries strings, DBS, NPC scripts, MSG, STRINGDATA.DAT and font/UI STI. Uninstaller included.
- Targets original retail 1.0.0 to 1.2.4 (2001/2003 era). It predates FanPatch 1.28, and nothing found says whether the two are compatible: "not obtained".
- Tooling: the author's private editor ("wiz8editor" by Ulf Wohlers). No public download found. EXE strings (formation words, kill, poison) left untranslated because the EXE is SafeDisc-packed.

## 1. Source, authors, version, license
- Link (verified): the PCGW wikitext row is `language = German ... notes = Fan translation, [http://www.wizardry-8.de/download.htm download here.]`. Source: https://www.pcgamingwiki.com/w/api.php?action=parse&page=Wizardry_8&prop=wikitext&format=json (the normal wiki page returns a Cloudflare 403).
- Download page (verified, HTTP 200, Last-Modified 2008-02-14): http://www.wizardry-8.de/download.htm
  - Standard patch, extended combat text: http://www.wizardry-8.de/download/W8PatchInstall.exe
    - 1202470 bytes, sha256 383546836874e77d7062ef8dc71c2446fd509b6e139896ec7b77f589b41b1469
  - Alternate patch, shortened combat text: http://www.wizardry-8.de/download/W8PatchInstall-kurz.exe
    - 1202471 bytes, sha256 f6ea501df344ea50e12159d3b79bcdd026ca7a89fd3a93af0488d01b571ca958
  - Same page also offers Sir-Tech patch 12_23 (/download/Wiz8Patch2001_12_23.zip), a Portrait Editor (/download/Wiz8PortEdit.zip) and a savegame editor (http://www.mangazine.de/wizardry8editor2.zip). Not downloaded.
- Version (verified, in-installer Readme_DeutscheVersion.txt, extracted from the installer): v0.97, 28.06.03.
  - History: v0.95 public on 09.04.03, v0.96 on 26.04.03, v0.97 on 28.06.03.
  - Work began in Jan 2002: prealpha 02.01.02, full patch 15.03.02.
  - Paused for the planned official German release (Crucial Entertainment). Public beta started 26.02.03.
  - The page says the standard and alternate patches are interchangeable.
- Team (verified, readme section 8 and http://www.wizardry-8.de/kontakt.htm):
  - Ulf Wohlers: project initiator / coder / translator. Site says he worked on Gothic (Piranha Bytes) and DOSBox.
  - Andreas Valley: translator / beta coordinator; also named responsible in the site's Impressum (http://www.wizardry-8.de/impressum.htm).
  - Sebastian Bombera: translator.
  - Spawn: translator.
  - Manfred (manfred69_2002): graphics / font.
  - Allmende: alpha/beta tester.
  - Effort (verified, http://www.wizardry-8.de/faq.htm): translation 2 months, about 550 person-hours, plus 1 month alpha and 1 month beta.
- License (verified, readme 7b and the installer EULA text, with http://www.wizardry-8.de/faq.htm):
  - The translation text is copyrighted.
  - No free redistribution on websites, magazine CDs or other central distribution unless they permit it, and permission is "almost always" granted on request to Download@Wizardry-8.de.
  - Permission can be revoked at any time.
  - Non-central sharing is fine (file-sharing services are an exception).
  - No fees may be charged.
  - FAQ: they have a contract with Sir-Tech (Robert Sirotek) allowing creation and free distribution.
  - Implication for us: do not bundle their file in a repo or release. Link only, or ask.

## 2. Mechanism and install procedure
- Installer (verified): NSIS v2.0b3 self-extractor. I did not run it. I parsed the header and deflate blocks with a small python script.
  - Reads the game path from the registry key `software\sirtech\wizardry 8\path`.
  - Checks for `\Wiz8.exe`.
  - Looks for `\Patches\Patch.001`, which is Sir-Tech's patch 12_23. If missing it warns, and it still installs if the version is already 1.2.4.
  - Writes `\Patches\PATCH.002` (decompressed size 4726322 bytes, sha256 0d89a42db2f900a72c92ead787dcadf313a5ac95d36f91dfd4970ff1fa9d48e6), plus `Readme_DeutscheVersion.txt` and `Wiz8PatchUninst.exe` in the game root.
  - The registry key and `Patch.001` path are verified from strings. The installer's instruction order was not decoded.
- Uninstall (verified): Wiz8PatchUninst.exe restores English (readme 1c, FAQ "voll funktionsfähige englische Version").
- Why it is not a localization folder (verified, FAQ): "we build on the Wizardry 8 engine and use the game's internal patch system ... no changes to the game itself".
  - Original files are untouched. No EXE is modified. The FAQ asks for help because EXE-embedded strings could not be changed and SafeDisc must keep working.
- PATCH.002 format (verified by hex and strings, structure inferred):
  - Starts with a 256-byte zero-padded name field `PATCH.002`, then a 256-byte `Data\` record, then entries.
  - Entries contain resource names such as `DATABASES\FACT.DBS`. I saw FACT_*, item and monster strings inside, in a mix of ASCII and UTF-16LE.
  - Resource paths named in it (verified, 229 found by regex; the list is not necessarily exhaustive and may include non-entries):
    - DATABASES\{FACT,ITEMDESC,ITEMS,MONSTERS,SPELLDESC,SPELLTABLES}.DBS
    - STRINGS\STRINGDATA.DAT
    - MESSAGES\*.MSG (15)
    - QUOTES\PCS\{F,M}_*.MSG (36)
    - NPC SCRIPTS\*.NSF (159: NPC_, RPC_, VOC_)
    - FONTS\{FONT10ARIAL,FONT12POINT1,OPT_DETAIL_FONT,SMFNT}.STI, JOURNAL\JOURNAL_FONT.STI, LEVEL LOAD\LEVELLOAD_FONT.STI
    - OPTIONS\{INTROTEXT_*,OPTIONS_BUTTONS,SAVE_LOAD_BUTTONS}.STI
  - So it replaces text, NPC dialogue scripts and fonts/menu bitmaps inside the game's virtual file system, via Patches.
- Subtitles only: no German voice (verified, FAQ and readme: all subtitles translated, speech not dubbed). The readme says to enable "Detaillierte Kampfnachrichten" for party chatter.
- Known untranslated parts (verified, FAQ and readme 3): formation words (Left/Right/Front/Center/Rear), keywords such as kill/poison in combat, location keywords in NPC dialogue lists (Monastery/Swamp). All are in the packed EXE.
- Save-game caveat (verified, readme 2): the NPC dialogue keyword list is stored in the save, so old saves mix English keywords. Start a new game.

## 3. Tooling, umlauts, length
- Tools (partly verified): the readme history says someone posted in Frank's MM-Forum that he had decoded the patch file format and wanted translators (readme 5). The coder is Ulf Wohlers, whose contact in the readme is `wiz8editor@onlinehome.de`. The site's Portrait Editor and the savegame editor (mangazine.de) are other tools. I found no public repository or download of an SLF/STI/Patch extractor or text editor (a web search found nothing relevant; wiz8vd on pkg.go.dev only edits LEVELS.SLF view distance: https://pkg.go.dev/codeberg.org/bsdcode/wiz8vd). "Not obtained": the source of their editor, and any statement about which unpackers they used.
- Umlauts (partly verified):
  - Fonts and UI bitmaps are replaced: FONTS\*.STI plus the INTROTEXT_* STIs are in the patch, and Manfred is credited for "the font" and the intro menu graphic (kontakt page).
  - Raw bytes consistent with cp1252 umlaut values and with UTF-16LE umlaut codes both occur in PATCH.002, so the text encoding is inferred, not determined (counts of 0xE4 0xF6 0xFC 0xDF appear at thousands each; not decisive since binary data is mixed).
  - The in-game parser input cannot accept umlauts (verified, FAQ). They chose keywords without umlauts, or you click the word in the NPC text. v0.97 replaced remaining umlauts in keywords (e.g. the "Bestientöteraxt").
- Length limits (verified, FAQ): strings are mostly unrestricted, but item names and enemy names have a fixed length limit, which forced shortened German. The battle sentence "X shoots at Y" ("at" generic) forced "nach" everywhere. The two patch variants exist because of the extended/short combat-text display bug (v0.96 history: "Spieler schießt mit Monster").
- Terms: they leaned on Wizardry 7 German terminology and kept some names in English (Dark Savant) (verified, FAQ).

## 4. FanPatch 1.28 compatibility
- Not obtained. The German patch (2003) predates FanPatch 1.28 and says it works with 1.0.0 to 1.2.4; nothing found mentions 1.28.
- Inferred only: the patch uses the Patches\ chain, while FanPatch 1.28 ships its own Localization\ mechanism with a changed EXE (Steam thread: https://steamcommunity.com/app/245450/discussions/0/3949155256707786367/?ctp=1, which mentions DBS localization support, no German details). Collisions with a Patch.002 file are untested.
- Needs an experiment to settle.

## Not done / limits
- Installer not executed; the NSIS instruction stream (exact file copy list and order) was not decoded, only the strings and data blocks.
- PATCH.002 was not parsed into its entry table.
- Extracted data is in /tmp/claude-1000/-home-mg-Projects-games-wiz8-mod/d06f7575-86be-46cb-bd4f-47f006a4e13e/scratchpad/dl-de/ (installers, blob0.bin = PATCH.002, blob1.bin = readme, hdr-de.bin).
- The standard-vs-short installer difference was not analyzed beyond a size/sha comparison (they differ in about 1.16M bytes, mostly payload).

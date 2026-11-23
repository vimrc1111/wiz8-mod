# A2：ddraw.dll 代理載入與串接

## 發現
- 樣本 Wiz8.exe 3580513 B、pei-i386，objdump 匯入表共 9 個 DLL（含 DDRAW.DLL、GDI32.DLL），與單上背景一致 [已驗證](Wiz8.exe:3580513)
- 未找到 Wiz8 專用開源 ddraw 代理；公開討論皆用通用包裝加 Wiz8 專用渲染外掛，未見針對 Wiz8 文字的代理專案 [未取得](https://www.pcgamingwiki.com/wiki/Wizardry_8)
- PCGamingWiki 建議 Wiz8 用 dgVoodoo2（拷 MS 目錄 x86 下全部檔到遊戲目錄）解解析度與縱橫比，另有 Steam 玩家回報修好畫面 [推測](https://www.pcgamingwiki.com/wiki/Wizardry_8)
- GOG 論壇把 dgVoodoo2 列為 Wiz8 舊 Glide 與舊 Direct3D 相容手段之一（與 nGlide 並列，非 Wiz8 專用） [推測](https://www.gog.com/forum/wizardry_series/wiz_8_no_longer_runs_after_latest_win10_update/page1)
- Wiz8Extender 是 srVP 系列渲染外掛（解包到 Dll 目錄加改 3DVideo.CFG 做 Hor+），不是 ddraw 代理 [推測](https://www.pcgamingwiki.com/wiki/Wizardry_8)
- 本機 3DVideo.CFG 首行渲染器為 Glide2x（1280x1024x16），預設走 Glide 路徑而非 DirectX [已驗證](3DVideo.CFG:1)
- cnc-ddraw 開源 MIT 但只做 DirectDraw 軟體渲染，README 明說不支援 Direct3D 遊戲，Wiz8（DirectX6／7／Glide 渲染器）不適用 [推測](https://github.com/FunkyFr3sh/cnc-ddraw)
- DDrawCompat 開源（LICENSE.txt 為 ISC 式授權），做 DirectDraw 與 Direct3D 1-7 相容及效能修正 [推測](https://github.com/narzoul/DDrawCompat)
- dxwrapper 開源，整合 DDrawCompat 並可載入自訂 DLL 與 asi 外掛 [推測](https://github.com/elishacloud/dxwrapper)
- Wiz8 低幀率修正是裝相容 shim（DXPrimaryEmulation、DisableMaxWindowedMode），不是 ddraw 代理 [推測](https://community.pcgamingwiki.com/files/file/348-wizardry-8-low-frame-rate-fix/)
- dgVoodoo2 的授權條款與原始碼位置在本次唯讀調查中未取得（官方站僅歷史二進位發布頁） [未取得](http://dege.freeweb.hu/dgVoodoo2/dgVoodoo2.html)
- 業界串接慣例是改名 chain-load：把下一層 wrapper 的 ddraw.dll 改名（如 wrapper_ddraw、wdraw、fdraw），自家 ddraw.dll 留原位轉發 [推測](https://github.com/norain211/barbie-secret-agent-win-11-patch/blob/HEAD/FRAMEWORK.md)
- Dege 本人給過 dgVoodoo 串接法（改名 fdraw.dll 加 hex-edit 相鄰模組內 ddraw.dll 字串），但屬第三方研究筆記轉述、非官方文件 [推測](https://github.com/code-herder/ta-impure-patch/blob/HEAD/research/notes/api-wrappers.md)
- dgVoodoo2 官方是否支援被包或包別人，本次未在公開文件找到說法 [未取得](http://dege.freeweb.hu/dgVoodoo2/dgVoodoo2.html)
- cnc-ddraw 公開 README 安裝節只寫放 ddraw.dll 加 config.exe，未寫轉發下一層的參數 [未取得](https://github.com/FunkyFr3sh/cnc-ddraw)
- Ultimate ASI Loader 是現成代理先例（dinput8 與 dsound 代理加載入 asi 外掛），可比照做 ddraw 代理加 sidecar [推測](https://github.com/ThirteenAG/Ultimate-ASI-Loader)
- Wiz8.exe 對 DDRAW.DLL 只匯入 DirectDrawCreate 一個函數（檔偏移 0x27912a 處） [已驗證](Wiz8.exe:0x27912a)
- GDI32.DLL 只匯入 GetStockObject；408 個匯入名中無 TextOut、ExtTextOut、DrawText、GetDC、CreateFont、BitBlt [已驗證](Wiz8.exe:0x27913d)
- 反組譯重建的 SGP Font.cpp 是自有字型引擎（mprintf 與 gprintf 經 Blt8BPPDataTo 系列 blit STI 點陣字），全檔零 GDI 文字呼叫 [推測](https://raw.githubusercontent.com/agluszak/wizardry-8-decomp/main/src/sgp/Font.cpp)
- 推論：代理即使掛住 GDI 文字函數也攔不到 Wiz8 文字；該掛 DirectDrawCreate 回傳 COM 物件的 vtable（Blt、BltFast、Lock） [推測](Wiz8.exe:0x27912a)
- sr.dll 與 srDD_DirectX7.dll 靜態匯入無 DDRAW（後者僅 USER32、MSVCRT、KERNEL32），DDraw 建構集中在 Wiz8.exe [已驗證](Wiz8.exe:0x2790c8)
- w8u 附 d3dim700.dll（797184 B）且其匯入表含 DDRAW.dll，為 FanPatch 系 D3D 包裝層候選 [已驗證](d3dim700.dll:797184)
- Wiz8.exe 無內嵌 manifest（requestedExecutionLevel、compatibility、TRUSTINFO 皆零命中），提權只可能來自外部 manifest、AppCompatFlags 或手動 [已驗證](Wiz8.exe:3580513)
- C2 已實測遊戲目錄 ddraw.dll 先於系統版載入（22 匯出全解），與 MS 標準搜尋順序（exe 目錄先於系統目錄）一致 [已驗證](notes/2026-10-07-c2-ddraw-proxy.md:6)
- RUNASADMIN 與 HIGHDPIAWARE 不改變 DLL 搜尋順序；HIGHDPIAWARE 只關 DPI 虛擬化 [推測](https://learn.microsoft.com/en-us/windows/win32/dlls/dynamic-link-library-search-order)
- 同一使用者提權前後 %TEMP% 為同一路徑（換使用者才變）；__COMPAT_LAYER 設 RunAsInvoker 可擋自動提權 [推測](https://devblogs.microsoft.com/oldnewthing/20161117-00/?p=94735)
- DDrawCompat 官方指南：AcSpecfc 類 shim 會強制載系統 ddraw，繞法為改 exe 名或加外部 manifest [推測](https://github.com/narzoul/DDrawCompat/wiki/Installation-guide)
- DDrawCompat 自身是轉發式包裝（仍呼叫系統 ddraw）；與自研代理同佔遊戲目錄 ddraw.dll 檔名，只能二選一或改名串接 [推測](https://github.com/narzoul/DDrawCompat/wiki/Installation-guide)
- 同一指南註明管理員權限下 COM 重導向失效，exe 勿勾選以管理員執行 [推測](https://github.com/narzoul/DDrawCompat/wiki/Installation-guide)
- Win8 起 DirectDraw 由系統模擬層承接；Wiz8 選項另有 Correct for Blurred Text 修糊字 [推測](https://www.pcgamingwiki.com/wiki/Wizardry_8)
- 本機無 Windows，shim 與 DDrawCompat 加自研代理三層疊加的實際行為未驗 [未取得](https://github.com/narzoul/DDrawCompat/wiki/Installation-guide)

## 命令紀錄

樣本經無空白 symlink 掛入沙箱，只讀（`w8`=Steam Wizardry8 目錄，`w8u`=Wizardry8_Uninstall）。網頁抓取皆用 `curl -A "wiz8-mod personal research (non-commercial)"`，未下載、未執行任何第三方工具。

    stat -c %s w8/Wiz8.exe
    # 3580513（與單上背景一致）

    objdump -p w8/Wiz8.exe | grep "DLL Name"
    # DDRAW.DLL / GDI32.DLL / KERNEL32.DLL / MSVCP60.DLL / MSVCRT.DLL / USER32.DLL / BINKW32.DLL / MSS32.DLL / SR.DLL（共 9 個）
    # 注：此 binutils 的 objdump -p 只列 DLL 名、不列函數名，故函數名改從檔內 raw 字串解析（見下）。

    python3 raw 解析匯入名區（DLL 名單在檔偏移 0x2790c8，函數名起於第二個 DirectDrawCreate=0x27912a，GetStockObject=0x27913d）
    # 408 個匯入名：DirectDrawCreate 唯一 DDRAW 函數；GetStockObject 唯一 GDI32 函數
    # DrawText/TextOut/ExtText/GetDC/ReleaseDC/CreateFont/CreateDC/BitBlt/StretchBlt/SetBk/GetGlyph/SelectObject/DeleteDC/DeleteObject/CreateCompatible/LoadString/wsprintf/Direct3D 全部 NONE
    # 唯一 DrawText 命中是 0x24bec0 的除錯字串 DrawTextMessage（非匯入）
    # 第一處 DirectDrawCreate=0x1fe082 為 bound-import 殘留，非真匯入表

    python3 計數 manifest 關鍵字
    # requestedExecutionLevel 0 / HIGHDPIAWARE 0 / RUNASADMIN 0 / compatibility 0 / manifest 0 / TRUSTINFO 0

    objdump -p w8/Dll/srDD_DirectX7.dll | grep "DLL Name"
    # USER32.dll / MSVCRT.dll / KERNEL32.dll（無 DDRAW）
    objdump -p w8/Dll/srDD_Software.dll | grep "DLL Name"
    # KERNEL32 / USER32 / GDI32 / MSVCRT（無 DDRAW）
    python3 計數 w8/sr.dll 內 DirectDrawCreate/TextOut/ExtTextOut/DrawText/GetDC/DDRAW
    # 全 0
    objdump -p w8u/d3dim700.dll | grep "DLL Name"
    # msvcrt / USER32 / ADVAPI32 / DDRAW.dll / KERNEL32（797184 B）

    cat w8/3DVideo.CFG
    # Glide2x / 1280 / 1024 / 16 / Miles Fast 2D Positional Audio
    ls w8/Dll/ | grep -i -c extender
    # 0（本機無 Wiz8Extender；srVP_* 僅 486/AMD3DNow/Generic/KNI/Pentium 系）
    sed -n '80,95p' w8u/wiz8.ini
    # 87 行 Fan_Patch_1_28_Localization=（空值），91 行 Language=2
    ls w8u/Localization/Fan-Patch1.28/
    # 1.28 default eng text.txt / POL

    curl GitHub API（narzoul/DDrawCompat、FunkyFr3sh/cnc-ddraw、elishacloud/dxwrapper、ThirteenAG/Ultimate-ASI-Loader、agluszak/wizardry-8-decomp）
    # DDrawCompat 描述 DirectDraw and Direct3D 1-7 compatibility；LICENSE.txt 為 ISC 式授權；wiki Installation-guide 有 Method 1-4（含管理員下 COM 重導向失效、AcSpecfc 繞法）
    # cnc-ddraw 描述 GDI/OpenGL/D3D9 重寫 DirectDraw，license MIT；README 明說不支援 Direct3D 遊戲
    # dxwrapper README 明載 DDrawCompat 整合與載入自訂 dll／asi
    # decomp src/sgp 下有 Font.cpp、vobject_blitters.h 等；Font.cpp 全文 grep GetDC|TextOut|ExtTextOut|DrawText|CreateFont 計數 0，匯出函數含 mprintf、gprintf、gprintfDirty、LoadFontFile
    curl PCGamingWiki Wiz8、GOG 論壇串、Steam 討論串、VOGONS、DDrawCompat wiki、MS DLL 搜尋順序文件、OldNewThing RunAsInvoker
    # 關鍵引文見發現段各條 URL；dgVoodoo2 官方站（dege.freeweb.hu）本次直接抓取無回應

## cc 抽查（2026-10-07，3 條，全數核實）
- 第 20 行（DDRAW.DLL 只匯入 DirectDrawCreate，0x27912a）：`dd skip=0x279128` 讀出 `..DirectDrawCreate`；objdump 匯入表 DDRAW.DLL 只有一列。核實。
- 第 21 行（無 TextOut／ExtTextOut／DrawText／GetDC／CreateFont／BitBlt）：`objdump -p Wiz8.exe | grep -ciE` 六個名字 0 命中。核實（「所見範圍內未見」）。
- 第 22 行（agluszak/wizardry-8-decomp `src/sgp/Font.cpp` 有 mprintf／gprintf／Blt8BPPDataTo）：curl HTTP 200，三個符號共 26 處。核實存在；「全檔零 GDI 呼叫」未逐行驗。
- rc=6「沙箱繞道特徵」為誤報：命中第 54 行斜線分隔的 API 名單（被當成長 base64），diff 內無 LD_PRELOAD／execve。

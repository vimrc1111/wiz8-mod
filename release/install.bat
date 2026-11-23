@echo off
rem wiz8-mod install: builds Patches\PATCH.002 from YOUR original game + our translations, then copies ddraw.dll.
rem Usage: install.bat "C:\path\to\Wizardry8"   (default: Steam path)
setlocal
set "G=%~1"
if "%G%"=="" set "G=C:\Program Files (x86)\Steam\steamapps\common\Wizardry8"
set "H=%~dp0"
if not exist "%G%\Data\DATA.SLF" (echo [wiz8-mod] Not a Wizardry 8 folder: "%G%" & exit /b 1)
if not exist "%G%\Patches\PATCH.001" (echo [wiz8-mod] Missing Patches\PATCH.001 - install the official 1.2.4 patch first & exit /b 1)
rem Upgrade: an older wiz8-mod is removed first; uninstall.bat refuses (and we stop) if ddraw.dll belongs to something else.
if exist "%G%\ddraw.dll" (call "%H%uninstall.bat" "%G%" || (echo [wiz8-mod] "%G%\ddraw.dll" is in the way - nothing installed & exit /b 1))
if exist "%G%\Patches\PATCH.002" (echo [wiz8-mod] "%G%\Patches\PATCH.002" already exists - run uninstall.bat or move it away first & exit /b 1)
"%H%zhpatch.exe" "%G%" "%H%translations" "%G%\Patches\PATCH.002" || (echo [wiz8-mod] zhpatch failed - nothing installed & exit /b 1)
copy /b "%H%ddraw.dll" "%G%\ddraw.dll" >nul && goto copied
del "%G%\Patches\PATCH.002"
if exist "%G%\Patches\PATCH.002" (echo [wiz8-mod] copy ddraw.dll failed and "%G%\Patches\PATCH.002" could not be removed - delete it by hand & exit /b 1)
echo [wiz8-mod] copy ddraw.dll failed - rolled back & exit /b 1
:copied
echo [wiz8-mod] Installed: "%G%\ddraw.dll", "%G%\Patches\PATCH.002"

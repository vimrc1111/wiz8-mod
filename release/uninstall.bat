@echo off
rem wiz8-mod uninstall: removes the two files install.bat added, plus wiz8mod.ini (settings the in-game menu writes back). Original game files are never touched.
setlocal
set "G=%~1"
if "%G%"=="" set "G=C:\Program Files (x86)\Steam\steamapps\common\Wizardry8"
if not exist "%G%\Data\DATA.SLF" (echo [wiz8-mod] Not a Wizardry 8 folder: "%G%" & exit /b 1)
rem wiz8mod.ini only ever comes from our menu; drop it even when the dll is already gone.
if exist "%G%\wiz8mod.ini" del "%G%\wiz8mod.ini"
rem PATCH.002 has no identity of its own: only removed together with OUR ddraw.dll.
rem "Ours" = any wiz8-mod build (it embeds its log name); builds are not byte-identical, so fc against this package would refuse older installs.
if not exist "%G%\ddraw.dll" if not exist "%G%\Patches\PATCH.002" (echo [wiz8-mod] Nothing to remove in "%G%" & exit /b 0)
if not exist "%G%\ddraw.dll" (echo [wiz8-mod] No ddraw.dll - cannot tell whether "%G%\Patches\PATCH.002" is ours, left alone. Delete it by hand only if wiz8-mod made it & exit /b 1)
findstr /m /c:"wiz8-mod-ddraw.log" "%G%\ddraw.dll" >nul || (echo [wiz8-mod] "%G%\ddraw.dll" is not ours - left alone, nothing removed & exit /b 1)
del "%G%\ddraw.dll"
if exist "%G%\Patches\PATCH.002" del "%G%\Patches\PATCH.002"
if exist "%G%\ddraw.dll" (echo [wiz8-mod] Could not remove "%G%\ddraw.dll" - is the game running? & exit /b 1)
if exist "%G%\Patches\PATCH.002" (echo [wiz8-mod] Could not remove "%G%\Patches\PATCH.002" & exit /b 1)
echo [wiz8-mod] Removed ddraw.dll, Patches\PATCH.002 and wiz8mod.ini from "%G%"

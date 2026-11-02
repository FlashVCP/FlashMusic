@echo off
setlocal enabledelayedexpansion

echo ===================================================
echo   FlashMusic Windows Official Build Pipeline
echo ===================================================

cd /d "%~dp0\.."

:: 1. Terminate any running FlashMusic instances
taskkill /F /IM FlashMusic.exe >nul 2>&1

:: 2. Verify prerequisites
if not exist "mygo.json" (
    echo [ERROR] mygo.json not found in %CD%!
    exit /b 1
)

if not exist "resources\icon.png" (
    echo [ERROR] resources\icon.png not found!
    exit /b 1
)

:: 3. Clean root, bin, dist and release from accidental stray executables
if exist "FlashMusic.exe" del /f /q "FlashMusic.exe"
if exist "netease-music-mygo.exe" del /f /q "netease-music-mygo.exe"
if exist "bin\FlashMusic.exe" del /f /q "bin\FlashMusic.exe"
for /d %%i in ("dist\.old*") do rmdir /s /q "%%i" 2>nul
for /d %%i in ("release\.old*") do rmdir /s /q "%%i" 2>nul
for /d %%i in ("release\.staging*") do rmdir /s /q "%%i" 2>nul
for /d %%i in ("dist\.staging*") do rmdir /s /q "%%i" 2>nul
if exist "release\windows-amd64" rmdir /s /q "release\windows-amd64" 2>nul
if exist "dist\windows-amd64" rmdir /s /q "dist\windows-amd64" 2>nul

:: 4. Run official MyGo build pipeline strictly in repository root
echo [*] Executing: go tool mygo build -platform windows/amd64 ...
go tool mygo build -platform windows/amd64
if %errorlevel% neq 0 (
    echo [ERROR] Build failed with exit code %errorlevel%!
    exit /b %errorlevel%
)

:: 5. Verify outputs
set "OUTDIR="
if exist "release\windows-amd64\FlashMusic.exe" set "OUTDIR=release"
if exist "dist\windows-amd64\FlashMusic.exe" set "OUTDIR=dist"

if defined OUTDIR (
    echo.
    echo ===================================================
    echo   [SUCCESS] Build completed successfully!
    echo   - Executable: !OUTDIR!\windows-amd64\FlashMusic.exe
    if exist "!OUTDIR!\windows-amd64\flashmusic-setup-0.1.0-amd64.exe" (
        echo   - Installer:  !OUTDIR!\windows-amd64\flashmusic-setup-0.1.0-amd64.exe
    )
    if exist "!OUTDIR!\windows-amd64\update-windows-amd64.json" (
        echo   - Update Feed: !OUTDIR!\windows-amd64\update-windows-amd64.json
    )
    if exist "!OUTDIR!\windows-amd64\flashmusic-0.1.0-windows-amd64.tar.gz" (
        echo   - Update Archive: !OUTDIR!\windows-amd64\flashmusic-0.1.0-windows-amd64.tar.gz
    )
    echo ===================================================
) else (
    echo [ERROR] Output binary was not created in dist\ or release\!
    exit /b 1
)

endlocal

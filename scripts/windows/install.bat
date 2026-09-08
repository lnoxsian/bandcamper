@echo off
setlocal enabledelayedexpansion

:: ============================================================================
:: Bandcamper Windows Installer (Batch)
:: Installs bandcamper.exe and adds it to the user's PATH environment variable.
:: ============================================================================

title Bandcamper Installer

echo ============================================================================
echo  Bandcamper Windows Installer
echo ============================================================================
echo.

:: Determine script directory (with trailing slash)
set "SCRIPT_DIR=%~dp0"

:: 1. Locate bandcamper.exe binary
set "BIN_SRC="
if exist "%SCRIPT_DIR%bandcamper.exe" (
    set "BIN_SRC=%SCRIPT_DIR%bandcamper.exe"
) else if exist "%SCRIPT_DIR%..\..\bin\bandcamper.exe" (
    set "BIN_SRC=%SCRIPT_DIR%..\..\bin\bandcamper.exe"
) else if exist "%SCRIPT_DIR%..\..\dist\bandcamper-windows-amd64.exe" (
    set "BIN_SRC=%SCRIPT_DIR%..\..\dist\bandcamper-windows-amd64.exe"
) else if exist "%SCRIPT_DIR%..\..\dist\bandcamper-windows-arm64.exe" (
    set "BIN_SRC=%SCRIPT_DIR%..\..\dist\bandcamper-windows-arm64.exe"
) else if exist "%SCRIPT_DIR%..\..\bin\bandcamper-windows-amd64.exe" (
    set "BIN_SRC=%SCRIPT_DIR%..\..\bin\bandcamper-windows-amd64.exe"
) else if exist "%SCRIPT_DIR%..\..\bin\bandcamper-windows-arm64.exe" (
    set "BIN_SRC=%SCRIPT_DIR%..\..\bin\bandcamper-windows-arm64.exe"
)

if not defined BIN_SRC (
    echo [ERROR] Could not find 'bandcamper.exe' in:
    echo         %SCRIPT_DIR%
    echo Please make sure 'bandcamper.exe' is in the same folder as this installer.
    goto :FAIL
)

:: 2. Determine installation directory
:: Default: %LOCALAPPDATA%\Programs\bandcamper (standard user-level install directory on Windows)
if not "%~1"=="" (
    set "INSTALL_DIR=%~1"
) else (
    set "INSTALL_DIR=%LOCALAPPDATA%\Programs\bandcamper"
)

echo [INFO] Source binary: %BIN_SRC%
echo [INFO] Target folder: %INSTALL_DIR%
echo.

:: 3. Create target directory
if not exist "%INSTALL_DIR%" (
    mkdir "%INSTALL_DIR%" >nul 2>&1
    if errorlevel 1 (
        echo [ERROR] Failed to create installation directory: "%INSTALL_DIR%"
        goto :FAIL
    )
)

:: 4. Copy binary
echo [INFO] Copying bandcamper.exe...
copy /y "%BIN_SRC%" "%INSTALL_DIR%\bandcamper.exe" >nul
if errorlevel 1 (
    echo [ERROR] Failed to copy 'bandcamper.exe' to "%INSTALL_DIR%".
    echo         Make sure bandcamper is not currently running.
    goto :FAIL
)
echo [OK]   Installed bandcamper.exe to %INSTALL_DIR%

:: Copy documentation if present
if exist "%SCRIPT_DIR%README.md" (
    copy /y "%SCRIPT_DIR%README.md" "%INSTALL_DIR%\" >nul 2>&1
) else if exist "%SCRIPT_DIR%..\..\README.md" (
    copy /y "%SCRIPT_DIR%..\..\README.md" "%INSTALL_DIR%\" >nul 2>&1
)

if exist "%SCRIPT_DIR%LICENSE" (
    copy /y "%SCRIPT_DIR%LICENSE" "%INSTALL_DIR%\" >nul 2>&1
) else if exist "%SCRIPT_DIR%..\..\LICENSE" (
    copy /y "%SCRIPT_DIR%..\..\LICENSE" "%INSTALL_DIR%\" >nul 2>&1
)

:: 5. Add to User PATH via PowerShell
echo [INFO] Checking PATH environment variable...
powershell -NoProfile -ExecutionPolicy Bypass -Command ^
    "$installPath = '%INSTALL_DIR%'.TrimEnd('\'); " ^
    "$userPath = [Environment]::GetEnvironmentVariable('Path', 'User'); " ^
    "$paths = @(); if ($userPath) { $paths = $userPath -split ';' | Where-Object { $_ -ne '' } }; " ^
    "if ($paths -contains $installPath) { " ^
    "    Write-Host '[INFO] Target directory is already in your user PATH.' -ForegroundColor Cyan; " ^
    "} else { " ^
    "    $newPath = if ($userPath) { ($userPath.TrimEnd(';') + ';' + $installPath) } else { $installPath }; " ^
    "    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User'); " ^
    "    Write-Host '[OK]   Added ' $installPath ' to user PATH.' -ForegroundColor Green; " ^
    "}"

if errorlevel 1 (
    echo [WARN] Could not automatically update user PATH.
    echo        Please manually add "%INSTALL_DIR%" to your PATH environment variable.
)

:: Update current cmd session PATH
set "PATH=%INSTALL_DIR%;%PATH%"

echo.
echo ============================================================================
echo [OK] Installation completed successfully!
echo ============================================================================
echo.
echo You can run bandcamper from any new Command Prompt or PowerShell window:
echo     bandcamper --help
echo.
echo NOTE: Existing terminal windows must be restarted to pick up the updated PATH.
echo.

goto :END

:FAIL
echo.
echo [ERROR] Installation failed.
echo.

:END
:: Pause if launched from Windows Explorer (double-click)
echo %cmdcmdline% | find /i "%~0" >nul && pause

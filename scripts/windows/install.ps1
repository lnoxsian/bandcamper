<#
.SYNOPSIS
    Bandcamper Windows Installer (PowerShell)
    Installs bandcamper.exe and adds it to the PATH environment variable.

.DESCRIPTION
    Installs the bandcamper executable into the user profile programs directory
    (or System directory if requested with administrative privileges) and ensures
    the directory is in the environment PATH.

.PARAMETER InstallDir
    Custom installation directory. Defaults to "$env:LOCALAPPDATA\Programs\bandcamper".

.PARAMETER System
    Install system-wide to "$env:ProgramFiles\bandcamper" (requires Administrator privileges).

.EXAMPLE
    .\install.ps1

.EXAMPLE
    .\install.ps1 -InstallDir "C:\Tools\bandcamper"

.EXAMPLE
    .\install.ps1 -System
#>

[CmdletBinding()]
param(
    [Parameter(Position = 0)]
    [string]$InstallDir = "",

    [Parameter()]
    [switch]$System
)

$ErrorActionPreference = "Stop"

function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host "[OK]   $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-Err {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

Write-Host "============================================================================" -ForegroundColor Cyan
Write-Host " Bandcamper Windows Installer" -ForegroundColor Cyan
Write-Host "============================================================================" -ForegroundColor Cyan
Write-Host ""

# Determine script directory
$scriptDir = $PSScriptRoot
if (-not $scriptDir) {
    $scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
}

# 1. Locate bandcamper.exe
$candidatePaths = @(
    (Join-Path $scriptDir "bandcamper.exe"),
    (Join-Path $scriptDir "..\..\bin\bandcamper.exe"),
    (Join-Path $scriptDir "..\..\dist\bandcamper-windows-amd64.exe"),
    (Join-Path $scriptDir "..\..\dist\bandcamper-windows-arm64.exe"),
    (Join-Path $scriptDir "..\..\bin\bandcamper-windows-amd64.exe"),
    (Join-Path $scriptDir "..\..\bin\bandcamper-windows-arm64.exe")
)

$binSrc = $null
foreach ($path in $candidatePaths) {
    if (Test-Path -Path $path -PathType Leaf) {
        $binSrc = (Resolve-Path $path).ProviderPath
        break
    }
}

if (-not $binSrc) {
    Write-Err "Could not find 'bandcamper.exe' in: $scriptDir"
    Write-Host "Please ensure 'bandcamper.exe' is present before running the installer."
    exit 1
}

# 2. Determine target directory
$pathScope = "User"
if ($System) {
    $isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    if (-not $isAdmin) {
        Write-Err "System-wide installation requires Administrator privileges. Please run PowerShell as Administrator."
        exit 1
    }
    $pathScope = "Machine"
    if ([string]::IsNullOrWhiteSpace($InstallDir)) {
        $InstallDir = Join-Path $env:ProgramFiles "bandcamper"
    }
} elseif ([string]::IsNullOrWhiteSpace($InstallDir)) {
    $InstallDir = Join-Path $env:LOCALAPPDATA "Programs\bandcamper"
}

Write-Info "Source binary: $binSrc"
Write-Info "Target folder: $InstallDir ($pathScope level)"
Write-Host ""

# 3. Create destination directory
if (-not (Test-Path -Path $InstallDir)) {
    try {
        $null = New-Item -ItemType Directory -Path $InstallDir -Force
    } catch {
        Write-Err "Failed to create installation directory '$InstallDir': $($_.Exception.Message)"
        exit 1
    }
}

# 4. Copy binary
$destBin = Join-Path $InstallDir "bandcamper.exe"
Write-Info "Copying bandcamper.exe..."
try {
    Copy-Item -Path $binSrc -Destination $destBin -Force
    Write-Success "Installed binary to $destBin"
} catch {
    Write-Err "Failed to copy '$binSrc' to '$destBin': $($_.Exception.Message)"
    Write-Host "Please ensure bandcamper is not currently in use."
    exit 1
}

# Copy documentation if present
$docSources = @(
    (Join-Path $scriptDir "README.md"),
    (Join-Path $scriptDir "..\..\README.md"),
    (Join-Path $scriptDir "LICENSE"),
    (Join-Path $scriptDir "..\..\LICENSE")
)
foreach ($doc in $docSources) {
    if (Test-Path -Path $doc -PathType Leaf) {
        Copy-Item -Path $doc -Destination (Join-Path $InstallDir (Split-Path -Leaf $doc)) -Force -ErrorAction SilentlyContinue
    }
}

# 5. Check and update PATH environment variable
Write-Info "Checking PATH environment variable ($pathScope)..."
try {
    $currentPath = [Environment]::GetEnvironmentVariable('Path', $pathScope)
    $cleanInstallDir = $InstallDir.TrimEnd('\')

    $existingEntries = @()
    if ($currentPath) {
        $existingEntries = $currentPath -split ';' | Where-Object { $_ -ne "" }
    }

    $alreadyInPath = $false
    foreach ($entry in $existingEntries) {
        if ($entry.TrimEnd('\') -ieq $cleanInstallDir) {
            $alreadyInPath = $true
            break
        }
    }

    if ($alreadyInPath) {
        Write-Info "Target directory is already in your $pathScope PATH."
    } else {
        $newPath = if ($currentPath) { ($currentPath.TrimEnd(';') + ';' + $cleanInstallDir) } else { $cleanInstallDir }
        [Environment]::SetEnvironmentVariable('Path', $newPath, $pathScope)
        Write-Success "Added $cleanInstallDir to $pathScope PATH."
    }

    # Update current session PATH as well
    if ($env:PATH -split ';' -notcontains $cleanInstallDir) {
        $env:PATH = "$cleanInstallDir;$env:PATH"
    }
} catch {
    Write-Warn "Could not update PATH automatically: $($_.Exception.Message)"
    Write-Host "You can manually add '$InstallDir' to your PATH environment variable."
}

Write-Host ""
Write-Host "============================================================================" -ForegroundColor Green
Write-Success "Installation completed successfully!"
Write-Host "============================================================================" -ForegroundColor Green
Write-Host ""
Write-Host "You can run bandcamper from any new Command Prompt or PowerShell terminal:"
Write-Host "    bandcamper --help" -ForegroundColor Yellow
Write-Host ""
Write-Host "NOTE: Existing terminal windows must be restarted to pick up the updated PATH."
Write-Host ""

# ==============================================================================
# mncode 1-Line Universal Installer for Windows PowerShell
# Usage: irm https://raw.githubusercontent.com/mncuchiinhuttt/mncode/main/install.ps1 | iex
# ==============================================================================

$ErrorActionPreference = 'Stop'

$Repo = "mncuchiinhuttt/mncode"
$BinaryName = "mncode.exe"
$InstallDir = "$env:LOCALAPPDATA\Programs\mncode"
$Target = "$InstallDir\$BinaryName"

Write-Host ""
Write-Host "  __  __ _  _  ____ ___  ____  ____ " -ForegroundColor Magenta
Write-Host " (  \/  ( \( )/ ___/ _ \(    \(  __)  mncode CLI Installer (Windows)" -ForegroundColor Cyan
Write-Host "  )    ( )  (( (__ )(_) )) D ( ) _)   Claude Code Golang Clone" -ForegroundColor Cyan
Write-Host " (_/\/\_(_)\_)\____\___/(____/(____) " -ForegroundColor Magenta
Write-Host ""

# 1. Ensure Install Directory Exists
if (!(Test-Path -Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$TempDir = [System.IO.Path]::GetTempPath() + [System.Guid]::NewGuid().ToString()
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null
$TempFile = "$TempDir\$BinaryName"

try {
    # 2. Check if Go is installed to build from source
    $GoInstalled = Get-Command go -ErrorAction SilentlyContinue
    $Built = $false

    if ($GoInstalled) {
        Write-Host "Building latest mncode from source using Go..." -ForegroundColor Cyan
        $SrcDir = "$TempDir\src"
        git clone --depth 1 "https://github.com/$Repo.git" $SrcDir 2>$null
        if (Test-Path "$SrcDir\cmd\mncode") {
            Push-Location $SrcDir
            go build -ldflags="-s -w" -o $TempFile ./cmd/mncode
            Pop-Location
            if (Test-Path $TempFile) {
                $Built = $true
            }
        }
    }

    # 3. Fallback: Download pre-built release binary from GitHub Releases
    if (!$Built) {
        $Tag = "v0.1.2-beta"
        try {
            $ReleaseData = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases" -UseBasicParsing
            if ($ReleaseData -and $ReleaseData.Count -gt 0) {
                $Tag = $ReleaseData[0].tag_name
            }
        } catch {}
        $ReleaseUrl = "https://github.com/$Repo/releases/download/$Tag/mncode-windows-amd64.exe"
        Write-Host "Downloading pre-built $Tag binary from GitHub Releases..." -ForegroundColor Cyan
        Invoke-WebRequest -Uri $ReleaseUrl -OutFile $TempFile -UseBasicParsing
    }

    if (!(Test-Path $TempFile) -or (Get-Item $TempFile).Length -eq 0) {
        Write-Host "Failed to build or download mncode binary." -ForegroundColor Red
        exit 1
    }

    # 4. Copy to Target
    Copy-Item -Path $TempFile -Destination $Target -Force
    Write-Host "Installed to $Target" -ForegroundColor Green

    # 5. Ensure Directory is in User PATH
    $UserPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User)
    $PathUpdated = $false
    if ($UserPath -notlike "*$InstallDir*") {
        Write-Host "Adding $InstallDir to User PATH..." -ForegroundColor Yellow
        $NewPath = if ([string]::IsNullOrWhiteSpace($UserPath)) { $InstallDir } else { "$UserPath;$InstallDir" }
        [Environment]::SetEnvironmentVariable("Path", $NewPath, [EnvironmentVariableTarget]::User)
        $env:Path += ";$InstallDir"
        $PathUpdated = $true
    }

    # 6. Initialize Config Directory
    $ConfigDir = "$HOME\.mncode"
    if (!(Test-Path $ConfigDir)) {
        New-Item -ItemType Directory -Path $ConfigDir -Force | Out-Null
    }

    Write-Host ""
    Write-Host "=================================================================" -ForegroundColor DarkGray
    Write-Host "✓ Successfully installed mncode to $Target!" -ForegroundColor Green
    Write-Host "=================================================================" -ForegroundColor DarkGray
    Write-Host ""
    Write-Host "🚀 To start using mncode right now:" -ForegroundColor White
    Write-Host "   1. Restart your PowerShell, Terminal, or VS Code window to reload PATH." -ForegroundColor Yellow
    Write-Host "   2. Type 'mncode' and press Enter:" -ForegroundColor White
    Write-Host "      mncode" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "📌 MANUAL PATH SETUP GUIDE (If 'mncode' is not recognized):" -ForegroundColor DarkCyan
    Write-Host "   If your shell does not recognize 'mncode', add the install folder to PATH:" -ForegroundColor Gray
    Write-Host "   - Path to copy: $InstallDir" -ForegroundColor White
    Write-Host "   - Step 1: Press Win + R, type 'sysdm.cpl' and press Enter" -ForegroundColor Gray
    Write-Host "   - Step 2: Select 'Advanced' tab -> Click 'Environment Variables...'" -ForegroundColor Gray
    Write-Host "   - Step 3: Under 'User variables', select 'Path' and click 'Edit...'" -ForegroundColor Gray
    Write-Host "   - Step 4: Click 'New' -> Paste: $InstallDir" -ForegroundColor Gray
    Write-Host "   - Step 5: Click OK on all dialogs and restart your terminal." -ForegroundColor Gray
    Write-Host ""
}
finally {
    if (Test-Path $TempDir) {
        Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

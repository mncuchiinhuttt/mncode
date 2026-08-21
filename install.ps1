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
        $ReleaseUrl = "https://github.com/$Repo/releases/latest/download/mncode-windows-amd64.exe"
        Write-Host "Downloading pre-built binary from GitHub Releases..." -ForegroundColor Cyan
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
    if ($UserPath -notlike "*$InstallDir*") {
        Write-Host "Adding $InstallDir to User PATH..." -ForegroundColor Yellow
        [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", [EnvironmentVariableTarget]::User)
        $env:Path += ";$InstallDir"
    }

    # 6. Initialize Config Directory
    $ConfigDir = "$HOME\.mncode"
    if (!(Test-Path $ConfigDir)) {
        New-Item -ItemType Directory -Path $ConfigDir -Force | Out-Null
    }

    Write-Host ""
    Write-Host "✓ Successfully installed mncode to $Target!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Run mncode in PowerShell or CMD to start pair programming:" -ForegroundColor White
    Write-Host "  mncode" -ForegroundColor Cyan
    Write-Host ""
}
finally {
    if (Test-Path $TempDir) {
        Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

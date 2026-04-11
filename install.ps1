$ErrorActionPreference = "Stop"

$REPO = "everlier/claune"

Write-Host "Installing Claune..."

# Detect Architecture
$ARCH = $env:PROCESSOR_ARCHITECTURE
if ($ARCH -eq "AMD64") {
    $ARCH = "amd64"
} elseif ($ARCH -eq "ARM64") {
    $ARCH = "arm64"
} else {
    Write-Error "Unsupported architecture: $ARCH"
    exit 1
}

Write-Host "Detected Architecture: $ARCH"

# Find latest release
try {
    $LATEST_RELEASE = Invoke-RestMethod -Uri "https://api.github.com/repos/$REPO/releases/latest"
    $LATEST_TAG = $LATEST_RELEASE.tag_name
} catch {
    Write-Error "Failed to fetch latest release version. Check your internet connection."
    exit 1
}

if (-not $LATEST_TAG) {
    Write-Error "Could not determine latest release version."
    exit 1
}

Write-Host "Latest version: $LATEST_TAG"

# Construct download URL
$FILENAME = "claune-windows-${ARCH}.exe"
$DOWNLOAD_URL = "https://github.com/$REPO/releases/download/${LATEST_TAG}/${FILENAME}"

$INSTALL_DIR = "$HOME\.local\bin"
if (-not (Test-Path -Path $INSTALL_DIR)) {
    New-Item -ItemType Directory -Force -Path $INSTALL_DIR | Out-Null
}

$DEST_FILE = "$INSTALL_DIR\claune.exe"

Write-Host "Downloading $FILENAME to $DEST_FILE..."
try {
    Invoke-WebRequest -Uri $DOWNLOAD_URL -OutFile $DEST_FILE
} catch {
    Write-Error "Failed to download $FILENAME. Ensure the URL is correct."
    exit 1
}

Write-Host ""
Write-Host "Claune installed successfully to $DEST_FILE"

Write-Host "Setting up shell completions..."
try {
    $PROFILE_DIR = Split-Path -Parent $PROFILE
    if (-not (Test-Path -Path $PROFILE_DIR)) {
        New-Item -ItemType Directory -Force -Path $PROFILE_DIR | Out-Null
    }
    $COMPLETION_FILE = Join-Path $PROFILE_DIR "claune-completion.ps1"
    & $DEST_FILE completion powershell | Out-File -FilePath $COMPLETION_FILE -Encoding utf8
    Write-Host "Installed powershell completion to $COMPLETION_FILE."

    if (Test-Path -Path $PROFILE) {
        $PROFILE_CONTENT = Get-Content -Path $PROFILE -Raw
        if ($PROFILE_CONTENT -notmatch "claune-completion.ps1") {
            Add-Content -Path $PROFILE -Value "`n. `"$COMPLETION_FILE`""
            Write-Host "Added completion to your PowerShell profile."
        }
    } else {
        New-Item -ItemType File -Force -Path $PROFILE | Out-Null
        Add-Content -Path $PROFILE -Value ". `"$COMPLETION_FILE`""
        Write-Host "Created PowerShell profile and added completion."
    }
} catch {
    Write-Host "Warning: Could not install powershell completion." -ForegroundColor Yellow
}

Write-Host ""
if ($env:PATH -notmatch [regex]::Escape($INSTALL_DIR)) {
    Write-Host "Warning: $INSTALL_DIR is not in your PATH." -ForegroundColor Yellow
    Write-Host "You may need to add it to your System or User environment variables." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "To get started, run:" -ForegroundColor Green
Write-Host "  claune install" -ForegroundColor Green
Write-Host ""

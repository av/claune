$ErrorActionPreference = "Stop"

$REPO = "av/claune"

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

if ($env:PATH -notmatch [regex]::Escape($INSTALL_DIR)) {
    Write-Host "Warning: $INSTALL_DIR is not in your PATH." -ForegroundColor Yellow
    Write-Host "You may need to add it to your System or User environment variables." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "To get started, run:" -ForegroundColor Green
Write-Host "  claune install" -ForegroundColor Green
Write-Host ""

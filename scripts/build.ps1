# Build script for TalkRealm (Windows PowerShell)

Write-Host "Building TalkRealm..." -ForegroundColor Green

# Set build variables
$APP_NAME = "talkrealm"
$OUTPUT_DIR = "bin"
$VERSION = "dev"
$BUILD_TIME = Get-Date -Format "yyyy-MM-dd_HH:mm:ss"

# Try to get git version
try {
    $VERSION = git describe --tags --always --dirty 2>$null
    if (-not $VERSION) { $VERSION = "dev" }
} catch {
    $VERSION = "dev"
}

# Create output directory
New-Item -ItemType Directory -Force -Path $OUTPUT_DIR | Out-Null

# Build for Windows
Write-Host "Building for Windows..." -ForegroundColor Yellow
go build -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" `
  -o "$OUTPUT_DIR\$APP_NAME.exe" `
  cmd\server\main.go

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build complete: $OUTPUT_DIR\$APP_NAME.exe" -ForegroundColor Green
} else {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}

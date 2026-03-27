# PowerShell build script for Windows development
# Usage: .\scripts\build.ps1 <target>
# Targets: build, test, test-all, coverage, lint, deps, clean, help

param(
    [Parameter(Position = 0)]
    [string]$Target = "help"
)

$ErrorActionPreference = "Stop"

$BinaryName = "repodocs"
$BuildDir = ".\build"

# Version info from git
$Version = git describe --tags --always --dirty 2>$null
if (-not $Version) { $Version = "dev" }
$BuildTime = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$Commit = git rev-parse --short HEAD 2>$null
if (-not $Commit) { $Commit = "unknown" }

$LDFlags = "-X github.com/quantmind-br/repodocs/pkg/version.Version=$Version -X github.com/quantmind-br/repodocs/pkg/version.BuildTime=$BuildTime -X github.com/quantmind-br/repodocs/pkg/version.Commit=$Commit -s -w"

function Invoke-Build {
    Write-Host "Building $BinaryName..." -ForegroundColor Cyan
    New-Item -ItemType Directory -Force -Path $BuildDir | Out-Null
    $env:CGO_ENABLED = "0"
    go build -ldflags $LDFlags -o "$BuildDir\$BinaryName.exe" ./cmd/repodocs
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    Write-Host "Built: $BuildDir\$BinaryName.exe" -ForegroundColor Green
}

function Invoke-Test {
    Write-Host "Running unit tests..." -ForegroundColor Cyan
    go test -v -race -short ./...
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

function Invoke-TestAll {
    Write-Host "Running all tests..." -ForegroundColor Cyan
    go test -v -race ./...
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

function Invoke-Coverage {
    Write-Host "Generating coverage report..." -ForegroundColor Cyan
    New-Item -ItemType Directory -Force -Path .\coverage | Out-Null
    go test -coverprofile=.\coverage\coverage.out -covermode=atomic ./...
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    go tool cover -html=.\coverage\coverage.out -o .\coverage\coverage.html
    go tool cover -func=.\coverage\coverage.out
    Write-Host "Report: .\coverage\coverage.html" -ForegroundColor Green
}

function Invoke-Lint {
    Write-Host "Running linters..." -ForegroundColor Cyan
    gofmt -s -w .
    golangci-lint run ./...
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

function Invoke-Deps {
    Write-Host "Downloading dependencies..." -ForegroundColor Cyan
    go mod download
    go mod tidy
}

function Invoke-Clean {
    Write-Host "Cleaning build artifacts..." -ForegroundColor Cyan
    if (Test-Path $BuildDir) { Remove-Item -Recurse -Force $BuildDir }
    if (Test-Path .\coverage) { Remove-Item -Recurse -Force .\coverage }
    if (Test-Path .\dist) { Remove-Item -Recurse -Force .\dist }
    Write-Host "Clean." -ForegroundColor Green
}

function Show-Help {
    Write-Host ""
    Write-Host "RepoDocs Build Script (PowerShell)" -ForegroundColor Cyan
    Write-Host "Usage: .\scripts\build.ps1 <target>" -ForegroundColor White
    Write-Host ""
    Write-Host "Targets:" -ForegroundColor Yellow
    Write-Host "  build      Build the binary"
    Write-Host "  test       Run unit tests (-short, -race)"
    Write-Host "  test-all   Run all tests (unit + integration + e2e)"
    Write-Host "  coverage   Generate HTML coverage report"
    Write-Host "  lint       Run gofmt + golangci-lint"
    Write-Host "  deps       Download and tidy dependencies"
    Write-Host "  clean      Remove build artifacts"
    Write-Host "  help       Show this help"
    Write-Host ""
}

switch ($Target) {
    "build"    { Invoke-Build }
    "test"     { Invoke-Test }
    "test-all" { Invoke-TestAll }
    "coverage" { Invoke-Coverage }
    "lint"     { Invoke-Lint }
    "deps"     { Invoke-Deps }
    "clean"    { Invoke-Clean }
    "help"     { Show-Help }
    default    { Write-Host "Unknown target: $Target" -ForegroundColor Red; Show-Help; exit 1 }
}

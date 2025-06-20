#!/usr/bin/env pwsh

# Test script with linting
param(
    [switch]$Verbose
)

$ErrorActionPreference = "Stop"

Write-Host "Running Go tests with linting..." -ForegroundColor Green

# Check code formatting
Write-Host "`nChecking code formatting..." -ForegroundColor Yellow
$formatResult = go fmt ./...
if ($formatResult) {
    Write-Host "Code formatting issues found:" -ForegroundColor Red
    Write-Host $formatResult
    exit 1
}
Write-Host "Code formatting OK" -ForegroundColor Green

# Run go vet
Write-Host "`nRunning go vet..." -ForegroundColor Yellow
go vet ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host "go vet found issues" -ForegroundColor Red
    exit 1
}
Write-Host "go vet OK" -ForegroundColor Green

# Run staticcheck
Write-Host "`nRunning staticcheck..." -ForegroundColor Yellow
staticcheck ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host "staticcheck found issues" -ForegroundColor Red
    exit 1
}
Write-Host "staticcheck OK" -ForegroundColor Green

# Run tests
Write-Host "`nRunning tests..." -ForegroundColor Yellow
if ($Verbose) {
    go test -v ./...
} else {
    go test ./...
}

if ($LASTEXITCODE -ne 0) {
    Write-Host "Tests failed" -ForegroundColor Red
    exit 1
}

Write-Host "`nAll checks passed!" -ForegroundColor Green

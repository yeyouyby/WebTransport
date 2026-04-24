param(
  [switch]$SkipWingetUpgrade
)

$ErrorActionPreference = "Stop"

function Ensure-Command {
  param([string]$Name)
  if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
    throw "missing command: $Name"
  }
}

Ensure-Command winget

if (-not $SkipWingetUpgrade) {
  winget source update
}

$commonArgs = @("--accept-package-agreements", "--accept-source-agreements", "--silent")

winget install --id Git.Git -e @commonArgs
winget install --id GoLang.Go -e @commonArgs
winget install --id jqlang.jq -e @commonArgs
winget install --id cURL.cURL -e @commonArgs

Write-Host "windows one-click install finished"
Write-Host "verify tools:"
git --version
go version
jq --version
curl --version

Write-Host ""
Write-Host "next: run pressure test"
Write-Host "  powershell -ExecutionPolicy Bypass -File scripts/run_pressure_test.ps1 -Url https://127.0.0.1:8443/fallback -Requests 1000 -Concurrency 20"

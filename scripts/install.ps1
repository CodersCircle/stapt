# One-command Windows install:
#   irm https://raw.githubusercontent.com/CodersCircle/stapt/main/scripts/install.ps1 | iex
$ErrorActionPreference = "Stop"
$url = "https://github.com/CodersCircle/stapt/releases/latest/download/STAPT-Windows-x64-setup.exe"
$out = Join-Path $env:TEMP "STAPT-Windows-x64-setup.exe"
Write-Host "Downloading STAPT installer..."
Invoke-WebRequest -Uri $url -OutFile $out -UseBasicParsing
Write-Host "Installing (no admin required)..."
Start-Process -FilePath $out -ArgumentList "/S" -Wait
$exe = Join-Path $env:LOCALAPPDATA "Programs\STAPT\STAPT.exe"
if (-not (Test-Path $exe)) {
    $exe = Join-Path ${env:ProgramFiles} "STAPT\STAPT.exe"
}
if (Test-Path $exe) {
    Write-Host "STAPT installed. Opening from the Start Menu shortcut."
    Start-Process $exe
} else {
    Write-Host "Installer finished. Open STAPT from the Start Menu."
}

# One-time install: STAPT → %LOCALAPPDATA%\Programs\STAPT
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

$src = Join-Path $PSScriptRoot "build\bin\STAPT.exe"
if (-not (Test-Path $src)) {
    Write-Host "Building STAPT..."
    & cmd /c build.bat
    $src = Join-Path $PSScriptRoot "build\bin\STAPT.exe"
}

$destDir = Join-Path $env:LOCALAPPDATA "Programs\STAPT"
New-Item -ItemType Directory -Force -Path $destDir | Out-Null
Copy-Item -Force $src (Join-Path $destDir "STAPT.exe")

$startMenu = [Environment]::GetFolderPath("Programs")
$shortcut = Join-Path $startMenu "STAPT.lnk"
$wsh = New-Object -ComObject WScript.Shell
$sc = $wsh.CreateShortcut($shortcut)
$sc.TargetPath = Join-Path $destDir "STAPT.exe"
$sc.WorkingDirectory = $destDir
$sc.Save()

Write-Host ""
Write-Host "STAPT installed. Start from Start Menu -> STAPT"
Start-Process (Join-Path $destDir "STAPT.exe")

# One-time install from a local build: STAPT → Start Menu
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
$sc.Description = "STAPT — SSH terminal and SFTP"
$sc.Save()

$desktop = [Environment]::GetFolderPath("Desktop")
$deskSc = Join-Path $desktop "STAPT.lnk"
$sc2 = $wsh.CreateShortcut($deskSc)
$sc2.TargetPath = Join-Path $destDir "STAPT.exe"
$sc2.WorkingDirectory = $destDir
$sc2.Save()

Write-Host ""
Write-Host "STAPT installed. Open it from the Start Menu or the desktop shortcut."
Start-Process (Join-Path $destDir "STAPT.exe")

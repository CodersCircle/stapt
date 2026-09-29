@echo off
setlocal
cd /d "%~dp0"

where go >nul 2>&1 || (
  echo Go 1.22+ is required: https://go.dev/dl/
  exit /b 1
)

where wails >nul 2>&1 || (
  echo Installing Wails CLI...
  go install github.com/wailsapp/wails/v2/cmd/wails@latest
)

wails build -clean %*
echo.
echo Built: build\bin\STAPT.exe
echo Install: powershell -ExecutionPolicy Bypass -File install-windows.ps1

@echo off
cd /d "%~dp0"

echo ============================================
echo   0538pay-api backend (go run, dev mode)
echo ============================================
echo.

where go >/dev/null 2>nul
if errorlevel 1 (
  echo [ERROR] go command not found. Install Go and add it to PATH.
  echo.
  pause
  exit /b 1
)

if not exist "configs\config.yaml" (
  echo [ERROR] configs\config.yaml missing.
  echo.
  pause
  exit /b 1
)

netstat -ano ^| findstr ":8080" ^| findstr "LISTENING" >nul
if not errorlevel 1 (
  echo [WARN] Port 8080 already in use. Stop the running backend first.
  echo.
  pause
  exit /b 1
)

echo Starting backend on :8080 (foreground). Close this window to stop.
echo --------------------------------------------
echo.
set GOPROXY=https://goproxy.cn,direct
go run ./cmd/server

echo.
echo Backend stopped.
pause

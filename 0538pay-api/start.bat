@echo off
cd /d "%~dp0"

echo ============================================
echo   0538pay-api backend launcher
echo ============================================
echo.

if not exist "%~dp00538pay-api.exe" (
  echo [ERROR] 0538pay-api.exe not found. Please build it first.
  echo         Run: go build -o 0538pay-api.exe ./cmd/server
  echo.
  pause
  exit /b 1
)

netstat -ano ^| findstr ":8080" ^| findstr "LISTENING" >nul
if not errorlevel 1 (
  echo [WARN] Port 8080 is already in use. Backend may be running.
  echo        Stop it first with stop.bat if you want to restart.
  echo.
  pause
  exit /b 1
)

echo Starting backend on :8080 ...
start "0538pay-api" /min "%~dp00538pay-api.exe" -config "%~dp0configs"

echo.
echo Backend started in background (minimized window).
echo Health check: http://127.0.0.1:8080/health
echo Stop it with: stop.bat
echo.
timeout /t 3 >nul

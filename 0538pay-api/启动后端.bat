@echo off
chcp 65001 >nul
cd /d "%~dp0"

echo ============================================
echo   0538pay-api 后端启动
echo ============================================
echo.

if not exist "%~dp00538pay-api.exe" (
  echo [错误] 未找到 0538pay-api.exe，请先编译。
  echo.
  pause
  exit /b 1
)

echo 正在后台启动后端（监听 :8080）...
start "0538pay-api" /min "%~dp00538pay-api.exe" -config "%~dp0configs"

echo.
echo 已在后台启动。可关闭本窗口，后端会继续运行。
echo 停止后端：任务管理器结束 0538pay-api.exe，或运行「停止后端.bat」。
echo.
timeout /t 3 >nul

@echo off
chcp 65001 >nul
echo 正在停止 0538pay-api 后端...
taskkill /F /IM 0538pay-api.exe >nul 2>nul
if errorlevel 1 (
  echo 未发现正在运行的后端进程。
) else (
  echo 后端已停止。
)
echo.
timeout /t 2 >nul

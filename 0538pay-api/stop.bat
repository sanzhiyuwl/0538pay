@echo off
echo Stopping 0538pay-api backend ...
taskkill /F /IM 0538pay-api.exe >/dev/null 2>nul
if errorlevel 1 (
  echo No running backend process found.
) else (
  echo Backend stopped.
)
echo.
timeout /t 2 >nul

@echo off
chcp 65001 >nul
setlocal
title 0538pay-api 后端服务

REM 切到脚本所在目录（即 0538pay-api 根目录）
cd /d "%~dp0"

echo ============================================
echo   0538pay-api 后端启动脚本
echo ============================================
echo.

REM 1) 检查 Go 是否可用
where go >nul 2>nul
if errorlevel 1 (
  echo [错误] 未找到 go 命令，请先安装 Go 或将其加入 PATH。
  echo.
  pause
  exit /b 1
)

REM 2) 检查配置文件
if not exist "configs\config.yaml" (
  echo [错误] 缺少 configs\config.yaml，请先复制样例并配置数据库。
  echo.
  pause
  exit /b 1
)

REM 3) 检查 MySQL:3306 是否在监听（后端强依赖）
netstat -ano | findstr ":3306" | findstr "LISTENING" >nul
if errorlevel 1 (
  echo [警告] 未检测到 MySQL(:3306) 在运行。
  echo         请先在 phpStudy 启动 MySQL，否则后端连库会失败。
  echo.
) else (
  echo [OK] MySQL(:3306) 正在运行。
)

REM 4) 检查 Redis:6379（非强依赖，仅提示）
netstat -ano | findstr ":6379" | findstr "LISTENING" >nul
if errorlevel 1 (
  echo [提示] 未检测到 Redis(:6379)。登录等基础功能不受影响；
  echo         若用到缓存/验证码/限流等功能再启动 Redis。
) else (
  echo [OK] Redis(:6379) 正在运行。
)

REM 5) 检查 :8080 是否已被占用（避免重复启动）
netstat -ano | findstr ":8080" | findstr "LISTENING" >nul
if not errorlevel 1 (
  echo.
  echo [警告] 端口 :8080 已被占用，后端可能已在运行。
  echo         如需重启，请先关闭占用该端口的进程再运行本脚本。
  echo.
  pause
  exit /b 1
)

echo.
echo 正在启动后端服务（监听 :8080）...
echo 关闭本窗口即可停止后端。
echo --------------------------------------------
echo.

REM 6) 前台运行，日志直接打印在本窗口
go run ./cmd/server

echo.
echo 后端已停止。
pause

# Go 迁移收尾脚本 —— 必须用「管理员」身份运行
# 右键此文件 → 使用 PowerShell 运行，或在管理员 PowerShell 里执行
$ErrorActionPreference = "Stop"

Write-Host "== 1. 从系统 PATH 移除旧的 C:\Program Files\Go\bin ==" -ForegroundColor Cyan
$sys = [Environment]::GetEnvironmentVariable("Path", "Machine")
$kept = $sys.Split(";") | Where-Object { $_ -ne "" -and $_.TrimEnd("\") -ne "C:\Program Files\Go\bin" }
$newSys = $kept -join ";"
[Environment]::SetEnvironmentVariable("Path", $newSys, "Machine")
Write-Host "新系统 PATH:`n$newSys`n"

Write-Host "== 2. 删除旧安装目录 ==" -ForegroundColor Cyan
if (Test-Path "C:\Program Files\Go") {
    Remove-Item "C:\Program Files\Go" -Recurse -Force
    Write-Host "已删除 C:\Program Files\Go"
} else { Write-Host "C:\Program Files\Go 不存在，跳过" }

if (Test-Path "C:\Users\Zhang\go") {
    Remove-Item "C:\Users\Zhang\go" -Recurse -Force
    Write-Host "已删除 C:\Users\Zhang\go (旧 GOPATH)"
} else { Write-Host "C:\Users\Zhang\go 不存在，跳过" }

Write-Host "`n全部完成。请重开终端后运行:  go env GOROOT GOPATH" -ForegroundColor Green

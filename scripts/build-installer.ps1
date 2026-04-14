$ErrorActionPreference = "Stop"

Set-Location (Join-Path $PSScriptRoot "..")

if (!(Test-Path "dist")) { New-Item -ItemType Directory -Path "dist" | Out-Null }

go mod tidy
go build -o "dist\power-scheduler.exe" ".\cmd\power-scheduler"
go build -o "dist\power-desktop.exe" ".\cmd\power-desktop"

if (Test-Path "dist\web") { Remove-Item -Recurse -Force "dist\web" }
Copy-Item -Recurse "web" "dist\web"

makensis ".\installer\power-scheduler.nsi"
Write-Host "安装包已生成：dist\PowerSchedulerSetup.exe"

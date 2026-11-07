@echo off
cd /d "%~dp0"
set GO111MODULE=off
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-H=windowsgui -s -w" -o "Ebala_0.64.exe" .
if errorlevel 1 (echo Build failed.& pause & exit /b 1)
echo Built Ebala_0.64.exe
pause

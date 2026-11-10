@echo off
cd /d "%~dp0"
if "%~1"=="" (
  go run -buildvcs=false . artisan help
) else (
  go run -buildvcs=false . artisan %*
)
exit /b %errorlevel%

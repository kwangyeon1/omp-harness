@echo off
setlocal
set "OMP_PS=%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe"
if exist "%SystemRoot%\Sysnative\WindowsPowerShell\v1.0\powershell.exe" set "OMP_PS=%SystemRoot%\Sysnative\WindowsPowerShell\v1.0\powershell.exe"
"%OMP_PS%" -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0install-omp-tui.ps1"
exit /b %errorlevel%

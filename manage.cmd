@echo off
setlocal
set "OMP_MANAGER_ARCH=%PROCESSOR_ARCHITECTURE%"
if defined PROCESSOR_ARCHITEW6432 set "OMP_MANAGER_ARCH=%PROCESSOR_ARCHITEW6432%"
if /I "%OMP_MANAGER_ARCH%"=="AMD64" (
    "%~dp0manager\manage-windows-amd64.exe" %*
    exit /b
)
if /I "%OMP_MANAGER_ARCH%"=="ARM64" (
    "%~dp0manager\manage-windows-arm64.exe" %*
    exit /b
)
echo Unsupported architecture
exit /b 1

@echo off
setlocal
if /I "%~1"=="tui" (
    call "%~dp0install-omp-tui.bat"
) else (
    call "%~dp0install-omp.bat"
)
set "OMP_INSTALL_EXIT=%errorlevel%"
echo.
echo Installer finished with exit code %OMP_INSTALL_EXIT%.
pause
exit /b %OMP_INSTALL_EXIT%

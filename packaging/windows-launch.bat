@echo off
setlocal
call "%~dp0manage.cmd" %*
set "OMP_INSTALL_EXIT=%errorlevel%"
echo.
echo Finished with exit code %OMP_INSTALL_EXIT%.
pause
exit /b %OMP_INSTALL_EXIT%

@echo off
call "%~dp0manage.cmd" restore %*
exit /b %errorlevel%

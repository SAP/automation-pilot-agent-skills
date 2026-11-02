@echo off
setlocal EnableExtensions DisableDelayedExpansion
rem Use cmd's status value even if the caller defines an ERRORLEVEL variable.
set "ERRORLEVEL="
rem Verify before starting the bundled binary; keep native argument forwarding in cmd.
"%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe" -NoLogo -NoProfile -NonInteractive -File "%~dp0verify-autopi.ps1"
rem Require exactly zero, including when PowerShell exits with a negative status.
if errorlevel 1 exit /b 1
if not errorlevel 0 exit /b 1
"%~dp0bin\windows-amd64\autopi.exe" %*
exit /b %errorlevel%

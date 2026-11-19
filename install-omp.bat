@echo off
setlocal
rem Self-contained CMD/PowerShell hybrid. Windows 10/11 or Server 2016+.
set "OMP_INSTALL_SELF=%~f0"
set "OMP_INSTALL_REF=%~1"
if not "%~2"=="" (
    echo Usage: install-omp.bat [vVERSION]
    exit /b 1
)
set "OMP_PS=%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe"
if exist "%SystemRoot%\Sysnative\WindowsPowerShell\v1.0\powershell.exe" set "OMP_PS=%SystemRoot%\Sysnative\WindowsPowerShell\v1.0\powershell.exe"
if not exist "%OMP_PS%" (
    echo Error: Windows PowerShell 5.1 is required. Use Windows 10/11 or Server 2016+.
    exit /b 1
)
"%OMP_PS%" -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -Command "try { $text = [IO.File]::ReadAllText($env:OMP_INSTALL_SELF); & ([scriptblock]::Create(($text -split '(?m)^# POWERSHELL_PAYLOAD\r?$', 2)[1])); exit 0 } catch { [Console]::Error.WriteLine($_); exit 1 }"
exit /b %errorlevel%
# POWERSHELL_PAYLOAD
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
if ($PSVersionTable.PSVersion -lt [version]'5.1') { throw 'PowerShell 5.1 or newer is required.' }
if ($env:OMP_INSTALL_REF -and $env:OMP_INSTALL_REF -notmatch '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$') {
    throw 'Expected a release tag such as v18.8.9.'
}
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
$env:PI_BUILD_URL = 'https://build.stencil.so'
if (-not $env:PI_INSTALL_DIR) { $env:PI_INSTALL_DIR = Join-Path $env:LOCALAPPDATA 'omp' }
$installer = Join-Path ([IO.Path]::GetTempPath()) ('omp-bootstrap-' + [guid]::NewGuid().ToString('N') + '.ps1')
try {
    Invoke-WebRequest -UseBasicParsing -Uri 'https://omp.sh/install.ps1' -OutFile $installer -TimeoutSec 120
    $parameters = @{ Binary = $true }
    if ($env:OMP_INSTALL_REF) { $parameters.Ref = $env:OMP_INSTALL_REF }
    # PS 5.1's web cmdlet otherwise depends on legacy Internet Explorer setup.
    $PSDefaultParameterValues['Invoke-WebRequest:UseBasicParsing'] = $true
    & $installer @parameters
    $binary = Join-Path $env:PI_INSTALL_DIR 'omp.exe'
    & $binary --version
    if ($LASTEXITCODE -ne 0) { throw "OMP startup check failed (exit $LASTEXITCODE)." }
    Write-Host "Official OMP installed: $binary"
    Write-Host 'Open a new terminal to use omp from PATH.'
    if ($env:OMP_HARNESS_MANAGER) {
        $itemId = if ($env:OMP_HARNESS_ITEM_ID) { $env:OMP_HARNESS_ITEM_ID } else { 'omp' }
        & $env:OMP_HARNESS_MANAGER state set $itemId true
        if ($LASTEXITCODE -ne 0) { throw 'Failed to record installation state.' }
    }
} finally {
    Remove-Item -LiteralPath $installer -Force -ErrorAction SilentlyContinue
}

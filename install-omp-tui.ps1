$ErrorActionPreference = 'Stop'
& (Join-Path $PSScriptRoot 'manage.cmd') @args
exit $LASTEXITCODE

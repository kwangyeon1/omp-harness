$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot
$version = (Get-Content "$PSScriptRoot/VERSION" -Raw).Trim()
$setup = Join-Path $root "dist/omp-harness-scripts-$version-windows-setup.exe"
$expected = ((Get-Content "$setup.sha256" -Raw) -split '\s+')[0]
if ((Get-FileHash $setup -Algorithm SHA256).Hash -ne $expected) { throw 'Package checksum mismatch' }
$dest = Join-Path $env:TEMP ('omp harness test ' + [guid]::NewGuid())
$oldStatePath = $env:OMP_HARNESS_STATE_FILE
$state = "$dest-state.json"
$env:OMP_HARNESS_STATE_FILE = $state
Set-Content $state '{"omp":true}' -Encoding ASCII
try {
    foreach ($attempt in 1..2) {
        $process = Start-Process -FilePath $setup -ArgumentList @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART', "/DIR=`"$dest`"") -Wait -PassThru
        if ($process.ExitCode -ne 0) { throw "Installation failed: $($process.ExitCode)" }
        foreach ($name in @('install-omp.bat', 'install-omp-tui.bat', 'install-omp-tui.ps1', 'restore.bat', 'restore-tui.bat', 'README.md')) {
            if ((Get-FileHash (Join-Path $dest $name)).Hash -ne (Get-FileHash (Join-Path $root $name)).Hash) { throw "Payload mismatch: $name" }
        }
        if ((Get-Content "$dest/VERSION" -Raw).Trim() -ne $version) { throw 'Version mismatch' }
        $status = & "$dest/manage.cmd" list
        if ($LASTEXITCODE -ne 0 -or ($status -join "`n") -notmatch 'omp \| installed') { throw 'Manager did not read JSON state' }
        $helpText = & "$dest/restore.bat" --help
        if ($LASTEXITCODE -ne 0 -or ($helpText -join "`n") -notmatch 'restore run') { throw 'Restore entry failed' }
        if (!(Test-Path "$dest/launch-installer.bat")) { throw 'Missing launcher' }
    }
    Set-Content "$dest/user-file.txt" 'preserve me'
    $process = Start-Process -FilePath "$dest/unins000.exe" -ArgumentList @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART') -Wait -PassThru
    if ($process.ExitCode -ne 0) { throw 'Uninstall failed' }
    if (Test-Path "$dest/install-omp.bat") { throw 'Package files remain after uninstall' }
    if ((Get-Content "$dest/user-file.txt" -Raw).Trim() -ne 'preserve me') { throw 'User file was removed' }
    if (!(Get-Content $state -Raw | ConvertFrom-Json).omp) { throw 'User state changed' }
} finally {
    $env:OMP_HARNESS_STATE_FILE = $oldStatePath
    Remove-Item $state -Force -ErrorAction SilentlyContinue
    if (Test-Path $dest) { Remove-Item $dest -Recurse -Force }
}

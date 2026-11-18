$ErrorActionPreference = 'Stop'
function Ask($message, $default) { $value = Read-Host "$message [$default]"; if (!$value) { return $default }; return $value }
function Yes($message, $default = 'n') { do { $v = Ask "$message (y/n)" $default } while ($v -notin @('y','n')); return $v -eq 'y' }
Write-Host '=== Official OMP installation wizard ==='
if (Get-Command omp -ErrorAction SilentlyContinue) { & omp --version; if (!(Yes 'Update/replace existing OMP?')) { exit 0 } }
$release = Ask 'Release: latest or vVERSION (older tags downgrade)' 'latest'
if ($release -ne 'latest' -and $release -notmatch '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$') { throw 'Invalid release tag' }
$scope = Ask 'Scope: user or system (system requires an elevated terminal)' 'user'
if ($scope -notin @('user','system')) { throw 'Invalid scope' }
$defaultDir = Join-Path $env:LOCALAPPDATA 'omp'
$target = 'User'
if ($scope -eq 'system') {
 if (!([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) { throw 'Run from an elevated terminal for system installation.' }
 $defaultDir = Join-Path $env:ProgramFiles 'omp'; $target = 'Machine'
}
$dest = Ask 'Installation directory' $defaultDir
if (![IO.Path]::IsPathRooted($dest)) { throw 'Absolute path required' }
$exe = Join-Path $dest 'omp.exe'
if ((Test-Path $exe) -and !(Yes 'Replace destination omp.exe?')) { exit 0 }
$proxy = Ask 'Proxy: keep or URL (use system settings for credentials)' 'keep'
Write-Host 'HTTPS uses Windows certificate trust. Import any private CA into Windows before continuing.'
if (!(Yes 'Use the current Windows certificate trust?' 'y')) { exit 0 }
$pathChange = Yes 'Add installation directory first in PATH?' 'y'
$reset = Yes 'Back up and reset existing settings.json?'
$shell = Ask 'OMP shell: keep, builtin, or full Git Bash executable path' 'keep'
if ($shell -notin @('keep','builtin') -and !(Test-Path -LiteralPath $shell -PathType Leaf)) { throw 'Shell executable not found' }
$extra = Ask 'Optional tools via winget: none, git, python, both' 'none'
if ($extra -notin @('none','git','python','both')) { throw 'Invalid tools selection' }
Write-Host 'Shell completions are available for Bash/Zsh/Fish. Generate them with omp completions SHELL in that shell.'
$action = Ask 'After installation: exit, run, login' 'exit'
if ($action -notin @('exit','run','login')) { throw 'Invalid action' }
Write-Host "Release: $release; directory: $dest; scope: $scope; PATH: $pathChange; reset: $reset; shell: $shell; tools: $extra"
if (!(Yes 'Install now?' 'n')) { exit 0 }
if ($extra -ne 'none') {
 if (!(Get-Command winget -ErrorAction SilentlyContinue)) { throw 'Install App Installer (winget), or choose no optional tools.' }
 $ids = @(); if ($extra -in @('git','both')) { $ids += 'Git.Git' }; if ($extra -in @('python','both')) { $ids += 'Python.Python.3.13' }
 foreach ($id in $ids) { & winget install --id $id --exact; if ($LASTEXITCODE -ne 0) { throw "winget failed: $id" } }
}
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
$arch = $env:PROCESSOR_ARCHITECTURE; if ($env:PROCESSOR_ARCHITEW6432) { $arch = $env:PROCESSOR_ARCHITEW6432 }
$platform = switch ($arch) { AMD64 { 'windows-x86_64' }; ARM64 { 'windows-arm64' }; default { throw 'Unsupported architecture' } }
$url = "https://build.stencil.so/api/products/omp/latest/${platform}?channel=stable"
if ($release -ne 'latest') { $url = "https://build.stencil.so/api/products/omp/versions/$($release.Substring(1))/$platform" }
$web = @{ UseBasicParsing = $true; TimeoutSec = 900 }; if ($proxy -ne 'keep') { $web.Proxy = $proxy }
$metadata = (Invoke-WebRequest @web -Uri $url).Content | ConvertFrom-Json
if ($metadata.file.sha256 -notmatch '^[a-fA-F0-9]{64}$' -or $metadata.download -notlike 'https://*') { throw 'Invalid build metadata' }
New-Item -ItemType Directory -Force -Path $dest | Out-Null
$temp = Join-Path $dest ('.omp-' + [guid]::NewGuid() + '.exe')
try {
 Invoke-WebRequest @web -Uri $metadata.download -OutFile $temp
 if ((Get-FileHash -LiteralPath $temp -Algorithm SHA256).Hash -ne $metadata.file.sha256) { throw 'Checksum mismatch' }
 & $temp --version; if ($LASTEXITCODE -ne 0) { throw 'Downloaded OMP cannot start' }
 Move-Item -LiteralPath $temp -Destination $exe -Force
} finally { Remove-Item -LiteralPath $temp -Force -ErrorAction SilentlyContinue }
$settings = Join-Path $env:USERPROFILE '.omp\agent\settings.json'
if (($reset -or $shell -ne 'keep') -and (Test-Path $settings)) { Copy-Item -LiteralPath $settings -Destination "$settings.backup.$([guid]::NewGuid())" }
$config = @{}
if (!$reset -and $shell -ne 'keep' -and (Test-Path $settings)) { $old = Get-Content -LiteralPath $settings -Raw | ConvertFrom-Json; foreach ($p in $old.PSObject.Properties) { $config[$p.Name] = $p.Value } }
if ($reset) { Remove-Item -LiteralPath $settings -ErrorAction SilentlyContinue }
if ($shell -ne 'keep') {
 if ($shell -eq 'builtin') { $config.Remove('shellPath') } else { $config.shellPath = $shell }
 New-Item -ItemType Directory -Force -Path (Split-Path $settings) | Out-Null
 $config | ConvertTo-Json -Depth 100 | Set-Content -LiteralPath $settings -Encoding UTF8
}
if ($pathChange) { $oldPath = [Environment]::GetEnvironmentVariable('Path', $target); $parts = @($oldPath -split ';' | Where-Object { $_ -and $_ -ne $dest }); [Environment]::SetEnvironmentVariable('Path', (@($dest) + $parts -join ';'), $target) }
Write-Host "Installed: $exe. Open a new terminal for PATH changes."
# Record only after this leaf has completed installation and configuration.
if ($env:OMP_HARNESS_MANAGER) {
 & $env:OMP_HARNESS_MANAGER state set $(if ($env:OMP_HARNESS_ITEM_ID) { $env:OMP_HARNESS_ITEM_ID } else { 'omp' }) true
 if ($LASTEXITCODE -ne 0) { throw 'OMP installed, but state recording failed' }
} elseif ((Test-Path (Join-Path $PSScriptRoot 'manager/manage-windows-amd64.exe')) -or (Test-Path (Join-Path $PSScriptRoot 'manager/manage-windows-arm64.exe'))) {
 & (Join-Path $PSScriptRoot 'manage.cmd') state set omp true
 if ($LASTEXITCODE -ne 0) { throw 'OMP installed, but state recording failed' }
}
if ($action -eq 'login') { Write-Host 'Use /login in OMP.' }
if ($action -ne 'exit') { & $exe }

#ifndef PackageVersion
  #error PackageVersion is required
#endif

[Setup]
AppId=omp-harness-script-installer
AppName=OMP Harness Scripts
AppVersion={#PackageVersion}
DefaultDirName={localappdata}\Programs\omp-harness
DefaultGroupName=OMP Harness Scripts
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible arm64
MinVersion=10.0
OutputDir=..\dist
OutputBaseFilename=omp-harness-scripts-{#PackageVersion}-windows-setup
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
UninstallDisplayName=OMP Harness Scripts
CloseApplications=no

[Files]
Source: "..\install-omp.bat"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\install-omp-tui.bat"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\install-omp-tui.ps1"; DestDir: "{app}"; Flags: ignoreversion
Source: "windows-launch.bat"; DestDir: "{app}"; DestName: "launch-installer.bat"; Flags: ignoreversion
Source: "VERSION"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\README.md"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\Install or update official OMP"; Filename: "{app}\launch-installer.bat"; WorkingDir: "{app}"
Name: "{group}\Interactive OMP installer"; Filename: "{app}\launch-installer.bat"; Parameters: "tui"; WorkingDir: "{app}"
Name: "{group}\Uninstall script package"; Filename: "{uninstallexe}"

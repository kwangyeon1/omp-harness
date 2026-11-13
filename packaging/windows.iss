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
Source: "..\dist\bundle\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{group}\Script and plugin manager"; Filename: "{app}\launch-installer.bat"; WorkingDir: "{app}"
Name: "{group}\OMP installer (ask before installing)"; Filename: "{app}\launch-installer.bat"; Parameters: "run omp"; WorkingDir: "{app}"
Name: "{group}\Uninstall script package"; Filename: "{uninstallexe}"

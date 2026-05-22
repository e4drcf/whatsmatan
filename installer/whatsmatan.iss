; Inno Setup script for whatsmatan.
; Builds whatsmatan-setup.exe — a tiny stub that, on install, downloads
; ffmpeg, whisper-cli (BLAS Windows x64), and the large-v3 whisper model.
;
; Build locally:
;   "C:\Program Files (x86)\Inno Setup 6\ISCC.exe" installer\whatsmatan.iss
; Output: installer\Output\whatsmatan-setup.exe
;
; CI: see .github/workflows/release.yml — runs on every git tag matching v*.

#define MyAppName    "whatsmatan"
#define MyAppVersion GetEnv("WHATSMATAN_VERSION")
#if MyAppVersion == ""
  #define MyAppVersion "0.0.0"
#endif
#define MyAppPublisher "e4drcf"
#define MyAppURL       "https://github.com/e4drcf/whatsmatan"
#define MyAppExeName   "whatsmatan.exe"

[Setup]
AppId={{6F2D7B41-9B7E-4F5D-8C1A-9E2C4D1F8E33}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}/issues
DefaultDirName={localappdata}\Programs\{#MyAppName}
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
OutputDir=Output
OutputBaseFilename=whatsmatan-setup
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
UninstallDisplayIcon={app}\{#MyAppExeName}

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
Name: "ukrainian"; MessagesFile: "compiler:Languages\Ukrainian.isl"

[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut"; GroupDescription: "Additional shortcuts:"

[Files]
; The Windows whatsmatan binary, built by the workflow before ISCC runs.
Source: "..\build\whatsmatan.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "install-deps.ps1";        DestDir: "{app}"; Flags: ignoreversion
Source: "launch.bat";              DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{#MyAppName}";           Filename: "{app}\launch.bat"; IconFilename: "{app}\{#MyAppExeName}"
Name: "{group}\Uninstall {#MyAppName}"; Filename: "{uninstallexe}"
Name: "{commondesktop}\{#MyAppName}";   Filename: "{app}\launch.bat"; IconFilename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Run]
; Download ffmpeg + whisper-cli + large-v3 model after files are extracted.
; The PowerShell script prints progress to its own console; the installer
; just shows a status message while it runs.
Filename: "powershell.exe"; \
  Parameters: "-NoProfile -ExecutionPolicy Bypass -File ""{app}\install-deps.ps1"" -InstallDir ""{app}"""; \
  StatusMsg: "Downloading ffmpeg, whisper-cli, and the Ukrainian model (~3.2 GB, allow 10–30 min)..."; \
  Flags: waituntilterminated

; Optional: launch app right after install.
Filename: "{app}\launch.bat"; Description: "Launch {#MyAppName} now"; \
  Flags: postinstall nowait skipifsilent unchecked

[UninstallDelete]
; Keep user session in %LOCALAPPDATA%\whatsmatan\data so re-install re-auths.
; Only wipe the downloaded big files from the program dir.
Type: filesandordirs; Name: "{app}\ffmpeg"
Type: filesandordirs; Name: "{app}\whisper"
Type: files;          Name: "{app}\ggml-large-v3.bin"

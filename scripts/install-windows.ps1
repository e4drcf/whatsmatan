# Bootstrap dependencies for whatsmatan on Windows (PowerShell).
# Requires Windows 10/11 with winget.
# Run in an elevated PowerShell (right-click > Run as administrator) on first install.
#
# Env overrides:
#   $env:WHISPER_DIR  default: $HOME\whisper.cpp
#   $env:MODEL        default: large-v3-turbo
#   $env:CUDA         set to 1 to build whisper.cpp with CUDA (needs CUDA Toolkit)
$ErrorActionPreference = "Stop"

if (-not $env:WHISPER_DIR) { $env:WHISPER_DIR = Join-Path $HOME "whisper.cpp" }
if (-not $env:MODEL)       { $env:MODEL = "large-v3-turbo" }
if (-not $env:CUDA)        { $env:CUDA = "0" }

function Log($msg) { Write-Host ("`n==> " + $msg) -ForegroundColor Green }

# ----- winget packages -----
function Ensure-Winget($id, $cmd) {
    if (Get-Command $cmd -ErrorAction SilentlyContinue) {
        Log "$cmd already installed"
        return
    }
    Log "winget install $id"
    winget install --id $id --silent --accept-package-agreements --accept-source-agreements
}

Ensure-Winget "GoLang.Go"          "go"
Ensure-Winget "Kitware.CMake"      "cmake"
Ensure-Winget "Gyan.FFmpeg"        "ffmpeg"
Ensure-Winget "Git.Git"            "git"

# winget updates PATH for new shells; refresh current session.
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" +
            [System.Environment]::GetEnvironmentVariable("Path","User")

# ----- whisper.cpp -----
if (-not (Test-Path $env:WHISPER_DIR)) {
    Log "Cloning whisper.cpp to $env:WHISPER_DIR"
    git clone --depth=1 https://github.com/ggml-org/whisper.cpp.git $env:WHISPER_DIR
}

$buildDir = Join-Path $env:WHISPER_DIR "build"
$whisperCli = Join-Path $buildDir "bin\Release\whisper-cli.exe"

$cmakeArgs = @("-S", $env:WHISPER_DIR, "-B", $buildDir, "-DCMAKE_BUILD_TYPE=Release")
if ($env:CUDA -eq "1") {
    Log "Building whisper.cpp with CUDA"
    $cmakeArgs += "-DGGML_CUDA=1"
} else {
    Log "Building whisper.cpp (CPU). Set `$env:CUDA = '1' for GPU build."
}

if (-not (Test-Path $whisperCli)) {
    cmake @cmakeArgs
    cmake --build $buildDir -j --config Release
}

# ----- model -----
$modelDir = Join-Path $env:WHISPER_DIR "models"
$modelPath = Join-Path $modelDir ("ggml-" + $env:MODEL + ".bin")
if (-not (Test-Path $modelPath)) {
    Log ("Downloading model " + $env:MODEL)
    # The shell script uses bash; on Windows fall back to direct curl.
    $url = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-" + $env:MODEL + ".bin"
    curl.exe -L -o $modelPath $url
} else {
    Log ("Model " + $env:MODEL + " already present")
}

# ----- ffmpeg path -----
$ffmpeg = (Get-Command ffmpeg).Source

# ----- build whatsmatan -----
Log "Building whatsmatan"
Set-Location (Join-Path $PSScriptRoot "..")
go build -o whatsmatan.exe ./cmd/whatsmatan

@"

Done. Run:

  .\whatsmatan.exe `
    --whisper-cli  '$whisperCli' `
    --whisper-model '$modelPath' `
    --ffmpeg       '$ffmpeg' `
    --lang uk

"@ | Write-Host

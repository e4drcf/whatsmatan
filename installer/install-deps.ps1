# Post-install dependency fetcher for whatsmatan (Windows).
# Invoked by the Inno Setup [Run] section after the whatsmatan.exe is copied.
#
# Downloads:
#   * ffmpeg (gyan.dev release-essentials build, ~100 MB)
#   * whisper.cpp Windows x64 BLAS build (~10 MB) — pinned to a known release
#   * ggml-large-v3 whisper model (~3.1 GB) from HuggingFace
#
# Each download is skipped if the target file already exists, so re-running
# the installer or recovering from a network drop is safe.

param(
    [Parameter(Mandatory=$true)][string]$InstallDir
)

$ErrorActionPreference = "Stop"
$ProgressPreference    = "Continue"

# Pinned versions / URLs.
$WhisperRelease  = "v1.8.4"
$WhisperZipUrl   = "https://github.com/ggml-org/whisper.cpp/releases/download/$WhisperRelease/whisper-blas-bin-x64.zip"
$FfmpegUrl       = "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip"
$ModelUrl        = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3.bin"

$ffmpegDir   = Join-Path $InstallDir "ffmpeg"
$whisperDir  = Join-Path $InstallDir "whisper"
$modelPath   = Join-Path $InstallDir "ggml-large-v3.bin"

function Log($msg) { Write-Host ("[whatsmatan] " + $msg) -ForegroundColor Cyan }
function Err($msg) { Write-Host ("[whatsmatan] ERROR: " + $msg) -ForegroundColor Red }

function Download-File($Url, $OutPath) {
    if (Test-Path $OutPath) {
        Log "Already present: $OutPath"
        return
    }
    Log "Downloading $Url"
    $tmp = "$OutPath.part"
    # BITS gives resumable, throttled, progress-aware transfers.
    try {
        Start-BitsTransfer -Source $Url -Destination $tmp -DisplayName "Downloading $(Split-Path $OutPath -Leaf)"
    } catch {
        Log "BITS failed ($($_.Exception.Message)); falling back to Invoke-WebRequest"
        Invoke-WebRequest -Uri $Url -OutFile $tmp -UseBasicParsing
    }
    Move-Item -Force -Path $tmp -Destination $OutPath
}

# ----- ffmpeg -----
if (-not (Test-Path (Join-Path $ffmpegDir "ffmpeg.exe"))) {
    $zip = Join-Path $env:TEMP "whatsmatan-ffmpeg.zip"
    Download-File -Url $FfmpegUrl -OutPath $zip
    Log "Extracting ffmpeg"
    $stage = Join-Path $env:TEMP "whatsmatan-ffmpeg-stage"
    if (Test-Path $stage) { Remove-Item -Recurse -Force $stage }
    Expand-Archive -Path $zip -DestinationPath $stage
    # gyan.dev zips have a top-level folder like ffmpeg-*-essentials_build\bin\ffmpeg.exe
    $exe = Get-ChildItem -Path $stage -Recurse -Filter ffmpeg.exe | Select-Object -First 1
    if (-not $exe) { throw "ffmpeg.exe not found in extracted zip" }
    New-Item -ItemType Directory -Force -Path $ffmpegDir | Out-Null
    Copy-Item $exe.FullName (Join-Path $ffmpegDir "ffmpeg.exe")
    Remove-Item -Recurse -Force $stage
    Remove-Item -Force $zip
} else {
    Log "ffmpeg already installed"
}

# ----- whisper-cli -----
if (-not (Test-Path (Join-Path $whisperDir "whisper-cli.exe"))) {
    $zip = Join-Path $env:TEMP "whatsmatan-whisper.zip"
    Download-File -Url $WhisperZipUrl -OutPath $zip
    Log "Extracting whisper-cli"
    if (Test-Path $whisperDir) { Remove-Item -Recurse -Force $whisperDir }
    Expand-Archive -Path $zip -DestinationPath $whisperDir
    Remove-Item -Force $zip
} else {
    Log "whisper-cli already installed"
}

# ----- model (3.1 GB) -----
Download-File -Url $ModelUrl -OutPath $modelPath

# Sanity check.
$expectedMin = 2.5GB
if ((Get-Item $modelPath).Length -lt $expectedMin) {
    Err "Model file is smaller than expected (corrupt or partial download). Deleting; re-run installer."
    Remove-Item -Force $modelPath
    exit 2
}

Log "All dependencies installed under $InstallDir"

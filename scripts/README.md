# Install scripts

One-shot bootstrap of every dependency: Go, ffmpeg, git, cmake, whisper.cpp (built with the best available accelerator), a whisper model, and the `whatsmatan` binary.

All scripts are **idempotent** — re-running skips anything already in place.

## macOS

```sh
./scripts/install-macos.sh
```

Detects Apple Silicon and builds whisper.cpp with `-DGGML_METAL=1`. Installs Homebrew if missing.

## Linux

Supports apt (Debian/Ubuntu), dnf (Fedora/RHEL), pacman (Arch).

```sh
./scripts/install-linux.sh           # CPU build
CUDA=1 ./scripts/install-linux.sh    # CUDA build (needs CUDA Toolkit + nvcc on PATH)
```

For other distros: install `go (>= 1.23)`, `cmake`, `ffmpeg`, `git`, a C/C++ toolchain — then rerun.

## Windows

```powershell
# Run in elevated PowerShell on first install (winget needs admin for some packages).
powershell -ExecutionPolicy Bypass -File .\scripts\install-windows.ps1

# CUDA build:
$env:CUDA = "1"; powershell -ExecutionPolicy Bypass -File .\scripts\install-windows.ps1
```

Uses winget. Requires Windows 10/11.

## Env overrides (all platforms)

| Var          | Default                          | Purpose                                                      |
|--------------|----------------------------------|--------------------------------------------------------------|
| `WHISPER_DIR`| `$HOME/whisper.cpp`              | Where to clone + build whisper.cpp                           |
| `MODEL`      | `large-v3-turbo`                 | Model name (`base`, `small`, `medium`, `large-v3`, `large-v3-turbo`, plus quantized variants like `large-v3-turbo-q5_0`) |
| `CUDA`       | `0` (Linux/Win only)             | `1` to build whisper.cpp with CUDA                           |

Example — small model, CPU only, custom install dir:

```sh
WHISPER_DIR=/opt/whisper.cpp MODEL=small ./scripts/install-linux.sh
```

## What gets installed where

| Item             | macOS                          | Linux                                | Windows                                       |
|------------------|--------------------------------|--------------------------------------|-----------------------------------------------|
| Go, ffmpeg, etc. | Homebrew                       | apt/dnf/pacman                       | winget                                        |
| whisper.cpp      | `$WHISPER_DIR`                 | `$WHISPER_DIR`                       | `$env:WHISPER_DIR`                            |
| whisper-cli      | `$WHISPER_DIR/build/bin/whisper-cli` | same                          | `$env:WHISPER_DIR\build\bin\Release\whisper-cli.exe` |
| Model            | `$WHISPER_DIR/models/ggml-$MODEL.bin` | same                          | same                                          |
| whatsmatan       | repo root: `./whatsmatan`      | same                                 | repo root: `.\whatsmatan.exe`                 |

Nothing is installed into system directories beyond what the package manager handles. `$WHISPER_DIR` is fully under your home directory by default.

## After install

```sh
cp examples/run-large-uk.sh.example run.sh
chmod +x run.sh
./run.sh
```

Then scan the QR code in the terminal, open http://localhost:8080, pick chats to track.

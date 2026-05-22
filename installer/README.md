# Windows installer

Online installer that ships as a single `whatsmatan-setup.exe` (~5 MB). At install time it downloads, into the install directory:

- `ffmpeg.exe` — gyan.dev "release-essentials" build (~100 MB)
- `whisper-cli.exe` — pinned [whisper.cpp release](https://github.com/ggml-org/whisper.cpp/releases) Windows x64 BLAS build (~10 MB)
- `ggml-large-v3.bin` — best-quality whisper model for Ukrainian (~3.1 GB)

Total install footprint after first run: **~3.3 GB**. Allow 10–30 min on a typical connection.

## Where things end up

| Path                                                          | Contents                                  |
|---------------------------------------------------------------|-------------------------------------------|
| `%LOCALAPPDATA%\Programs\whatsmatan\whatsmatan.exe`           | Server binary                             |
| `%LOCALAPPDATA%\Programs\whatsmatan\ffmpeg\ffmpeg.exe`        | Bundled ffmpeg                            |
| `%LOCALAPPDATA%\Programs\whatsmatan\whisper\whisper-cli.exe` | Bundled whisper-cli                       |
| `%LOCALAPPDATA%\Programs\whatsmatan\ggml-large-v3.bin`        | Whisper model                             |
| `%LOCALAPPDATA%\whatsmatan\data\`                             | Session DB, tracked.json, chats.json, tmp |

No admin rights required — everything installs under `%LOCALAPPDATA%`. Uninstall leaves `%LOCALAPPDATA%\whatsmatan\data\` in place so you don't have to re-pair after a reinstall; delete it manually to forget the WhatsApp session.

## Building locally (Windows)

1. Install Go ≥ 1.23 and [Inno Setup 6](https://jrsoftware.org/isdl.php).
2. Cross-compile or native-build the Windows binary:
   ```powershell
   $env:GOOS = 'windows'; $env:GOARCH = 'amd64'; $env:CGO_ENABLED = '0'
   go build -trimpath -ldflags "-s -w" -o build\whatsmatan.exe .\cmd\whatsmatan
   ```
3. Compile the installer:
   ```powershell
   $env:WHATSMATAN_VERSION = '0.1.0'
   & 'C:\Program Files (x86)\Inno Setup 6\ISCC.exe' installer\whatsmatan.iss
   ```
4. Output: `installer\Output\whatsmatan-setup.exe`.

## CI / Releases

[`.github/workflows/release.yml`](../.github/workflows/release.yml) runs on every git tag starting with `v` (e.g. `v0.1.0`). It:

1. Builds `whatsmatan.exe` on `windows-latest`.
2. Installs Inno Setup via Chocolatey.
3. Compiles `whatsmatan-setup.exe`.
4. Uploads it as a workflow artifact and attaches it to the GitHub Release for that tag.

To cut a release:

```sh
git tag v0.1.0
git push origin v0.1.0
```

The installer appears on the [Releases page](https://github.com/e4drcf/whatsmatan/releases) within a few minutes.

## Notes / caveats

- The installer is **unsigned**. SmartScreen will warn on first run; users must click "More info → Run anyway". Code signing requires a paid certificate.
- whisper-cli build is CPU + BLAS, not CUDA. Users with NVIDIA GPUs who want CUDA speed can swap in `whisper-cublas-*-bin-x64.zip` manually after install.
- The 3.1 GB model download is the slow step. The script uses BITS for resumability — if it fails partway, re-running the installer resumes.
- Inno Setup's [Run] step blocks the wizard while PowerShell downloads. Users see a "Downloading..." status during this time.

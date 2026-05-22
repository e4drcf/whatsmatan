# whatsmatan

WhatsApp userbot that transcribes voice messages with local whisper.cpp.

- Connects to WhatsApp via [whatsmeow](https://github.com/tulir/whatsmeow) (multi-device, QR-pair on first run).
- Small web UI to pick which chats to track.
- When a voice/PTT message arrives in a tracked chat, downloads it, decodes with `ffmpeg`, transcribes with `whisper-cli`, replies in-thread with the text.

## Quick start

One-shot bootstrap of Go, ffmpeg, whisper.cpp, a model, and the `whatsmatan` binary:

```sh
# macOS (Intel + Apple Silicon)
./scripts/install-macos.sh

# Linux (Debian/Ubuntu, Fedora/RHEL, Arch)
./scripts/install-linux.sh
# or with CUDA: CUDA=1 ./scripts/install-linux.sh

# Windows (PowerShell, Windows 10/11 with winget)
powershell -ExecutionPolicy Bypass -File .\scripts\install-windows.ps1
```

Override defaults with env vars: `WHISPER_DIR=/opt/whisper.cpp MODEL=base ./scripts/install-linux.sh`.
See [`scripts/README.md`](scripts/README.md) for the full table.

## Requirements (if installing manually)

- Go 1.23+
- `ffmpeg` on `$PATH` (or pass `--ffmpeg`)
- `whisper.cpp` built with the `whisper-cli` binary. On Apple Silicon build with `-DGGML_METAL=1`; with `-DGGML_CUDA=1` on Nvidia. Set `--whisper-cli` and `--whisper-model` to the binary and a `.bin` ggml model.

Get a model, e.g.:

```sh
# inside whisper.cpp checkout
./models/download-ggml-model.sh base
```

## Run

```sh
go build -o whatsmatan ./cmd/whatsmatan

./whatsmatan \
  --data-dir ./data \
  --http-addr :8080 \
  --whisper-cli /path/to/whisper.cpp/build/bin/whisper-cli \
  --whisper-model /path/to/whisper.cpp/models/ggml-base.bin \
  --ffmpeg $(which ffmpeg) \
  --lang auto
```

First start prints a QR code in the terminal. Scan it from WhatsApp → Settings → Linked Devices → Link a Device. Session is saved in `data/session.db`.

Then open http://localhost:8080 and tick the chats you want to track. Tracked JIDs are stored in `data/tracked.json`.

Send a voice note from another account in a tracked chat — the bot replies (quoted) with the transcript.

## Flags

| Flag              | Default                  | Purpose                             |
|-------------------|--------------------------|-------------------------------------|
| `--data-dir`      | `./data`                 | session DB, tracked.json, tmp/      |
| `--http-addr`     | `:8080`                  | web UI listen address               |
| `--whisper-cli`   | `whisper-cli`            | path to whisper-cli binary          |
| `--whisper-model` | `./models/ggml-base.bin` | ggml model path                     |
| `--ffmpeg`        | `ffmpeg`                 | path to ffmpeg                      |
| `--lang`          | `auto`                   | whisper language code               |
| `--concurrency`   | `1`                      | max concurrent transcriptions       |
| `--timeout`       | `2m`                     | per-message budget                  |
| `--debug`         | `false`                  | verbose whatsmeow logs              |

## Layout

```
cmd/whatsmatan/main.go    # entry, flag wiring
internal/wa/              # whatsmeow client + voice event handler
internal/transcribe/      # ffmpeg + whisper-cli subprocess
internal/tracked/         # JSON-backed set of tracked JIDs
internal/web/             # REST + static UI
frontend/                 # embedded HTML/JS UI
```

Frontend is embedded into the binary via `//go:embed`, so a single `whatsmatan` binary is self-contained.

## Security

The web UI has no auth — bind to localhost only. Exposing it publicly would let anyone change what your account transcribes.

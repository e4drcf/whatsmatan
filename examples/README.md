# Example run scripts

These are templates — copy one to the repo root as `run.sh`, edit, then run.

```sh
cp examples/run-large-uk.sh.example run.sh
chmod +x run.sh
./run.sh
```

The root `.gitignore` covers `/run.sh` and `/run-*.sh` so your local copy stays out of git.

| File                             | Model              | RAM    | Speed (M-series Metal) | Best for                              |
|----------------------------------|--------------------|--------|-------------------------|---------------------------------------|
| `run-base.sh.example`            | `ggml-base.bin`    | ~500 MB | very fast               | quick smoke test, English             |
| `run-large-uk.sh.example`        | `ggml-large-v3-turbo.bin` | ~2.5 GB | ~2-4× realtime | Ukrainian, mixed-language, best quality |

## Required tools

- `ffmpeg` on `$PATH`. macOS: `brew install ffmpeg`.
- `whisper.cpp` built locally with `whisper-cli` binary. See repo root `README.md` for the build steps.
- Whisper ggml model file. Download via `~/whisper.cpp/models/download-ggml-model.sh <name>`.

## Flag reference

| Flag              | Meaning                                                                  |
|-------------------|--------------------------------------------------------------------------|
| `--data-dir`      | Where `session.db`, `tracked.json`, `chats.json`, `tmp/` live. **Never commit.** |
| `--http-addr`     | Web UI listen address. Keep on `localhost` — no auth.                    |
| `--whisper-cli`   | Path to `whisper-cli` binary.                                            |
| `--whisper-model` | Path to `.bin` ggml model file.                                          |
| `--ffmpeg`        | Path to `ffmpeg` binary.                                                 |
| `--lang`          | Language code: `uk`, `ru`, `en`, … or `auto`. Pinning beats auto for non-English. |
| `--concurrency`   | Max parallel transcriptions. `1` is safe; bump only with spare RAM.      |
| `--timeout`       | Per-message budget. Bigger model + long voice notes need more.           |
| `--debug`         | Verbose whatsmeow logs. Off by default.                                  |

#!/usr/bin/env bash
# Bootstrap dependencies for whatsmatan on macOS (Intel or Apple Silicon).
# Idempotent: re-running skips anything already installed.
#
# Installs: Homebrew (if missing), go, cmake, ffmpeg, git.
# Builds:   whisper.cpp at $WHISPER_DIR with Metal acceleration on arm64.
# Pulls:    whisper ggml model named $MODEL (default: large-v3-turbo).
#
# Env overrides:
#   WHISPER_DIR  default: $HOME/whisper.cpp
#   MODEL        default: large-v3-turbo  (other: base, small, medium, large-v3)
set -euo pipefail

WHISPER_DIR="${WHISPER_DIR:-$HOME/whisper.cpp}"
MODEL="${MODEL:-large-v3-turbo}"

log() { printf "\n\033[1;32m==>\033[0m %s\n" "$*"; }

# ----- Homebrew -----
if ! command -v brew >/dev/null 2>&1; then
  log "Installing Homebrew"
  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
fi
eval "$(/opt/homebrew/bin/brew shellenv 2>/dev/null || /usr/local/bin/brew shellenv)"

# ----- brew packages -----
for pkg in go cmake ffmpeg git; do
  if ! brew list "$pkg" >/dev/null 2>&1; then
    log "brew install $pkg"
    brew install "$pkg"
  else
    log "$pkg already installed"
  fi
done

# ----- whisper.cpp -----
if [ ! -d "$WHISPER_DIR" ]; then
  log "Cloning whisper.cpp to $WHISPER_DIR"
  git clone --depth=1 https://github.com/ggml-org/whisper.cpp.git "$WHISPER_DIR"
fi

CMAKE_FLAGS=()
if [ "$(uname -m)" = "arm64" ]; then
  CMAKE_FLAGS+=(-DGGML_METAL=1)
fi

if [ ! -x "$WHISPER_DIR/build/bin/whisper-cli" ]; then
  log "Building whisper.cpp (Release${CMAKE_FLAGS:+, Metal})"
  cmake -S "$WHISPER_DIR" -B "$WHISPER_DIR/build" \
    -DCMAKE_BUILD_TYPE=Release "${CMAKE_FLAGS[@]}"
  cmake --build "$WHISPER_DIR/build" -j --config Release
fi

# ----- model -----
MODEL_PATH="$WHISPER_DIR/models/ggml-$MODEL.bin"
if [ ! -f "$MODEL_PATH" ]; then
  log "Downloading model $MODEL"
  "$WHISPER_DIR/models/download-ggml-model.sh" "$MODEL"
else
  log "Model $MODEL already present"
fi

# ----- build whatsmatan -----
log "Building whatsmatan"
cd "$(dirname "$0")/.."
go build -o whatsmatan ./cmd/whatsmatan

cat <<EOF

Done. Run:

  ./whatsmatan \\
    --whisper-cli  $WHISPER_DIR/build/bin/whisper-cli \\
    --whisper-model $MODEL_PATH \\
    --ffmpeg       \$(command -v ffmpeg) \\
    --lang uk

Or copy a template:  cp examples/run-large-uk.sh.example run.sh
EOF

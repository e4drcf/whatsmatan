#!/usr/bin/env bash
# Bootstrap dependencies for whatsmatan on Linux.
# Supports Debian/Ubuntu (apt), Fedora/RHEL (dnf), Arch (pacman).
# For other distros, install go/cmake/ffmpeg/git manually and rerun.
#
# Env overrides:
#   WHISPER_DIR  default: $HOME/whisper.cpp
#   MODEL        default: large-v3-turbo
#   CUDA         set to 1 to build whisper.cpp with CUDA (needs nvcc + libs)
set -euo pipefail

WHISPER_DIR="${WHISPER_DIR:-$HOME/whisper.cpp}"
MODEL="${MODEL:-large-v3-turbo}"
CUDA="${CUDA:-0}"

log() { printf "\n\033[1;32m==>\033[0m %s\n" "$*"; }
die() { printf "\033[1;31mERROR:\033[0m %s\n" "$*" >&2; exit 1; }

# ----- package manager -----
if command -v apt-get >/dev/null 2>&1; then
  PM=apt
elif command -v dnf >/dev/null 2>&1; then
  PM=dnf
elif command -v pacman >/dev/null 2>&1; then
  PM=pacman
else
  die "No supported package manager found (apt/dnf/pacman). Install go, cmake, ffmpeg, git manually."
fi
log "Package manager: $PM"

install_pkgs() {
  case "$PM" in
    apt)
      sudo apt-get update
      sudo apt-get install -y golang-go cmake ffmpeg git build-essential
      ;;
    dnf)
      sudo dnf install -y golang cmake ffmpeg git gcc gcc-c++ make
      ;;
    pacman)
      sudo pacman -Sy --noconfirm go cmake ffmpeg git base-devel
      ;;
  esac
}

need=()
for c in go cmake ffmpeg git make; do
  command -v "$c" >/dev/null 2>&1 || need+=("$c")
done
if [ "${#need[@]}" -gt 0 ]; then
  log "Installing: ${need[*]}"
  install_pkgs
else
  log "go, cmake, ffmpeg, git, make already installed"
fi

# Debian-bundled golang is often outdated. Warn if < 1.22.
if go_ver=$(go version 2>/dev/null | awk '{print $3}'); then
  log "Go version: $go_ver"
  case "$go_ver" in
    go1.[0-9].*|go1.1[0-9].*|go1.20.*|go1.21.*)
      die "Go $go_ver too old. whatsmatan needs >= 1.23. Install from https://go.dev/dl/ or use 'snap install go --classic'."
      ;;
  esac
fi

# ----- whisper.cpp -----
if [ ! -d "$WHISPER_DIR" ]; then
  log "Cloning whisper.cpp to $WHISPER_DIR"
  git clone --depth=1 https://github.com/ggml-org/whisper.cpp.git "$WHISPER_DIR"
fi

CMAKE_FLAGS=(-DCMAKE_BUILD_TYPE=Release)
if [ "$CUDA" = "1" ]; then
  CMAKE_FLAGS+=(-DGGML_CUDA=1)
  log "Building whisper.cpp with CUDA"
else
  log "Building whisper.cpp (CPU). Set CUDA=1 for GPU build."
fi

if [ ! -x "$WHISPER_DIR/build/bin/whisper-cli" ]; then
  cmake -S "$WHISPER_DIR" -B "$WHISPER_DIR/build" "${CMAKE_FLAGS[@]}"
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

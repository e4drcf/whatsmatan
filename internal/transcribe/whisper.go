package transcribe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Whisper struct {
	BinPath   string // whisper-cli
	ModelPath string // ggml model
	FFmpeg    string // ffmpeg
	TmpDir    string // base dir for scratch files
	Language  string // e.g. "auto", "en", "uk"; empty -> whisper default

	sem chan struct{}
}

func NewWhisper(binPath, modelPath, ffmpegPath, tmpDir, language string, concurrency int) *Whisper {
	if concurrency <= 0 {
		concurrency = 1
	}
	return &Whisper{
		BinPath:   binPath,
		ModelPath: modelPath,
		FFmpeg:    ffmpegPath,
		TmpDir:    tmpDir,
		Language:  language,
		sem:       make(chan struct{}, concurrency),
	}
}

// Run takes an OGG/Opus audio blob and returns the transcribed text.
func (w *Whisper) Run(ctx context.Context, ogg []byte) (string, error) {
	if len(ogg) == 0 {
		return "", errors.New("empty audio")
	}
	select {
	case w.sem <- struct{}{}:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	defer func() { <-w.sem }()

	if err := os.MkdirAll(w.TmpDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir tmp: %w", err)
	}
	dir, err := os.MkdirTemp(w.TmpDir, "stt-*")
	if err != nil {
		return "", fmt.Errorf("mktemp: %w", err)
	}
	defer os.RemoveAll(dir)

	oggPath := filepath.Join(dir, "in.ogg")
	wavPath := filepath.Join(dir, "in.wav")
	outBase := filepath.Join(dir, "out")

	if err := os.WriteFile(oggPath, ogg, 0o644); err != nil {
		return "", fmt.Errorf("write ogg: %w", err)
	}

	// ffmpeg: decode OGG/Opus → 16kHz mono PCM WAV.
	ff := exec.CommandContext(ctx, w.FFmpeg,
		"-hide_banner", "-loglevel", "error", "-y",
		"-i", oggPath,
		"-ar", "16000", "-ac", "1", "-f", "wav", wavPath,
	)
	var ffErr bytes.Buffer
	ff.Stderr = &ffErr
	if err := ff.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg: %w: %s", err, ffErr.String())
	}

	args := []string{
		"-m", w.ModelPath,
		"-f", wavPath,
		"-otxt",
		"-of", outBase,
		"-nt", // no timestamps in output
	}
	if w.Language != "" {
		args = append(args, "-l", w.Language)
	}
	wcmd := exec.CommandContext(ctx, w.BinPath, args...)
	var wErr bytes.Buffer
	wcmd.Stderr = &wErr
	if err := wcmd.Run(); err != nil {
		return "", fmt.Errorf("whisper: %w: %s", err, wErr.String())
	}

	txt, err := os.ReadFile(outBase + ".txt")
	if err != nil {
		return "", fmt.Errorf("read transcript: %w", err)
	}
	return strings.TrimSpace(string(txt)), nil
}

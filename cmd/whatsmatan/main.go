package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/e4drcf/whatsmatan/frontend"
	"github.com/e4drcf/whatsmatan/internal/chats"
	"github.com/e4drcf/whatsmatan/internal/tracked"
	"github.com/e4drcf/whatsmatan/internal/transcribe"
	"github.com/e4drcf/whatsmatan/internal/wa"
	"github.com/e4drcf/whatsmatan/internal/web"
)

func main() {
	dataDir := flag.String("data-dir", "./data", "directory for session.db + tracked.json + tmp/")
	httpAddr := flag.String("http-addr", ":8080", "HTTP listen address for web UI")
	whisperBin := flag.String("whisper-cli", "whisper-cli", "path to whisper.cpp whisper-cli binary")
	whisperModel := flag.String("whisper-model", "./models/ggml-base.bin", "path to whisper ggml model")
	ffmpegBin := flag.String("ffmpeg", "ffmpeg", "path to ffmpeg binary")
	language := flag.String("lang", "auto", "whisper language code, e.g. en, uk, auto")
	concurrency := flag.Int("concurrency", 1, "max concurrent transcriptions")
	timeout := flag.Duration("timeout", 2*time.Minute, "per-message transcribe timeout")
	debug := flag.Bool("debug", false, "verbose whatsmeow logs")
	flag.Parse()

	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := wa.Build(ctx, *dataDir, *debug)
	if err != nil {
		log.Fatalf("wa init: %v", err)
	}

	store, err := tracked.New(filepath.Join(*dataDir, "tracked.json"))
	if err != nil {
		log.Fatalf("tracked store: %v", err)
	}

	chatCache, err := chats.New(filepath.Join(*dataDir, "chats.json"))
	if err != nil {
		log.Fatalf("chats cache: %v", err)
	}

	stt := transcribe.NewWhisper(
		*whisperBin, *whisperModel, *ffmpegBin,
		filepath.Join(*dataDir, "tmp"),
		*language, *concurrency,
	)

	handler := &wa.Handler{
		Client:      client,
		Tracker:     store,
		Transcriber: stt,
		Chats:       chatCache,
		Timeout:     *timeout,
	}
	client.AddEventHandler(handler.Handle)

	srv := &web.Server{Client: client, Tracker: store, Chats: chatCache, Static: frontend.FS}
	httpSrv := &http.Server{
		Addr:              *httpAddr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		fmt.Printf("web UI: http://localhost%s\n", *httpAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	fmt.Println("shutting down…")
	shutdownCtx, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	_ = httpSrv.Shutdown(shutdownCtx)
	client.Disconnect()
}

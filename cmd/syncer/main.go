package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"device-content-sync/internal/downloader"
	"device-content-sync/internal/manifest"
	"device-content-sync/internal/publisher"
	"device-content-sync/internal/syncer"
)

func main() {
	manifestURL := envOrDefault("MANIFEST_URL", "http://localhost:8080")
	deviceToken := envOrDefault("DEVICE_TOKEN", "test-device-001")
	contentDir := envOrDefault("CONTENT_DIR", "/tmp/winnow")
	statePath := envOrDefault("STATE_PATH", "/tmp/winnow/.sync-state.json")
	pollInterval := envOrDefault("POLL_INTERVAL", "30s")

	interval, err := time.ParseDuration(pollInterval)
	if err != nil {
		log.Fatalf("invalid POLL_INTERVAL: %v", err)
	}

	client := manifest.NewClient(manifestURL, deviceToken)
	dl := downloader.NewHTTPDownloader()
	pub := publisher.NewStdoutPublisher()

	s := syncer.New(syncer.Config{
		PollInterval: interval,
		ContentDir:   contentDir,
		StatePath:    statePath,
	}, client, dl, pub)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown on SIGINT/SIGTERM.
	// "Devices can be turned off at any time" — we save state before exiting.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("received %s, shutting down...", sig)
		cancel()
	}()

	log.Printf("starting syncer (poll every %s, content dir: %s)", interval, contentDir)
	if err := s.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("syncer error: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

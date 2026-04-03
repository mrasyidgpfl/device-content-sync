package syncer

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"device-content-sync/internal/downloader"
	"device-content-sync/internal/manifest"
	"device-content-sync/internal/publisher"
)

type Config struct {
	PollInterval time.Duration
	ContentDir   string
	StatePath    string
}

type Syncer struct {
	config     Config
	client     *manifest.Client
	downloader downloader.Downloader
	publisher  publisher.Publisher
	state      *State
}

func New(cfg Config, client *manifest.Client, dl downloader.Downloader, pub publisher.Publisher) *Syncer {
	return &Syncer{
		config:     cfg,
		client:     client,
		downloader: dl,
		publisher:  pub,
		state:      NewState(cfg.StatePath),
	}
}

func (s *Syncer) Run(ctx context.Context) error {
	if err := s.state.Load(); err != nil {
		log.Printf("warning: failed to load state, starting fresh: %v", err)
	}

	// Run once immediately on startup, then on ticker
	s.sync(ctx)

	ticker := time.NewTicker(s.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("shutting down syncer")
			return s.state.Save()
		case <-ticker.C:
			s.sync(ctx)
		}
	}
}

func (s *Syncer) sync(ctx context.Context) {
	log.Println("polling manifest...")

	m, changed, err := s.client.FetchManifest(ctx)
	if err != nil {
		log.Printf("error fetching manifest: %v", err)
		return
	}

	if !changed {
		log.Println("manifest unchanged (304)")
		return
	}

	log.Println("manifest changed, processing...")
	s.processManifest(ctx, m)

	if err := s.state.Save(); err != nil {
		log.Printf("error saving state: %v", err)
	}
}

func (s *Syncer) processManifest(ctx context.Context, m manifest.ManifestResponse) {
	// Track which keys are present in the new manifest
	seen := make(map[string]bool)

	for contentType, item := range m {
		if item == nil {
			continue
		}

		// If the whole content type is unavailable, skip but don't remove.
		// "Stale content is preferable to no content."
		if item.Unavailable {
			log.Printf("content type %q is unavailable, keeping existing content", contentType)
			for _, key := range s.state.Keys() {
				st, _ := s.state.Get(key)
				if st.ContentType == contentType {
					seen[key] = true
				}
			}
			continue
		}

		for _, ci := range item.Items {
			if ctx.Err() != nil {
				return
			}
			key := fmt.Sprintf("%s/%s", contentType, ci.Name)
			seen[key] = true

			// Item-level unavailable: skip but keep existing
			if ci.Unavailable {
				log.Printf("item %q is unavailable, keeping existing", key)
				continue
			}

			existing, exists := s.state.Get(key)

			// Skip if ETag matches (content unchanged)
			if exists && existing.ETag == ci.ETag {
				continue
			}

			destPath := filepath.Join(s.config.ContentDir, contentType, ci.Name)

			if err := s.downloader.Download(ctx, ci.URI, destPath); err != nil {
				log.Printf("error downloading %s: %v", key, err)
				// Don't remove existing content on download failure.
				// "Stale content is preferable to no content."
				continue
			}

			action := "UPDATED"
			if !exists {
				action = "ADDED"
			}

			s.state.Set(key, ItemState{
				ETag:        ci.ETag,
				ContentType: contentType,
				FilePath:    destPath,
			})

			s.publisher.Publish(publisher.Event{
				Action: action,
				Key:    ci.Name,
			})
		}
	}

	// Remove items that are no longer in the manifest
	for _, key := range s.state.Keys() {
		if !seen[key] {
			st, _ := s.state.Get(key)
			if err := s.downloader.Remove(st.FilePath); err != nil {
				log.Printf("error removing %s: %v", key, err)
				continue
			}
			s.state.Delete(key)
			s.publisher.Publish(publisher.Event{
				Action: "REMOVED",
				Key:    filepath.Base(st.FilePath),
			})
		}
	}
}

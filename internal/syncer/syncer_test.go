package syncer

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"device-content-sync/internal/manifest"
	"device-content-sync/internal/publisher"
)

// Mock downloader that records calls
type mockDownloader struct {
	mu          sync.Mutex
	downloaded  map[string]string // destPath -> uri
	removed     []string
	shouldError bool
}

func newMockDownloader() *mockDownloader {
	return &mockDownloader{downloaded: make(map[string]string)}
}

func (m *mockDownloader) Download(_ context.Context, uri, destPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.shouldError {
		return fmt.Errorf("mock download error")
	}
	m.downloaded[destPath] = uri
	return nil
}

func (m *mockDownloader) Remove(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.removed = append(m.removed, path)
	return nil
}

// Mock publisher that records events
type mockPublisher struct {
	mu     sync.Mutex
	events []publisher.Event
}

func (m *mockPublisher) Publish(event publisher.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

func TestSyncer_NewContent(t *testing.T) {
	dl := newMockDownloader()
	pub := &mockPublisher{}
	state := NewState(t.TempDir() + "/state.json")

	s := &Syncer{
		config: Config{
			ContentDir: t.TempDir(),
		},
		downloader: dl,
		publisher:  pub,
		state:      state,
	}

	m := manifest.ManifestResponse{
		"icons": &manifest.ManifestItem{
			Items: []manifest.ContentItem{
				{Name: "icon-1.png", URI: "http://example.com/icon-1.png", ETag: "v1"},
			},
		},
	}

	s.processManifest(context.Background(), m)

	if len(pub.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(pub.events))
	}
	if pub.events[0].Action != "ADDED" {
		t.Errorf("expected ADDED, got %s", pub.events[0].Action)
	}
	if pub.events[0].Key != "icon-1.png" {
		t.Errorf("expected icon-1.png, got %s", pub.events[0].Key)
	}
}

func TestSyncer_UpdatedContent(t *testing.T) {
	dl := newMockDownloader()
	pub := &mockPublisher{}
	state := NewState(t.TempDir() + "/state.json")

	// Pre-populate state as if we already downloaded this item
	state.Set("icons/icon-1.png", ItemState{
		ETag:        "v1",
		ContentType: "icons",
		FilePath:    "/tmp/winnow/icons/icon-1.png",
	})

	s := &Syncer{
		config: Config{
			ContentDir: t.TempDir(),
		},
		downloader: dl,
		publisher:  pub,
		state:      state,
	}

	m := manifest.ManifestResponse{
		"icons": &manifest.ManifestItem{
			Items: []manifest.ContentItem{
				{Name: "icon-1.png", URI: "http://example.com/icon-1.png", ETag: "v2"},
			},
		},
	}

	s.processManifest(context.Background(), m)

	if len(pub.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(pub.events))
	}
	if pub.events[0].Action != "UPDATED" {
		t.Errorf("expected UPDATED, got %s", pub.events[0].Action)
	}
}

func TestSyncer_RemovedContent(t *testing.T) {
	dl := newMockDownloader()
	pub := &mockPublisher{}
	state := NewState(t.TempDir() + "/state.json")

	state.Set("icons/icon-1.png", ItemState{
		ETag:        "v1",
		ContentType: "icons",
		FilePath:    "/tmp/winnow/icons/icon-1.png",
	})

	s := &Syncer{
		config: Config{
			ContentDir: t.TempDir(),
		},
		downloader: dl,
		publisher:  pub,
		state:      state,
	}

	// Empty manifest — icon-1.png should be removed
	m := manifest.ManifestResponse{
		"icons": &manifest.ManifestItem{
			Items: []manifest.ContentItem{},
		},
	}

	s.processManifest(context.Background(), m)

	if len(pub.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(pub.events))
	}
	if pub.events[0].Action != "REMOVED" {
		t.Errorf("expected REMOVED, got %s", pub.events[0].Action)
	}
	if len(dl.removed) != 1 {
		t.Errorf("expected 1 file removed, got %d", len(dl.removed))
	}
}

func TestSyncer_UnavailableContentType(t *testing.T) {
	dl := newMockDownloader()
	pub := &mockPublisher{}
	state := NewState(t.TempDir() + "/state.json")

	state.Set("icons/icon-1.png", ItemState{
		ETag:        "v1",
		ContentType: "icons",
		FilePath:    "/tmp/winnow/icons/icon-1.png",
	})

	s := &Syncer{
		config: Config{
			ContentDir: t.TempDir(),
		},
		downloader: dl,
		publisher:  pub,
		state:      state,
	}

	m := manifest.ManifestResponse{
		"icons": &manifest.ManifestItem{
			Unavailable: true,
			Items:       []manifest.ContentItem{},
		},
	}

	s.processManifest(context.Background(), m)

	// Should not remove existing content or publish any events
	if len(pub.events) != 0 {
		t.Errorf("expected 0 events, got %d", len(pub.events))
	}
	if len(dl.removed) != 0 {
		t.Errorf("expected 0 removals, got %d", len(dl.removed))
	}
}

func TestSyncer_DownloadFailureKeepsExisting(t *testing.T) {
	dl := newMockDownloader()
	dl.shouldError = true
	pub := &mockPublisher{}
	state := NewState(t.TempDir() + "/state.json")

	state.Set("icons/icon-1.png", ItemState{
		ETag:        "v1",
		ContentType: "icons",
		FilePath:    "/tmp/winnow/icons/icon-1.png",
	})

	s := &Syncer{
		config: Config{
			ContentDir: t.TempDir(),
		},
		downloader: dl,
		publisher:  pub,
		state:      state,
	}

	m := manifest.ManifestResponse{
		"icons": &manifest.ManifestItem{
			Items: []manifest.ContentItem{
				{Name: "icon-1.png", URI: "http://example.com/icon-1.png", ETag: "v2"},
			},
		},
	}

	s.processManifest(context.Background(), m)

	// Download failed, so state should still have v1, not v2
	item, exists := state.Get("icons/icon-1.png")
	if !exists {
		t.Fatal("expected item to still exist in state")
	}
	if item.ETag != "v1" {
		t.Errorf("expected ETag v1 (unchanged), got %s", item.ETag)
	}
	if len(pub.events) != 0 {
		t.Errorf("expected 0 events on failure, got %d", len(pub.events))
	}
}

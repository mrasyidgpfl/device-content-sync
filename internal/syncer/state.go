package syncer

import (
	"encoding/json"
	"os"
	"sync"
)

// ItemState tracks what we've downloaded for a single content item.
type ItemState struct {
	ETag        string `json:"etag"`
	ContentType string `json:"content_type"`
	FilePath    string `json:"file_path"`
}

// State tracks all downloaded content, keyed by "contentType/itemName".
type State struct {
	mu    sync.RWMutex
	items map[string]ItemState
	path  string // file path for persistence
}

func NewState(path string) *State {
	return &State{
		items: make(map[string]ItemState),
		path:  path,
	}
}

func (s *State) Get(key string) (ItemState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[key]
	return item, ok
}

func (s *State) Set(key string, item ItemState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = item
}

func (s *State) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key)
}

func (s *State) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.items))
	for k := range s.items {
		keys = append(keys, k)
	}
	return keys
}

// Load restores state from disk. If the file doesn't exist, starts fresh.
// This handles the "device can be turned off at any time" requirement.
func (s *State) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.items)
}

// Save persists state to disk using atomic write.
func (s *State) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s.items, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.path)
}

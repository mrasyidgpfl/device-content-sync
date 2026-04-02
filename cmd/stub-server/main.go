package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
)

// Reuse the same model shape as the real manifest API
type ContentItem struct {
	Unavailable bool   `json:"unavailable,omitempty"`
	Name        string `json:"name"`
	URI         string `json:"uri,omitempty"`
	ExpiresAt   string `json:"expiresAt,omitempty"`
	ETag        string `json:"ETag,omitempty"`
}

type ManifestItem struct {
	Unavailable bool          `json:"unavailable,omitempty"`
	Items       []ContentItem `json:"items"`
}

type ManifestResponse struct {
	Menus *ManifestItem `json:"menus,omitempty"`
	Icons *ManifestItem `json:"icons,omitempty"`
}

type stubServer struct {
	mu       sync.RWMutex
	baseURL  string
	manifest ManifestResponse
}

func newStubServer(baseURL string) *stubServer {
	return &stubServer{
		baseURL: baseURL,
		manifest: ManifestResponse{
			Menus: &ManifestItem{
				Items: []ContentItem{
					{
						Name:      "current-menu",
						URI:       baseURL + "/content/menus/current-menu.json",
						ExpiresAt: "2026-12-31T23:59:59Z",
						ETag:      "menu-v1",
					},
				},
			},
			Icons: &ManifestItem{
				Items: []ContentItem{
					{
						Name:      "icon-1.png",
						URI:       baseURL + "/content/icons/icon-1.png",
						ExpiresAt: "2026-12-31T23:59:59Z",
						ETag:      "icon1-v1",
					},
					{
						Name:      "icon-2.png",
						URI:       baseURL + "/content/icons/icon-2.png",
						ExpiresAt: "2026-12-31T23:59:59Z",
						ETag:      "icon2-v1",
					},
				},
			},
		},
	}
}

func (s *stubServer) computeETag() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, _ := json.Marshal(s.manifest)
	return fmt.Sprintf(`"%x"`, md5.Sum(data))
}

func (s *stubServer) handleManifest(w http.ResponseWriter, r *http.Request) {
	etag := s.computeETag()

	if match := r.Header.Get("If-None-Match"); match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	w.Header().Set("ETag", etag)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.manifest)
}

func (s *stubServer) handleContent(w http.ResponseWriter, r *http.Request) {
	// Serve dummy content for any content path
	w.Header().Set("Content-Type", "application/octet-stream")
	fmt.Fprintf(w, "stub-content-for-%s", r.URL.Path)
}

// Admin endpoints to mutate the manifest during testing
func (s *stubServer) handleAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	action := r.URL.Query().Get("action")

	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "add-icon":
		s.manifest.Icons.Items = append(s.manifest.Icons.Items, ContentItem{
			Name:      "icon-3.png",
			URI:       s.baseURL + "/content/icons/icon-3.png",
			ExpiresAt: "2026-12-31T23:59:59Z",
			ETag:      "icon3-v1",
		})
	case "update-menu":
		if len(s.manifest.Menus.Items) > 0 {
			s.manifest.Menus.Items[0].ETag = "menu-v2"
		}
	case "remove-icon":
		if len(s.manifest.Icons.Items) > 1 {
			s.manifest.Icons.Items = s.manifest.Icons.Items[:1]
		}
	case "unavailable-icons":
		s.manifest.Icons.Unavailable = true
	case "available-icons":
		s.manifest.Icons.Unavailable = false
	default:
		http.Error(w, "unknown action", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "done: %s\n", action)
}

func main() {
	baseURL := os.Getenv("CONTENT_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	s := newStubServer(baseURL)

	http.HandleFunc("/v2/manifest", s.handleManifest)
	http.HandleFunc("/content/", s.handleContent)
	http.HandleFunc("/admin", s.handleAdmin)

	log.Println("stub server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

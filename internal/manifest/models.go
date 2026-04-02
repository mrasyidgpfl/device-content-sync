package manifest

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

// ManifestResponse uses a map so we don't hardcode content types.
// The manifest could have "menus", "icons", "translations", whatever.
type ManifestResponse map[string]*ManifestItem

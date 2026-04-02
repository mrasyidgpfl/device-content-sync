package manifest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	baseURL     string
	deviceToken string
	httpClient  *http.Client
	lastETag    string
}

func NewClient(baseURL, deviceToken string) *Client {
	return &Client{
		baseURL:     baseURL,
		deviceToken: deviceToken,
		httpClient:  &http.Client{},
	}
}

// FetchManifest returns the manifest and a boolean indicating whether
// the manifest has changed since the last fetch. If unchanged (304),
// it returns nil, false, nil.
func (c *Client) FetchManifest(ctx context.Context) (ManifestResponse, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v2/manifest", nil)
	if err != nil {
		return nil, false, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("X-Authorization-Device", c.deviceToken)
	if c.lastETag != "" {
		req.Header.Set("If-None-Match", c.lastETag)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("fetching manifest: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotModified:
		return nil, false, nil
	case http.StatusOK:
		var manifest ManifestResponse
		if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
			return nil, false, fmt.Errorf("decoding manifest: %w", err)
		}
		if etag := resp.Header.Get("ETag"); etag != "" {
			c.lastETag = etag
		}
		return manifest, true, nil
	case http.StatusUnauthorized:
		return nil, false, fmt.Errorf("device token rejected (401)")
	default:
		return nil, false, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
}

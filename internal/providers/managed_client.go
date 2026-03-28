package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CLIVersion is sent as X-Termiflow-Version header on all managed-mode requests.
// Defaults to "dev"; overridden at build time or by SetCLIVersion().
var CLIVersion = "dev"

// SetCLIVersion sets the CLI version for managed API requests.
func SetCLIVersion(v string) {
	CLIVersion = v
}

// ManagedClient is the shared HTTP client for all managed-mode API calls.
// Used by LLM proxy, search proxy, and sync.
//
//	request flow:
//	caller ──► ManagedClient.NewRequest(method, path, body)
//	              │
//	              ├── set Authorization: Bearer {apiKey}
//	              ├── set X-Termiflow-Version header
//	              └── set Content-Type: application/json
//	              │
//	              ▼
//	         ManagedClient.Do(req)      ── 120s timeout (standard calls)
//	         ManagedClient.DoStream(req) ── no timeout (SSE connections)
type ManagedClient struct {
	APIKey       string
	BaseURL      string
	Client       *http.Client // 120s timeout for standard calls
	StreamClient *http.Client // no timeout for SSE connections
}

// NewManagedClient creates a shared client for managed-mode API calls.
func NewManagedClient(apiKey, baseURL string) *ManagedClient {
	if baseURL == "" {
		baseURL = "https://api.termiflow.com"
	}
	return &ManagedClient{
		APIKey:       apiKey,
		BaseURL:      baseURL,
		Client:       &http.Client{Timeout: 120 * time.Second},
		StreamClient: &http.Client{},
	}
}

// NewRequest creates an HTTP request with auth and version headers set.
func (c *ManagedClient) NewRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("X-Termiflow-Version", CLIVersion)
	return req, nil
}

// Do executes an HTTP request using the standard client (120s timeout).
func (c *ManagedClient) Do(req *http.Request) (*http.Response, error) {
	return c.Client.Do(req)
}

// DoStream executes an HTTP request using the stream client (no timeout).
// Use for SSE connections that stay open for the full response generation.
func (c *ManagedClient) DoStream(req *http.Request) (*http.Response, error) {
	return c.StreamClient.Do(req)
}

// DoJSON sends a JSON request and decodes the response.
// Uses DoWithRetry for automatic 429/503 handling.
// If respBody is nil, the response body is discarded.
func (c *ManagedClient) DoJSON(ctx context.Context, method, path string, reqBody, respBody interface{}) error {
	var jsonBody []byte
	if reqBody != nil {
		var err error
		jsonBody, err = json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
	}

	resp, err := DoWithRetry(ctx, func() (*http.Response, error) { //nolint:bodyclose // DoWithRetry manages body lifecycle
		var body io.Reader
		if jsonBody != nil {
			body = bytes.NewReader(jsonBody) // fresh reader on each retry attempt
		}
		req, err := c.NewRequest(ctx, method, path, body)
		if err != nil {
			return nil, err
		}
		return c.Do(req)
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("API error %s: %s", resp.Status, string(bodyBytes))
	}

	if respBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

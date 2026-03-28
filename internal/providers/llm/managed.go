package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/oluoyefeso/termiflow/internal/config"
	"github.com/oluoyefeso/termiflow/internal/providers"
)

// ManagedProvider calls the termiflow backend proxy instead of Anthropic directly.
// The backend validates the termiflow API key and forwards to Anthropic.
//
//	CLI ──► POST {baseURL}/v1/messages ──► termiflow backend ──► Anthropic
//	        Authorization: Bearer {apiKey}      x-api-key: server-side key
type ManagedProvider struct {
	mc *providers.ManagedClient
}

func NewManagedProvider(apiKey, baseURL string) *ManagedProvider {
	return &ManagedProvider{
		mc: providers.NewManagedClient(apiKey, baseURL),
	}
}

// Client returns the underlying ManagedClient for use by sync and other packages.
func (p *ManagedProvider) Client() *providers.ManagedClient {
	return p.mc
}

func (p *ManagedProvider) Name() string    { return "managed" }
func (p *ManagedProvider) Available() bool { return p.mc.APIKey != "" }

func (p *ManagedProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	var systemPrompt string
	var messages []anthropicMessage

	for _, m := range req.Messages {
		if m.Role == "system" {
			systemPrompt = m.Content
		} else {
			messages = append(messages, anthropicMessage(m))
		}
	}

	body := anthropicRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: req.MaxTokens,
		Messages:  messages,
		System:    systemPrompt,
		Stream:    false,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	resp, err := providers.DoWithRetry(ctx, func() (*http.Response, error) { //nolint:bodyclose // DoWithRetry manages body lifecycle
		httpReq, err := p.mc.NewRequest(ctx, "POST", "/v1/messages", bytes.NewReader(jsonBody))
		if err != nil {
			return nil, err
		}
		return p.mc.Do(httpReq)
	})
	if err != nil {
		return nil, fmt.Errorf("managed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("managed: API error %s: %s", resp.Status, string(bodyBytes))
	}

	var anthropicResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("managed: unexpected response from Termiflow API — try again or check api.termiflow.com/health")
	}

	var content string
	for _, c := range anthropicResp.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	return &CompletionResponse{
		Content:      content,
		FinishReason: anthropicResp.StopReason,
		Usage: Usage{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
	}, nil
}

// Stream sends a streaming request to the managed API and returns SSE chunks in real-time.
// Uses ManagedClient.DoStream (no timeout) because SSE connections stay open for the full generation.
func (p *ManagedProvider) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	var systemPrompt string
	var messages []anthropicMessage

	for _, m := range req.Messages {
		if m.Role == "system" {
			systemPrompt = m.Content
		} else {
			messages = append(messages, anthropicMessage(m))
		}
	}

	body := anthropicRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: req.MaxTokens,
		Messages:  messages,
		System:    systemPrompt,
		Stream:    true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	var resp *http.Response
	for attempt := 0; ; attempt++ {
		httpReq, err := p.mc.NewRequest(ctx, "POST", "/v1/messages", bytes.NewReader(jsonBody))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Accept", "text/event-stream")

		resp, err = p.mc.DoStream(httpReq) //nolint:bodyclose // closed in goroutine, 429 path, or 503 retry path
		if err != nil {
			return nil, fmt.Errorf("managed: stream request failed: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := providers.ParseRateLimitResponse(resp)
			resp.Body.Close()
			return nil, &providers.RateLimitError{RetryAfter: retryAfter}
		}

		if resp.StatusCode == http.StatusServiceUnavailable && attempt < providers.MaxRetries() {
			resp.Body.Close()
			delay := providers.RetryDelay(attempt, resp)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			continue
		}
		break
	}

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("managed: API error %s: %s", resp.Status, string(bodyBytes))
	}

	chunks := make(chan StreamChunk)

	go func() {
		defer close(chunks)
		defer resp.Body.Close()
		// Always send Done before close so consumers checking Done don't hang on connection drops
		defer func() {
			select {
			case chunks <- StreamChunk{Done: true}:
			case <-ctx.Done():
			}
		}()

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			var event anthropicStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			switch event.Type {
			case "content_block_delta":
				if event.Delta.Text != "" {
					select {
					case chunks <- StreamChunk{Content: event.Delta.Text}:
					case <-ctx.Done():
						return
					}
				}
			case "message_stop":
				return // deferred Done send handles this
			}
		}

		if err := scanner.Err(); err != nil {
			select {
			case chunks <- StreamChunk{Error: err}:
			case <-ctx.Done():
			}
		}
	}()

	return chunks, nil
}

// healthClient is a short-timeout HTTP client for health checks only.
var healthClient = &http.Client{Timeout: 3 * time.Second}

// CheckHealth calls the /health endpoint and returns the status string ("ok" or "degraded").
func CheckHealth(baseURL string) (string, error) {
	if baseURL == "" {
		baseURL = "https://api.termiflow.com"
	}
	resp, err := healthClient.Get(baseURL + "/health")
	if err != nil {
		return "", fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		return "", fmt.Errorf("health check returned %s", resp.Status)
	}

	var result struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&result); err != nil {
		return "", fmt.Errorf("health check: invalid JSON: %w", err)
	}
	if result.Status == "" {
		return "", fmt.Errorf("health check: empty status")
	}
	return result.Status, nil
}

// healthCache is the on-disk cache format.
type healthCache struct {
	Status string `json:"status"`
	TS     int64  `json:"ts"`
}

const healthCacheTTL = 5 * time.Minute

// healthRefreshMu prevents multiple concurrent background refreshes.
var healthRefreshMu sync.Mutex

// CheckHealthCached returns the cached health status, refreshing in the background if stale.
// On first-ever call (no cache), blocks for up to 3s. Never returns an error to callers.
func CheckHealthCached(baseURL string) string {
	if baseURL == "" {
		baseURL = "https://api.termiflow.com"
	}
	cachePath := healthCachePath(baseURL)

	// Try to read cache
	data, err := os.ReadFile(cachePath)
	if err == nil {
		var cached healthCache
		if json.Unmarshal(data, &cached) == nil && cached.Status != "" {
			age := time.Since(time.Unix(cached.TS, 0))
			if age < healthCacheTTL {
				return cached.Status // fresh cache
			}
			// Stale cache: return it immediately, refresh in background
			go refreshHealthCache(baseURL, cachePath)
			return cached.Status
		}
	}

	// No valid cache: block for first call
	status, err := CheckHealth(baseURL)
	if err != nil {
		return "unknown"
	}
	writeHealthCache(cachePath, status)
	return status
}

func healthCachePath(baseURL string) string {
	hash := crc32.ChecksumIEEE([]byte(baseURL))
	return filepath.Join(config.GetCacheDir(), fmt.Sprintf("health-%08x.json", hash))
}

func refreshHealthCache(baseURL, cachePath string) {
	if !healthRefreshMu.TryLock() {
		return // another refresh is already running
	}
	defer healthRefreshMu.Unlock()

	status, err := CheckHealth(baseURL)
	if err != nil {
		return // keep stale cache
	}
	writeHealthCache(cachePath, status)
}

func writeHealthCache(cachePath, status string) {
	data, _ := json.Marshal(healthCache{Status: status, TS: time.Now().Unix()})
	_ = os.MkdirAll(filepath.Dir(cachePath), 0o755)
	_ = os.WriteFile(cachePath, data, 0o644)
}

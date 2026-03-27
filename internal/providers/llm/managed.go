package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/oluoyefeso/termiflow/internal/providers"
)

// CLIVersion is set at build time and sent as X-Termiflow-Version header.
var CLIVersion = "dev"

// ManagedProvider calls the termiflow backend proxy instead of Anthropic directly.
// The backend validates the termiflow API key and forwards to Anthropic.
//
//	CLI ──► POST {baseURL}/v1/messages ──► termiflow backend ──► Anthropic
//	        Authorization: Bearer {apiKey}      x-api-key: server-side key
type ManagedProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewManagedProvider(apiKey, baseURL string) *ManagedProvider {
	if baseURL == "" {
		baseURL = "https://api.termiflow.com"
	}
	return &ManagedProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *ManagedProvider) Name() string    { return "managed" }
func (p *ManagedProvider) Available() bool { return p.apiKey != "" }

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
		httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/messages", bytes.NewReader(jsonBody))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
		httpReq.Header.Set("X-Termiflow-Version", CLIVersion)
		return p.client.Do(httpReq)
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

// streamClient is a separate HTTP client with no timeout for SSE connections,
// which stay open for the full response generation.
var streamClient = &http.Client{}

// Stream sends a streaming request to the managed API and returns SSE chunks in real-time.
// Uses a separate HTTP client with no timeout (SSE connections stay open for the full generation).
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
		httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/messages", bytes.NewReader(jsonBody))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
		httpReq.Header.Set("X-Termiflow-Version", CLIVersion)
		httpReq.Header.Set("Accept", "text/event-stream")

		resp, err = streamClient.Do(httpReq) //nolint:bodyclose // closed in goroutine, 429 path, or 503 retry path
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

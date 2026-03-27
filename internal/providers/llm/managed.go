package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

	var resp *http.Response
	for attempt := 0; ; attempt++ {
		httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/messages", bytes.NewReader(jsonBody))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
		httpReq.Header.Set("X-Termiflow-Version", CLIVersion)

		resp, err = p.client.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("managed: request failed: %w", err)
		}
		if !providers.IsRetryable(resp.StatusCode) || attempt >= providers.MaxRetries() {
			break
		}
		resp.Body.Close()
		delay := providers.RetryDelay(attempt, resp)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
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

// Stream buffers the response via Complete and emits it as a single chunk.
// SSE streaming is deferred to a future release.
func (p *ManagedProvider) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	resp, err := p.Complete(ctx, req)
	if err != nil {
		return nil, err
	}
	ch := make(chan StreamChunk, 2)
	go func() {
		ch <- StreamChunk{Content: resp.Content}
		ch <- StreamChunk{Done: true}
		close(ch)
	}()
	return ch, nil
}

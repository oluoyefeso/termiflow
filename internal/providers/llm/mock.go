//go:build mock

package llm

import (
	"context"
	"os"
	"strings"
)

type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (p *MockProvider) Name() string    { return "mock" }
func (p *MockProvider) Available() bool { return true }

// Complete inspects the prompt to return contextually appropriate mock content:
//   - relevance scoring prompts  → "0.85"
//   - summarization prompts      → realistic developer summary
//   - tag extraction prompts     → realistic comma-separated tags
//   - ask/stream prompts         → a plausible answer
func (p *MockProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	return &CompletionResponse{
		Content:      mockResponseForRequest(req),
		FinishReason: "end_turn",
		Usage:        Usage{PromptTokens: 80, CompletionTokens: 20, TotalTokens: 100},
	}, nil
}

func (p *MockProvider) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
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

func mockResponseForRequest(req CompletionRequest) string {
	// Allow full override via env
	if path := os.Getenv("TERMIFLOW_MOCK_FIXTURE"); path != "" {
		if b, err := os.ReadFile(path); err == nil {
			return strings.TrimSpace(string(b))
		}
	}

	// Detect call type from the last user message
	var prompt string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			prompt = req.Messages[i].Content
			break
		}
	}

	switch {
	case strings.Contains(prompt, "Rate the relevance") || strings.Contains(prompt, "0.0 to 1.0"):
		return "0.85"

	case strings.Contains(prompt, "Summarize"):
		return "This piece examines recent developments in systems-level programming, covering async runtime improvements, zero-cost abstraction patterns, and production deployment strategies. The technical focus is on compile-time safety guarantees and their impact on developer productivity in large codebases."

	case strings.Contains(prompt, "Extract") && strings.Contains(prompt, "tags"):
		return "rust, async, systems, performance"

	default:
		// ask command or anything else
		return "Based on the latest developments, async Rust has seen significant improvements in 2025. The Tokio runtime introduced work-stealing schedulers that reduce tail latency by 40% in I/O-heavy workloads. Axum 0.8 landed with first-class support for streaming responses and improved middleware composition. The main ecosystem trend is consolidation around tower-compatible middleware and a move away from callback-heavy patterns toward structured concurrency."
	}
}

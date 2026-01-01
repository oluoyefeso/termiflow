package llm

import (
	"context"
	"fmt"

	"github.com/termiflow/termiflow/internal/config"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CompletionRequest struct {
	Messages    []Message
	MaxTokens   int
	Temperature float64
	Stream      bool
}

type CompletionResponse struct {
	Content      string
	FinishReason string
	Usage        Usage
}

type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type StreamChunk struct {
	Content string
	Done    bool
	Error   error
}

type Provider interface {
	Name() string
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
	Available() bool
}

func GetProvider(name string, cfg *config.Config) (Provider, error) {
	switch name {
	case "openai":
		return NewOpenAIProvider(
			cfg.Providers.OpenAI.APIKey,
			cfg.Providers.OpenAI.BaseURL,
			cfg.Providers.OpenAI.Model,
		), nil
	case "anthropic":
		return NewAnthropicProvider(
			cfg.Providers.Anthropic.APIKey,
			cfg.Providers.Anthropic.Model,
		), nil
	case "local":
		return NewLocalProvider(
			cfg.Providers.Local.BaseURL,
			cfg.Providers.Local.Model,
		), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}

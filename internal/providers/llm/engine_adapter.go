package llm

import (
	"context"

	engine "github.com/oluoyefeso/termiflow-engine"
)

// EngineAdapter wraps a CLI llm.Provider to implement engine.LLMProvider.
// This allows CLI providers (Anthropic, OpenAI, Local, Managed) to be passed
// to the engine's Curator without the engine knowing about CLI-specific methods
// like Name(), Available(), or Stream().
type EngineAdapter struct {
	Provider Provider
}

// AsEngine wraps a CLI Provider for use with the termiflow-engine.
func AsEngine(p Provider) engine.LLMProvider {
	return &EngineAdapter{Provider: p}
}

func (a *EngineAdapter) Complete(ctx context.Context, req engine.CompletionRequest) (*engine.CompletionResponse, error) {
	// Convert engine request → CLI request
	cliMessages := make([]Message, len(req.Messages))
	for i, m := range req.Messages {
		cliMessages[i] = Message{Role: m.Role, Content: m.Content}
	}

	resp, err := a.Provider.Complete(ctx, CompletionRequest{
		Messages:    cliMessages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      req.Stream,
	})
	if err != nil {
		return nil, err
	}

	// Convert CLI response → engine response
	return &engine.CompletionResponse{
		Content:      resp.Content,
		FinishReason: resp.FinishReason,
		Usage: engine.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

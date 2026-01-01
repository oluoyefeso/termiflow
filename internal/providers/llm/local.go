package llm

import (
	"context"
)

// LocalProvider wraps OpenAIProvider for OpenAI-compatible local servers
// like Ollama, llama.cpp, LM Studio, etc.
type LocalProvider struct {
	*OpenAIProvider
}

func NewLocalProvider(baseURL, model string) *LocalProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1"
	}
	if model == "" {
		model = "llama3"
	}

	// Local providers don't need an API key, but we set a dummy one
	// to satisfy the OpenAI client
	return &LocalProvider{
		OpenAIProvider: NewOpenAIProvider("local", baseURL, model),
	}
}

func (p *LocalProvider) Name() string {
	return "local"
}

func (p *LocalProvider) Available() bool {
	// Local is available if we have a base URL configured
	return p.baseURL != ""
}

func (p *LocalProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	return p.OpenAIProvider.Complete(ctx, req)
}

func (p *LocalProvider) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	return p.OpenAIProvider.Stream(ctx, req)
}

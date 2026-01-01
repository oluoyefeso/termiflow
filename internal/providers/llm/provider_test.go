package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/termiflow/termiflow/internal/config"
)

func TestGetProvider(t *testing.T) {
	cfg := &config.Config{
		Providers: config.ProvidersConfig{
			OpenAI: config.OpenAIConfig{
				APIKey:  "test-key",
				BaseURL: "https://api.openai.com/v1",
				Model:   "gpt-4o",
			},
			Anthropic: config.AnthropicConfig{
				APIKey: "test-anthropic-key",
				Model:  "claude-3-opus",
			},
			Local: config.LocalConfig{
				BaseURL: "http://localhost:11434/v1",
				Model:   "llama3",
			},
		},
	}

	tests := []struct {
		name        string
		provider    string
		expectError bool
		expectName  string
	}{
		{"openai", "openai", false, "openai"},
		{"anthropic", "anthropic", false, "anthropic"},
		{"local", "local", false, "local"},
		{"unknown", "unknown", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := GetProvider(tt.provider, cfg)

			if tt.expectError {
				if err == nil {
					t.Error("GetProvider() should return error")
				}
				return
			}

			if err != nil {
				t.Errorf("GetProvider() error = %v", err)
				return
			}

			if provider.Name() != tt.expectName {
				t.Errorf("Name() = %q, want %q", provider.Name(), tt.expectName)
			}
		})
	}
}

func TestOpenAIProvider_Name(t *testing.T) {
	p := NewOpenAIProvider("key", "", "")
	if p.Name() != "openai" {
		t.Errorf("Name() = %q, want %q", p.Name(), "openai")
	}
}

func TestOpenAIProvider_Available(t *testing.T) {
	tests := []struct {
		apiKey   string
		expected bool
	}{
		{"", false},
		{"sk-test", true},
	}

	for _, tt := range tests {
		p := NewOpenAIProvider(tt.apiKey, "", "")
		if p.Available() != tt.expected {
			t.Errorf("Available() with key %q = %v, want %v", tt.apiKey, p.Available(), tt.expected)
		}
	}
}

func TestOpenAIProvider_Defaults(t *testing.T) {
	p := NewOpenAIProvider("key", "", "")

	if p.baseURL != "https://api.openai.com/v1" {
		t.Errorf("baseURL = %q, want default", p.baseURL)
	}
	if p.model != "gpt-4o" {
		t.Errorf("model = %q, want gpt-4o", p.model)
	}
}

func TestOpenAIProvider_CustomValues(t *testing.T) {
	p := NewOpenAIProvider("key", "https://custom.api.com/v1/", "gpt-4-turbo")

	if p.baseURL != "https://custom.api.com/v1" {
		t.Errorf("baseURL = %q, want trimmed URL", p.baseURL)
	}
	if p.model != "gpt-4-turbo" {
		t.Errorf("model = %q, want gpt-4-turbo", p.model)
	}
}

func TestOpenAIProvider_Complete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected /chat/completions, got %s", r.URL.Path)
		}

		// Check headers
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Wrong Authorization header")
		}

		response := map[string]interface{}{
			"id": "test-id",
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"role":    "assistant",
						"content": "Hello, world!",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{
				"prompt_tokens":     10,
				"completion_tokens": 5,
				"total_tokens":      15,
			},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	p := NewOpenAIProvider("test-key", server.URL, "gpt-4o")

	req := CompletionRequest{
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 100,
	}

	resp, err := p.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if resp.Content != "Hello, world!" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello, world!")
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("TotalTokens = %d, want %d", resp.Usage.TotalTokens, 15)
	}
}

func TestOpenAIProvider_CompleteError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "bad request"}`))
	}))
	defer server.Close()

	p := NewOpenAIProvider("test-key", server.URL, "gpt-4o")

	req := CompletionRequest{
		Messages: []Message{{Role: "user", Content: "Hello"}},
	}

	_, err := p.Complete(context.Background(), req)
	if err == nil {
		t.Error("Complete() should return error on bad status")
	}
}

func TestAnthropicProvider_Name(t *testing.T) {
	p := NewAnthropicProvider("key", "")
	if p.Name() != "anthropic" {
		t.Errorf("Name() = %q, want %q", p.Name(), "anthropic")
	}
}

func TestAnthropicProvider_Available(t *testing.T) {
	tests := []struct {
		apiKey   string
		expected bool
	}{
		{"", false},
		{"sk-ant-test", true},
	}

	for _, tt := range tests {
		p := NewAnthropicProvider(tt.apiKey, "")
		if p.Available() != tt.expected {
			t.Errorf("Available() with key %q = %v, want %v", tt.apiKey, p.Available(), tt.expected)
		}
	}
}

func TestAnthropicProvider_Defaults(t *testing.T) {
	p := NewAnthropicProvider("key", "")
	if p.model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %q, want default", p.model)
	}
}

func TestLocalProvider_Name(t *testing.T) {
	p := NewLocalProvider("", "")
	if p.Name() != "local" {
		t.Errorf("Name() = %q, want %q", p.Name(), "local")
	}
}

func TestLocalProvider_Available(t *testing.T) {
	// LocalProvider with default URL (from empty string) is available
	p := NewLocalProvider("", "")
	// With defaults applied, it has a URL so should be available
	if !p.Available() {
		t.Error("LocalProvider with default URL should be available")
	}

	// With explicit URL
	p2 := NewLocalProvider("http://localhost:11434/v1", "")
	if !p2.Available() {
		t.Error("LocalProvider with explicit URL should be available")
	}
}

func TestLocalProvider_Defaults(t *testing.T) {
	p := NewLocalProvider("", "")

	if p.baseURL != "http://localhost:11434/v1" {
		t.Errorf("baseURL = %q, want default Ollama URL", p.baseURL)
	}
	if p.model != "llama3" {
		t.Errorf("model = %q, want llama3", p.model)
	}
}

func TestMessage(t *testing.T) {
	m := Message{Role: "user", Content: "test"}
	if m.Role != "user" {
		t.Errorf("Role = %q, want %q", m.Role, "user")
	}
	if m.Content != "test" {
		t.Errorf("Content = %q, want %q", m.Content, "test")
	}
}

func TestCompletionRequest(t *testing.T) {
	req := CompletionRequest{
		Messages: []Message{
			{Role: "user", Content: "test"},
		},
		MaxTokens:   100,
		Temperature: 0.7,
		Stream:      true,
	}

	if len(req.Messages) != 1 {
		t.Errorf("Messages length = %d, want 1", len(req.Messages))
	}
	if req.MaxTokens != 100 {
		t.Errorf("MaxTokens = %d, want 100", req.MaxTokens)
	}
	if req.Temperature != 0.7 {
		t.Errorf("Temperature = %f, want 0.7", req.Temperature)
	}
	if !req.Stream {
		t.Error("Stream should be true")
	}
}

func TestUsage(t *testing.T) {
	u := Usage{
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
	}

	if u.PromptTokens != 10 {
		t.Errorf("PromptTokens = %d, want 10", u.PromptTokens)
	}
	if u.CompletionTokens != 20 {
		t.Errorf("CompletionTokens = %d, want 20", u.CompletionTokens)
	}
	if u.TotalTokens != 30 {
		t.Errorf("TotalTokens = %d, want 30", u.TotalTokens)
	}
}

func TestStreamChunk(t *testing.T) {
	chunk := StreamChunk{
		Content: "test content",
		Done:    false,
		Error:   nil,
	}

	if chunk.Content != "test content" {
		t.Errorf("Content = %q, want %q", chunk.Content, "test content")
	}
	if chunk.Done {
		t.Error("Done should be false")
	}
	if chunk.Error != nil {
		t.Error("Error should be nil")
	}
}

package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestManagedProviderComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer tf_testkey" {
			t.Errorf("expected Bearer tf_testkey, got %s", r.Header.Get("Authorization"))
		}
		// Verify stream is forced false
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if stream, ok := body["stream"]; ok && stream != false {
			t.Errorf("expected stream:false, got %v", stream)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(anthropicResponse{
			StopReason: "end_turn",
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{{"text", "hello from managed"}},
			Usage: struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			}{10, 5},
		})
	}))
	defer srv.Close()

	p := NewManagedProvider("tf_testkey", srv.URL)
	resp, err := p.Complete(context.Background(), CompletionRequest{
		Messages:  []Message{{Role: "user", Content: "hi"}},
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "hello from managed" {
		t.Errorf("expected 'hello from managed', got %q", resp.Content)
	}
}

func TestManagedProviderUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid key"}`))
	}))
	defer srv.Close()

	p := NewManagedProvider("tf_badkey", srv.URL)
	_, err := p.Complete(context.Background(), CompletionRequest{
		Messages:  []Message{{Role: "user", Content: "hi"}},
		MaxTokens: 100,
	})
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}

func TestManagedProviderStreamBuffers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(anthropicResponse{
			StopReason: "end_turn",
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{{"text", "buffered response"}},
			Usage: struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			}{10, 5},
		})
	}))
	defer srv.Close()

	p := NewManagedProvider("tf_testkey", srv.URL)
	ch, err := p.Stream(context.Background(), CompletionRequest{
		Messages:  []Message{{Role: "user", Content: "hi"}},
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var chunks []StreamChunk
	for c := range ch {
		chunks = append(chunks, c)
	}
	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks (content + done), got %d", len(chunks))
	}
	if chunks[0].Content != "buffered response" {
		t.Errorf("expected 'buffered response', got %q", chunks[0].Content)
	}
	if !chunks[1].Done {
		t.Error("expected last chunk to be Done")
	}
}

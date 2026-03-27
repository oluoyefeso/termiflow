package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/oluoyefeso/termiflow/internal/providers"
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

func TestManagedProvider429RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "3540")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":               "rate limit exceeded",
			"retry_after_seconds": 3540,
		})
	}))
	defer srv.Close()

	p := NewManagedProvider("tf_testkey", srv.URL)
	_, err := p.Complete(context.Background(), CompletionRequest{
		Messages:  []Message{{Role: "user", Content: "hi"}},
		MaxTokens: 100,
	})
	if err == nil {
		t.Fatal("expected error for 429, got nil")
	}
	var rle *providers.RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("expected RateLimitError, got %T: %v", err, err)
	}
	if rle.RetryAfter.Seconds() != 3540 {
		t.Errorf("expected RetryAfter 3540s, got %v", rle.RetryAfter)
	}
}

func TestManagedProvider503Retry(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(anthropicResponse{
			StopReason: "end_turn",
			Content: []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}{{"text", "recovered"}},
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
	if resp.Content != "recovered" {
		t.Errorf("expected 'recovered', got %q", resp.Content)
	}
	if got := int(attempts.Load()); got != 2 {
		t.Errorf("expected 2 attempts, got %d", got)
	}
}

func TestManagedProviderStreamSSE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify stream:true was sent
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if stream, ok := body["stream"].(bool); !ok || !stream {
			t.Errorf("expected stream:true, got %v", body["stream"])
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)

		// Send SSE events
		fmt.Fprint(w, "event: message_start\ndata: {\"type\":\"message_start\"}\n\n")
		flusher.Flush()
		fmt.Fprint(w, "event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0}\n\n")
		flusher.Flush()
		fmt.Fprint(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n")
		flusher.Flush()
		fmt.Fprint(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\" world\"}}\n\n")
		flusher.Flush()
		fmt.Fprint(w, "data: {\"type\":\"message_stop\"}\n\n")
		flusher.Flush()
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
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks (2 content + done), got %d", len(chunks))
	}
	if chunks[0].Content != "Hello" {
		t.Errorf("expected 'Hello', got %q", chunks[0].Content)
	}
	if chunks[1].Content != " world" {
		t.Errorf("expected ' world', got %q", chunks[1].Content)
	}
	if !chunks[2].Done {
		t.Error("expected last chunk to be Done")
	}
}

func TestManagedProviderStreamRetry503(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		fmt.Fprint(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"recovered\"}}\n\n")
		flusher.Flush()
		fmt.Fprint(w, "data: {\"type\":\"message_stop\"}\n\n")
		flusher.Flush()
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
	if got := int(attempts.Load()); got != 3 {
		t.Errorf("expected 3 total requests, got %d", got)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks (content + done), got %d", len(chunks))
	}
	if chunks[0].Content != "recovered" {
		t.Errorf("expected 'recovered', got %q", chunks[0].Content)
	}
	if !chunks[len(chunks)-1].Done {
		t.Error("expected last chunk to be Done")
	}
}

func TestManagedProviderStreamLargeSSEEvent(t *testing.T) {
	// Generate a string larger than the default 64KB scanner buffer
	largeContent := make([]byte, 80*1024)
	for i := range largeContent {
		largeContent[i] = 'A'
	}
	largeStr := string(largeContent)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)

		// Send a large SSE event that exceeds default 64KB bufio.Scanner buffer
		fmt.Fprintf(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":%q}}\n\n", largeStr)
		flusher.Flush()
		fmt.Fprint(w, "data: {\"type\":\"message_stop\"}\n\n")
		flusher.Flush()
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
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks (content + done), got %d", len(chunks))
	}
	if chunks[0].Content != largeStr {
		t.Errorf("large content not received intact: got %d bytes, want %d bytes", len(chunks[0].Content), len(largeStr))
	}
}

func TestManagedProviderStreamNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal server error"}`))
	}))
	defer srv.Close()

	p := NewManagedProvider("tf_testkey", srv.URL)
	_, err := p.Stream(context.Background(), CompletionRequest{
		Messages:  []Message{{Role: "user", Content: "hi"}},
		MaxTokens: 100,
	})
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
	if !contains(err.Error(), "API error") {
		t.Errorf("expected API error message, got %q", err.Error())
	}
}

func TestManagedProviderStream429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":               "rate limit exceeded",
			"retry_after_seconds": 2400,
		})
	}))
	defer srv.Close()

	p := NewManagedProvider("tf_testkey", srv.URL)
	_, err := p.Stream(context.Background(), CompletionRequest{
		Messages:  []Message{{Role: "user", Content: "hi"}},
		MaxTokens: 100,
	})

	var rle *providers.RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("expected RateLimitError, got %T: %v", err, err)
	}
	if rle.RetryAfter.Seconds() != 2400 {
		t.Errorf("expected RetryAfter 2400s, got %v", rle.RetryAfter)
	}
}

func TestManagedProviderStreamMalformedSSE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)

		// Malformed data line (invalid JSON), should be skipped
		fmt.Fprint(w, "data: {not valid json}\n\n")
		flusher.Flush()
		// Non-data line, should be skipped
		fmt.Fprint(w, "event: ping\n\n")
		flusher.Flush()
		// Valid content
		fmt.Fprint(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"ok\"}}\n\n")
		flusher.Flush()
		fmt.Fprint(w, "data: {\"type\":\"message_stop\"}\n\n")
		flusher.Flush()
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
	// Should only get the valid content + done, malformed lines skipped
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks (1 content + done), got %d", len(chunks))
	}
	if chunks[0].Content != "ok" {
		t.Errorf("expected 'ok', got %q", chunks[0].Content)
	}
}

func TestManagedProviderStreamEmptyContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)

		// Empty text delta should be skipped
		fmt.Fprint(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"\"}}\n\n")
		flusher.Flush()
		fmt.Fprint(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"content\"}}\n\n")
		flusher.Flush()
		fmt.Fprint(w, "data: {\"type\":\"message_stop\"}\n\n")
		flusher.Flush()
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
	// Empty content skipped, so 1 content + 1 done
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if chunks[0].Content != "content" {
		t.Errorf("expected 'content', got %q", chunks[0].Content)
	}
}

func TestManagedProviderStreamConnectionDrop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)

		fmt.Fprint(w, "data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"partial\"}}\n\n")
		flusher.Flush()
		// Close connection without message_stop (simulates connection drop)
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
	// Should get partial content, channel closes when server closes connection
	if len(chunks) < 1 {
		t.Fatal("expected at least 1 chunk")
	}
	if chunks[0].Content != "partial" {
		t.Errorf("expected 'partial', got %q", chunks[0].Content)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

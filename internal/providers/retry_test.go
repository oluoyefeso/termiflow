package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRateLimitError_Error(t *testing.T) {
	tests := []struct {
		name       string
		retryAfter time.Duration
		want       string
	}{
		{"minutes", 47 * time.Minute, "Rate limited. Try again in 47m."},
		{"rounds up", 119 * time.Second, "Rate limited. Try again in 2m."},
		{"seconds only", 30 * time.Second, "Rate limited. Try again in 30s."},
		{"sub-minute", 5 * time.Second, "Rate limited. Try again in 5s."},
		{"zero", 0, "Rate limited. Try again later."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &RateLimitError{RetryAfter: tt.retryAfter}
			if got := e.Error(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRateLimitError_ErrorsAs(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", &RateLimitError{RetryAfter: 10 * time.Minute})
	var rle *RateLimitError
	if !errors.As(err, &rle) {
		t.Fatal("errors.As failed to unwrap RateLimitError")
	}
	if rle.RetryAfter != 10*time.Minute {
		t.Errorf("got RetryAfter %v, want 10m", rle.RetryAfter)
	}
}

func TestDoWithRetry_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	resp, err := DoWithRetry(context.Background(), func() (*http.Response, error) {
		return http.Get(srv.URL)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDoWithRetry_503RetrySucceeds(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	resp, err := DoWithRetry(context.Background(), func() (*http.Response, error) {
		return http.Get(srv.URL)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 after retries, got %d", resp.StatusCode)
	}
	if got := int(attempts.Load()); got != 3 {
		t.Errorf("expected 3 attempts, got %d", got)
	}
}

func TestDoWithRetry_503RetriesExhausted(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	resp, err := DoWithRetry(context.Background(), func() (*http.Response, error) {
		return http.Get(srv.URL)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 after exhausted retries, got %d", resp.StatusCode)
	}
	// maxRetries is 3, so attempt 0 + 3 retries = 4 total attempts
	if got := int(attempts.Load()); got != maxRetries+1 {
		t.Errorf("expected %d attempts, got %d", maxRetries+1, got)
	}
}

func TestDoWithRetry_429ReturnsImmediately(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":               "rate limit exceeded",
			"retry_after_seconds": 3540,
		})
	}))
	defer srv.Close()

	_, err := DoWithRetry(context.Background(), func() (*http.Response, error) {
		return http.Get(srv.URL)
	})

	var rle *RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("expected RateLimitError, got %v", err)
	}
	if rle.RetryAfter != 3540*time.Second {
		t.Errorf("expected RetryAfter 3540s, got %v", rle.RetryAfter)
	}
	if got := int(attempts.Load()); got != 1 {
		t.Errorf("expected 1 attempt (no retry on 429), got %d", got)
	}
}

func TestDoWithRetry_429FallbackToHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "1800")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate limit exceeded"}`)) // no retry_after_seconds field
	}))
	defer srv.Close()

	_, err := DoWithRetry(context.Background(), func() (*http.Response, error) {
		return http.Get(srv.URL)
	})

	var rle *RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("expected RateLimitError, got %v", err)
	}
	if rle.RetryAfter != 1800*time.Second {
		t.Errorf("expected RetryAfter 1800s from header, got %v", rle.RetryAfter)
	}
}

func TestDoWithRetry_429NoRetryInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate limit exceeded"}`))
	}))
	defer srv.Close()

	_, err := DoWithRetry(context.Background(), func() (*http.Response, error) {
		return http.Get(srv.URL)
	})

	var rle *RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("expected RateLimitError, got %v", err)
	}
	if rle.RetryAfter != 0 {
		t.Errorf("expected RetryAfter 0 (no info), got %v", rle.RetryAfter)
	}
	if rle.Error() != "Rate limited. Try again later." {
		t.Errorf("expected generic message, got %q", rle.Error())
	}
}

func TestDoWithRetry_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately so the retry wait is interrupted
	cancel()

	_, err := DoWithRetry(ctx, func() (*http.Response, error) {
		return http.Get(srv.URL)
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestDoWithRetry_NetworkError(t *testing.T) {
	_, err := DoWithRetry(context.Background(), func() (*http.Response, error) {
		return nil, fmt.Errorf("dial tcp: connection refused")
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "dial tcp: connection refused" {
		t.Errorf("expected connection refused error, got %v", err)
	}
}

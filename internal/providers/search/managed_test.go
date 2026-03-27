package search

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/oluoyefeso/termiflow/internal/providers"
)

func TestManagedSearchProviderSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tf_testkey" {
			t.Errorf("expected Bearer tf_testkey, got %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"title":          "Test Article",
					"url":            "https://example.com/1",
					"content":        "Test content",
					"snippet":        "Test snippet",
					"published_date": "2026-03-25T00:00:00Z",
					"source":         "example.com",
				},
			},
		})
	}))
	defer srv.Close()

	p := NewManagedSearchProvider("tf_testkey", srv.URL)
	results, err := p.Search(context.Background(), SearchRequest{Query: "test", MaxResults: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "Test Article" {
		t.Errorf("expected 'Test Article', got %q", results[0].Title)
	}
}

func TestManagedSearchProvider429RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":               "rate limit exceeded",
			"retry_after_seconds": 1800,
		})
	}))
	defer srv.Close()

	p := NewManagedSearchProvider("tf_testkey", srv.URL)
	_, err := p.Search(context.Background(), SearchRequest{Query: "test"})
	if err == nil {
		t.Fatal("expected error for 429, got nil")
	}
	var rle *providers.RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("expected RateLimitError, got %T: %v", err, err)
	}
	if rle.RetryAfter.Seconds() != 1800 {
		t.Errorf("expected RetryAfter 1800s, got %v", rle.RetryAfter)
	}
}

func TestManagedSearchProvider503Retry(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"title":          "Result After Retry",
					"url":            "https://example.com/1",
					"content":        "content",
					"snippet":        "snippet",
					"published_date": "2026-03-25T00:00:00Z",
					"source":         "example.com",
				},
			},
		})
	}))
	defer srv.Close()

	p := NewManagedSearchProvider("tf_testkey", srv.URL)
	results, err := p.Search(context.Background(), SearchRequest{Query: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Title != "Result After Retry" {
		t.Errorf("unexpected results: %+v", results)
	}
	if got := int(attempts.Load()); got != 2 {
		t.Errorf("expected 2 attempts, got %d", got)
	}
}

func TestManagedSearchProviderUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid key"}`))
	}))
	defer srv.Close()

	p := NewManagedSearchProvider("tf_badkey", srv.URL)
	_, err := p.Search(context.Background(), SearchRequest{Query: "test"})
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}

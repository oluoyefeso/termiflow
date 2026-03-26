package search

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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

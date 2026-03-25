//go:build mock

package search

import (
	"context"
	"fmt"
	"time"
)

type MockSearchProvider struct{}

func NewMockSearchProvider() *MockSearchProvider {
	return &MockSearchProvider{}
}

func (p *MockSearchProvider) Name() string    { return "mock" }
func (p *MockSearchProvider) Available() bool { return true }

// Search returns realistic mock results. URLs are timestamped so each refresh
// produces genuinely new items (bypassing the DB dedup check on identical URLs).
func (p *MockSearchProvider) Search(ctx context.Context, req SearchRequest) ([]SearchResult, error) {
	now := time.Now()
	ts := now.Unix()

	articles := []struct {
		title   string
		content string
		source  string
	}{
		{
			title:   fmt.Sprintf("[%s] Tokio 2.0 ships work-stealing scheduler with 40%% lower tail latency", req.Query),
			content: "The Tokio runtime released version 2.0 with a redesigned work-stealing scheduler that reduces P99 latency by 40% in I/O-bound workloads. The change affects all users of async/await in Rust. New API surface includes structured concurrency primitives and first-class cancellation tokens. Migration from 1.x is largely mechanical with a provided codemod.",
			source:  "blog.tokio.rs",
		},
		{
			title:   fmt.Sprintf("[%s] Axum 0.8: streaming responses and tower middleware changes", req.Query),
			content: "Axum 0.8 shipped with native streaming response support, removing the need for manual Body implementations. The middleware layer now uses a simpler trait bound. Breaking changes affect projects using custom extractors. The Axum team published a migration guide covering the 15 most common patterns.",
			source:  "github.com",
		},
		{
			title:   fmt.Sprintf("[%s] Rust 2025 edition: let chains and async closures stabilized", req.Query),
			content: "The Rust 2025 edition stabilizes let chains (let x = ... && let y = ...) and async closures. Both features were in nightly for over a year. Let chains simplify deeply nested if-let patterns common in error handling. Async closures enable higher-order async functions without workaround trait objects.",
			source:  "blog.rust-lang.org",
		},
	}

	var results []SearchResult
	for i, a := range articles {
		results = append(results, SearchResult{
			Title:       a.title,
			URL:         fmt.Sprintf("https://%s/mock/%d/%d", a.source, ts, i),
			Content:     a.content,
			Snippet:     a.content[:min(len(a.content), 200)],
			PublishedAt: now.Add(-time.Duration(i) * 4 * time.Hour),
			Source:      a.source,
		})
	}
	return results, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

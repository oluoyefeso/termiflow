package search

import (
	engine "github.com/oluoyefeso/termiflow-engine"
)

// ToEngineResults converts CLI search results to engine search results.
func ToEngineResults(results []SearchResult) []engine.SearchResult {
	out := make([]engine.SearchResult, len(results))
	for i, r := range results {
		out[i] = engine.SearchResult{
			Title:       r.Title,
			URL:         r.URL,
			Snippet:     r.Snippet,
			Content:     r.Content,
			PublishedAt: r.PublishedAt,
			Source:      r.Source,
		}
	}
	return out
}

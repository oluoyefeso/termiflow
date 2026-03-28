package search

import (
	"context"
	"fmt"
	"time"

	"github.com/oluoyefeso/termiflow/internal/providers"
)

// ManagedFeedFetcher calls the termiflow backend to fetch and parse RSS feeds.
//
//	CLI ──► POST {baseURL}/v1/sources/fetch ──► termiflow backend ──► upstream RSS
//	        Authorization: Bearer {apiKey}       (cached, SSRF-protected)
type ManagedFeedFetcher struct {
	mc *providers.ManagedClient
}

func NewManagedFeedFetcher(apiKey, baseURL string) *ManagedFeedFetcher {
	return &ManagedFeedFetcher{
		mc: providers.NewManagedClient(apiKey, baseURL),
	}
}

type sourceFetchRequest struct {
	URL   string `json:"url"`
	Since string `json:"since,omitempty"`
}

type sourceFetchResponse struct {
	Results []struct {
		Title         string `json:"title"`
		URL           string `json:"url"`
		Content       string `json:"content"`
		Snippet       string `json:"snippet"`
		Source        string `json:"source"`
		PublishedDate string `json:"published_date"`
	} `json:"results"`
}

func (f *ManagedFeedFetcher) FetchFeed(ctx context.Context, feedURL string, since *time.Time) ([]SearchResult, error) {
	req := sourceFetchRequest{URL: feedURL}
	if since != nil {
		req.Since = since.UTC().Format(time.RFC3339)
	}

	var resp sourceFetchResponse
	if err := f.mc.DoJSON(ctx, "POST", "/v1/sources/fetch", req, &resp); err != nil {
		return nil, fmt.Errorf("managed feed fetch: %w", err)
	}

	var results []SearchResult
	for _, r := range resp.Results {
		var pub time.Time
		if r.PublishedDate != "" {
			if parsed, err := time.Parse(time.RFC3339, r.PublishedDate); err == nil {
				pub = parsed
			}
		}
		results = append(results, SearchResult{
			Title:       r.Title,
			URL:         r.URL,
			Content:     r.Content,
			Snippet:     r.Snippet,
			PublishedAt: pub,
			Source:      r.Source,
		})
	}
	return results, nil
}

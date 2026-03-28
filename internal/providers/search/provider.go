package search

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/oluoyefeso/termiflow/internal/config"
)

type SearchResult struct {
	Title       string
	URL         string
	Snippet     string
	Content     string
	PublishedAt time.Time
	Source      string
}

type SearchRequest struct {
	Query      string
	MaxResults int
	TimeRange  string // "day", "week", "month", "year"
}

type Provider interface {
	Name() string
	Search(ctx context.Context, req SearchRequest) ([]SearchResult, error)
	Available() bool
}

// GetSearchProvider returns the appropriate search provider based on config and environment.
func GetSearchProvider(cfg *config.Config) (Provider, error) {
	if os.Getenv("TERMIFLOW_MOCK") == "true" {
		p := NewMockSearchProvider()
		if p == nil {
			return nil, fmt.Errorf("TERMIFLOW_MOCK=true requires building with -tags mock")
		}
		return p, nil
	}
	if config.IsManagedMode() {
		return NewManagedSearchProvider(
			cfg.Providers.Managed.APIKey,
			cfg.Providers.Managed.BaseURL,
		), nil
	}
	if cfg.Search.Tavily.APIKey != "" {
		return NewTavilyProvider(cfg.Search.Tavily.APIKey), nil
	}
	return NewRSSProvider(), nil
}

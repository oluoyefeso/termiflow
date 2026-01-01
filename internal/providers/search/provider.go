package search

import (
	"context"
	"time"
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

package search

import (
	"context"
	"time"
)

// FeedFetcher fetches and parses an RSS/Atom feed URL.
// Implemented by RSSProvider (self-hosted, direct fetch) and
// ManagedFeedFetcher (managed, via POST /v1/sources/fetch).
type FeedFetcher interface {
	FetchFeed(ctx context.Context, feedURL string, since *time.Time) ([]SearchResult, error)
}

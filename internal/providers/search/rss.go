package search

import (
	"context"
	"time"

	"github.com/mmcdole/gofeed"
)

type RSSProvider struct {
	parser *gofeed.Parser
}

func NewRSSProvider() *RSSProvider {
	return &RSSProvider{
		parser: gofeed.NewParser(),
	}
}

func (p *RSSProvider) Name() string {
	return "rss"
}

func (p *RSSProvider) Available() bool {
	return true
}

func (p *RSSProvider) Search(ctx context.Context, req SearchRequest) ([]SearchResult, error) {
	// RSS provider doesn't support search - use FetchFeed instead
	return nil, nil
}

func (p *RSSProvider) FetchFeed(ctx context.Context, feedURL string, since *time.Time) ([]SearchResult, error) {
	feed, err := p.parser.ParseURLWithContext(feedURL, ctx)
	if err != nil {
		return nil, err
	}

	var results []SearchResult
	for _, item := range feed.Items {
		// Skip items older than 'since'
		if since != nil && item.PublishedParsed != nil && item.PublishedParsed.Before(*since) {
			continue
		}

		var publishedAt time.Time
		if item.PublishedParsed != nil {
			publishedAt = *item.PublishedParsed
		}

		result := SearchResult{
			Title:       item.Title,
			URL:         item.Link,
			Snippet:     item.Description,
			Content:     item.Content,
			PublishedAt: publishedAt,
			Source:      feed.Title,
		}

		results = append(results, result)
	}

	return results, nil
}

func (p *RSSProvider) FetchMultipleFeeds(ctx context.Context, feedURLs []string, since *time.Time) ([]SearchResult, error) {
	var allResults []SearchResult

	for _, url := range feedURLs {
		results, err := p.FetchFeed(ctx, url, since)
		if err != nil {
			// Log error but continue with other feeds
			continue
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

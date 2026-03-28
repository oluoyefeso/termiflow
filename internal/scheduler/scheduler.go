package scheduler

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	engine "github.com/oluoyefeso/termiflow-engine"
	"github.com/oluoyefeso/termiflow/internal/config"
	"github.com/oluoyefeso/termiflow/internal/db"
	"github.com/oluoyefeso/termiflow/internal/providers/llm"
	"github.com/oluoyefeso/termiflow/internal/providers/search"
	"github.com/oluoyefeso/termiflow/pkg/models"
)

type Scheduler struct {
	llmProvider    llm.Provider
	searchProvider search.Provider
	rssProvider    *search.RSSProvider
	scraper        *search.Scraper
	curator        *engine.Curator
}

// NewFromConfig creates a Scheduler with the appropriate LLM and search providers
// resolved from the config. Used by both CLI and TUI.
func NewFromConfig(cfg *config.Config, providerName string) (*Scheduler, error) {
	llmProvider, err := llm.GetProvider(providerName, cfg)
	if err != nil {
		return nil, err
	}

	searchProvider, err := search.GetSearchProvider(cfg)
	if err != nil {
		return nil, err
	}

	return New(llmProvider, searchProvider), nil
}

func New(llmProvider llm.Provider, searchProvider search.Provider) *Scheduler {
	return &Scheduler{
		llmProvider:    llmProvider,
		searchProvider: searchProvider,
		rssProvider:    search.NewRSSProvider(),
		scraper:        search.NewScraper("", 0),
		curator:        engine.NewCurator(llm.AsEngine(llmProvider)),
	}
}

// RefreshSubscription fetches and processes new items for a subscription.
// Branches on source_type: "feed" and "scrape" use direct fetch, "topic" uses search.
func (s *Scheduler) RefreshSubscription(ctx context.Context, sub *models.Subscription) ([]*models.FeedItem, error) {
	var allResults []search.SearchResult
	scoringTopic := sub.ScoringTopic()

	switch sub.SourceType {
	case "feed":
		// Source subscription: fetch RSS feed directly
		results, err := s.rssProvider.FetchFeed(ctx, sub.SourceURL, sub.LastFetchedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch feed %s: %w", sub.SourceURL, err)
		}
		// Strip HTML from RSS content before LLM processing
		for i := range results {
			results[i].Content = stripHTML(results[i].Content)
			results[i].Snippet = stripHTML(results[i].Snippet)
			if runes := []rune(results[i].Content); len(runes) > 2000 {
				results[i].Content = string(runes[:2000])
			}
		}
		allResults = results

	case "scrape":
		// Scrape subscription: fetch page content
		result, err := s.scraper.Scrape(ctx, sub.SourceURL)
		if err != nil {
			return nil, fmt.Errorf("failed to scrape %s: %w", sub.SourceURL, err)
		}
		if result != nil {
			allResults = []search.SearchResult{*result}
		}

	default:
		// Topic subscription: existing flow (search + category RSS)
		if s.searchProvider != nil && s.searchProvider.Available() {
			results, err := s.searchProvider.Search(ctx, search.SearchRequest{
				Query:      sub.Topic,
				MaxResults: 10,
				TimeRange:  sub.GetTimeRange(),
			})
			if err == nil {
				allResults = append(allResults, results...)
			}
		}

		// Fetch from RSS feeds if this is a category with default RSS
		category := models.GetCategoryByName(sub.Topic)
		if category != nil && len(category.DefaultRSS) > 0 {
			results, err := s.rssProvider.FetchMultipleFeeds(ctx, category.DefaultRSS, sub.LastFetchedAt)
			if err == nil {
				allResults = append(allResults, results...)
			}
		}
	}

	// Deduplicate by URL
	allResults = deduplicateByURL(allResults)

	if len(allResults) == 0 {
		return nil, nil
	}

	// Convert CLI search results to engine search results and curate
	engineResults := search.ToEngineResults(allResults)
	curatedItems, err := s.curator.Curate(ctx, scoringTopic, engineResults)
	if err != nil {
		return nil, err
	}

	// Convert engine FeedItems to CLI FeedItems and save to database
	items := db.FromEngineFeedItems(curatedItems)

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var newItems []*models.FeedItem
	for _, item := range items {
		item.SubscriptionID = sub.ID

		// INSERT OR IGNORE handles duplicates atomically via unique index on (subscription_id, source_url)
		if err := db.CreateFeedItemTx(tx, item); err != nil {
			continue
		}
		if item.ID > 0 {
			newItems = append(newItems, item)
		}
	}

	// Update last fetched time
	if err := db.UpdateLastFetchedTx(tx, sub.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit refresh transaction: %w", err)
	}

	return newItems, nil
}

// RefreshAllSubscriptions refreshes all active subscriptions
func (s *Scheduler) RefreshAllSubscriptions(ctx context.Context) error {
	subs, err := db.GetActiveSubscriptions()
	if err != nil {
		return err
	}

	for _, sub := range subs {
		if shouldRefresh(sub) {
			_, err := s.RefreshSubscription(ctx, sub)
			if err != nil {
				// Log error but continue with other subscriptions
				continue
			}
		}
	}

	return nil
}

// RefreshAllSubscriptionsConcurrent refreshes all due subscriptions concurrently,
// bounded by maxConcurrent goroutines. The onResult callback fires once per subscription
// with the topic name, new item count, and any error (thread-safe for bubbletea p.Send).
func (s *Scheduler) RefreshAllSubscriptionsConcurrent(ctx context.Context, maxConcurrent int, onResult func(topic string, newItems int, err error)) error {
	subs, err := db.GetActiveSubscriptions()
	if err != nil {
		return err
	}

	var toRefresh []*models.Subscription
	for _, sub := range subs {
		if shouldRefresh(sub) {
			toRefresh = append(toRefresh, sub)
		}
	}

	if len(toRefresh) == 0 {
		return nil
	}

	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}

	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, sub := range toRefresh {
		wg.Add(1)
		sem <- struct{}{}

		go func(sub *models.Subscription) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					if onResult != nil {
						onResult(sub.Topic, 0, fmt.Errorf("panic during refresh: %v", r))
					}
				}
			}()

			newItems, err := s.RefreshSubscription(ctx, sub)
			if onResult != nil {
				onResult(sub.Topic, len(newItems), err)
			}
		}(sub)
	}

	wg.Wait()
	return nil
}

func shouldRefresh(sub *models.Subscription) bool {
	if sub.LastFetchedAt == nil {
		return true
	}

	now := time.Now()
	switch sub.Frequency {
	case "hourly":
		return now.Sub(*sub.LastFetchedAt) >= time.Hour
	case "daily":
		return now.Sub(*sub.LastFetchedAt) >= 24*time.Hour
	case "weekly":
		return now.Sub(*sub.LastFetchedAt) >= 7*24*time.Hour
	default:
		return now.Sub(*sub.LastFetchedAt) >= 24*time.Hour
	}
}

// htmlTagRe matches HTML tags for stripping.
var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// stripHTML removes HTML tags and collapses whitespace.
func stripHTML(s string) string {
	s = htmlTagRe.ReplaceAllString(s, " ")
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}

func deduplicateByURL(results []search.SearchResult) []search.SearchResult {
	seen := make(map[string]bool)
	var unique []search.SearchResult

	for _, r := range results {
		if !seen[r.URL] {
			seen[r.URL] = true
			unique = append(unique, r)
		}
	}

	return unique
}

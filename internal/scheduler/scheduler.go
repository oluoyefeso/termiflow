package scheduler

import (
	"context"
	"time"

	"github.com/termiflow/termiflow/internal/db"
	"github.com/termiflow/termiflow/internal/intelligence"
	"github.com/termiflow/termiflow/internal/providers/llm"
	"github.com/termiflow/termiflow/internal/providers/search"
	"github.com/termiflow/termiflow/pkg/models"
)

type Scheduler struct {
	llmProvider    llm.Provider
	searchProvider search.Provider
	rssProvider    *search.RSSProvider
	curator        *intelligence.Curator
}

func New(llmProvider llm.Provider, searchProvider search.Provider) *Scheduler {
	return &Scheduler{
		llmProvider:    llmProvider,
		searchProvider: searchProvider,
		rssProvider:    search.NewRSSProvider(),
		curator:        intelligence.NewCurator(llmProvider),
	}
}

// RefreshSubscription fetches and processes new items for a subscription
func (s *Scheduler) RefreshSubscription(ctx context.Context, sub *models.Subscription) ([]*models.FeedItem, error) {
	var allResults []search.SearchResult

	// Fetch from search provider (Tavily)
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

	// Deduplicate by URL
	allResults = deduplicateByURL(allResults)

	// Curate results
	items, err := s.curator.CurateResults(ctx, sub.Topic, allResults)
	if err != nil {
		return nil, err
	}

	// Set subscription ID and save to database
	for _, item := range items {
		item.SubscriptionID = sub.ID

		// Check if item already exists
		exists, _ := db.ItemExistsByURL(item.SourceURL)
		if !exists {
			if err := db.CreateFeedItem(item); err != nil {
				// Log but continue
				continue
			}
		}
	}

	// Update last fetched time
	if err := db.UpdateLastFetched(sub.ID); err != nil {
		return nil, err
	}

	return items, nil
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

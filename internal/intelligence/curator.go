package intelligence

import (
	"context"
	"sort"
	"sync"

	"github.com/oluoyefeso/termiflow/internal/providers/llm"
	"github.com/oluoyefeso/termiflow/internal/providers/search"
	"github.com/oluoyefeso/termiflow/pkg/models"
)

const maxConcurrentArticles = 5

type Curator struct {
	llmProvider llm.Provider
}

func NewCurator(provider llm.Provider) *Curator {
	return &Curator{
		llmProvider: provider,
	}
}

// CurateResults processes search results concurrently and returns curated feed items.
// Uses bounded goroutines (max 5) to parallelize LLM calls per article.
func (c *Curator) CurateResults(ctx context.Context, topic string, results []search.SearchResult) ([]*models.FeedItem, error) {
	type indexedItem struct {
		index int
		item  *models.FeedItem
	}

	var (
		mu      sync.Mutex
		items   = make([]*models.FeedItem, len(results))
		wg      sync.WaitGroup
		sem     = make(chan struct{}, maxConcurrentArticles)
	)

	for i, result := range results {
		wg.Add(1)
		sem <- struct{}{} // acquire semaphore

		go func(idx int, r search.SearchResult) {
			defer wg.Done()
			defer func() { <-sem }() // release semaphore

			item := &models.FeedItem{
				Title:       r.Title,
				SourceName:  r.Source,
				SourceURL:   r.URL,
				Content:     truncateContent(r.Content, 2000),
				PublishedAt: &r.PublishedAt,
			}

			// Score relevance
			score, err := ScoreRelevance(ctx, c.llmProvider, topic, r.Title, r.Snippet)
			if err != nil {
				score = 0.5
			}
			item.RelevanceScore = score

			// Only process items above threshold
			if score > 0.5 {
				summary, err := Summarize(ctx, c.llmProvider, topic, r.Title, r.Content)
				if err == nil {
					item.Summary = summary
				}

				tags, err := ExtractTags(ctx, c.llmProvider, r.Title, r.Content)
				if err == nil {
					item.Tags = tags
				}
			}

			mu.Lock()
			items[idx] = item
			mu.Unlock()
		}(i, result)
	}

	wg.Wait()

	// Collect non-nil items (all should be non-nil, but be safe)
	var collected []*models.FeedItem
	for _, item := range items {
		if item != nil {
			collected = append(collected, item)
		}
	}

	// Filter and sort by relevance
	collected = filterByRelevance(collected, 0.5)
	sortByRelevanceAndRecency(collected)

	return collected, nil
}

func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen]
}

func filterByRelevance(items []*models.FeedItem, threshold float64) []*models.FeedItem {
	var filtered []*models.FeedItem
	for _, item := range items {
		if item.RelevanceScore >= threshold {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func sortByRelevanceAndRecency(items []*models.FeedItem) {
	sort.Slice(items, func(i, j int) bool {
		// Combine relevance (70%) and recency (30%)
		scoreI := items[i].RelevanceScore * 0.7
		scoreJ := items[j].RelevanceScore * 0.7

		if items[i].PublishedAt != nil && items[j].PublishedAt != nil {
			// More recent items get higher recency score
			if items[i].PublishedAt.After(*items[j].PublishedAt) {
				scoreI += 0.3
			} else {
				scoreJ += 0.3
			}
		}

		return scoreI > scoreJ
	})
}

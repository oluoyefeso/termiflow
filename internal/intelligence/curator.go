package intelligence

import (
	"context"
	"sort"

	"github.com/termiflow/termiflow/internal/providers/llm"
	"github.com/termiflow/termiflow/internal/providers/search"
	"github.com/termiflow/termiflow/pkg/models"
)

type Curator struct {
	llmProvider llm.Provider
}

func NewCurator(provider llm.Provider) *Curator {
	return &Curator{
		llmProvider: provider,
	}
}

// CurateResults processes search results and returns curated feed items
func (c *Curator) CurateResults(ctx context.Context, topic string, results []search.SearchResult) ([]*models.FeedItem, error) {
	var items []*models.FeedItem

	for _, result := range results {
		item := &models.FeedItem{
			Title:       result.Title,
			SourceName:  result.Source,
			SourceURL:   result.URL,
			Content:     truncateContent(result.Content, 2000),
			PublishedAt: &result.PublishedAt,
		}

		// Score relevance
		score, err := ScoreRelevance(ctx, c.llmProvider, topic, result.Title, result.Snippet)
		if err != nil {
			score = 0.5 // Default score on error
		}
		item.RelevanceScore = score

		// Only process items above threshold
		if score > 0.5 {
			// Generate summary
			summary, err := Summarize(ctx, c.llmProvider, topic, result.Title, result.Content)
			if err == nil {
				item.Summary = summary
			}

			// Extract tags
			tags, err := ExtractTags(ctx, c.llmProvider, result.Title, result.Content)
			if err == nil {
				item.Tags = tags
			}
		}

		items = append(items, item)
	}

	// Filter and sort by relevance
	items = filterByRelevance(items, 0.5)
	sortByRelevanceAndRecency(items)

	return items, nil
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

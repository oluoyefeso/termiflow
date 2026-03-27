package db

import (
	"time"

	engine "github.com/oluoyefeso/termiflow-engine"
	"github.com/oluoyefeso/termiflow/pkg/models"
)

// FromEngineFeedItems converts engine FeedItems to CLI FeedItems for database storage.
// DB-specific fields (ID, SubscriptionID, IsRead, FetchedAt) are set by the caller.
func FromEngineFeedItems(items []*engine.FeedItem) []*models.FeedItem {
	out := make([]*models.FeedItem, len(items))
	for i, item := range items {
		out[i] = FromEngineFeedItem(item)
	}
	return out
}

// FromEngineFeedItem converts a single engine FeedItem to a CLI FeedItem.
func FromEngineFeedItem(item *engine.FeedItem) *models.FeedItem {
	return &models.FeedItem{
		Title:          item.Title,
		Summary:        item.Summary,
		Content:        item.Content,
		SourceName:     item.SourceName,
		SourceURL:      item.SourceURL,
		PublishedAt:    item.PublishedAt,
		FetchedAt:      time.Now(),
		RelevanceScore: item.RelevanceScore,
		Tags:           item.Tags,
	}
}

// ToEngineFeedItem converts a CLI FeedItem to an engine FeedItem.
// DB-specific fields are dropped.
func ToEngineFeedItem(item *models.FeedItem) *engine.FeedItem {
	return &engine.FeedItem{
		Title:          item.Title,
		Summary:        item.Summary,
		Content:        item.Content,
		SourceName:     item.SourceName,
		SourceURL:      item.SourceURL,
		PublishedAt:    item.PublishedAt,
		RelevanceScore: item.RelevanceScore,
		Tags:           item.Tags,
	}
}

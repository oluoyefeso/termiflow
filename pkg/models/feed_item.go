package models

import (
	"encoding/json"
	"fmt"
	"time"
)

type FeedItem struct {
	ID             int64      `json:"id"`
	SubscriptionID int64      `json:"subscription_id"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary,omitempty"`
	Content        string     `json:"content,omitempty"`
	SourceName     string     `json:"source_name,omitempty"`
	SourceURL      string     `json:"source_url,omitempty"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
	FetchedAt      time.Time  `json:"fetched_at"`
	IsRead         bool       `json:"is_read"`
	RelevanceScore float64    `json:"relevance_score,omitempty"`
	Tags           []string   `json:"tags,omitempty"`
}

func (f *FeedItem) GetTagsJSON() string {
	if len(f.Tags) == 0 {
		return "[]"
	}
	data, _ := json.Marshal(f.Tags)
	return string(data)
}

func (f *FeedItem) SetTagsFromJSON(data string) error {
	if data == "" || data == "null" {
		f.Tags = nil
		return nil
	}
	return json.Unmarshal([]byte(data), &f.Tags)
}

func (f *FeedItem) TimeAgo() string {
	if f.PublishedAt == nil {
		return "unknown"
	}

	duration := time.Since(*f.PublishedAt)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1m ago"
		}
		return formatDuration(mins, "m")
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return formatDuration(hours, "h")
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return formatDuration(days, "d")
	default:
		weeks := int(duration.Hours() / 24 / 7)
		if weeks == 1 {
			return "1w ago"
		}
		return formatDuration(weeks, "w")
	}
}

func formatDuration(value int, unit string) string {
	return fmt.Sprintf("%d%s ago", value, unit)
}

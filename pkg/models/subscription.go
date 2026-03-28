package models

import (
	"encoding/json"
	"time"
)

type Subscription struct {
	ID            int64      `json:"id"`
	Topic         string     `json:"topic"`
	Category      string     `json:"category,omitempty"`
	Frequency     string     `json:"frequency"`
	Sources       []string   `json:"sources,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LastFetchedAt *time.Time `json:"last_fetched_at,omitempty"`
	IsActive      bool       `json:"is_active"`
	SourceURL     string     `json:"source_url,omitempty"`
	SourceType    string     `json:"source_type,omitempty"` // "topic", "feed", "scrape"
	DisplayName   string     `json:"display_name,omitempty"`
	Context       string     `json:"context,omitempty"`
}

// IsSourceSubscription returns true if this is a feed or scrape subscription (not topic-based).
func (s *Subscription) IsSourceSubscription() bool {
	return s.SourceType == "feed" || s.SourceType == "scrape"
}

// ScoringTopic returns the topic string used for LLM relevance scoring.
// For source subscriptions, uses Context (if provided) or DisplayName.
// For topic subscriptions, uses Topic.
func (s *Subscription) ScoringTopic() string {
	if s.IsSourceSubscription() {
		if s.Context != "" {
			return s.Context
		}
		if s.DisplayName != "" {
			return s.DisplayName
		}
	}
	return s.Topic
}

func (s *Subscription) GetSourcesJSON() string {
	if len(s.Sources) == 0 {
		return "[]"
	}
	data, _ := json.Marshal(s.Sources)
	return string(data)
}

func (s *Subscription) SetSourcesFromJSON(data string) error {
	if data == "" || data == "null" {
		s.Sources = nil
		return nil
	}
	return json.Unmarshal([]byte(data), &s.Sources)
}

func (s *Subscription) GetTimeRange() string {
	switch s.Frequency {
	case "hourly":
		return "day"
	case "daily":
		return "week"
	case "weekly":
		return "month"
	default:
		return "week"
	}
}

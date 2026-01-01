package models

import (
	"testing"
	"time"
)

func TestFeedItem_GetTagsJSON(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		expected string
	}{
		{
			name:     "empty tags",
			tags:     nil,
			expected: "[]",
		},
		{
			name:     "empty slice",
			tags:     []string{},
			expected: "[]",
		},
		{
			name:     "single tag",
			tags:     []string{"rust"},
			expected: `["rust"]`,
		},
		{
			name:     "multiple tags",
			tags:     []string{"rust", "async", "tokio"},
			expected: `["rust","async","tokio"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &FeedItem{Tags: tt.tags}
			result := item.GetTagsJSON()
			if result != tt.expected {
				t.Errorf("GetTagsJSON() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFeedItem_SetTagsFromJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
		wantErr  bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "null string",
			input:    "null",
			expected: nil,
			wantErr:  false,
		},
		{
			name:     "empty array",
			input:    "[]",
			expected: []string{},
			wantErr:  false,
		},
		{
			name:     "single tag",
			input:    `["rust"]`,
			expected: []string{"rust"},
			wantErr:  false,
		},
		{
			name:     "multiple tags",
			input:    `["rust", "async", "tokio"]`,
			expected: []string{"rust", "async", "tokio"},
			wantErr:  false,
		},
		{
			name:     "invalid JSON",
			input:    "not valid json",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &FeedItem{}
			err := item.SetTagsFromJSON(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("SetTagsFromJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(item.Tags) != len(tt.expected) {
					t.Errorf("SetTagsFromJSON() tags length = %d, want %d", len(item.Tags), len(tt.expected))
					return
				}
				for i, tag := range item.Tags {
					if tag != tt.expected[i] {
						t.Errorf("SetTagsFromJSON() tags[%d] = %q, want %q", i, tag, tt.expected[i])
					}
				}
			}
		})
	}
}

func TestFeedItem_TimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		publishedAt *time.Time
		expected    string
	}{
		{
			name:        "nil time",
			publishedAt: nil,
			expected:    "unknown",
		},
		{
			name:        "just now",
			publishedAt: timePtr(now.Add(-30 * time.Second)),
			expected:    "just now",
		},
		{
			name:        "1 minute ago",
			publishedAt: timePtr(now.Add(-1 * time.Minute)),
			expected:    "1m ago",
		},
		{
			name:        "5 minutes ago",
			publishedAt: timePtr(now.Add(-5 * time.Minute)),
			expected:    "5m ago",
		},
		{
			name:        "1 hour ago",
			publishedAt: timePtr(now.Add(-1 * time.Hour)),
			expected:    "1h ago",
		},
		{
			name:        "3 hours ago",
			publishedAt: timePtr(now.Add(-3 * time.Hour)),
			expected:    "3h ago",
		},
		{
			name:        "1 day ago",
			publishedAt: timePtr(now.Add(-24 * time.Hour)),
			expected:    "1d ago",
		},
		{
			name:        "3 days ago",
			publishedAt: timePtr(now.Add(-3 * 24 * time.Hour)),
			expected:    "3d ago",
		},
		{
			name:        "1 week ago",
			publishedAt: timePtr(now.Add(-7 * 24 * time.Hour)),
			expected:    "1w ago",
		},
		{
			name:        "3 weeks ago",
			publishedAt: timePtr(now.Add(-21 * 24 * time.Hour)),
			expected:    "3w ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &FeedItem{PublishedAt: tt.publishedAt}
			result := item.TimeAgo()
			if result != tt.expected {
				t.Errorf("TimeAgo() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFeedItem_TagsRoundTrip(t *testing.T) {
	original := []string{"rust", "async", "tokio"}
	item := &FeedItem{}

	// Set tags
	item.Tags = original
	json := item.GetTagsJSON()

	// Reset and restore
	item.Tags = nil
	err := item.SetTagsFromJSON(json)
	if err != nil {
		t.Fatalf("SetTagsFromJSON() error = %v", err)
	}

	// Verify
	if len(item.Tags) != len(original) {
		t.Errorf("Round trip failed: got %d tags, want %d", len(item.Tags), len(original))
	}
	for i, tag := range item.Tags {
		if tag != original[i] {
			t.Errorf("Round trip tags[%d] = %q, want %q", i, tag, original[i])
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		value    int
		unit     string
		expected string
	}{
		{5, "m", "5m ago"},
		{3, "h", "3h ago"},
		{2, "d", "2d ago"},
		{4, "w", "4w ago"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDuration(tt.value, tt.unit)
			if result != tt.expected {
				t.Errorf("formatDuration(%d, %q) = %q, want %q", tt.value, tt.unit, result, tt.expected)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

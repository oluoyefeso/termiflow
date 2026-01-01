package models

import (
	"testing"
)

func TestSubscription_GetSourcesJSON(t *testing.T) {
	tests := []struct {
		name     string
		sources  []string
		expected string
	}{
		{
			name:     "empty sources",
			sources:  nil,
			expected: "[]",
		},
		{
			name:     "empty slice",
			sources:  []string{},
			expected: "[]",
		},
		{
			name:     "single source",
			sources:  []string{"tavily"},
			expected: `["tavily"]`,
		},
		{
			name:     "multiple sources",
			sources:  []string{"tavily", "rss", "scrape"},
			expected: `["tavily","rss","scrape"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := &Subscription{Sources: tt.sources}
			result := sub.GetSourcesJSON()
			if result != tt.expected {
				t.Errorf("GetSourcesJSON() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSubscription_SetSourcesFromJSON(t *testing.T) {
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
			name:     "single source",
			input:    `["tavily"]`,
			expected: []string{"tavily"},
			wantErr:  false,
		},
		{
			name:     "multiple sources",
			input:    `["tavily", "rss", "scrape"]`,
			expected: []string{"tavily", "rss", "scrape"},
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
			sub := &Subscription{}
			err := sub.SetSourcesFromJSON(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("SetSourcesFromJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(sub.Sources) != len(tt.expected) {
					t.Errorf("SetSourcesFromJSON() sources length = %d, want %d", len(sub.Sources), len(tt.expected))
					return
				}
				for i, s := range sub.Sources {
					if s != tt.expected[i] {
						t.Errorf("SetSourcesFromJSON() sources[%d] = %q, want %q", i, s, tt.expected[i])
					}
				}
			}
		})
	}
}

func TestSubscription_GetTimeRange(t *testing.T) {
	tests := []struct {
		frequency string
		expected  string
	}{
		{"hourly", "day"},
		{"daily", "week"},
		{"weekly", "month"},
		{"unknown", "week"}, // default case
		{"", "week"},        // empty string
	}

	for _, tt := range tests {
		t.Run(tt.frequency, func(t *testing.T) {
			sub := &Subscription{Frequency: tt.frequency}
			result := sub.GetTimeRange()
			if result != tt.expected {
				t.Errorf("GetTimeRange() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSubscription_RoundTrip(t *testing.T) {
	original := []string{"tavily", "rss", "scrape"}
	sub := &Subscription{}

	// Set sources
	sub.Sources = original
	json := sub.GetSourcesJSON()

	// Reset and restore
	sub.Sources = nil
	err := sub.SetSourcesFromJSON(json)
	if err != nil {
		t.Fatalf("SetSourcesFromJSON() error = %v", err)
	}

	// Verify
	if len(sub.Sources) != len(original) {
		t.Errorf("Round trip failed: got %d sources, want %d", len(sub.Sources), len(original))
	}
	for i, s := range sub.Sources {
		if s != original[i] {
			t.Errorf("Round trip sources[%d] = %q, want %q", i, s, original[i])
		}
	}
}

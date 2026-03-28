package db

import (
	"fmt"
	"strings"

	"github.com/oluoyefeso/termiflow/internal/config"
)

// BuildUserContext returns a string describing the user's termiflow state
// (subscriptions, unread counts, mode) for injection into LLM system prompts.
// Returns empty string if no active subscriptions or on error.
func BuildUserContext() string {
	subs, err := GetActiveSubscriptions()
	if err != nil || len(subs) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("The user is running termiflow, a terminal news tool. Here is their current state:\n")

	if config.IsManagedMode() {
		sb.WriteString("Mode: Managed (using termiflow API)\n")
	} else {
		cfg := config.Get()
		fmt.Fprintf(&sb, "Mode: Self-hosted (provider: %s)\n", cfg.General.DefaultProvider)
	}

	fmt.Fprintf(&sb, "Subscriptions: %d active\n", len(subs))
	for _, sub := range subs {
		total, unread, _ := GetSubscriptionItemCount(sub.ID)
		lastFetch := "never"
		if sub.LastFetchedAt != nil {
			lastFetch = sub.LastFetchedAt.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(&sb, "- %s (%s, %d items, %d unread, last fetched: %s)\n",
			sub.Topic, sub.Frequency, total, unread, lastFetch)
	}
	return sb.String()
}

package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/termiflow/termiflow/internal/config"
	"github.com/termiflow/termiflow/internal/db"
	"github.com/termiflow/termiflow/internal/providers/llm"
	"github.com/termiflow/termiflow/internal/providers/search"
	"github.com/termiflow/termiflow/internal/scheduler"
	"github.com/termiflow/termiflow/internal/ui"
	"github.com/termiflow/termiflow/pkg/models"
)

var feedTopic string
var feedToday bool
var feedWeek bool
var feedLimit int
var feedRefresh bool
var feedAll bool
var feedMarkRead bool
var feedCleanup bool

var feedCmd = &cobra.Command{
	Use:   "feed",
	Short: "Display curated feed items from subscriptions",
	Long: `Display curated feed items from subscriptions.

Examples:
  termiflow feed                           # Show all unread items
  termiflow feed --topic silicon-chips     # Filter by topic
  termiflow feed --today                   # Today's items only
  termiflow feed --limit 10                # Limit number of items
  termiflow feed --refresh                 # Fetch new items first`,
	RunE: runFeed,
}

func init() {
	feedCmd.Flags().StringVar(&feedTopic, "topic", "", "filter by subscription topic")
	feedCmd.Flags().BoolVar(&feedToday, "today", false, "show only today's items")
	feedCmd.Flags().BoolVar(&feedWeek, "week", false, "show items from the past week")
	feedCmd.Flags().IntVar(&feedLimit, "limit", 0, "maximum items to display")
	feedCmd.Flags().BoolVar(&feedRefresh, "refresh", false, "fetch fresh items before displaying")
	feedCmd.Flags().BoolVar(&feedAll, "all", false, "include already-read items")
	feedCmd.Flags().BoolVar(&feedMarkRead, "mark-read", true, "mark displayed items as read")
	feedCmd.Flags().BoolVar(&feedCleanup, "cleanup", false, "remove items older than 30 days")
}

func runFeed(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	// Handle cleanup
	if feedCleanup {
		return cleanupOldItems()
	}

	// Handle refresh
	if feedRefresh {
		if err := refreshFeeds(cfg, feedTopic); err != nil {
			fmt.Print(ui.Error(fmt.Sprintf("Refresh failed: %v", err)))
			fmt.Println()
			// Continue to show existing items
		}
	}

	// Build filter
	filter := db.FeedItemFilter{
		Unread: !feedAll,
	}

	if feedTopic != "" {
		filter.Topic = feedTopic
	}

	if feedToday {
		since := time.Now().Truncate(24 * time.Hour)
		filter.Since = &since
	} else if feedWeek {
		since := time.Now().AddDate(0, 0, -7)
		filter.Since = &since
	}

	if feedLimit > 0 {
		filter.Limit = feedLimit
	} else {
		filter.Limit = cfg.General.FeedLimit
	}

	// Get subscriptions
	subs, err := db.GetActiveSubscriptions()
	if err != nil {
		return err
	}

	if len(subs) == 0 {
		fmt.Println()
		fmt.Print(ui.Warning("No active subscriptions"))
		fmt.Println()
		fmt.Printf("   Get started with:\n")
		fmt.Printf("     %s\n", ui.TitleStyle.Render("termiflow subscribe silicon-chips"))
		fmt.Printf("     %s\n", ui.TitleStyle.Render("termiflow subscribe \"your custom topic\""))
		fmt.Println()
		fmt.Printf("   See available topics with %s\n", ui.TitleStyle.Render("termiflow topics --available"))
		fmt.Println()
		return nil
	}

	// Get feed items
	items, err := db.GetFeedItems(filter)
	if err != nil {
		return err
	}

	// Print header
	fmt.Println(ui.HeaderWithDate("termiflow feed"))

	if len(items) == 0 {
		fmt.Println()
		fmt.Print(ui.MutedStyle.Render("   No new items in your feed.\n"))
		fmt.Println()
		fmt.Printf("   Run %s to fetch updates.\n", ui.TitleStyle.Render("termiflow feed --refresh"))
		fmt.Println()
		return nil
	}

	// Group items by subscription
	groupedItems := groupBySubscription(items, subs)

	// Track items to mark as read
	var itemIDs []int64
	totalItems := 0
	topicCount := 0

	for _, sub := range subs {
		subItems, ok := groupedItems[sub.ID]
		if !ok || len(subItems) == 0 {
			continue
		}

		topicCount++
		fmt.Print(ui.Section(sub.Topic, len(subItems), "new items"))

		for i, item := range subItems {
			fmt.Println(ui.FormatFeedItem(
				item.Title,
				item.SourceName,
				item.TimeAgo(),
				item.Summary,
				item.Tags,
			))

			if i < len(subItems)-1 {
				fmt.Print(ui.Divider())
			}

			itemIDs = append(itemIDs, item.ID)
			totalItems++
		}
	}

	// Print footer
	fmt.Print(ui.Footer(totalItems, topicCount, "just now"))

	// Mark items as read
	if feedMarkRead && len(itemIDs) > 0 {
		if err := db.MarkItemsRead(itemIDs); err != nil {
			// Log error but don't fail
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to mark items as read: %v\n", err)
		}
	}

	return nil
}

func groupBySubscription(items []*models.FeedItem, subs []*models.Subscription) map[int64][]*models.FeedItem {
	result := make(map[int64][]*models.FeedItem)
	for _, item := range items {
		result[item.SubscriptionID] = append(result[item.SubscriptionID], item)
	}
	return result
}

func cleanupOldItems() error {
	olderThan := time.Now().AddDate(0, 0, -30)
	count, err := db.DeleteOldItems(olderThan)
	if err != nil {
		return fmt.Errorf("failed to cleanup: %w", err)
	}

	if count == 0 {
		fmt.Print(ui.Success("No old items to remove"))
	} else {
		fmt.Print(ui.Success(fmt.Sprintf("Removed %d items older than 30 days", count)))
	}
	return nil
}

func refreshFeeds(cfg *config.Config, topicFilter string) error {
	// Get subscriptions to refresh
	subs, err := db.GetActiveSubscriptions()
	if err != nil {
		return err
	}

	if len(subs) == 0 {
		return nil
	}

	// Filter by topic if specified
	if topicFilter != "" {
		var filtered []*models.Subscription
		for _, sub := range subs {
			if sub.Topic == topicFilter {
				filtered = append(filtered, sub)
			}
		}
		subs = filtered
		if len(subs) == 0 {
			return fmt.Errorf("no subscription found for topic: %s", topicFilter)
		}
	}

	// Initialize LLM provider
	providerName := getProvider()
	llmProvider, err := llm.GetProvider(providerName, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM provider: %w", err)
	}

	if !llmProvider.Available() {
		return fmt.Errorf("LLM provider '%s' not configured - run 'termiflow config init'", providerName)
	}

	// Initialize search provider (Tavily)
	var searchProvider search.Provider
	if cfg.Search.Tavily.APIKey != "" {
		searchProvider = search.NewTavilyProvider(cfg.Search.Tavily.APIKey)
	}

	// Create scheduler
	sched := scheduler.New(llmProvider, searchProvider)

	// Show spinner
	sp := ui.NewSpinner(fmt.Sprintf("Fetching updates for %d subscription(s)...", len(subs)))
	sp.Start()

	ctx := context.Background()
	totalNewItems := 0

	for _, sub := range subs {
		items, err := sched.RefreshSubscription(ctx, sub)
		if err != nil {
			// Log error but continue with other subscriptions
			continue
		}
		totalNewItems += len(items)
	}

	if totalNewItems > 0 {
		sp.Success(fmt.Sprintf("Fetched %d new item(s)", totalNewItems))
	} else {
		sp.Success("No new items found")
	}

	return nil
}

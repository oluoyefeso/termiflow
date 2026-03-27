package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"

	"github.com/oluoyefeso/termiflow/internal/config"
	"github.com/oluoyefeso/termiflow/internal/db"
	"github.com/oluoyefeso/termiflow/internal/providers"
	"github.com/oluoyefeso/termiflow/internal/providers/llm"
	"github.com/oluoyefeso/termiflow/internal/providers/search"
	"github.com/oluoyefeso/termiflow/internal/scheduler"
	"github.com/oluoyefeso/termiflow/internal/ui"
	"github.com/oluoyefeso/termiflow/pkg/models"
)

var feedTopic string
var feedToday bool
var feedWeek bool
var feedLimit int
var feedRefresh bool
var feedAll bool
var feedMarkRead bool
var feedCleanup bool
var feedWatch bool
var feedInterval time.Duration

var feedCmd = &cobra.Command{
	Use:   "feed",
	Short: "Display curated feed items from subscriptions",
	Long: `Display curated feed items from subscriptions.

Examples:
  termiflow feed                           # Show all unread items
  termiflow feed --topic silicon-chips     # Filter by topic
  termiflow feed --today                   # Today's items only
  termiflow feed --limit 10                # Limit number of items
  termiflow feed --refresh                 # Fetch new items first
  termiflow feed --watch                   # Continuously refresh feed`,
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
	feedCmd.Flags().BoolVar(&feedWatch, "watch", false, "run continuously, auto-refreshing on interval")
	feedCmd.Flags().DurationVar(&feedInterval, "interval", 30*time.Minute, "refresh interval for --watch mode")
}

func runFeed(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	// Handle cleanup
	if feedCleanup {
		return cleanupOldItems()
	}

	// Watch mode: loop with auto-refresh until Ctrl+C
	if feedWatch {
		return runFeedWatch(cmd, cfg)
	}

	// Handle refresh
	if feedRefresh {
		if err := refreshFeeds(cfg, feedTopic); err != nil {
			fmt.Print(ui.Error(fmt.Sprintf("Refresh failed: %v", err)))
			// Continue to show existing items
		}
	}

	// Check for subscriptions before rendering
	subs, err := db.GetActiveSubscriptions()
	if err != nil {
		return err
	}
	if len(subs) == 0 {
		fmt.Println()
		fmt.Print(ui.Warning("No active subscriptions"))
		fmt.Println()
		fmt.Printf("  Get started with:\n")
		fmt.Printf("    %s\n", ui.TitleStyle.Render("termiflow subscribe silicon-chips"))
		fmt.Printf("    %s\n", ui.TitleStyle.Render("termiflow subscribe \"your custom topic\""))
		fmt.Println()
		fmt.Printf("  See available topics with %s\n", ui.TitleStyle.Render("termiflow topics --available"))
		fmt.Println()
		return nil
	}

	return displayFeedItems(cmd, cfg)
}

func runFeedWatch(cmd *cobra.Command, cfg *config.Config) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		cancel()
	}()

	for {
		// Clear screen
		fmt.Print("\033[H\033[2J")

		if err := refreshFeeds(cfg, feedTopic); err != nil {
			fmt.Print(ui.Error(fmt.Sprintf("Refresh failed: %v", err)))
		}
		if err := displayFeedItems(cmd, cfg); err != nil {
			fmt.Print(ui.Error(fmt.Sprintf("Display failed: %v", err)))
		}

		fmt.Printf("\n%s\n", ui.MutedStyle.Render(fmt.Sprintf("  watching · next refresh in %s · ctrl+c to stop", feedInterval)))

		select {
		case <-ctx.Done():
			fmt.Println()
			return nil
		case <-time.After(feedInterval):
		}
	}
}

// displayFeedItems renders items from the DB without triggering a refresh.
func displayFeedItems(cmd *cobra.Command, cfg *config.Config) error {
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

	subs, err := db.GetActiveSubscriptions()
	if err != nil {
		return err
	}

	items, err := db.GetFeedItems(filter)
	if err != nil {
		return err
	}

	fmt.Println(ui.HeaderWithDate("termiflow feed"))

	if len(items) == 0 {
		fmt.Println()
		fmt.Print(ui.MutedStyle.Render("  no new items\n"))
		fmt.Println()
		return nil
	}

	groupedItems := groupBySubscription(items, subs)
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

	fmt.Print(ui.Footer(totalItems, topicCount, "just now"))

	if feedMarkRead && len(itemIDs) > 0 {
		_ = db.MarkItemsRead(itemIDs)
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

	// Initialize search provider
	searchProvider, err := search.GetSearchProvider(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize search provider: %w", err)
	}

	// Create scheduler
	sched := scheduler.New(llmProvider, searchProvider)

	// Show spinner
	sp := ui.NewSpinner(fmt.Sprintf("Fetching updates for %d subscription(s)...", len(subs)))
	sp.Start()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		wg            sync.WaitGroup
		mu            sync.Mutex
		totalNewItems int
		offline       atomic.Bool
		rateLimited   []string // topic names that hit rate limits
	)

	for _, sub := range subs {
		wg.Add(1)
		go func(sub *models.Subscription) {
			defer wg.Done()

			items, err := sched.RefreshSubscription(ctx, sub)
			if err != nil {
				if isOfflineError(err) {
					offline.Store(true)
					cancel()
					return
				}
				var rle *providers.RateLimitError
				if errors.As(err, &rle) {
					mu.Lock()
					rateLimited = append(rateLimited, sub.Topic)
					mu.Unlock()
					return
				}
				// Non-fatal: continue with other subscriptions
				return
			}

			mu.Lock()
			totalNewItems += len(items)
			mu.Unlock()
		}(sub)
	}

	wg.Wait()

	if offline.Load() {
		sp.Stop()
		fmt.Println()
		fmt.Println(ui.Warning("Offline — showing cached feed."))
		return nil
	}

	if totalNewItems > 0 {
		sp.Success(fmt.Sprintf("Fetched %d new item(s)", totalNewItems))
	} else {
		sp.Success("No new items found")
	}

	// Show rate limit warnings after spinner stops (avoids garbled output)
	for _, topic := range rateLimited {
		fmt.Printf("  %s Rate limited on %s — try again later.\n", ui.ErrorStyle.Render("!"), topic)
	}

	return nil
}

// isOfflineError detects network errors indicating the user is offline.
func isOfflineError(err error) bool {
	if err == nil {
		return false
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return false
}

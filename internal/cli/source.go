package cli

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/oluoyefeso/termiflow/internal/config"
	"github.com/oluoyefeso/termiflow/internal/db"
	"github.com/oluoyefeso/termiflow/internal/sources"
	tfSync "github.com/oluoyefeso/termiflow/internal/sync"
	"github.com/oluoyefeso/termiflow/internal/ui"
	"github.com/oluoyefeso/termiflow/pkg/models"
)

var srcContext string
var srcHourly bool
var srcDaily bool
var srcWeekly bool

var sourceCmd = &cobra.Command{
	Use:   "source",
	Short: "Manage RSS/blog source subscriptions",
	Long: `Follow specific blogs and RSS feeds with AI-curated summaries.

  termiflow source add https://simonwillison.net
  termiflow source add https://jvns.ca --context "systems programming"
  termiflow source list
  termiflow source remove "Julia Evans"`,
}

var sourceAddCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Subscribe to a blog or RSS feed",
	Args:  cobra.ExactArgs(1),
	RunE:  runSourceAdd,
}

var sourceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List source subscriptions",
	RunE:  runSourceList,
}

var sourceRemoveCmd = &cobra.Command{
	Use:   "remove <name-or-url>",
	Short: "Remove a source subscription",
	Args:  cobra.ExactArgs(1),
	RunE:  runSourceRemove,
}

func init() {
	sourceAddCmd.Flags().StringVar(&srcContext, "context", "", "relevance context for AI scoring (e.g., \"AI and ML\")")
	sourceAddCmd.Flags().BoolVar(&srcHourly, "hourly", false, "check for new posts every hour")
	sourceAddCmd.Flags().BoolVar(&srcDaily, "daily", false, "check daily (default)")
	sourceAddCmd.Flags().BoolVar(&srcWeekly, "weekly", false, "check weekly")

	sourceCmd.AddCommand(sourceAddCmd)
	sourceCmd.AddCommand(sourceListCmd)
	sourceCmd.AddCommand(sourceRemoveCmd)
}

func runSourceAdd(cmd *cobra.Command, args []string) error {
	rawURL := args[0]

	// Determine frequency
	frequency := "daily"
	if srcHourly {
		frequency = "hourly"
	} else if srcWeekly {
		frequency = "weekly"
	}

	// Check if already subscribed to this source
	existing, _ := db.GetSubscriptionBySourceURL(rawURL)
	if existing != nil {
		fmt.Print(ui.Warning(fmt.Sprintf("Already subscribed to %s", existing.DisplayName)))
		return nil
	}

	// Run autodiscovery with spinner
	fmt.Printf("  Checking feed... ")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	info, err := sources.Discover(ctx, rawURL)
	if err != nil {
		if info != nil && info.ScrapeOnly {
			fmt.Println(ui.WarningStyle.Render("no RSS feed"))
			fmt.Print(ui.Warning("No RSS feed found. Subscribing with web scraping (less reliable)."))
			return createSourceSubscription(rawURL, rawURL, domainFromURL(rawURL), "scrape", frequency)
		}
		fmt.Println(ui.ErrorStyle.Render("failed"))
		return fmt.Errorf("autodiscovery failed: %w", err)
	}

	fmt.Println(ui.SuccessStyle.Render("found"))

	// Check for duplicate by resolved feed URL (different input URL, same feed)
	existing, _ = db.GetSubscriptionBySourceURL(info.FeedURL)
	if existing != nil {
		fmt.Print(ui.Warning(fmt.Sprintf("Already subscribed to this feed as %q", existing.DisplayName)))
		return nil
	}

	displayName := info.Title
	if displayName == "" {
		displayName = domainFromURL(rawURL)
	}

	return createSourceSubscription(info.FeedURL, rawURL, displayName, "feed", frequency)
}

func createSourceSubscription(feedURL, originalURL, displayName, sourceType, frequency string) error {
	sub := &models.Subscription{
		Topic:       displayName, // topic field used as display key
		Frequency:   frequency,
		Sources:     []string{sourceType},
		IsActive:    true,
		SourceURL:   feedURL,
		SourceType:  sourceType,
		DisplayName: displayName,
		Context:     srcContext,
	}

	if err := db.CreateSubscription(sub); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			fmt.Print(ui.Warning(fmt.Sprintf("Already subscribed to %s", displayName)))
			return nil
		}
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	// Sync to server (managed mode, fails silently)
	tfSync.PushSubscription(context.Background(), sub)

	// Print success
	fmt.Print(ui.Success(fmt.Sprintf("Subscribed to %s", displayName)))
	fmt.Println()
	fmt.Print(ui.Info("Source", feedURL))
	fmt.Print(ui.Info("Type", sourceType))
	fmt.Print(ui.Info("Frequency", formatFrequency(frequency, config.Get().Schedule.DailyTime)))
	if srcContext != "" {
		fmt.Print(ui.Info("Context", srcContext))
	}
	fmt.Println()
	fmt.Printf("   Run %s to see curated posts.\n", ui.TitleStyle.Render("termiflow feed --refresh"))
	fmt.Println()

	return nil
}

func runSourceList(cmd *cobra.Command, args []string) error {
	subs, err := db.GetActiveSubscriptions()
	if err != nil {
		return err
	}

	var sourceSubs []*models.Subscription
	for _, sub := range subs {
		if sub.IsSourceSubscription() {
			sourceSubs = append(sourceSubs, sub)
		}
	}

	if len(sourceSubs) == 0 {
		fmt.Println()
		fmt.Print(ui.MutedStyle.Render("  No source subscriptions yet.\n"))
		fmt.Println()
		fmt.Printf("  Add one with %s\n", ui.TitleStyle.Render("termiflow source add <url>"))
		fmt.Println()
		return nil
	}

	fmt.Println(ui.Header("termiflow sources"))
	fmt.Println()

	for _, sub := range sourceSubs {
		total, unread, _ := db.GetSubscriptionItemCount(sub.ID)
		domain := domainFromURL(sub.SourceURL)

		name := sub.DisplayName
		if name == "" {
			name = domain
		}

		stats := ""
		if total > 0 {
			stats = fmt.Sprintf("%d items", total)
			if unread > 0 {
				stats += fmt.Sprintf(" (%d unread)", unread)
			}
		}

		typeTag := ui.MutedStyle.Render(fmt.Sprintf("[%s]", sub.SourceType))
		fmt.Printf("  %-30s %s  %-20s %-10s %s\n",
			ui.AccentStyle.Render(name),
			typeTag,
			ui.MutedStyle.Render(domain),
			ui.MutedStyle.Render(sub.Frequency),
			ui.MutedStyle.Render(stats),
		)
	}

	fmt.Println()
	return nil
}

func runSourceRemove(cmd *cobra.Command, args []string) error {
	nameOrURL := args[0]

	// Try matching by display_name first (case-insensitive)
	subs, err := db.GetActiveSubscriptions()
	if err != nil {
		return err
	}

	for _, sub := range subs {
		if sub.IsSourceSubscription() && strings.EqualFold(sub.DisplayName, nameOrURL) {
			return removeSource(sub)
		}
	}

	// Try matching by source_url (exact match or domain match)
	for _, sub := range subs {
		if sub.IsSourceSubscription() && (sub.SourceURL == nameOrURL || domainFromURL(sub.SourceURL) == nameOrURL) {
			return removeSource(sub)
		}
	}

	return fmt.Errorf("no source subscription matching %q found", nameOrURL)
}

func removeSource(sub *models.Subscription) error {
	if err := db.DeleteSubscriptionBySourceURL(sub.SourceURL); err != nil {
		return fmt.Errorf("failed to remove: %w", err)
	}
	// Sync deletion to server (managed mode, fails silently)
	tfSync.DeleteSubscription(context.Background(), sub.Topic)
	fmt.Print(ui.Success(fmt.Sprintf("Removed source: %s", sub.DisplayName)))
	return nil
}

func domainFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	host := u.Hostname()
	// Strip www. prefix
	host = strings.TrimPrefix(host, "www.")
	return host
}

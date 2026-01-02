package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/oluoyefeso/termiflow/internal/db"
	"github.com/oluoyefeso/termiflow/internal/ui"
	"github.com/oluoyefeso/termiflow/pkg/models"
)

var topicsAvailable bool
var topicsSubscribed bool

var topicsCmd = &cobra.Command{
	Use:   "topics",
	Short: "List available predefined topics and current subscriptions",
	Long: `List available predefined topics and current subscriptions.

Examples:
  termiflow topics                  # List all available + subscribed
  termiflow topics --available      # Only predefined categories
  termiflow topics --subscribed     # Only your subscriptions`,
	RunE: runTopics,
}

func init() {
	topicsCmd.Flags().BoolVar(&topicsAvailable, "available", false, "show only predefined categories")
	topicsCmd.Flags().BoolVar(&topicsSubscribed, "subscribed", false, "show only active subscriptions")
}

func runTopics(cmd *cobra.Command, args []string) error {
	fmt.Println(ui.Header("termiflow topics"))
	fmt.Println()

	// Get subscriptions
	subs, err := db.GetActiveSubscriptions()
	if err != nil {
		return err
	}

	// Build map of subscribed topics
	subscribedTopics := make(map[string]bool)
	for _, sub := range subs {
		subscribedTopics[sub.Topic] = true
	}

	// Show subscriptions section
	if !topicsAvailable {
		fmt.Println(ui.BoldStyle.Render(" Your Subscriptions"))
		fmt.Println(" " + ui.SmallDivider())

		if len(subs) == 0 {
			fmt.Println(ui.MutedStyle.Render("   No active subscriptions"))
		} else {
			for _, sub := range subs {
				total, unread, _ := db.GetSubscriptionItemCount(sub.ID)

				topicDisplay := sub.Topic
				if sub.Category == "" && !isKnownCategory(sub.Topic) {
					topicDisplay = fmt.Sprintf("\"custom: %s\"", truncate(sub.Topic, 18))
				}

				fmt.Print(ui.SubscriptionRow(topicDisplay, capitalize(sub.Frequency), total, unread, sub.Category != ""))
			}
		}
		fmt.Println()
	}

	// Show available categories section
	if !topicsSubscribed {
		fmt.Println(ui.BoldStyle.Render(" Available Categories"))
		fmt.Println(" " + ui.SmallDivider())

		hasUnsubscribed := false
		for _, cat := range models.DefaultCategories {
			if !subscribedTopics[cat.Name] {
				fmt.Print(ui.CategoryRow(cat.Name, cat.DisplayName))
				hasUnsubscribed = true
			}
		}

		if !hasUnsubscribed {
			fmt.Println(ui.MutedStyle.Render("   All categories subscribed!"))
		}
		fmt.Println()
	}

	fmt.Print(ui.Tip(fmt.Sprintf("Subscribe with %s", ui.TitleStyle.Render("termiflow subscribe <topic>"))))
	fmt.Println()

	return nil
}

func isKnownCategory(name string) bool {
	return models.GetCategoryByName(name) != nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}

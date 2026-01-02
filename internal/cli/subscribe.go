package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/oluoyefeso/termiflow/internal/config"
	"github.com/oluoyefeso/termiflow/internal/db"
	"github.com/oluoyefeso/termiflow/internal/ui"
	"github.com/oluoyefeso/termiflow/pkg/models"
)

var subHourly bool
var subDaily bool
var subWeekly bool
var subSources string

var subscribeCmd = &cobra.Command{
	Use:   "subscribe <topic>",
	Short: "Subscribe to a topic for curated updates",
	Long: `Subscribe to a topic for curated updates.

You can subscribe to predefined categories or free-form topics:

  termiflow subscribe "silicon-chips"                    # Predefined category
  termiflow subscribe "RISC-V adoption in automotive"    # Free-form topic
  termiflow subscribe "rust async ecosystem" --hourly
  termiflow subscribe "quantum error correction" --weekly`,
	Args: cobra.ExactArgs(1),
	RunE: runSubscribe,
}

func init() {
	subscribeCmd.Flags().BoolVar(&subHourly, "hourly", false, "get updates every hour")
	subscribeCmd.Flags().BoolVar(&subDaily, "daily", false, "get updates once per day (default)")
	subscribeCmd.Flags().BoolVar(&subWeekly, "weekly", false, "get updates once per week")
	subscribeCmd.Flags().StringVar(&subSources, "sources", "", "comma-separated source preferences (tavily,rss,scrape)")
}

func runSubscribe(cmd *cobra.Command, args []string) error {
	topic := args[0]
	cfg := config.Get()

	// Determine frequency
	frequency := cfg.Schedule.DefaultFrequency
	if subHourly {
		frequency = "hourly"
	} else if subDaily {
		frequency = "daily"
	} else if subWeekly {
		frequency = "weekly"
	}

	// Check if already subscribed
	existing, err := db.GetSubscription(topic)
	if err == nil && existing != nil {
		fmt.Print(ui.Warning(fmt.Sprintf("Already subscribed to %s", topic)))
		return nil
	}

	// Check if this is a predefined category
	category := models.GetCategoryByName(topic)

	// Parse sources
	var sources []string
	if subSources != "" {
		sources = strings.Split(subSources, ",")
	} else {
		sources = []string{"tavily", "rss"}
	}

	// Create subscription
	sub := &models.Subscription{
		Topic:     topic,
		Frequency: frequency,
		Sources:   sources,
		IsActive:  true,
	}

	if category != nil {
		sub.Category = category.Name
	}

	if err := db.CreateSubscription(sub); err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}

	// Print success message
	fmt.Print(ui.Success(fmt.Sprintf("Subscribed to %s", topic)))
	fmt.Println()

	if category != nil {
		fmt.Print(ui.Info("Category", category.DisplayName))
	} else {
		fmt.Print(ui.Info("Type", "Custom topic"))
		// For custom topics, we could use LLM to generate keywords
		// For now, just use the topic as-is
		fmt.Print(ui.Info("Keywords", topic))
	}

	fmt.Print(ui.Info("Frequency", formatFrequency(frequency, cfg.Schedule.DailyTime)))
	fmt.Print(ui.Info("Sources", formatSources(sources)))

	fmt.Println()
	fmt.Printf("   Run %s to see your updates.\n", ui.TitleStyle.Render("termiflow feed"))
	fmt.Println()

	return nil
}

func formatFrequency(frequency, dailyTime string) string {
	switch frequency {
	case "hourly":
		return "Hourly"
	case "daily":
		return fmt.Sprintf("Daily at %s", dailyTime)
	case "weekly":
		return "Weekly"
	default:
		return frequency
	}
}

func formatSources(sources []string) string {
	var parts []string
	for _, s := range sources {
		switch s {
		case "tavily":
			parts = append(parts, "Tavily search")
		case "rss":
			parts = append(parts, "RSS feeds")
		case "scrape":
			parts = append(parts, "Web scraping")
		default:
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, ", ")
}

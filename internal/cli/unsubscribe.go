package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/termiflow/termiflow/internal/db"
	"github.com/termiflow/termiflow/internal/ui"
)

var unsubAll bool
var unsubForce bool

var unsubscribeCmd = &cobra.Command{
	Use:   "unsubscribe <topic>",
	Short: "Remove a topic subscription",
	Long: `Remove a topic subscription.

Examples:
  termiflow unsubscribe "silicon-chips"
  termiflow unsubscribe --all     # Remove all subscriptions`,
	Args: cobra.MaximumNArgs(1),
	RunE: runUnsubscribe,
}

func init() {
	unsubscribeCmd.Flags().BoolVar(&unsubAll, "all", false, "unsubscribe from all topics")
	unsubscribeCmd.Flags().BoolVar(&unsubForce, "force", false, "skip confirmation prompt")
}

func runUnsubscribe(cmd *cobra.Command, args []string) error {
	if unsubAll {
		return unsubscribeAll()
	}

	if len(args) == 0 {
		return fmt.Errorf("please specify a topic or use --all")
	}

	topic := args[0]

	// Check if subscription exists
	sub, err := db.GetSubscription(topic)
	if err != nil || sub == nil {
		fmt.Print(ui.Error(fmt.Sprintf("Not subscribed to %s", topic)))
		return nil
	}

	// Delete subscription
	if err := db.DeleteSubscription(topic); err != nil {
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}

	fmt.Print(ui.Success(fmt.Sprintf("Unsubscribed from %s", topic)))
	fmt.Println()
	fmt.Printf("   Your feed items for this topic have been preserved.\n")
	fmt.Printf("   Run %s to remove old items.\n", ui.TitleStyle.Render("termiflow feed --cleanup"))
	fmt.Println()

	return nil
}

func unsubscribeAll() error {
	subs, err := db.GetActiveSubscriptions()
	if err != nil {
		return err
	}

	if len(subs) == 0 {
		fmt.Print(ui.Warning("No active subscriptions"))
		return nil
	}

	if !unsubForce {
		fmt.Printf("This will remove all %d subscriptions. Are you sure? [y/N] ", len(subs))
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := db.DeleteAllSubscriptions(); err != nil {
		return fmt.Errorf("failed to unsubscribe: %w", err)
	}

	fmt.Print(ui.Success(fmt.Sprintf("Unsubscribed from all %d topics", len(subs))))
	return nil
}

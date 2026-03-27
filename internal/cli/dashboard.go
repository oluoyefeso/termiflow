package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/oluoyefeso/termiflow/internal/db"
	"github.com/oluoyefeso/termiflow/internal/ui"
)

// JSON output types for dashboard.

type DashboardOutputJSON struct {
	Subscriptions []DashboardSubJSON `json:"subscriptions"`
	TotalUnread   int                `json:"total_unread"`
	Version       string             `json:"version"`
}

type DashboardSubJSON struct {
	Topic  string `json:"topic"`
	Unread int    `json:"unread"`
	Total  int    `json:"total"`
}

func runDashboard(cmd *cobra.Command) error {
	subs, err := db.GetActiveSubscriptions()
	if err != nil {
		return err
	}

	if len(subs) == 0 {
		if jsonOutput {
			out := DashboardOutputJSON{
				Subscriptions: []DashboardSubJSON{},
				TotalUnread:   0,
				Version:       version,
			}
			return ui.WriteJSON(out, version)
		}
		showGettingStarted()
		return nil
	}

	type subInfo struct {
		topic  string
		total  int
		unread int
	}

	var infos []subInfo
	totalUnread := 0

	for _, sub := range subs {
		total, unread, err := db.GetSubscriptionItemCount(sub.ID)
		if err != nil {
			continue
		}
		infos = append(infos, subInfo{
			topic:  sub.Topic,
			total:  total,
			unread: unread,
		})
		totalUnread += unread
	}

	if jsonOutput {
		jsonSubs := make([]DashboardSubJSON, 0, len(infos))
		for _, info := range infos {
			s := DashboardSubJSON{
				Topic:  info.topic,
				Unread: info.unread,
				Total:  info.total,
			}
			jsonSubs = append(jsonSubs, s)
		}
		out := DashboardOutputJSON{
			Subscriptions: jsonSubs,
			TotalUnread:   totalUnread,
			Version:       version,
		}
		return ui.WriteJSON(out, version)
	}

	// Terminal render
	fmt.Println(ui.HeaderWithDate("termiflow"))
	fmt.Println()

	for _, info := range infos {
		unreadLabel := fmt.Sprintf("%d unread", info.unread)
		if info.unread == 0 {
			unreadLabel = "up to date"
		}
		fmt.Printf("  %s %-26s %s\n",
			ui.TitleStyle.Render("▸"),
			info.topic,
			ui.MutedStyle.Render(unreadLabel),
		)
	}

	if totalUnread > 0 {
		fmt.Printf("  %s%s\n",
			fmt.Sprintf("%28s", ""),
			ui.MutedStyle.Render(fmt.Sprintf("─── %d total", totalUnread)),
		)
	}

	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("  Commands:"))
	commands := []struct{ cmd, desc string }{
		{"feed", "show your feed"},
		{"ask <question>", "ask anything"},
		{"subscribe <topic>", "add a topic"},
		{"status", "full config info"},
		{"upgrade", "check for updates"},
	}
	for _, c := range commands {
		fmt.Printf("    %-22s %s\n",
			ui.TitleStyle.Render(c.cmd),
			ui.MutedStyle.Render(c.desc),
		)
	}
	fmt.Println()

	return nil
}

func showGettingStarted() {
	fmt.Println()
	fmt.Print(ui.Warning("No active subscriptions"))
	fmt.Println()
	fmt.Printf("  Get started with:\n")
	fmt.Printf("    %s\n", ui.TitleStyle.Render("termiflow subscribe silicon-chips"))
	fmt.Printf("    %s\n", ui.TitleStyle.Render("termiflow subscribe \"your custom topic\""))
	fmt.Println()
	fmt.Printf("  See available topics with %s\n", ui.TitleStyle.Render("termiflow topics --available"))
	fmt.Println()
}

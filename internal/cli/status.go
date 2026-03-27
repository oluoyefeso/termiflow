package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/oluoyefeso/termiflow/internal/config"
	"github.com/oluoyefeso/termiflow/internal/db"
	"github.com/oluoyefeso/termiflow/internal/ui"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current configuration summary and database stats",
	RunE:  runStatus,
}

// JSON output types for status command.

type StatusOutputJSON struct {
	Mode          string             `json:"mode"`
	Provider      string             `json:"provider,omitempty"`
	APIKey        string             `json:"api_key,omitempty"`
	BaseURL       string             `json:"base_url,omitempty"`
	Subscriptions []SubscriptionJSON `json:"subscriptions"`
	Database      DatabaseJSON       `json:"database"`
	ConfigPath    string             `json:"config_path"`
}

type SubscriptionJSON struct {
	Topic       string `json:"topic"`
	Frequency   string `json:"frequency"`
	LastFetched string `json:"last_fetched,omitempty"`
	ItemCount   int    `json:"item_count"`
	UnreadCount int    `json:"unread_count"`
}

type DatabaseJSON struct {
	Path   string `json:"path"`
	SizeKB int64  `json:"size_kb"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg := config.Get()

	if jsonOutput {
		return renderStatusJSON(cfg)
	}

	fmt.Println(ui.Header("termiflow status"))
	fmt.Println()

	// Mode
	if config.IsManagedMode() {
		fmt.Print(ui.Info("Mode", "Managed"))
		apiKey := cfg.Providers.Managed.APIKey
		if len(apiKey) > 20 {
			masked := apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
			fmt.Print(ui.Info("API Key", masked))
		} else if apiKey != "" {
			fmt.Print(ui.Info("API Key", "***"))
		}
		baseURL := cfg.Providers.Managed.BaseURL
		if baseURL == "" {
			baseURL = "https://api.termiflow.com"
		}
		fmt.Print(ui.Info("Base URL", baseURL))
	} else {
		fmt.Print(ui.Info("Mode", "Self-hosted"))
		fmt.Print(ui.Info("Provider", cfg.General.DefaultProvider))
	}
	fmt.Println()

	// Subscriptions
	subs, err := db.GetActiveSubscriptions()
	if err == nil {
		fmt.Print(ui.Info("Subscriptions", fmt.Sprintf("%d active", len(subs))))
		for _, sub := range subs {
			lastFetched := "never"
			if sub.LastFetchedAt != nil {
				lastFetched = sub.LastFetchedAt.Format("2006-01-02 15:04")
			}
			fmt.Printf("    %s  (last: %s)\n", ui.TitleStyle.Render(sub.Topic), ui.MutedStyle.Render(lastFetched))
		}
	}
	fmt.Println()

	// Database size
	dbPath := filepath.Join(config.GetDataDir(), "termiflow.db")
	if info, err := os.Stat(dbPath); err == nil {
		sizeKB := info.Size() / 1024
		if sizeKB < 1024 {
			fmt.Print(ui.Info("Database", fmt.Sprintf("%d KB (%s)", sizeKB, dbPath)))
		} else {
			fmt.Print(ui.Info("Database", fmt.Sprintf("%.1f MB (%s)", float64(sizeKB)/1024, dbPath)))
		}
	}

	// Config path
	fmt.Print(ui.Info("Config", config.GetConfigPath()))
	fmt.Println()

	return nil
}

func renderStatusJSON(cfg *config.Config) error {
	out := StatusOutputJSON{
		ConfigPath: config.GetConfigPath(),
	}

	if config.IsManagedMode() {
		out.Mode = "managed"
		apiKey := cfg.Providers.Managed.APIKey
		if len(apiKey) > 20 {
			out.APIKey = apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
		} else if apiKey != "" {
			out.APIKey = "***"
		}
		baseURL := cfg.Providers.Managed.BaseURL
		if baseURL == "" {
			baseURL = "https://api.termiflow.com"
		}
		out.BaseURL = baseURL
	} else {
		out.Mode = "self-hosted"
		out.Provider = cfg.General.DefaultProvider
	}

	subs, err := db.GetActiveSubscriptions()
	if err == nil {
		out.Subscriptions = make([]SubscriptionJSON, 0, len(subs))
		for _, sub := range subs {
			subJSON := SubscriptionJSON{
				Topic:     sub.Topic,
				Frequency: sub.Frequency,
			}
			if sub.LastFetchedAt != nil {
				subJSON.LastFetched = sub.LastFetchedAt.Format("2006-01-02T15:04:05Z07:00")
			}
			total, unread, err := db.GetSubscriptionItemCount(sub.ID)
			if err == nil {
				subJSON.ItemCount = total
				subJSON.UnreadCount = unread
			}
			out.Subscriptions = append(out.Subscriptions, subJSON)
		}
	} else {
		out.Subscriptions = []SubscriptionJSON{}
	}

	dbPath := filepath.Join(config.GetDataDir(), "termiflow.db")
	out.Database = DatabaseJSON{Path: dbPath}
	if info, err := os.Stat(dbPath); err == nil {
		out.Database.SizeKB = info.Size() / 1024
	}

	return ui.WriteJSON(out, version)
}

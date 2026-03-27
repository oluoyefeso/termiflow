package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/oluoyefeso/termiflow/internal/config"
	"github.com/oluoyefeso/termiflow/internal/db"
	"github.com/oluoyefeso/termiflow/internal/notifications"
	"github.com/oluoyefeso/termiflow/internal/tui"
	"github.com/oluoyefeso/termiflow/internal/ui"
)

var (
	cfgFile    string
	provider   string
	quiet      bool
	debug      bool
	noColor    bool
	jsonOutput bool

	version string
	commit  string
	date    string

	// cachedBanners holds notification banners loaded during PersistentPreRunE,
	// so the TUI can display them instead of printing to stdout.
	cachedBanners []tui.BannerMsg
)

var rootCmd = &cobra.Command{
	Use:   "termiflow",
	Short: "Terminal-native AI intelligence tool",
	Long: `Termflow is a terminal-native AI intelligence tool that lets developers
ask questions and subscribe to curated topic updates, all from the command line.

Information comes to you where you already are — the terminal.
No browser switching, no context loss, no noise. Just signal.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Launch TUI when running interactively (TTY, not JSON, not quiet)
		if !jsonOutput && !quiet && term.IsTerminal(int(os.Stdin.Fd())) {
			return tui.Run(cachedBanners)
		}
		return runDashboard(cmd)
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// JSON mode implies quiet to prevent spinners/banners corrupting JSON output
		if jsonOutput {
			quiet = true
		}

		// Skip init for certain commands
		if cmd.Name() == "version" || cmd.Name() == "help" {
			return nil
		}
		if cmd.Parent() != nil && cmd.Parent().Name() == "config" && cmd.Name() == "init" {
			return nil
		}

		// Apply no-color setting
		if noColor {
			ui.NoColor(true)
		}

		// Load config
		_, err := config.Load(cfgFile)
		if err != nil {
			// Config file not found is okay for init
			if cmd.Name() == "init" {
				return nil
			}
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Initialize database
		if err := db.Init(); err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}

		// Load notification banners and start background fetch
		if !quiet {
			nm := notifications.NewManager(
				config.Get().Providers.Managed.BaseURL,
				config.Get().Providers.Managed.APIKey,
				version,
				config.GetCacheDir(),
				config.IsManagedMode(),
			)
			nm.LoadCache()
			banners := nm.GetBanners()

			// If this is the root command in TUI mode (interactive TTY, not JSON),
			// cache banners for the TUI. Otherwise, print them to stdout as before.
			isTUI := cmd.Name() == "termiflow" && !jsonOutput && term.IsTerminal(int(os.Stdin.Fd()))
			if isTUI {
				cachedBanners = nil
				for _, b := range banners {
					cachedBanners = append(cachedBanners, tui.BannerMsg{
						Type:    b.Type,
						Message: b.Message,
					})
				}
			} else {
				for _, b := range banners {
					fmt.Print(ui.Banner(b.Type, b.Message))
				}
			}

			if len(banners) > 0 {
				nm.MarkDisplayed(banners)
			}
			go nm.FetchAsync(context.Background())
		}

		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		db.Close()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func SetVersionInfo(v, c, d string) {
	version = v
	commit = c
	date = d
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ~/.config/termiflow/config.toml)")
	rootCmd.PersistentFlags().StringVar(&provider, "provider", "", "override LLM provider (openai, anthropic, local)")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress non-essential output")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output as JSON")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(askCmd)
	rootCmd.AddCommand(subscribeCmd)
	rootCmd.AddCommand(unsubscribeCmd)
	rootCmd.AddCommand(feedCmd)
	rootCmd.AddCommand(topicsCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(changelogCmd)
}

func getProvider() string {
	if provider != "" {
		return provider
	}
	return config.Get().General.DefaultProvider
}

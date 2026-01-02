package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/oluoyefeso/termiflow/internal/config"
	"github.com/oluoyefeso/termiflow/internal/db"
	"github.com/oluoyefeso/termiflow/internal/ui"
)

var (
	cfgFile  string
	provider string
	quiet    bool
	debug    bool
	noColor  bool

	version string
	commit  string
	date    string
)

var rootCmd = &cobra.Command{
	Use:   "termiflow",
	Short: "Terminal-native AI intelligence tool",
	Long: `Termflow is a terminal-native AI intelligence tool that lets developers
ask questions and subscribe to curated topic updates, all from the command line.

Information comes to you where you already are — the terminal.
No browser switching, no context loss, no noise. Just signal.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
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

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(askCmd)
	rootCmd.AddCommand(subscribeCmd)
	rootCmd.AddCommand(unsubscribeCmd)
	rootCmd.AddCommand(feedCmd)
	rootCmd.AddCommand(topicsCmd)
}

func getProvider() string {
	if provider != "" {
		return provider
	}
	return config.Get().General.DefaultProvider
}

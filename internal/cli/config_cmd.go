package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/oluoyefeso/termiflow/internal/config"
	"github.com/oluoyefeso/termiflow/internal/ui"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and modify configuration",
	Run:   showConfig,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create default config file with interactive setup",
	RunE:  runConfigInit,
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open config file in $EDITOR",
	RunE:  runConfigEdit,
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print config file path",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(config.GetConfigPath())
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		value := viper.Get(args[0])
		if value == nil {
			fmt.Println("(not set)")
		} else {
			fmt.Printf("%v\n", value)
		}
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]

		// Validate theme names
		if key == "general.theme" {
			if err := ui.ValidateThemeName(value); err != nil {
				return err
			}
		}

		config.Set(key, value)
		if err := config.SaveConfig(config.GetConfigPath()); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Print(ui.Success(fmt.Sprintf("Set %s = %s", key, value)))
		return nil
	},
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)

	configCmd.Flags().Bool("edit", false, "Open config file in $EDITOR")
}

func showConfig(cmd *cobra.Command, args []string) {
	edit, _ := cmd.Flags().GetBool("edit")
	if edit {
		_ = runConfigEdit(cmd, args)
		return
	}

	cfg := config.Get()

	fmt.Println(ui.Header("termiflow config"))
	fmt.Println()
	fmt.Printf(" Config file: %s\n", ui.MutedStyle.Render(config.GetConfigPath()))
	fmt.Println()

	// General section
	fmt.Println(ui.BoldStyle.Render(" General"))
	fmt.Print(ui.Info("Default provider", cfg.General.DefaultProvider))
	fmt.Print(ui.Info("Theme", cfg.General.Theme))
	fmt.Print(ui.Info("Feed limit", fmt.Sprintf("%d", cfg.General.FeedLimit)))
	fmt.Println()

	// Providers section
	fmt.Println(ui.BoldStyle.Render(" Providers"))
	printProviderStatus("OpenAI", cfg.Providers.OpenAI.APIKey != "", cfg.Providers.OpenAI.Model)
	printProviderStatus("Anthropic", cfg.Providers.Anthropic.APIKey != "", cfg.Providers.Anthropic.Model)
	printProviderStatus("Local", cfg.Providers.Local.BaseURL != "", cfg.Providers.Local.Model)
	fmt.Println()

	// Search section
	fmt.Println(ui.BoldStyle.Render(" Search"))
	printProviderStatus("Tavily", cfg.Search.Tavily.APIKey != "", "")
	fmt.Print(ui.Info("RSS feeds", fmt.Sprintf("%d global feeds", len(cfg.Search.RSS.Feeds))))
	fmt.Println()

	// Schedule section
	fmt.Println(ui.BoldStyle.Render(" Schedule"))
	fmt.Print(ui.Info("Default frequency", cfg.Schedule.DefaultFrequency))
	fmt.Print(ui.Info("Daily time", cfg.Schedule.DailyTime))
	fmt.Println()

	fmt.Printf(" Run %s to modify settings.\n", ui.TitleStyle.Render("termiflow config --edit"))
}

func printProviderStatus(name string, configured bool, model string) {
	status := ui.ErrorStyle.Render("✗ not configured")
	if configured {
		if model != "" {
			status = ui.SuccessStyle.Render(fmt.Sprintf("✓ configured (%s)", model))
		} else {
			status = ui.SuccessStyle.Render("✓ configured")
		}
	}
	fmt.Print(ui.Info(name, status))
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Welcome to termiflow! Let's get you set up.")
	fmt.Println()

	// Ensure config directory exists
	if err := config.EnsureDirectories(); err != nil {
		return err
	}

	// Check for managed API key first (default: yes, since most users sign up via termiflow.com)
	fmt.Print("Do you have a termiflow API key? (Y/n): ")
	managedAnswer, _ := reader.ReadString('\n')
	managedAnswer = strings.TrimSpace(strings.ToLower(managedAnswer))

	if managedAnswer != "n" && managedAnswer != "no" {
		// Check if key already exists
		existingKey := config.GetString("providers.managed.api_key")
		if existingKey != "" {
			fmt.Print("Update existing managed key? (y/N): ")
			updateAnswer, _ := reader.ReadString('\n')
			updateAnswer = strings.TrimSpace(strings.ToLower(updateAnswer))
			if updateAnswer != "y" && updateAnswer != "yes" {
				fmt.Print(ui.Success("Keeping existing managed key"))
				return nil
			}
		}

		fmt.Print("Enter your termiflow API key: ")
		managedKey, _ := reader.ReadString('\n')
		managedKey = strings.TrimSpace(managedKey)
		if managedKey != "" {
			config.Set("providers.managed.api_key", managedKey)
			config.Set("general.default_provider", "managed")
			fmt.Print(ui.Success("Managed mode configured. No other keys needed."))

			configPath := config.GetConfigPath()
			configDir := filepath.Dir(configPath)
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}
			if err := writeManagedConfig(configPath, managedKey); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			return nil
		}
	}

	// Fall through to self-hosted setup
	fmt.Println()
	fmt.Println("Setting up self-hosted configuration...")
	fmt.Println()

	// OpenAI API key
	fmt.Print("Enter your OpenAI API key (or press Enter to skip): ")
	openaiKey, _ := reader.ReadString('\n')
	openaiKey = strings.TrimSpace(openaiKey)
	if openaiKey != "" {
		config.Set("providers.openai.api_key", openaiKey)
		fmt.Print(ui.Success("OpenAI configured"))
	}

	// Anthropic API key
	fmt.Print("Enter your Anthropic API key (or press Enter to skip): ")
	anthropicKey, _ := reader.ReadString('\n')
	anthropicKey = strings.TrimSpace(anthropicKey)
	if anthropicKey != "" {
		config.Set("providers.anthropic.api_key", anthropicKey)
		fmt.Print(ui.Success("Anthropic configured"))
	}

	// Tavily API key
	fmt.Print("Enter your Tavily API key (or press Enter to skip): ")
	tavilyKey, _ := reader.ReadString('\n')
	tavilyKey = strings.TrimSpace(tavilyKey)
	if tavilyKey != "" {
		config.Set("search.tavily.api_key", tavilyKey)
		fmt.Print(ui.Success("Tavily configured"))
	}

	// Set default provider based on what's configured
	if openaiKey != "" {
		config.Set("general.default_provider", "openai")
	} else if anthropicKey != "" {
		config.Set("general.default_provider", "anthropic")
	}

	// Save config
	configPath := config.GetConfigPath()
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := writeDefaultConfig(configPath, openaiKey, anthropicKey, tavilyKey); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	fmt.Print(ui.Success(fmt.Sprintf("Configuration saved to %s", configPath)))

	return nil
}

func writeManagedConfig(path, managedKey string) error {
	content := fmt.Sprintf(`# Termflow Configuration — Managed Mode

[general]
default_provider = "managed"
theme = "amber"
cache_dir = "~/.cache/termiflow"
feed_limit = 20

[providers.managed]
api_key = "%s"
base_url = "https://api.termiflow.com"

[schedule]
default_frequency = "daily"
daily_time = "08:00"
weekly_day = 1
`, managedKey)
	return os.WriteFile(path, []byte(content), 0600)
}

func writeDefaultConfig(path, openaiKey, anthropicKey, tavilyKey string) error {
	content := fmt.Sprintf(`# Termflow Configuration

[general]
# Default LLM provider: "openai", "anthropic", "local"
default_provider = "%s"

# Color theme: "amber", "light", "dracula"
theme = "amber"

# Cache directory for offline mode
cache_dir = "~/.cache/termiflow"

# How many feed items to show by default
feed_limit = 20

[providers.openai]
api_key = "%s"
model = "gpt-4o"
base_url = "https://api.openai.com/v1"

[providers.anthropic]
api_key = "%s"
model = "claude-sonnet-4-20250514"

[providers.local]
# OpenAI-compatible local server (Ollama, llama.cpp, LM Studio, etc.)
base_url = "http://localhost:11434/v1"
model = "llama3"

[search.tavily]
api_key = "%s"

[search.rss]
# Global RSS feeds to include
feeds = []

[search.scraper]
user_agent = "termiflow/1.0"
timeout = 30
respect_robots = true

[schedule]
default_frequency = "daily"
daily_time = "08:00"
weekly_day = 1
`,
		getDefaultProvider(openaiKey, anthropicKey),
		openaiKey,
		anthropicKey,
		tavilyKey,
	)

	return os.WriteFile(path, []byte(content), 0600)
}

func getDefaultProvider(openaiKey, anthropicKey string) string {
	if openaiKey != "" {
		return "openai"
	}
	if anthropicKey != "" {
		return "anthropic"
	}
	return "openai"
}

func runConfigEdit(cmd *cobra.Command, args []string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	configPath := config.GetConfigPath()

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("Config file not found. Run %s first.\n", ui.TitleStyle.Render("termiflow config init"))
		return nil
	}

	editorCmd := exec.Command(editor, configPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	return editorCmd.Run()
}

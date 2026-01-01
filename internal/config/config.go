package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	General   GeneralConfig   `mapstructure:"general"`
	Providers ProvidersConfig `mapstructure:"providers"`
	Search    SearchConfig    `mapstructure:"search"`
	Schedule  ScheduleConfig  `mapstructure:"schedule"`
}

type GeneralConfig struct {
	DefaultProvider string `mapstructure:"default_provider"`
	OutputStyle     string `mapstructure:"output_style"`
	CacheDir        string `mapstructure:"cache_dir"`
	FeedLimit       int    `mapstructure:"feed_limit"`
}

type ProvidersConfig struct {
	OpenAI    OpenAIConfig    `mapstructure:"openai"`
	Anthropic AnthropicConfig `mapstructure:"anthropic"`
	Local     LocalConfig     `mapstructure:"local"`
}

type OpenAIConfig struct {
	APIKey  string `mapstructure:"api_key"`
	Model   string `mapstructure:"model"`
	BaseURL string `mapstructure:"base_url"`
}

type AnthropicConfig struct {
	APIKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
}

type LocalConfig struct {
	BaseURL string `mapstructure:"base_url"`
	Model   string `mapstructure:"model"`
}

type SearchConfig struct {
	Tavily  TavilyConfig  `mapstructure:"tavily"`
	RSS     RSSConfig     `mapstructure:"rss"`
	Scraper ScraperConfig `mapstructure:"scraper"`
}

type TavilyConfig struct {
	APIKey string `mapstructure:"api_key"`
}

type RSSConfig struct {
	Feeds []string `mapstructure:"feeds"`
}

type ScraperConfig struct {
	UserAgent     string `mapstructure:"user_agent"`
	Timeout       int    `mapstructure:"timeout"`
	RespectRobots bool   `mapstructure:"respect_robots"`
}

type ScheduleConfig struct {
	DefaultFrequency string `mapstructure:"default_frequency"`
	DailyTime        string `mapstructure:"daily_time"`
	WeeklyDay        int    `mapstructure:"weekly_day"`
}

var cfg *Config

func Get() *Config {
	if cfg == nil {
		cfg = &Config{}
		setDefaults()
	}
	return cfg
}

func Load(configPath string) (*Config, error) {
	setDefaults()

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		configDir := expandPath("~/.config/termiflow")
		viper.AddConfigPath(configDir)
		viper.AddConfigPath(expandPath("~"))
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
	}

	viper.SetEnvPrefix("TERMFLOW")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Map environment variables
	_ = viper.BindEnv("providers.openai.api_key", "TERMFLOW_OPENAI_API_KEY")
	_ = viper.BindEnv("providers.anthropic.api_key", "TERMFLOW_ANTHROPIC_API_KEY")
	_ = viper.BindEnv("search.tavily.api_key", "TERMFLOW_TAVILY_API_KEY")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config: %w", err)
		}
	}

	cfg = &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	return cfg, nil
}

func setDefaults() {
	viper.SetDefault("general.default_provider", DefaultProvider)
	viper.SetDefault("general.output_style", DefaultOutputStyle)
	viper.SetDefault("general.cache_dir", DefaultCacheDir())
	viper.SetDefault("general.feed_limit", DefaultFeedLimit)

	viper.SetDefault("providers.openai.model", DefaultOpenAIModel)
	viper.SetDefault("providers.openai.base_url", DefaultOpenAIBaseURL)
	viper.SetDefault("providers.anthropic.model", DefaultAnthropicModel)
	viper.SetDefault("providers.local.base_url", DefaultLocalBaseURL)
	viper.SetDefault("providers.local.model", DefaultLocalModel)

	viper.SetDefault("search.scraper.user_agent", DefaultScraperUserAgent)
	viper.SetDefault("search.scraper.timeout", DefaultScraperTimeout)
	viper.SetDefault("search.scraper.respect_robots", DefaultRespectRobots)

	viper.SetDefault("schedule.default_frequency", DefaultFrequency)
	viper.SetDefault("schedule.daily_time", DefaultDailyTime)
	viper.SetDefault("schedule.weekly_day", DefaultWeeklyDay)
}

func GetConfigPath() string {
	if viper.ConfigFileUsed() != "" {
		return viper.ConfigFileUsed()
	}
	return filepath.Join(expandPath("~/.config/termiflow"), "config.toml")
}

func GetDataDir() string {
	return expandPath(DefaultDataDir())
}

func GetCacheDir() string {
	cacheDir := DefaultCacheDir()
	if cfg != nil && cfg.General.CacheDir != "" {
		cacheDir = cfg.General.CacheDir
	}
	return expandPath(cacheDir)
}

func EnsureDirectories() error {
	dirs := []string{
		expandPath("~/.config/termiflow"),
		GetDataDir(),
		GetCacheDir(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func SaveConfig(path string) error {
	return viper.WriteConfigAs(path)
}

func Set(key string, value interface{}) {
	viper.Set(key, value)
}

func GetString(key string) string {
	return viper.GetString(key)
}

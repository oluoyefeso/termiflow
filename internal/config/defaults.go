package config

const (
	DefaultProvider    = "openai"
	DefaultOutputStyle = "pretty"
	DefaultFeedLimit   = 20
	DefaultFrequency   = "daily"
	DefaultDailyTime   = "08:00"
	DefaultWeeklyDay   = 1

	DefaultOpenAIModel    = "gpt-4o"
	DefaultOpenAIBaseURL  = "https://api.openai.com/v1"
	DefaultAnthropicModel = "claude-sonnet-4-20250514"
	DefaultLocalBaseURL   = "http://localhost:11434/v1"
	DefaultLocalModel     = "llama3"

	DefaultScraperUserAgent = "termiflow/1.0"
	DefaultScraperTimeout   = 30
	DefaultRespectRobots    = true
)

func DefaultConfigDir() string {
	return "~/.config/termiflow"
}

func DefaultDataDir() string {
	return "~/.local/share/termiflow"
}

func DefaultCacheDir() string {
	return "~/.cache/termiflow"
}

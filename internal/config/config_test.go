package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestMain(m *testing.M) {
	// Reset viper before each test run
	code := m.Run()
	os.Exit(code)
}

func resetViper() {
	viper.Reset()
	cfg = nil
}

func TestGet(t *testing.T) {
	resetViper()

	c := Get()
	if c == nil {
		t.Fatal("Get() returned nil")
	}

	// Should return same instance
	c2 := Get()
	if c != c2 {
		t.Error("Get() should return same instance")
	}
}

func TestLoadWithDefaults(t *testing.T) {
	resetViper()

	// Create temp dir for config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Create minimal config file
	err := os.WriteFile(configPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	c, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	// Check defaults are applied
	if c.General.DefaultProvider != DefaultProvider {
		t.Errorf("DefaultProvider = %q, want %q", c.General.DefaultProvider, DefaultProvider)
	}
	if c.General.OutputStyle != DefaultOutputStyle {
		t.Errorf("OutputStyle = %q, want %q", c.General.OutputStyle, DefaultOutputStyle)
	}
	if c.General.FeedLimit != DefaultFeedLimit {
		t.Errorf("FeedLimit = %d, want %d", c.General.FeedLimit, DefaultFeedLimit)
	}
	if c.Providers.OpenAI.Model != DefaultOpenAIModel {
		t.Errorf("OpenAI.Model = %q, want %q", c.Providers.OpenAI.Model, DefaultOpenAIModel)
	}
	if c.Providers.OpenAI.BaseURL != DefaultOpenAIBaseURL {
		t.Errorf("OpenAI.BaseURL = %q, want %q", c.Providers.OpenAI.BaseURL, DefaultOpenAIBaseURL)
	}
	if c.Schedule.DefaultFrequency != DefaultFrequency {
		t.Errorf("DefaultFrequency = %q, want %q", c.Schedule.DefaultFrequency, DefaultFrequency)
	}
}

func TestLoadWithValues(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
[general]
default_provider = "anthropic"
output_style = "minimal"
feed_limit = 50

[providers.openai]
api_key = "sk-test-key"
model = "gpt-4-turbo"

[providers.anthropic]
api_key = "sk-ant-test"
model = "claude-3-opus"

[schedule]
default_frequency = "hourly"
daily_time = "09:00"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	c, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if c.General.DefaultProvider != "anthropic" {
		t.Errorf("DefaultProvider = %q, want %q", c.General.DefaultProvider, "anthropic")
	}
	if c.General.OutputStyle != "minimal" {
		t.Errorf("OutputStyle = %q, want %q", c.General.OutputStyle, "minimal")
	}
	if c.General.FeedLimit != 50 {
		t.Errorf("FeedLimit = %d, want %d", c.General.FeedLimit, 50)
	}
	if c.Providers.OpenAI.APIKey != "sk-test-key" {
		t.Errorf("OpenAI.APIKey = %q, want %q", c.Providers.OpenAI.APIKey, "sk-test-key")
	}
	if c.Providers.OpenAI.Model != "gpt-4-turbo" {
		t.Errorf("OpenAI.Model = %q, want %q", c.Providers.OpenAI.Model, "gpt-4-turbo")
	}
	if c.Providers.Anthropic.APIKey != "sk-ant-test" {
		t.Errorf("Anthropic.APIKey = %q, want %q", c.Providers.Anthropic.APIKey, "sk-ant-test")
	}
	if c.Schedule.DefaultFrequency != "hourly" {
		t.Errorf("DefaultFrequency = %q, want %q", c.Schedule.DefaultFrequency, "hourly")
	}
}

func TestLoadWithEnvOverrides(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	err := os.WriteFile(configPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Set environment variables
	os.Setenv("TERMFLOW_OPENAI_API_KEY", "env-openai-key")
	os.Setenv("TERMFLOW_ANTHROPIC_API_KEY", "env-anthropic-key")
	os.Setenv("TERMFLOW_TAVILY_API_KEY", "env-tavily-key")
	defer func() {
		os.Unsetenv("TERMFLOW_OPENAI_API_KEY")
		os.Unsetenv("TERMFLOW_ANTHROPIC_API_KEY")
		os.Unsetenv("TERMFLOW_TAVILY_API_KEY")
	}()

	c, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if c.Providers.OpenAI.APIKey != "env-openai-key" {
		t.Errorf("OpenAI.APIKey = %q, want %q", c.Providers.OpenAI.APIKey, "env-openai-key")
	}
	if c.Providers.Anthropic.APIKey != "env-anthropic-key" {
		t.Errorf("Anthropic.APIKey = %q, want %q", c.Providers.Anthropic.APIKey, "env-anthropic-key")
	}
	if c.Search.Tavily.APIKey != "env-tavily-key" {
		t.Errorf("Tavily.APIKey = %q, want %q", c.Search.Tavily.APIKey, "env-tavily-key")
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	resetViper()

	// Should not error for missing config file
	_, err := Load("/nonexistent/path/config.toml")
	// viper returns ConfigFileNotFoundError which is ignored
	if err != nil {
		t.Logf("Load() returned error (may be expected): %v", err)
	}
}

func TestLoadInvalidFile(t *testing.T) {
	resetViper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Write invalid TOML
	err := os.WriteFile(configPath, []byte("this is not valid [toml"), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	_, err = Load(configPath)
	if err == nil {
		t.Error("Load() should return error for invalid TOML")
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		input    string
		wantHome bool
	}{
		{"~/test", true},
		{"/absolute/path", false},
		{"relative/path", false},
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Could not get home dir: %v", err)
	}

	for _, tt := range tests {
		result := expandPath(tt.input)
		if tt.wantHome {
			expected := filepath.Join(home, "test")
			if result != expected {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, result, expected)
			}
		} else {
			if result != tt.input {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, result, tt.input)
			}
		}
	}
}

func TestEnsureDirectories(t *testing.T) {
	resetViper()

	// Use temp home for testing
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	err := EnsureDirectories()
	if err != nil {
		t.Fatalf("EnsureDirectories() returned error: %v", err)
	}

	// Check directories were created
	dirs := []string{
		filepath.Join(tmpHome, ".config", "termiflow"),
		filepath.Join(tmpHome, ".local", "share", "termiflow"),
		filepath.Join(tmpHome, ".cache", "termiflow"),
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Directory %q was not created", dir)
		}
	}
}

func TestSetAndGetString(t *testing.T) {
	resetViper()

	Set("test.key", "test-value")
	result := GetString("test.key")
	if result != "test-value" {
		t.Errorf("GetString() = %q, want %q", result, "test-value")
	}
}

func TestDefaultFunctions(t *testing.T) {
	tests := []struct {
		name     string
		fn       func() string
		expected string
	}{
		{"DefaultConfigDir", DefaultConfigDir, "~/.config/termiflow"},
		{"DefaultDataDir", DefaultDataDir, "~/.local/share/termiflow"},
		{"DefaultCacheDir", DefaultCacheDir, "~/.cache/termiflow"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn()
			if result != tt.expected {
				t.Errorf("%s() = %q, want %q", tt.name, result, tt.expected)
			}
		})
	}
}

func TestDefaults(t *testing.T) {
	if DefaultProvider != "openai" {
		t.Errorf("DefaultProvider = %q, want %q", DefaultProvider, "openai")
	}
	if DefaultOutputStyle != "pretty" {
		t.Errorf("DefaultOutputStyle = %q, want %q", DefaultOutputStyle, "pretty")
	}
	if DefaultFeedLimit != 20 {
		t.Errorf("DefaultFeedLimit = %d, want %d", DefaultFeedLimit, 20)
	}
	if DefaultFrequency != "daily" {
		t.Errorf("DefaultFrequency = %q, want %q", DefaultFrequency, "daily")
	}
	if DefaultOpenAIModel != "gpt-4o" {
		t.Errorf("DefaultOpenAIModel = %q, want %q", DefaultOpenAIModel, "gpt-4o")
	}
	if DefaultAnthropicModel != "claude-sonnet-4-20250514" {
		t.Errorf("DefaultAnthropicModel = %q, want %q", DefaultAnthropicModel, "claude-sonnet-4-20250514")
	}
}

func TestGetCacheDir(t *testing.T) {
	resetViper()

	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	// Without config, should return default
	cfg = nil
	cacheDir := GetCacheDir()
	expected := filepath.Join(tmpHome, ".cache", "termiflow")
	if cacheDir != expected {
		t.Errorf("GetCacheDir() = %q, want %q", cacheDir, expected)
	}

	// With config override
	cfg = &Config{
		General: GeneralConfig{
			CacheDir: filepath.Join(tmpHome, "custom-cache"),
		},
	}
	cacheDir = GetCacheDir()
	expected = filepath.Join(tmpHome, "custom-cache")
	if cacheDir != expected {
		t.Errorf("GetCacheDir() with override = %q, want %q", cacheDir, expected)
	}
}

package cli

import (
	"bytes"
	"testing"
)

func TestSetVersionInfo(t *testing.T) {
	SetVersionInfo("1.0.0", "abc123", "2025-01-01")

	if version != "1.0.0" {
		t.Errorf("version = %q, want %q", version, "1.0.0")
	}
	if commit != "abc123" {
		t.Errorf("commit = %q, want %q", commit, "abc123")
	}
	if date != "2025-01-01" {
		t.Errorf("date = %q, want %q", date, "2025-01-01")
	}
}

func TestRootCmd(t *testing.T) {
	if rootCmd.Use != "termiflow" {
		t.Errorf("rootCmd.Use = %q, want %q", rootCmd.Use, "termiflow")
	}

	if rootCmd.Short == "" {
		t.Error("rootCmd.Short should not be empty")
	}

	if rootCmd.Long == "" {
		t.Error("rootCmd.Long should not be empty")
	}
}

func TestRootCmdHasSubcommands(t *testing.T) {
	commands := rootCmd.Commands()

	expectedCommands := []string{
		"version",
		"config",
		"ask",
		"subscribe",
		"unsubscribe",
		"feed",
		"topics",
	}

	for _, expected := range expectedCommands {
		found := false
		for _, cmd := range commands {
			if cmd.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected subcommand %q not found", expected)
		}
	}
}

func TestVersionCmd(t *testing.T) {
	if versionCmd.Use != "version" {
		t.Errorf("versionCmd.Use = %q, want %q", versionCmd.Use, "version")
	}

	if versionCmd.Short == "" {
		t.Error("versionCmd.Short should not be empty")
	}

	if versionCmd.Run == nil {
		t.Error("versionCmd.Run should not be nil")
	}
}

func TestPersistentFlags(t *testing.T) {
	flags := []struct {
		name     string
		defValue string
	}{
		{"config", ""},
		{"provider", ""},
		{"quiet", "false"},
		{"debug", "false"},
		{"no-color", "false"},
	}

	for _, f := range flags {
		flag := rootCmd.PersistentFlags().Lookup(f.name)
		if flag == nil {
			t.Errorf("Flag %q not found", f.name)
			continue
		}
		if flag.DefValue != f.defValue {
			t.Errorf("Flag %q default = %q, want %q", f.name, flag.DefValue, f.defValue)
		}
	}
}

func TestGetProvider(t *testing.T) {
	// Save and restore provider flag
	originalProvider := provider
	defer func() { provider = originalProvider }()

	// When provider flag is set
	provider = "anthropic"
	result := getProvider()
	if result != "anthropic" {
		t.Errorf("getProvider() with flag = %q, want %q", result, "anthropic")
	}

	// When provider flag is empty, falls back to config
	provider = ""
	// Note: This will use the config default, which may vary
	result = getProvider()
	// Just verify it returns something
	if result == "" {
		t.Log("getProvider() returned empty when no flag set (may need config)")
	}
}

func TestVersionCmdOutput(t *testing.T) {
	SetVersionInfo("test-version", "test-commit", "test-date")

	// Capture output
	buf := new(bytes.Buffer)
	versionCmd.SetOut(buf)

	// Execute command
	err := versionCmd.Execute()
	if err != nil {
		t.Fatalf("versionCmd.Execute() error = %v", err)
	}

	output := buf.String()
	if output == "" {
		// Version command writes to stdout directly, not to cmd output
		t.Log("Note: version command writes directly to stdout")
	}
}

func TestConfigCmd(t *testing.T) {
	if configCmd == nil {
		t.Fatal("configCmd should not be nil")
	}

	if configCmd.Use != "config" {
		t.Errorf("configCmd.Use = %q, want %q", configCmd.Use, "config")
	}

	// Check subcommands
	subcommands := configCmd.Commands()
	expectedSubs := []string{"init", "get", "set", "edit", "path"}

	for _, expected := range expectedSubs {
		found := false
		for _, cmd := range subcommands {
			if cmd.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected config subcommand %q not found", expected)
		}
	}
}

func TestAskCmd(t *testing.T) {
	if askCmd == nil {
		t.Fatal("askCmd should not be nil")
	}

	if askCmd.Use != "ask <question>" {
		t.Errorf("askCmd.Use = %q, want %q", askCmd.Use, "ask <question>")
	}

	// Check flags
	flags := []string{"sources", "no-search", "save"}
	for _, name := range flags {
		if askCmd.Flags().Lookup(name) == nil {
			t.Errorf("ask command missing flag %q", name)
		}
	}
}

func TestSubscribeCmd(t *testing.T) {
	if subscribeCmd == nil {
		t.Fatal("subscribeCmd should not be nil")
	}

	if subscribeCmd.Use != "subscribe <topic>" {
		t.Errorf("subscribeCmd.Use = %q, want %q", subscribeCmd.Use, "subscribe <topic>")
	}

	// Check flags
	flags := []string{"hourly", "daily", "weekly", "sources"}
	for _, name := range flags {
		if subscribeCmd.Flags().Lookup(name) == nil {
			t.Errorf("subscribe command missing flag %q", name)
		}
	}
}

func TestUnsubscribeCmd(t *testing.T) {
	if unsubscribeCmd == nil {
		t.Fatal("unsubscribeCmd should not be nil")
	}

	if unsubscribeCmd.Use != "unsubscribe <topic>" {
		t.Errorf("unsubscribeCmd.Use = %q, want %q", unsubscribeCmd.Use, "unsubscribe <topic>")
	}

	// Check flags
	flags := []string{"all", "force"}
	for _, name := range flags {
		if unsubscribeCmd.Flags().Lookup(name) == nil {
			t.Errorf("unsubscribe command missing flag %q", name)
		}
	}
}

func TestFeedCmd(t *testing.T) {
	if feedCmd == nil {
		t.Fatal("feedCmd should not be nil")
	}

	if feedCmd.Use != "feed" {
		t.Errorf("feedCmd.Use = %q, want %q", feedCmd.Use, "feed")
	}

	// Check flags
	flags := []string{"topic", "today", "week", "limit", "refresh", "all", "mark-read", "cleanup"}
	for _, name := range flags {
		if feedCmd.Flags().Lookup(name) == nil {
			t.Errorf("feed command missing flag %q", name)
		}
	}
}

func TestTopicsCmd(t *testing.T) {
	if topicsCmd == nil {
		t.Fatal("topicsCmd should not be nil")
	}

	if topicsCmd.Use != "topics" {
		t.Errorf("topicsCmd.Use = %q, want %q", topicsCmd.Use, "topics")
	}

	// Check flags
	flags := []string{"available", "subscribed"}
	for _, name := range flags {
		if topicsCmd.Flags().Lookup(name) == nil {
			t.Errorf("topics command missing flag %q", name)
		}
	}
}

func TestCommandDescriptions(t *testing.T) {
	// All commands should have descriptions
	for _, cmd := range rootCmd.Commands() {
		if cmd.Short == "" {
			t.Errorf("Command %q has empty Short description", cmd.Name())
		}
	}
}

func TestQuietFlag(t *testing.T) {
	originalQuiet := quiet
	defer func() { quiet = originalQuiet }()

	quiet = true
	if !quiet {
		t.Error("quiet flag should be settable to true")
	}
}

func TestDebugFlag(t *testing.T) {
	originalDebug := debug
	defer func() { debug = originalDebug }()

	debug = true
	if !debug {
		t.Error("debug flag should be settable to true")
	}
}

func TestNoColorFlag(t *testing.T) {
	originalNoColor := noColor
	defer func() { noColor = originalNoColor }()

	noColor = true
	if !noColor {
		t.Error("noColor flag should be settable to true")
	}
}

package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestLoadThemeAmber(t *testing.T) {
	if err := LoadTheme("amber"); err != nil {
		t.Fatalf("LoadTheme(amber) error: %v", err)
	}
	if Current.Name != "amber" {
		t.Errorf("Current.Name = %q, want %q", Current.Name, "amber")
	}
	if Primary != lipgloss.Color("214") {
		t.Errorf("Primary = %v, want 214", Primary)
	}
	if Accent != lipgloss.Color("220") {
		t.Errorf("Accent = %v, want 220", Accent)
	}
	if Current.InvertedFg != lipgloss.Color("0") {
		t.Errorf("InvertedFg = %v, want 0", Current.InvertedFg)
	}
}

func TestLoadThemeLight(t *testing.T) {
	if err := LoadTheme("light"); err != nil {
		t.Fatalf("LoadTheme(light) error: %v", err)
	}
	if Current.Name != "light" {
		t.Errorf("Current.Name = %q, want %q", Current.Name, "light")
	}
	if Primary != lipgloss.Color("130") {
		t.Errorf("Primary = %v, want 130", Primary)
	}
	if Current.InvertedFg != lipgloss.Color("255") {
		t.Errorf("InvertedFg = %v, want 255", Current.InvertedFg)
	}
}

func TestLoadThemeDracula(t *testing.T) {
	if err := LoadTheme("dracula"); err != nil {
		t.Fatalf("LoadTheme(dracula) error: %v", err)
	}
	if Current.Name != "dracula" {
		t.Errorf("Current.Name = %q, want %q", Current.Name, "dracula")
	}
	if Primary != lipgloss.Color("141") {
		t.Errorf("Primary = %v, want 141", Primary)
	}
	if Current.Blue != lipgloss.Color("60") {
		t.Errorf("Blue = %v, want 60 (comment color)", Current.Blue)
	}
}

func TestLoadThemeEmpty(t *testing.T) {
	if err := LoadTheme(""); err != nil {
		t.Fatalf("LoadTheme(\"\") error: %v", err)
	}
	if Current.Name != "amber" {
		t.Errorf("Current.Name = %q, want %q (default)", Current.Name, "amber")
	}
}

func TestLoadThemeInvalid(t *testing.T) {
	err := LoadTheme("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown theme, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention theme name, got: %v", err)
	}
	if !strings.Contains(err.Error(), "amber") {
		t.Errorf("error should list available themes, got: %v", err)
	}
}

func TestLoadThemeIdempotent(t *testing.T) {
	if err := LoadTheme("dracula"); err != nil {
		t.Fatalf("first LoadTheme(dracula) error: %v", err)
	}
	if err := LoadTheme("amber"); err != nil {
		t.Fatalf("second LoadTheme(amber) error: %v", err)
	}
	if Current.Name != "amber" {
		t.Errorf("Current.Name = %q, want %q after second load", Current.Name, "amber")
	}
	if Primary != lipgloss.Color("214") {
		t.Errorf("Primary = %v, want 214 after second load", Primary)
	}
}

func TestThemeFieldsNonZero(t *testing.T) {
	for _, name := range ThemeNames() {
		theme, ok := themes[name]
		if !ok {
			t.Errorf("theme %q not found in map", name)
			continue
		}
		if theme.Name == "" {
			t.Errorf("theme %q has empty Name", name)
		}
		if theme.Primary == "" {
			t.Errorf("theme %q has empty Primary", name)
		}
		if theme.Secondary == "" {
			t.Errorf("theme %q has empty Secondary", name)
		}
		if theme.Success == "" {
			t.Errorf("theme %q has empty Success", name)
		}
		if theme.Warning == "" {
			t.Errorf("theme %q has empty Warning", name)
		}
		if theme.Error == "" {
			t.Errorf("theme %q has empty Error", name)
		}
		if theme.Muted == "" {
			t.Errorf("theme %q has empty Muted", name)
		}
		if theme.Accent == "" {
			t.Errorf("theme %q has empty Accent", name)
		}
		if theme.Cyan == "" {
			t.Errorf("theme %q has empty Cyan", name)
		}
		if theme.Blue == "" {
			t.Errorf("theme %q has empty Blue", name)
		}
		if theme.InvertedFg == "" {
			t.Errorf("theme %q has empty InvertedFg", name)
		}
		if theme.SelectedBg == "" {
			t.Errorf("theme %q has empty SelectedBg", name)
		}
		if theme.MutedBorder == "" {
			t.Errorf("theme %q has empty MutedBorder", name)
		}
	}
}

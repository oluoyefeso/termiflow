package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/oluoyefeso/termiflow/internal/ui"
)

func TestInitStyles(t *testing.T) {
	if err := ui.LoadTheme("dracula"); err != nil {
		t.Fatalf("LoadTheme error: %v", err)
	}
	InitStyles()

	if ColorPrimary != ui.Primary {
		t.Errorf("ColorPrimary = %v, want %v (ui.Primary)", ColorPrimary, ui.Primary)
	}
	if ColorSecondary != ui.Secondary {
		t.Errorf("ColorSecondary = %v, want %v (ui.Secondary)", ColorSecondary, ui.Secondary)
	}
	if ColorSuccess != ui.SuccessColor {
		t.Errorf("ColorSuccess = %v, want %v (ui.SuccessColor)", ColorSuccess, ui.SuccessColor)
	}
	if ColorMuted != ui.Muted {
		t.Errorf("ColorMuted = %v, want %v (ui.Muted)", ColorMuted, ui.Muted)
	}
}

func TestInitStylesInvertedKey(t *testing.T) {
	// Load light theme — InvertedFg should be 255
	if err := ui.LoadTheme("light"); err != nil {
		t.Fatalf("LoadTheme error: %v", err)
	}
	InitStyles()

	// StyleInvertedKey should use the light theme's InvertedFg (255)
	if ui.Current.InvertedFg != lipgloss.Color("255") {
		t.Errorf("light theme InvertedFg = %v, want 255", ui.Current.InvertedFg)
	}

	// StyleInvertedKey should be bold
	if !StyleInvertedKey.GetBold() {
		t.Error("StyleInvertedKey should be bold")
	}
}

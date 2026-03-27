package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/oluoyefeso/termiflow/internal/ui"
)

// KeyHint represents a single keybinding hint for the status bar.
type KeyHint struct {
	Key  string
	Desc string
}

// StatusBar renders a bottom bar with keybinding hints.
type StatusBar struct {
	Hints []KeyHint
	Width int
}

// NewStatusBar creates a status bar with the given hints.
func NewStatusBar(hints []KeyHint, width int) StatusBar {
	return StatusBar{Hints: hints, Width: width}
}

// View renders the status bar.
func (s StatusBar) View() string {
	if len(s.Hints) == 0 {
		return ""
	}

	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.Primary)

	descStyle := lipgloss.NewStyle().
		Foreground(ui.Muted)

	var parts []string
	for _, h := range s.Hints {
		parts = append(parts, keyStyle.Render("["+h.Key+"]")+" "+descStyle.Render(h.Desc))
	}

	content := strings.Join(parts, "  ")

	bar := lipgloss.NewStyle().
		Width(s.Width).
		Padding(0, 1)

	separator := ui.MutedStyle.Render(strings.Repeat("─", s.Width))

	return separator + "\n" + bar.Render(content)
}

package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Bloomberg terminal palette — matches internal/ui/colors.go
var (
	ColorPrimary   = lipgloss.Color("214") // Amber
	ColorSecondary = lipgloss.Color("244") // Light gray
	ColorSuccess   = lipgloss.Color("82")  // Green
	ColorWarning   = lipgloss.Color("208") // Orange
	ColorError     = lipgloss.Color("196") // Red
	ColorMuted     = lipgloss.Color("240") // Dark gray
	ColorAccent    = lipgloss.Color("220") // Bright amber
	ColorCyan      = lipgloss.Color("51")  // Info banners
	ColorBlue      = lipgloss.Color("33")  // Update banners

	// Text styles
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	StyleAccent = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent)

	StyleMuted = lipgloss.NewStyle().
			Foreground(ColorMuted)

	StyleSuccess = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	StyleWarning = lipgloss.NewStyle().
			Foreground(ColorWarning)

	StyleError = lipgloss.NewStyle().
			Foreground(ColorError)

	StyleCyan = lipgloss.NewStyle().
			Foreground(ColorCyan)

	StyleBlue = lipgloss.NewStyle().
			Foreground(ColorBlue)

	// Selected item in lists
	StyleSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent)

	StyleSelectedIndicator = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary)

	// Unread badge
	StyleUnreadBadge = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary)

	// Tag style
	StyleTag = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	// Inverted key style for status bar (amber bg, black fg)
	StyleInvertedKey = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("0")).
				Background(ColorPrimary)

	// Selected item background
	StyleSelectedBg = lipgloss.NewStyle().
			Background(lipgloss.Color("236"))

	// Dotted separator
	StyleDotSeparator = lipgloss.NewStyle().
				Foreground(ColorMuted)
)

// HealthDot returns a colored dot for the given health status.
func HealthDot(status string) string {
	switch status {
	case "ok":
		return StyleSuccess.Render("●")
	case "degraded":
		return StyleError.Render("●")
	default:
		return StyleMuted.Render("●")
	}
}

// Bar returns a horizontal rule of the given character repeated width times.
func Bar(ch string, width int) string {
	if width <= 0 {
		return ""
	}
	return strings.Repeat(ch, width)
}

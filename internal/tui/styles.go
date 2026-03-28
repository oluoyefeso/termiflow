package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Termiflow palette — warm amber on dark, matching termiflow.com (#f5a623)
var (
	ColorPrimary   = lipgloss.Color("214") // Amber (#f5a623)
	ColorSecondary = lipgloss.Color("180") // Warm tan (#d7c3ae)
	ColorSuccess   = lipgloss.Color("78")  // Green (#22c55e)
	ColorWarning   = lipgloss.Color("208") // Orange
	ColorError     = lipgloss.Color("196") // Red
	ColorMuted     = lipgloss.Color("242") // Dark gray (#666666)
	ColorAccent    = lipgloss.Color("220") // Bright amber (#ffb955)
	ColorCyan      = lipgloss.Color("39")  // Cyan (#3ac2ff)
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

	// Warm muted for secondary text (source, time, descriptions)
	StyleWarmMuted = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	// Italic warm muted for summaries/descriptions
	StyleSummary = lipgloss.NewStyle().
			Italic(true).
			Foreground(ColorSecondary)

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

	// Tag style — uppercase, no brackets
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

	// Left border accent for unread/active items
	StyleLeftBorder = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	// Left border muted for read items
	StyleLeftBorderMuted = lipgloss.NewStyle().
				Foreground(lipgloss.Color("238"))

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

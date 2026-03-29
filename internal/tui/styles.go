package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/oluoyefeso/termiflow/internal/ui"
)

// Color aliases — set by InitStyles() from ui package vars.
var (
	ColorPrimary   lipgloss.Color
	ColorSecondary lipgloss.Color
	ColorSuccess   lipgloss.Color
	ColorWarning   lipgloss.Color
	ColorError     lipgloss.Color
	ColorMuted     lipgloss.Color
	ColorAccent    lipgloss.Color
	ColorCyan      lipgloss.Color
	ColorBlue      lipgloss.Color
)

// Style vars — recomputed by InitStyles().
var (
	StyleTitle             lipgloss.Style
	StyleAccent            lipgloss.Style
	StyleMuted             lipgloss.Style
	StyleWarmMuted         lipgloss.Style
	StyleSummary           lipgloss.Style
	StyleSuccess           lipgloss.Style
	StyleWarning           lipgloss.Style
	StyleError             lipgloss.Style
	StyleCyan              lipgloss.Style
	StyleBlue              lipgloss.Style
	StyleSelected          lipgloss.Style
	StyleSelectedIndicator lipgloss.Style
	StyleUnreadBadge       lipgloss.Style
	StyleTag               lipgloss.Style
	StyleInvertedKey       lipgloss.Style
	StyleSelectedBg        lipgloss.Style
	StyleLeftBorder        lipgloss.Style
	StyleLeftBorderMuted   lipgloss.Style
	StyleDotSeparator      lipgloss.Style
)

// InitStyles aliases tui color vars from ui and recomputes all tui styles.
// Must be called after ui.LoadTheme().
func InitStyles() {
	ColorPrimary = ui.Primary
	ColorSecondary = ui.Secondary
	ColorSuccess = ui.SuccessColor
	ColorWarning = ui.WarningColor
	ColorError = ui.ErrorColor
	ColorMuted = ui.Muted
	ColorAccent = ui.Accent
	ColorCyan = ui.CyanColor
	ColorBlue = ui.BlueColor

	StyleTitle = lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
	StyleAccent = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	StyleMuted = lipgloss.NewStyle().Foreground(ColorMuted)
	StyleWarmMuted = lipgloss.NewStyle().Foreground(ColorSecondary)
	StyleSummary = lipgloss.NewStyle().Italic(true).Foreground(ColorSecondary)
	StyleSuccess = lipgloss.NewStyle().Foreground(ColorSuccess)
	StyleWarning = lipgloss.NewStyle().Foreground(ColorWarning)
	StyleError = lipgloss.NewStyle().Foreground(ColorError)
	StyleCyan = lipgloss.NewStyle().Foreground(ColorCyan)
	StyleBlue = lipgloss.NewStyle().Foreground(ColorBlue)
	StyleSelected = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	StyleSelectedIndicator = lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
	StyleUnreadBadge = lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
	StyleTag = lipgloss.NewStyle().Foreground(ColorPrimary)

	StyleInvertedKey = lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.Current.InvertedFg).
		Background(ColorPrimary)

	StyleSelectedBg = lipgloss.NewStyle().
		Background(ui.Current.SelectedBg)

	StyleLeftBorder = lipgloss.NewStyle().Foreground(ColorPrimary)
	StyleLeftBorderMuted = lipgloss.NewStyle().Foreground(ui.Current.MutedBorder)
	StyleDotSeparator = lipgloss.NewStyle().Foreground(ColorMuted)
}

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

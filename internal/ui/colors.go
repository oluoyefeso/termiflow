package ui

import "github.com/charmbracelet/lipgloss"

// Color vars — populated by LoadTheme(). Zero values until a theme is loaded.
var (
	Primary      lipgloss.Color
	Secondary    lipgloss.Color
	SuccessColor lipgloss.Color
	WarningColor lipgloss.Color
	ErrorColor   lipgloss.Color
	Muted        lipgloss.Color
	Accent       lipgloss.Color
	CyanColor    lipgloss.Color
	BlueColor    lipgloss.Color
)

// Style vars — recomputed by LoadTheme().
var (
	TitleStyle     lipgloss.Style
	SubtitleStyle  lipgloss.Style
	SuccessStyle   lipgloss.Style
	ErrorStyle     lipgloss.Style
	WarningStyle   lipgloss.Style
	MutedStyle     lipgloss.Style
	WarmMutedStyle lipgloss.Style
	BoldStyle      lipgloss.Style
	AccentStyle    lipgloss.Style
	CyanStyle      lipgloss.Style
	BlueStyle      lipgloss.Style
	TagStyle       lipgloss.Style
	BoxStyle       lipgloss.Style
	HeaderBoxStyle lipgloss.Style
)

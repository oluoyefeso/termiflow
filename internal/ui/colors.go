package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Primary colors
	Primary      = lipgloss.Color("39")  // Blue
	Secondary    = lipgloss.Color("245") // Gray
	SuccessColor = lipgloss.Color("82")  // Green
	WarningColor = lipgloss.Color("214") // Orange
	ErrorColor   = lipgloss.Color("196") // Red
	Muted        = lipgloss.Color("241") // Dark gray

	// Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(Secondary)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(SuccessColor)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor)

	WarningStyle = lipgloss.NewStyle().
			Foreground(WarningColor)

	MutedStyle = lipgloss.NewStyle().
			Foreground(Muted)

	BoldStyle = lipgloss.NewStyle().
			Bold(true)

	TagStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Secondary).
			Padding(0, 1)

	HeaderBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(0, 1).
			Width(67)
)

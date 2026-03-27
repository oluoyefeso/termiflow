package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Bloomberg terminal palette: amber on dark, dense and functional
	Primary      = lipgloss.Color("214") // Amber
	Secondary    = lipgloss.Color("244") // Light gray
	SuccessColor = lipgloss.Color("82")  // Green
	WarningColor = lipgloss.Color("208") // Orange
	ErrorColor   = lipgloss.Color("196") // Red
	Muted        = lipgloss.Color("240") // Dark gray
	Accent       = lipgloss.Color("220") // Bright amber (headlines)

	// Text styles
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

	AccentStyle = lipgloss.NewStyle().
			Foreground(Accent).
			Bold(true)

	// Banner colors
	CyanColor = lipgloss.Color("51")  // Cyan (info banners)
	BlueColor = lipgloss.Color("33")  // Blue (update/version banners)

	CyanStyle = lipgloss.NewStyle().
			Foreground(CyanColor)

	BlueStyle = lipgloss.NewStyle().
			Foreground(BlueColor)

	// Tags: compact bracket style
	TagStyle = lipgloss.NewStyle().
			Foreground(Primary)

	// Header: plain border (Bloomberg uses flat lines, not rounded boxes)
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(Muted).
			Padding(0, 1)

	HeaderBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, false).
			Padding(0, 0).
			Width(69)
)

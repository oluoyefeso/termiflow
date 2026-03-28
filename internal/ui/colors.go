package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Termiflow palette — warm amber on dark, matching termiflow.com (#f5a623)
	Primary      = lipgloss.Color("214") // Amber (#f5a623)
	Secondary    = lipgloss.Color("180") // Warm tan (#d7c3ae)
	SuccessColor = lipgloss.Color("78")  // Green (#22c55e)
	WarningColor = lipgloss.Color("208") // Orange
	ErrorColor   = lipgloss.Color("196") // Red
	Muted        = lipgloss.Color("242") // Dark gray (#666666)
	Accent       = lipgloss.Color("220") // Bright amber (#ffb955)

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

	// Warm muted for secondary text (source, time, descriptions)
	WarmMutedStyle = lipgloss.NewStyle().
			Foreground(Secondary)

	BoldStyle = lipgloss.NewStyle().
			Bold(true)

	AccentStyle = lipgloss.NewStyle().
			Foreground(Accent).
			Bold(true)

	// Banner colors
	CyanColor = lipgloss.Color("39") // Cyan (#3ac2ff)
	BlueColor = lipgloss.Color("33") // Blue (update/version banners)

	CyanStyle = lipgloss.NewStyle().
			Foreground(CyanColor)

	BlueStyle = lipgloss.NewStyle().
			Foreground(BlueColor)

	// Tags: uppercase, no brackets
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

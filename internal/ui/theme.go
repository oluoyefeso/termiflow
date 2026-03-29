package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
)

// Theme defines a complete color palette for the terminal UI.
type Theme struct {
	Name        string
	Primary     lipgloss.Color
	Secondary   lipgloss.Color
	Success     lipgloss.Color
	Warning     lipgloss.Color
	Error       lipgloss.Color
	Muted       lipgloss.Color
	Accent      lipgloss.Color
	Cyan        lipgloss.Color
	Blue        lipgloss.Color
	InvertedFg  lipgloss.Color
	SelectedBg  lipgloss.Color
	MutedBorder lipgloss.Color
}

// Current holds the active theme, set by LoadTheme.
var Current Theme

// cachedGlamourStyle holds the glamour ansi.StyleConfig for the active theme.
var cachedGlamourStyle *ansi.StyleConfig

// themes maps theme names to their palette definitions.
var themes = map[string]Theme{
	"amber": {
		Name:        "amber",
		Primary:     lipgloss.Color("214"),
		Secondary:   lipgloss.Color("180"),
		Success:     lipgloss.Color("78"),
		Warning:     lipgloss.Color("208"),
		Error:       lipgloss.Color("196"),
		Muted:       lipgloss.Color("242"),
		Accent:      lipgloss.Color("220"),
		Cyan:        lipgloss.Color("39"),
		Blue:        lipgloss.Color("33"),
		InvertedFg:  lipgloss.Color("0"),
		SelectedBg:  lipgloss.Color("236"),
		MutedBorder: lipgloss.Color("238"),
	},
	"light": {
		Name:        "light",
		Primary:     lipgloss.Color("130"),
		Secondary:   lipgloss.Color("95"),
		Success:     lipgloss.Color("28"),
		Warning:     lipgloss.Color("166"),
		Error:       lipgloss.Color("124"),
		Muted:       lipgloss.Color("245"),
		Accent:      lipgloss.Color("172"),
		Cyan:        lipgloss.Color("30"),
		Blue:        lipgloss.Color("25"),
		InvertedFg:  lipgloss.Color("255"),
		SelectedBg:  lipgloss.Color("253"),
		MutedBorder: lipgloss.Color("250"),
	},
	"dracula": {
		Name:        "dracula",
		Primary:     lipgloss.Color("141"),
		Secondary:   lipgloss.Color("189"),
		Success:     lipgloss.Color("84"),
		Warning:     lipgloss.Color("215"),
		Error:       lipgloss.Color("210"),
		Muted:       lipgloss.Color("60"),
		Accent:      lipgloss.Color("212"),
		Cyan:        lipgloss.Color("117"),
		Blue:        lipgloss.Color("60"),
		InvertedFg:  lipgloss.Color("0"),
		SelectedBg:  lipgloss.Color("237"),
		MutedBorder: lipgloss.Color("60"),
	},
}

// ThemeNames returns the sorted list of available theme names.
func ThemeNames() []string {
	names := make([]string, 0, len(themes))
	for name := range themes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ValidateThemeName checks if a theme name is valid without mutating global state.
func ValidateThemeName(name string) error {
	if name == "" {
		return nil
	}
	if _, ok := themes[name]; !ok {
		return fmt.Errorf("unknown theme %q (available: %s)", name, strings.Join(ThemeNames(), ", "))
	}
	return nil
}

// LoadTheme sets the active theme and recomputes all style variables.
// An empty name defaults to "amber". Returns an error for unknown names.
func LoadTheme(name string) error {
	if name == "" {
		name = "amber"
	}

	t, ok := themes[name]
	if !ok {
		return fmt.Errorf("unknown theme %q (available: %s)", name, strings.Join(ThemeNames(), ", "))
	}

	Current = t

	// Reassign color vars
	Primary = t.Primary
	Secondary = t.Secondary
	SuccessColor = t.Success
	WarningColor = t.Warning
	ErrorColor = t.Error
	Muted = t.Muted
	Accent = t.Accent
	CyanColor = t.Cyan
	BlueColor = t.Blue

	// Recompute style vars
	TitleStyle = lipgloss.NewStyle().Bold(true).Foreground(t.Primary)
	SubtitleStyle = lipgloss.NewStyle().Foreground(t.Secondary)
	SuccessStyle = lipgloss.NewStyle().Foreground(t.Success)
	ErrorStyle = lipgloss.NewStyle().Foreground(t.Error)
	WarningStyle = lipgloss.NewStyle().Foreground(t.Warning)
	MutedStyle = lipgloss.NewStyle().Foreground(t.Muted)
	WarmMutedStyle = lipgloss.NewStyle().Foreground(t.Secondary)
	BoldStyle = lipgloss.NewStyle().Bold(true)
	AccentStyle = lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	CyanStyle = lipgloss.NewStyle().Foreground(t.Cyan)
	BlueStyle = lipgloss.NewStyle().Foreground(t.Blue)
	TagStyle = lipgloss.NewStyle().Foreground(t.Primary)
	BoxStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.Muted).
		Padding(0, 1)
	HeaderBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, false).
		Padding(0, 0).
		Width(69)

	// Recompute warmMutedItalic in output.go
	warmMutedItalic = lipgloss.NewStyle().Italic(true).Foreground(t.Secondary)

	// Cache glamour style config
	style := buildGlamourStyle(t)
	cachedGlamourStyle = &style

	return nil
}

// GetGlamourStyle returns the cached glamour style config, or nil if no theme is loaded.
func GetGlamourStyle() *ansi.StyleConfig {
	return cachedGlamourStyle
}

func buildGlamourStyle(t Theme) ansi.StyleConfig {
	primary := string(t.Primary)
	secondary := string(t.Secondary)
	muted := string(t.Muted)
	accent := string(t.Accent)
	cyan := string(t.Cyan)

	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: &secondary,
			},
			Margin: uintPtr(0),
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: &primary,
				Bold:  boolPtr(true),
			},
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: &accent,
				Bold:  boolPtr(true),
			},
		},
		H2: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: &primary,
				Bold:  boolPtr(true),
			},
		},
		Strong: ansi.StylePrimitive{
			Bold:  boolPtr(true),
			Color: &primary,
		},
		Emph: ansi.StylePrimitive{
			Italic: boolPtr(true),
			Color:  &secondary,
		},
		Link: ansi.StylePrimitive{
			Color:     &cyan,
			Underline: boolPtr(true),
		},
		LinkText: ansi.StylePrimitive{
			Color: &primary,
		},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: &accent,
			},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				Margin: uintPtr(1),
			},
			Chroma: &ansi.Chroma{
				Text: ansi.StylePrimitive{Color: &secondary},
			},
		},
		List: ansi.StyleList{
			StyleBlock:  ansi.StyleBlock{},
			LevelIndent: 2,
		},
		Table: ansi.StyleTable{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{
					Color: &secondary,
				},
			},
		},
		HorizontalRule: ansi.StylePrimitive{
			Color: &muted,
		},
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func uintPtr(u uint) *uint {
	return &u
}

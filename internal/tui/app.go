package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AppModel is the root model that routes between screens.
type AppModel struct {
	activeScreen Screen
	dashboard    DashboardModel
	width        int
	height       int
	banners      []BannerMsg
	showHelp     bool
}

// NewAppModel creates the root app model with optional notification banners.
func NewAppModel(banners []BannerMsg) AppModel {
	return AppModel{
		activeScreen: ScreenDashboard,
		dashboard:    NewDashboardModel(),
		banners:      banners,
	}
}

func (m AppModel) Init() tea.Cmd {
	return m.dashboard.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Forward to active screen (currently only dashboard exists)
		if m.activeScreen == ScreenDashboard {
			var cmd tea.Cmd
			m.dashboard, cmd = m.dashboard.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		// Global keys
		if key.Matches(msg, GlobalKeys.Quit) {
			return m, tea.Quit
		}
		if key.Matches(msg, GlobalKeys.Help) {
			m.showHelp = !m.showHelp
			return m, nil
		}
		// When help overlay is shown, consume all other keys
		if m.showHelp {
			return m, nil
		}
		// Escape returns to dashboard from any other screen
		if msg.String() == "esc" && m.activeScreen != ScreenDashboard {
			m.activeScreen = ScreenDashboard
			return m, nil
		}

	case SwitchScreenMsg:
		// Don't switch to screens that aren't implemented yet
		if msg.Screen != ScreenDashboard {
			return m, nil
		}
		m.activeScreen = msg.Screen
		return m, nil

	case BannersLoadedMsg:
		m.banners = msg.Banners
		return m, nil
	}

	// Route to active screen
	var cmd tea.Cmd
	switch m.activeScreen {
	case ScreenDashboard:
		m.dashboard, cmd = m.dashboard.Update(msg)
	}

	return m, cmd
}

func (m AppModel) View() string {
	var b strings.Builder

	// Notification banners at the top
	for _, banner := range m.banners {
		b.WriteString(renderBanner(banner))
	}

	// Help overlay
	if m.showHelp {
		b.WriteString(m.renderHelp())
		return b.String()
	}

	// Active screen
	switch m.activeScreen {
	case ScreenDashboard:
		b.WriteString(m.dashboard.View())
	default:
		b.WriteString(StyleMuted.Render("  Screen not implemented yet"))
	}

	return b.String()
}

func renderBanner(banner BannerMsg) string {
	icon := "▲"
	var s lipgloss.Style

	switch banner.Type {
	case "info":
		s = StyleCyan
	case "warning":
		s = StyleWarning
	case "update", "version":
		s = StyleBlue
	case "breaking":
		icon = "!"
		s = StyleError
	default:
		s = StyleMuted
	}

	return fmt.Sprintf(" %s %s\n", s.Render(icon), s.Render(banner.Message))
}

func (m AppModel) renderHelp() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(StyleAccent.Render("  KEYBINDINGS"))
	b.WriteString("\n\n")

	type helpEntry struct{ key, desc string }

	var entries []helpEntry

	switch m.activeScreen {
	case ScreenDashboard:
		entries = []helpEntry{
			{"j/k, arrows", "navigate subscriptions"},
			{"enter", "open feed for selected topic"},
			{"r", "refresh all feeds"},
			{"?", "toggle this help"},
			{"q, ctrl+c", "quit"},
		}
	}

	for _, e := range entries {
		b.WriteString(fmt.Sprintf("    %-16s %s\n",
			StyleTitle.Render(e.key),
			StyleMuted.Render(e.desc),
		))
	}

	b.WriteString("\n")
	b.WriteString(StyleMuted.Render("  Press ? to close"))
	b.WriteString("\n")

	return b.String()
}

// Run starts the Bubble Tea program.
func Run(banners []BannerMsg) error {
	model := NewAppModel(banners)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

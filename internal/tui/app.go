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
	feed         FeedModel
	detail       DetailModel
	ask          AskModel
	topics       TopicsModel
	status       StatusModel
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
		// Forward to active screen
		var cmd tea.Cmd
		switch m.activeScreen {
		case ScreenDashboard:
			m.dashboard, cmd = m.dashboard.Update(msg)
		case ScreenFeed:
			m.feed, cmd = m.feed.Update(msg)
		case ScreenDetail:
			m.detail, cmd = m.detail.Update(msg)
		case ScreenAsk:
			m.ask, cmd = m.ask.Update(msg)
		case ScreenTopics:
			m.topics, cmd = m.topics.Update(msg)
		case ScreenStatus:
			m.status, cmd = m.status.Update(msg)
		}
		return m, cmd

	case tea.KeyMsg:
		// Skip global keys when a screen is capturing text input
		if m.activeScreen == ScreenFeed && m.feed.filtering {
			var cmd tea.Cmd
			m.feed, cmd = m.feed.Update(msg)
			return m, cmd
		}
		if m.activeScreen == ScreenTopics && (m.topics.freqPicking || m.topics.confirming) {
			// During frequency picker or delete confirmation: route keys to topics
			var cmd tea.Cmd
			m.topics, cmd = m.topics.Update(msg)
			return m, cmd
		}
		if m.activeScreen == ScreenAsk && m.ask.phase != askPhaseDone {
			// During input, searching, and streaming: route all keys to ask screen
			var cmd tea.Cmd
			m.ask, cmd = m.ask.Update(msg)
			return m, cmd
		}
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

	case SwitchScreenMsg:
		switch msg.Screen {
		case ScreenDashboard:
			m.activeScreen = ScreenDashboard
			// Refresh dashboard data when returning
			return m, loadSubscriptions
		case ScreenFeed:
			if msg.Subscription != nil {
				// New feed: initialize from subscription
				m.feed = NewFeedModel(msg.Subscription)
				m.activeScreen = ScreenFeed
				return m, m.feed.Init()
			}
			// Returning from detail: reuse existing feed model
			m.activeScreen = ScreenFeed
		case ScreenAsk:
			// Cancel any inflight stream from a previous ask
			if m.ask.cancel != nil {
				m.ask.cancel()
			}
			m.ask = NewAskModel()
			m.activeScreen = ScreenAsk
			return m, m.ask.Init()
		case ScreenTopics:
			m.topics = NewTopicsModel()
			m.activeScreen = ScreenTopics
			return m, m.topics.Init()
		case ScreenStatus:
			m.status = NewStatusModel()
			m.activeScreen = ScreenStatus
			return m, m.status.Init()
		}
		return m, nil

	case OpenDetailMsg:
		m.detail = NewDetailModel(msg.Item, msg.Items, msg.Index)
		m.activeScreen = ScreenDetail
		return m, m.detail.Init()

	case BannersLoadedMsg:
		m.banners = msg.Banners
		return m, nil

	case AllRefreshDoneMsg, autoRefreshTickMsg:
		// Always route refresh messages to dashboard, regardless of active screen
		var cmd tea.Cmd
		m.dashboard, cmd = m.dashboard.Update(msg)
		return m, cmd
	}

	// Route to active screen
	var cmd tea.Cmd
	switch m.activeScreen {
	case ScreenDashboard:
		m.dashboard, cmd = m.dashboard.Update(msg)
	case ScreenFeed:
		m.feed, cmd = m.feed.Update(msg)
	case ScreenDetail:
		m.detail, cmd = m.detail.Update(msg)
	case ScreenAsk:
		m.ask, cmd = m.ask.Update(msg)
	case ScreenTopics:
		m.topics, cmd = m.topics.Update(msg)
	case ScreenStatus:
		m.status, cmd = m.status.Update(msg)
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
	case ScreenFeed:
		b.WriteString(m.feed.View())
	case ScreenDetail:
		b.WriteString(m.detail.View())
	case ScreenAsk:
		b.WriteString(m.ask.View())
	case ScreenTopics:
		b.WriteString(m.topics.View())
	case ScreenStatus:
		b.WriteString(m.status.View())
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
			{"a", "ask a question"},
			{"t", "topics browser"},
			{"s", "status info"},
			{"r", "refresh all feeds"},
			{"?", "toggle this help"},
			{"q, ctrl+c", "quit"},
		}
	case ScreenFeed:
		entries = []helpEntry{
			{"j/k, arrows", "navigate articles"},
			{"enter", "open article detail"},
			{"/", "filter articles"},
			{"u", "toggle unread only"},
			{"esc", "back to dashboard"},
			{"?", "toggle this help"},
			{"q, ctrl+c", "quit"},
		}
	case ScreenDetail:
		entries = []helpEntry{
			{"j/k, arrows", "scroll article"},
			{"n/p", "next/prev article"},
			{"o", "open in browser"},
			{"m", "toggle read/unread"},
			{"esc", "back to feed list"},
			{"?", "toggle this help"},
			{"q, ctrl+c", "quit"},
		}
	case ScreenAsk:
		entries = []helpEntry{
			{"enter", "submit question"},
			{"s", "save answer (when done)"},
			{"j/k", "scroll response"},
			{"ctrl+c", "cancel streaming"},
			{"esc", "new question / back"},
			{"q", "quit"},
		}
	case ScreenTopics:
		entries = []helpEntry{
			{"j/k", "navigate"},
			{"tab", "switch subscribed/available"},
			{"enter", "subscribe (available section)"},
			{"d", "unsubscribe"},
			{"e", "edit frequency"},
			{"esc", "back to dashboard"},
		}
	case ScreenStatus:
		entries = []helpEntry{
			{"esc", "back to dashboard"},
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

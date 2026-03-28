package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/oluoyefeso/termiflow/internal/config"
	"github.com/oluoyefeso/termiflow/internal/providers/llm"
	"github.com/oluoyefeso/termiflow/internal/tui/components"
)

// programHolder allows the model (which is copied by tea.NewProgram) to
// access the *tea.Program for sending messages from background goroutines.
type programHolder struct {
	p *tea.Program
}

// spinnerTickMsg drives the loading animation.
type spinnerTickMsg struct{}

// healthTickMsg fires every 5 minutes to re-check API health (managed mode only).
type healthTickMsg struct{}

const healthTickInterval = 5 * time.Minute

// AppModel is the root model that routes between screens.
type AppModel struct {
	activeScreen Screen
	dashboard    DashboardModel
	feed         FeedModel
	detail       DetailModel
	ask          AskModel
	topics       TopicsModel
	sources      SourcesModel
	status       StatusModel
	width        int
	height       int
	banners      []BannerMsg
	showHelp     bool
	totalUnread  int
	subCount     int
	lastRefresh  time.Time
	spinnerFrame int
	headerInfo   HeaderInfo
	programRef   *programHolder // shared ref, set in Run() before p.Run()
}

// NewAppModel creates the root app model with optional notification banners.
func NewAppModel(banners []BannerMsg, version string) AppModel {
	return AppModel{
		activeScreen: ScreenDashboard,
		dashboard:    NewDashboardModel(),
		banners:      banners,
		headerInfo:   NewHeaderInfo(version),
	}
}

func (m AppModel) Init() tea.Cmd {
	cmds := []tea.Cmd{m.dashboard.Init(), spinnerTick()}
	if config.IsManagedMode() {
		cmds = append(cmds, checkAPIHealthCmd(), healthTick())
	}
	return tea.Batch(cmds...)
}

// checkAPIHealthCmd performs an async health check against the managed API.
// Uses CheckHealthCached so TUI and CLI share the same disk cache.
func checkAPIHealthCmd() tea.Cmd {
	return func() tea.Msg {
		cfg := config.Get()
		status := llm.CheckHealthCached(cfg.Providers.Managed.BaseURL)
		return HealthCheckMsg{Status: status}
	}
}

// healthTick sends a tick message every 5 minutes for health re-checks.
func healthTick() tea.Cmd {
	return tea.Tick(healthTickInterval, func(t time.Time) tea.Msg {
		return healthTickMsg{}
	})
}

func spinnerTick() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Forward adjusted height to active screen
		contentH := ContentHeight(msg.Height, len(m.banners))
		adjustedMsg := tea.WindowSizeMsg{Width: msg.Width, Height: contentH}
		var cmd tea.Cmd
		switch m.activeScreen {
		case ScreenDashboard:
			m.dashboard, cmd = m.dashboard.Update(adjustedMsg)
		case ScreenFeed:
			m.feed, cmd = m.feed.Update(adjustedMsg)
		case ScreenDetail:
			m.detail, cmd = m.detail.Update(adjustedMsg)
		case ScreenAsk:
			m.ask, cmd = m.ask.Update(adjustedMsg)
		case ScreenTopics:
			m.topics, cmd = m.topics.Update(adjustedMsg)
		case ScreenSources:
			m.sources, cmd = m.sources.Update(adjustedMsg)
		case ScreenStatus:
			m.status, cmd = m.status.Update(adjustedMsg)
		}
		return m, cmd

	case spinnerTickMsg:
		m.spinnerFrame++
		// Only keep ticking if something is animating
		if m.needsAnimation() {
			return m, spinnerTick()
		}
		return m, nil

	case tea.KeyMsg:
		// Skip global keys when a screen is capturing text input
		if m.activeScreen == ScreenFeed && m.feed.filtering {
			var cmd tea.Cmd
			m.feed, cmd = m.feed.Update(msg)
			return m, cmd
		}
		if m.activeScreen == ScreenTopics && (m.topics.freqPicking || m.topics.confirming || m.topics.customAdding) {
			var cmd tea.Cmd
			m.topics, cmd = m.topics.Update(msg)
			return m, cmd
		}
		if m.activeScreen == ScreenSources && (m.sources.adding || m.sources.editingCtx || m.sources.freqPicking || m.sources.confirming) {
			var cmd tea.Cmd
			m.sources, cmd = m.sources.Update(msg)
			return m, cmd
		}
		if m.activeScreen == ScreenAsk && m.ask.phase != askPhaseDone {
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
		contentH := ContentHeight(m.height, len(m.banners))
		switch msg.Screen {
		case ScreenDashboard:
			m.activeScreen = ScreenDashboard
			m.dashboard.width = m.width
			m.dashboard.height = contentH
			return m, tea.Batch(loadSubscriptions, m.ensureSpinnerRunning())
		case ScreenFeed:
			if msg.Subscription != nil {
				m.feed = NewFeedModel(msg.Subscription)
				m.activeScreen = ScreenFeed
				m.feed.width = m.width
				m.feed.height = contentH
				return m, tea.Batch(m.feed.Init(), m.ensureSpinnerRunning())
			}
			m.activeScreen = ScreenFeed
			m.feed.width = m.width
			m.feed.height = contentH
		case ScreenAsk:
			if m.ask.cancel != nil {
				m.ask.cancel()
			}
			m.ask = NewAskModel()
			m.activeScreen = ScreenAsk
			m.ask.width = m.width
			m.ask.height = contentH
			return m, tea.Batch(m.ask.Init(), m.ensureSpinnerRunning())
		case ScreenTopics:
			m.topics = NewTopicsModel()
			m.activeScreen = ScreenTopics
			m.topics.width = m.width
			m.topics.height = contentH
			return m, tea.Batch(m.topics.Init(), m.ensureSpinnerRunning())
		case ScreenSources:
			m.sources = NewSourcesModel()
			m.activeScreen = ScreenSources
			m.sources.width = m.width
			m.sources.height = contentH
			return m, tea.Batch(m.sources.Init(), m.ensureSpinnerRunning())
		case ScreenStatus:
			m.status = NewStatusModel()
			m.activeScreen = ScreenStatus
			m.status.width = m.width
			m.status.height = contentH
			return m, tea.Batch(m.status.Init(), m.ensureSpinnerRunning())
		}
		return m, nil

	case OpenDetailMsg:
		contentH := ContentHeight(m.height, len(m.banners))
		m.detail = NewDetailModel(msg.Item, msg.Items, msg.Index)
		m.detail.topic = msg.Topic
		m.activeScreen = ScreenDetail
		m.detail.width = m.width
		m.detail.height = contentH
		return m, m.detail.Init()

	case BannersLoadedMsg:
		m.banners = msg.Banners
		return m, nil

	case SubscriptionsLoadedMsg:
		// Track total unread and sub count for header
		m.totalUnread = 0
		m.subCount = len(msg.Subs)
		for _, s := range msg.Subs {
			m.totalUnread += s.Unread
		}
		// Always route to dashboard
		var cmd tea.Cmd
		m.dashboard, cmd = m.dashboard.Update(msg)
		return m, cmd

	case AllRefreshDoneMsg:
		m.lastRefresh = time.Now()
		var cmd tea.Cmd
		m.dashboard, cmd = m.dashboard.Update(msg)
		return m, cmd

	case PerTopicRefreshMsg:
		// Route to dashboard for per-topic status update
		var cmd tea.Cmd
		m.dashboard, cmd = m.dashboard.Update(msg)
		return m, cmd

	case autoRefreshTickMsg:
		var cmd tea.Cmd
		m.dashboard, cmd = m.dashboard.Update(msg)
		return m, cmd

	case HealthCheckMsg:
		m.headerInfo.APIHealth = msg.Status
		return m, nil

	case healthTickMsg:
		if config.IsManagedMode() {
			return m, tea.Batch(checkAPIHealthCmd(), healthTick())
		}
		return m, nil
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
	case ScreenSources:
		m.sources, cmd = m.sources.Update(msg)
	case ScreenStatus:
		m.status, cmd = m.status.Update(msg)
	}

	return m, cmd
}

func (m AppModel) View() string {
	w := m.width
	if w == 0 {
		w = 69
	}

	var b strings.Builder

	// 1. Notification banners (bordered cards)
	for _, banner := range m.banners {
		b.WriteString(RenderBanner(banner, w))
		b.WriteString("\n")
	}

	// 2. Persistent header with ASCII logo + stats
	breadcrumb := m.activeBreadcrumb()
	b.WriteString(RenderHeader(w, breadcrumb, m.totalUnread, m.subCount, m.lastRefresh, m.headerInfo))
	b.WriteString("\n")

	// 3. Help overlay OR screen content
	if m.showHelp {
		b.WriteString(m.renderHelpOverlay(w))
	} else {
		b.WriteString(m.activeContentView())
	}

	// 4. Pad to push status bar to bottom
	currentLines := strings.Count(b.String(), "\n")
	targetLines := m.height - footerLines
	if targetLines < currentLines {
		targetLines = currentLines
	}
	if remaining := targetLines - currentLines; remaining > 0 {
		b.WriteString(strings.Repeat("\n", remaining))
	}

	// 5. Persistent status bar
	hints := m.activeStatusHints()
	// Always add help and quit
	hints = append(hints, components.KeyHint{Key: "?", Desc: "help"})
	if m.activeScreen == ScreenDashboard {
		hints = append(hints, components.KeyHint{Key: "q", Desc: "quit"})
	} else {
		hints = append(hints, components.KeyHint{Key: "esc", Desc: "back"})
	}
	b.WriteString(RenderStatusBar(hints, w))

	return b.String()
}

// needsAnimation returns true if any screen is in a loading/animating state.
func (m AppModel) needsAnimation() bool {
	switch m.activeScreen {
	case ScreenDashboard:
		return m.dashboard.loading || m.dashboard.refreshing
	case ScreenFeed:
		return m.feed.loading
	case ScreenAsk:
		return m.ask.phase == askPhaseSearching || m.ask.phase == askPhaseStreaming
	case ScreenSources:
		return m.sources.adding && m.sources.addPhase == addPhaseDiscovering
	}
	return false
}

// ensureSpinnerRunning restarts the spinner tick if animation is needed.
func (m AppModel) ensureSpinnerRunning() tea.Cmd {
	if m.needsAnimation() {
		return spinnerTick()
	}
	return nil
}

// activeBreadcrumb returns the breadcrumb segments for the current screen.
func (m AppModel) activeBreadcrumb() []string {
	switch m.activeScreen {
	case ScreenDashboard:
		return nil
	case ScreenFeed:
		return m.feed.Breadcrumb()
	case ScreenDetail:
		return m.detail.Breadcrumb()
	case ScreenAsk:
		return []string{"ASK"}
	case ScreenTopics:
		return []string{"TOPICS"}
	case ScreenSources:
		return []string{"SOURCES"}
	case ScreenStatus:
		return []string{"STATUS"}
	}
	return nil
}

// activeContentView returns the content from the active screen (no chrome).
func (m AppModel) activeContentView() string {
	switch m.activeScreen {
	case ScreenDashboard:
		return m.dashboard.ContentView(m.spinnerFrame)
	case ScreenFeed:
		return m.feed.ContentView(m.spinnerFrame)
	case ScreenDetail:
		return m.detail.ContentView()
	case ScreenAsk:
		return m.ask.ContentView(m.spinnerFrame)
	case ScreenTopics:
		return m.topics.ContentView()
	case ScreenSources:
		return m.sources.ContentView()
	case ScreenStatus:
		return m.status.ContentView()
	default:
		return StyleMuted.Render("  Screen not implemented yet")
	}
}

// activeStatusHints returns the keybinding hints for the current screen.
func (m AppModel) activeStatusHints() []components.KeyHint {
	switch m.activeScreen {
	case ScreenDashboard:
		return m.dashboard.StatusHints()
	case ScreenFeed:
		return m.feed.StatusHints()
	case ScreenDetail:
		return m.detail.StatusHints()
	case ScreenAsk:
		return m.ask.StatusHints()
	case ScreenTopics:
		return m.topics.StatusHints()
	case ScreenSources:
		return m.sources.StatusHints()
	case ScreenStatus:
		return m.status.StatusHints()
	}
	return nil
}

// renderHelpOverlay renders a centered bordered help card.
func (m AppModel) renderHelpOverlay(width int) string {
	type helpEntry struct{ key, desc string }

	var entries []helpEntry

	switch m.activeScreen {
	case ScreenDashboard:
		entries = []helpEntry{
			{"j/k, arrows", "navigate subscriptions"},
			{"enter", "open feed for selected topic"},
			{"a", "ask a question"},
			{"t", "topics browser"},
			{"s", "sources (RSS/blogs)"},
			{"r", "refresh all feeds"},
			{"i", "system info"},
			{"?", "close help"},
			{"q, ctrl+c", "quit"},
		}
	case ScreenFeed:
		entries = []helpEntry{
			{"j/k, arrows", "navigate articles"},
			{"enter", "open article detail"},
			{"/", "filter articles"},
			{"u", "toggle unread only"},
			{"esc", "back to dashboard"},
			{"?", "close help"},
		}
	case ScreenDetail:
		entries = []helpEntry{
			{"j/k, arrows", "scroll article"},
			{"n/p", "next/prev article"},
			{"o", "open in browser"},
			{"m", "toggle read/unread"},
			{"esc", "back to feed list"},
			{"?", "close help"},
		}
	case ScreenAsk:
		entries = []helpEntry{
			{"enter", "submit question"},
			{"s", "save answer (when done)"},
			{"j/k", "scroll response"},
			{"ctrl+c", "cancel streaming"},
			{"esc", "new question / back"},
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
	case ScreenSources:
		entries = []helpEntry{
			{"j/k", "navigate sources"},
			{"a", "add new source"},
			{"d", "delete source"},
			{"e", "edit frequency"},
			{"c", "edit context"},
			{"esc", "back to dashboard"},
		}
	case ScreenStatus:
		entries = []helpEntry{
			{"esc", "back to dashboard"},
		}
	}

	// Build card content
	var lines []string
	for _, e := range entries {
		lines = append(lines, fmt.Sprintf("%-16s %s",
			StyleTitle.Render(e.key),
			StyleMuted.Render(e.desc),
		))
	}

	// Center the card
	cardWidth := 48
	if cardWidth > width-4 {
		cardWidth = width - 4
	}

	card := RenderCard("KEYBINDINGS", lines, cardWidth)

	// Center horizontally
	cardLines := strings.Split(card, "\n")
	var centered []string
	for _, line := range cardLines {
		lineWidth := lipgloss.Width(line)
		pad := (width - lineWidth) / 2
		if pad < 0 {
			pad = 0
		}
		centered = append(centered, strings.Repeat(" ", pad)+line)
	}

	// Add vertical padding
	result := "\n\n" + strings.Join(centered, "\n") + "\n\n"
	closePad := (width - 18) / 2
	if closePad < 0 {
		closePad = 0
	}
	result += StyleMuted.Render(strings.Repeat(" ", closePad) + "Press ? to close")
	return result
}

// Run starts the Bubble Tea program.
func Run(banners []BannerMsg, version string) error {
	holder := &programHolder{}
	model := NewAppModel(banners, version)
	model.programRef = holder
	model.dashboard.programRef = holder
	p := tea.NewProgram(model, tea.WithAltScreen())
	holder.p = p
	_, err := p.Run()
	return err
}

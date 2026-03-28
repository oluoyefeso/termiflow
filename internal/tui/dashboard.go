package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/oluoyefeso/termiflow/internal/config"
	"github.com/oluoyefeso/termiflow/internal/db"
	"github.com/oluoyefeso/termiflow/internal/scheduler"
	"github.com/oluoyefeso/termiflow/internal/tui/components"
)

const autoRefreshInterval = 30 * time.Minute

// DashboardModel is the home screen showing subscriptions and unread counts.
type DashboardModel struct {
	subs          []SubInfo
	cursor        int
	totalUnread   int
	width         int
	height        int
	loading       bool
	err           error
	refreshing    bool
	lastRefresh   time.Time
	refreshErr    string
	refreshStatus map[string]string // per-topic: ""/"⦻"/"✓ +N"/"✗"
}

func NewDashboardModel() DashboardModel {
	return DashboardModel{loading: true, refreshStatus: make(map[string]string)}
}

// loadSubscriptions fetches subscription data from the database.
func loadSubscriptions() tea.Msg {
	subs, err := db.GetActiveSubscriptions()
	if err != nil {
		return SubscriptionsLoadedMsg{Err: err}
	}

	var infos []SubInfo
	for _, sub := range subs {
		total, unread, err := db.GetSubscriptionItemCount(sub.ID)
		if err != nil {
			continue
		}
		infos = append(infos, SubInfo{
			Sub:    sub,
			Total:  total,
			Unread: unread,
		})
	}

	return SubscriptionsLoadedMsg{Subs: infos}
}

// refreshAllFeeds refreshes all subscriptions, sending PerTopicRefreshMsg per topic.
func refreshAllFeeds(p *tea.Program) tea.Cmd {
	return func() tea.Msg {
		cfg := config.Get()
		sched, err := scheduler.NewFromConfig(cfg, cfg.General.DefaultProvider)
		if err != nil {
			return AllRefreshDoneMsg{Err: err}
		}

		subs, err := db.GetActiveSubscriptions()
		if err != nil {
			return AllRefreshDoneMsg{Err: err}
		}

		totalNew := 0
		errCount := 0
		for _, sub := range subs {
			newItems, err := sched.RefreshSubscription(context.Background(), sub)
			if err != nil {
				errCount++
				if p != nil {
					p.Send(PerTopicRefreshMsg{Topic: sub.Topic, Err: err})
				}
				continue
			}
			totalNew += len(newItems)
			if p != nil {
				p.Send(PerTopicRefreshMsg{Topic: sub.Topic, NewItems: len(newItems)})
			}
		}

		return AllRefreshDoneMsg{TotalNew: totalNew, Errors: errCount}
	}
}

// refreshAllFeedsSimple is the non-program version for compatibility.
func refreshAllFeedsSimple() tea.Cmd {
	return refreshAllFeeds(nil)
}

// autoRefreshTick sends a tick message for auto-refresh.
func autoRefreshTick() tea.Cmd {
	return tea.Tick(autoRefreshInterval, func(t time.Time) tea.Msg {
		return autoRefreshTickMsg{}
	})
}

// autoRefreshTickMsg is the internal tick message for auto-refresh.
type autoRefreshTickMsg struct{}

func (m DashboardModel) Init() tea.Cmd {
	return tea.Batch(loadSubscriptions, autoRefreshTick())
}

func (m DashboardModel) Update(msg tea.Msg) (DashboardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case SubscriptionsLoadedMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.subs = msg.Subs
		m.totalUnread = 0
		for _, s := range m.subs {
			m.totalUnread += s.Unread
		}
		if m.cursor >= len(m.subs) && len(m.subs) > 0 {
			m.cursor = len(m.subs) - 1
		}
		return m, nil

	case AllRefreshDoneMsg:
		m.refreshing = false
		m.lastRefresh = time.Now()
		m.refreshErr = ""
		m.refreshStatus = make(map[string]string) // clear stale status badges
		if msg.Err != nil {
			m.refreshErr = fmt.Sprintf("Refresh failed: %v", msg.Err)
		} else if msg.Errors > 0 {
			m.refreshErr = fmt.Sprintf("%d topic(s) failed to refresh", msg.Errors)
		}
		return m, loadSubscriptions

	case PerTopicRefreshMsg:
		if msg.Err != nil {
			m.refreshStatus[msg.Topic] = "✗"
		} else {
			if msg.NewItems > 0 {
				m.refreshStatus[msg.Topic] = fmt.Sprintf("✓ +%d", msg.NewItems)
			} else {
				m.refreshStatus[msg.Topic] = "✓"
			}
		}
		return m, nil

	case autoRefreshTickMsg:
		if !m.refreshing && len(m.subs) > 0 {
			m.refreshing = true
			m.refreshStatus = make(map[string]string)
			for _, s := range m.subs {
				m.refreshStatus[s.Sub.Topic] = "⦻"
			}
			return m, tea.Batch(refreshAllFeedsSimple(), autoRefreshTick())
		}
		return m, autoRefreshTick()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DashboardKeys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, DashboardKeys.Down):
			if m.cursor < len(m.subs)-1 {
				m.cursor++
			}
		case key.Matches(msg, DashboardKeys.Enter):
			if len(m.subs) > 0 && m.cursor < len(m.subs) {
				return m, func() tea.Msg {
					return SwitchScreenMsg{
						Screen:       ScreenFeed,
						Subscription: m.subs[m.cursor].Sub,
					}
				}
			}
		case key.Matches(msg, DashboardKeys.Refresh):
			if !m.refreshing {
				m.refreshing = true
				m.refreshStatus = make(map[string]string)
				for _, s := range m.subs {
					m.refreshStatus[s.Sub.Topic] = "⦻"
				}
				return m, refreshAllFeedsSimple()
			}
		case key.Matches(msg, DashboardKeys.Ask):
			return m, func() tea.Msg {
				return SwitchScreenMsg{Screen: ScreenAsk}
			}
		case key.Matches(msg, DashboardKeys.Topics):
			return m, func() tea.Msg {
				return SwitchScreenMsg{Screen: ScreenTopics}
			}
		case key.Matches(msg, DashboardKeys.Status):
			return m, func() tea.Msg {
				return SwitchScreenMsg{Screen: ScreenStatus}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// Breadcrumb returns breadcrumb segments for the dashboard.
func (m DashboardModel) Breadcrumb() []string {
	return nil // dashboard is root
}

// StatusHints returns keybinding hints for the dashboard.
func (m DashboardModel) StatusHints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "enter", Desc: "feed"},
		{Key: "a", Desc: "ask"},
		{Key: "t", Desc: "topics"},
		{Key: "r", Desc: "refresh"},
		{Key: "s", Desc: "status"},
	}
}

// ContentView renders the dashboard content without header/footer chrome.
func (m DashboardModel) ContentView(spinnerFrame int) string {
	w := m.width
	if w == 0 {
		w = 69
	}

	var b strings.Builder

	if m.loading && !m.refreshing {
		b.WriteString(fmt.Sprintf("\n  %s Loading...\n", AnimatedSpinner(spinnerFrame)))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(StyleError.Render(fmt.Sprintf("\n  Error: %v", m.err)))
		b.WriteString("\n")
		return b.String()
	}

	// Zero-subscription state
	if len(m.subs) == 0 {
		b.WriteString(m.renderZeroState(w))
		return b.String()
	}

	b.WriteString("\n")

	// Section header with total unread
	header := StyleAccent.Render("  SUBSCRIPTIONS")
	if m.totalUnread > 0 {
		unreadStr := StyleUnreadBadge.Render(fmt.Sprintf("%d unread", m.totalUnread))
		pad := w - lipgloss.Width(header) - lipgloss.Width(unreadStr) - 2
		if pad < 1 {
			pad = 1
		}
		header += strings.Repeat(" ", pad) + unreadStr
	}
	b.WriteString(header)
	b.WriteString("\n\n")

	// Subscription list with column alignment
	b.WriteString(m.renderSubscriptions(w))

	// Activity footer
	b.WriteString(m.renderActivityFooter(w))

	return b.String()
}

func (m DashboardModel) renderSubscriptions(width int) string {
	var b strings.Builder

	// Calculate column widths
	topicWidth := 20
	for _, info := range m.subs {
		if len(info.Sub.Topic) > topicWidth {
			topicWidth = len(info.Sub.Topic)
		}
	}
	topicWidth += 2 // padding

	for i, info := range m.subs {
		cursor := "  "
		topicStyle := lipgloss.NewStyle()
		if i == m.cursor {
			cursor = StyleSelectedIndicator.Render("▸ ")
			topicStyle = StyleSelected
		}

		// Topic name (padded)
		topic := PadRight(topicStyle.Render(info.Sub.Topic), topicWidth)

		// Activity bar or refresh status
		var statusCol string
		if rs, ok := m.refreshStatus[info.Sub.Topic]; ok && rs != "" {
			if rs == "⦻" {
				statusCol = StyleCyan.Render("  ⦻   ")
			} else if strings.HasPrefix(rs, "✓") {
				statusCol = StyleSuccess.Render(PadRight(rs, 6))
			} else {
				statusCol = StyleError.Render(PadRight(rs, 6))
			}
		} else {
			statusCol = ActivityBar(info.Unread, info.Total)
		}

		// Unread count
		unreadStr := StyleMuted.Render("  ─   ")
		if info.Unread > 0 {
			unreadStr = StyleUnreadBadge.Render(PadLeft(fmt.Sprintf("%d new", info.Unread), 6))
		}

		// Frequency
		freq := StyleMuted.Render(PadRight(info.Sub.Frequency, 7))

		// Last fetch time
		lastFetch := StyleMuted.Render("  ─   ")
		if info.Sub.LastFetchedAt != nil {
			ago := time.Since(*info.Sub.LastFetchedAt)
			label := "now"
			if ago > 24*time.Hour {
				label = fmt.Sprintf("%dd ago", int(ago.Hours()/24))
			} else if ago > time.Hour {
				label = fmt.Sprintf("%dh ago", int(ago.Hours()))
			} else if ago > time.Minute {
				label = fmt.Sprintf("%dm ago", int(ago.Minutes()))
			}
			lastFetch = StyleMuted.Render(PadLeft(label, 7))
		}

		b.WriteString(fmt.Sprintf("  %s%s %s  %s  %s  %s\n",
			cursor, topic, statusCol, unreadStr, freq, lastFetch))
	}

	return b.String()
}

func (m DashboardModel) renderActivityFooter(width int) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + LabeledRule("Activity", width-4))
	b.WriteString("\n")

	// Refresh status
	if m.refreshing {
		b.WriteString(fmt.Sprintf("  %s Refreshing feeds...\n", StyleCyan.Render("⦻")))
	} else if m.refreshErr != "" {
		b.WriteString(fmt.Sprintf("  %s %s\n", StyleWarning.Render("!"), m.refreshErr))
	} else if !m.lastRefresh.IsZero() {
		ago := time.Since(m.lastRefresh)
		label := "just now"
		if ago > time.Minute {
			label = fmt.Sprintf("%dm ago", int(ago.Minutes()))
		}
		b.WriteString(fmt.Sprintf("  %s Last refreshed %s\n",
			StyleSuccess.Render("✓"), StyleMuted.Render(label)))
	}

	// Next auto-refresh
	if !m.lastRefresh.IsZero() {
		nextIn := autoRefreshInterval - time.Since(m.lastRefresh)
		if nextIn > 0 {
			b.WriteString(fmt.Sprintf("  %s Next auto-refresh in %dm\n",
				StyleMuted.Render("⦻"), int(nextIn.Minutes())))
		}
	}

	return b.String()
}

func (m DashboardModel) renderZeroState(width int) string {
	lines := []string{
		"",
		StyleAccent.Render("Welcome to Termiflow"),
		"",
		"Subscribe to your first topic:",
		"Press " + StyleTitle.Render("t") + " to browse available topics",
		"",
		"Or from CLI:",
		StyleMuted.Render("$ termiflow subscribe rust-lang"),
		StyleMuted.Render("$ termiflow subscribe \"large language models\""),
		"",
	}

	return "\n" + RenderCard("GET STARTED", lines, width-4) + "\n"
}

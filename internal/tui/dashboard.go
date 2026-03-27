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
	subs        []SubInfo
	cursor      int
	totalUnread int
	width       int
	height      int
	loading     bool
	err         error
	refreshing  bool
	lastRefresh time.Time
	refreshErr  string
}

func NewDashboardModel() DashboardModel {
	return DashboardModel{loading: true}
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

// refreshAllFeeds creates a tea.Cmd that refreshes all subscriptions sequentially,
// sending a FeedRefreshedMsg per topic and AllRefreshDoneMsg when done.
func refreshAllFeeds() tea.Cmd {
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
				continue
			}
			totalNew += len(newItems)
		}

		return AllRefreshDoneMsg{TotalNew: totalNew, Errors: errCount}
	}
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
		return m, nil

	case AllRefreshDoneMsg:
		m.refreshing = false
		m.lastRefresh = time.Now()
		m.refreshErr = ""
		if msg.Err != nil {
			m.refreshErr = fmt.Sprintf("Refresh failed: %v", msg.Err)
		} else if msg.Errors > 0 {
			m.refreshErr = fmt.Sprintf("%d topic(s) failed to refresh", msg.Errors)
		}
		// Reload subscription counts from DB to reflect new items
		return m, loadSubscriptions

	case autoRefreshTickMsg:
		if !m.refreshing && len(m.subs) > 0 {
			m.refreshing = true
			return m, tea.Batch(refreshAllFeeds(), autoRefreshTick())
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
				return m, refreshAllFeeds()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m DashboardModel) View() string {
	w := m.width
	if w == 0 {
		w = 69
	}

	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader(w))
	b.WriteString("\n\n")

	if m.loading && !m.refreshing {
		b.WriteString(StyleMuted.Render("  Loading..."))
		b.WriteString("\n")
		return b.String()
	}

	if m.err != nil {
		b.WriteString(StyleError.Render(fmt.Sprintf("  Error: %v", m.err)))
		b.WriteString("\n")
		return b.String()
	}

	// Zero-subscription state
	if len(m.subs) == 0 {
		b.WriteString(m.renderZeroState())
		return b.String()
	}

	// Subscription list
	b.WriteString(m.renderSubscriptions())

	// Total unread
	if m.totalUnread > 0 {
		line := fmt.Sprintf("%28s", "")
		b.WriteString(fmt.Sprintf("  %s%s\n",
			line,
			StyleMuted.Render(fmt.Sprintf("─── %d total unread", m.totalUnread)),
		))
	}

	// Refresh status
	if m.refreshing {
		b.WriteString(fmt.Sprintf("\n  %s\n", StyleCyan.Render("⟳ Refreshing feeds...")))
	} else if m.refreshErr != "" {
		b.WriteString(fmt.Sprintf("\n  %s\n", StyleWarning.Render("! "+m.refreshErr)))
	} else if !m.lastRefresh.IsZero() {
		ago := time.Since(m.lastRefresh)
		label := "just now"
		if ago > time.Minute {
			label = fmt.Sprintf("%dm ago", int(ago.Minutes()))
		}
		b.WriteString(fmt.Sprintf("\n  %s\n", StyleMuted.Render("Last refreshed "+label)))
	}

	b.WriteString("\n")

	// Pad to push status bar to bottom
	contentHeight := strings.Count(b.String(), "\n")
	statusBarHeight := 2 // separator + hints
	if remaining := m.height - contentHeight - statusBarHeight; remaining > 0 {
		b.WriteString(strings.Repeat("\n", remaining))
	}

	// Status bar
	hints := []components.KeyHint{
		{Key: "enter", Desc: "feed"},
		{Key: "r", Desc: "refresh"},
		{Key: "?", Desc: "help"},
		{Key: "q", Desc: "quit"},
	}
	b.WriteString(components.NewStatusBar(hints, w).View())

	return b.String()
}

func (m DashboardModel) renderHeader(width int) string {
	date := time.Now().Format("02 Jan 06 · 15:04")
	title := "TERMIFLOW"

	// Badge
	badge := ""
	if m.totalUnread > 0 {
		badge = StyleUnreadBadge.Render(fmt.Sprintf(" %d unread", m.totalUnread))
	}

	// Use lipgloss.Width for ANSI-aware width measurement
	badgeWidth := lipgloss.Width(badge)
	titleWidth := lipgloss.Width(StyleAccent.Render(title))
	dateWidth := lipgloss.Width(StyleMuted.Render(date))
	padding := width - titleWidth - badgeWidth - dateWidth - 2 // 2 for leading indent
	if padding < 1 {
		padding = 1
	}

	top := StyleMuted.Render(Bar("═", width))
	content := fmt.Sprintf("  %s%s%s%s",
		StyleAccent.Render(title),
		badge,
		strings.Repeat(" ", padding),
		StyleMuted.Render(date),
	)
	bot := StyleMuted.Render(Bar("═", width))

	return fmt.Sprintf("%s\n%s\n%s", top, content, bot)
}

func (m DashboardModel) renderSubscriptions() string {
	var b strings.Builder

	sectionTitle := StyleAccent.Render("  SUBSCRIPTIONS")
	b.WriteString(sectionTitle)
	b.WriteString("\n\n")

	for i, info := range m.subs {
		cursor := "  "
		topicStyle := lipgloss.NewStyle()
		if i == m.cursor {
			cursor = StyleSelectedIndicator.Render("▸ ")
			topicStyle = StyleSelected
		}

		unreadLabel := StyleMuted.Render("up to date")
		if info.Unread > 0 {
			unreadLabel = StyleUnreadBadge.Render(fmt.Sprintf("%d new", info.Unread))
		}

		b.WriteString(fmt.Sprintf("  %s%-26s %s\n",
			cursor,
			topicStyle.Render(info.Sub.Topic),
			unreadLabel,
		))
	}

	return b.String()
}

func (m DashboardModel) renderZeroState() string {
	var b strings.Builder

	b.WriteString(StyleWarning.Render("  ! No active subscriptions"))
	b.WriteString("\n\n")
	b.WriteString("  Get started:\n")
	b.WriteString(fmt.Sprintf("    %s\n", StyleTitle.Render("termiflow subscribe silicon-chips")))
	b.WriteString(fmt.Sprintf("    %s\n", StyleTitle.Render("termiflow subscribe \"your custom topic\"")))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  See available topics with %s\n",
		StyleTitle.Render("termiflow topics --available")))
	b.WriteString("\n")

	return b.String()
}

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/oluoyefeso/termiflow/internal/db"
	"github.com/oluoyefeso/termiflow/internal/tui/components"
	"github.com/oluoyefeso/termiflow/pkg/models"
)

// FeedModel is the feed list screen showing articles for a subscription.
type FeedModel struct {
	topic       string
	subID       int64
	items       []*models.FeedItem
	filtered    []*models.FeedItem // items after filter applied
	cursor      int
	width       int
	height      int
	loading     bool
	err         error
	unreadOnly  bool
	filterInput textinput.Model
	filtering   bool
	filterText  string
}

func NewFeedModel(sub *models.Subscription) FeedModel {
	ti := textinput.New()
	ti.Placeholder = "filter..."
	ti.CharLimit = 64

	return FeedModel{
		topic:       sub.Topic,
		subID:       sub.ID,
		loading:     true,
		filterInput: ti,
	}
}

func loadFeedItems(subID int64, topic string) tea.Cmd {
	return func() tea.Msg {
		items, err := db.GetFeedItems(db.FeedItemFilter{
			SubscriptionID: subID,
			Limit:          100,
		})
		if err != nil {
			return FeedItemsLoadedMsg{Err: err, Topic: topic}
		}
		return FeedItemsLoadedMsg{Items: items, Topic: topic}
	}
}

func (m FeedModel) Init() tea.Cmd {
	return loadFeedItems(m.subID, m.topic)
}

func (m FeedModel) Update(msg tea.Msg) (FeedModel, tea.Cmd) {
	switch msg := msg.(type) {
	case FeedItemsLoadedMsg:
		if msg.Topic != m.topic {
			return m, nil
		}
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.items = msg.Items
		m.applyFilter()
		m.cursor = 0
		return m, nil

	case ItemMarkedReadMsg:
		if msg.Err == nil {
			for _, item := range m.items {
				if item.ID == msg.ItemID {
					item.IsRead = true
					break
				}
			}
			m.applyFilter()
		}
		return m, nil

	case tea.KeyMsg:
		if m.filtering {
			return m.updateFilter(msg)
		}

		switch {
		case key.Matches(msg, FeedKeys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, FeedKeys.Down):
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case key.Matches(msg, FeedKeys.Enter):
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				item := m.filtered[m.cursor]
				topic := m.topic
				return m, func() tea.Msg {
					return OpenDetailMsg{
						Item:  item,
						Items: m.filtered,
						Index: m.cursor,
						Topic: topic,
					}
				}
			}
		case key.Matches(msg, FeedKeys.Back):
			return m, func() tea.Msg {
				return SwitchScreenMsg{Screen: ScreenDashboard}
			}
		case key.Matches(msg, FeedKeys.Unread):
			m.unreadOnly = !m.unreadOnly
			m.applyFilter()
			m.cursor = 0
		case key.Matches(msg, FeedKeys.Filter):
			m.filtering = true
			m.filterInput.Focus()
			return m, textinput.Blink
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m *FeedModel) updateFilter(msg tea.KeyMsg) (FeedModel, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc":
		m.filtering = false
		m.filterInput.Blur()
		if msg.String() == "esc" {
			m.filterText = ""
			m.filterInput.SetValue("")
		} else {
			m.filterText = m.filterInput.Value()
		}
		m.applyFilter()
		m.cursor = 0
		return *m, nil
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.filterText = m.filterInput.Value()
	m.applyFilter()
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	return *m, cmd
}

func (m *FeedModel) applyFilter() {
	m.filtered = nil
	for _, item := range m.items {
		if m.unreadOnly && item.IsRead {
			continue
		}
		if m.filterText != "" {
			lower := strings.ToLower(m.filterText)
			if !strings.Contains(strings.ToLower(item.Title), lower) &&
				!strings.Contains(strings.ToLower(item.Summary), lower) &&
				!strings.Contains(strings.ToLower(item.SourceName), lower) {
				continue
			}
		}
		m.filtered = append(m.filtered, item)
	}
}

// Breadcrumb returns breadcrumb segments for the feed screen.
func (m FeedModel) Breadcrumb() []string {
	return []string{strings.ToUpper(m.topic)}
}

// StatusHints returns keybinding hints for the feed screen.
func (m FeedModel) StatusHints() []components.KeyHint {
	hints := []components.KeyHint{
		{Key: "enter", Desc: "open"},
		{Key: "/", Desc: "filter"},
		{Key: "u", Desc: "unread"},
	}
	return hints
}

// ContentView renders the feed list content without header/footer chrome.
func (m FeedModel) ContentView(spinnerFrame int) string {
	w := m.width
	if w == 0 {
		w = 69
	}

	var b strings.Builder

	if m.loading {
		b.WriteString(fmt.Sprintf("\n  %s Loading...\n", AnimatedSpinner(spinnerFrame)))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(StyleError.Render(fmt.Sprintf("\n  Error: %v", m.err)))
		return b.String()
	}

	// Count label
	countLabel := fmt.Sprintf("%d items", len(m.filtered))
	if m.unreadOnly {
		countLabel += " (unread)"
	}
	b.WriteString(fmt.Sprintf("\n  %s\n", StyleMuted.Render(countLabel)))

	if len(m.filtered) == 0 {
		msg := "No items"
		if m.unreadOnly {
			msg = "No unread items. Press u to show all."
		}
		if m.filterText != "" {
			msg = fmt.Sprintf("No items matching \"%s\"", m.filterText)
		}
		b.WriteString(StyleMuted.Render(fmt.Sprintf("\n  %s\n", msg)))
		return b.String()
	}

	// Filter bar
	if m.filtering {
		b.WriteString(fmt.Sprintf("  %s %s\n", StyleTitle.Render("Filter:"), m.filterInput.View()))
	} else if m.filterText != "" {
		b.WriteString(fmt.Sprintf("  %s %s\n",
			StyleTitle.Render("Filter:"),
			StyleMuted.Render(m.filterText),
		))
	}

	b.WriteString("\n")

	// Calculate visible area
	visibleHeight := m.height - strings.Count(b.String(), "\n") - 1
	if visibleHeight < 3 {
		visibleHeight = 10
	}
	// Each item takes 2 lines + 1 blank = 3 lines
	visibleItems := visibleHeight / 3
	if visibleItems < 1 {
		visibleItems = 5
	}

	// Scroll window
	startIdx := 0
	if m.cursor >= visibleItems {
		startIdx = m.cursor - visibleItems + 1
	}
	endIdx := startIdx + visibleItems
	if endIdx > len(m.filtered) {
		endIdx = len(m.filtered)
	}

	// Render items
	for i := startIdx; i < endIdx; i++ {
		item := m.filtered[i]
		b.WriteString(m.renderItem(item, i == m.cursor, w))
	}

	// Scroll indicator
	if len(m.filtered) > visibleItems {
		b.WriteString(StyleMuted.Render(fmt.Sprintf("  ... %d/%d items\n", m.cursor+1, len(m.filtered))))
	}

	return b.String()
}

func (m FeedModel) renderItem(item *models.FeedItem, selected bool, width int) string {
	var b strings.Builder

	// Read indicator
	readDot := StyleSuccess.Render("●")
	if item.IsRead {
		readDot = StyleMuted.Render("○")
	}

	// Cursor
	cursor := "  "
	titleStyle := StyleMuted
	if !item.IsRead {
		titleStyle = lipgloss.NewStyle().Foreground(ColorAccent)
	}
	if selected {
		cursor = StyleSelectedIndicator.Render("▸ ")
		if !item.IsRead {
			titleStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
		} else {
			titleStyle = StyleSelected
		}
	}

	// Relevance micro-bar (right-aligned)
	var scoreStr string
	if item.RelevanceScore > 0 {
		pct := fmt.Sprintf("%.0f%%", item.RelevanceScore*100)
		scoreStr = RelevanceBar(item.RelevanceScore) + " " + StyleMuted.Render(pct)
	}

	// Title (truncated with muted ellipsis)
	title := item.Title
	maxTitleWidth := width - 20 // space for cursor + dot + score + padding
	if maxTitleWidth < 10 {
		maxTitleWidth = 10
	}
	runes := []rune(title)
	if len(runes) > maxTitleWidth {
		cutAt := maxTitleWidth - 3
		if cutAt < 0 {
			cutAt = 0
		}
		title = titleStyle.Render(string(runes[:cutAt])) + StyleMuted.Render("...")
	} else {
		title = titleStyle.Render(title)
	}

	// Line 1: cursor + dot + title + score (right-aligned)
	line1Left := fmt.Sprintf("  %s%s %s", cursor, readDot, title)
	line1LeftWidth := lipgloss.Width(line1Left)
	scoreWidth := lipgloss.Width(scoreStr)
	pad := width - line1LeftWidth - scoreWidth - 1
	if pad < 1 {
		pad = 1
	}

	line1 := line1Left + strings.Repeat(" ", pad) + scoreStr

	// Apply selection background
	if selected {
		line1 = StyleSelectedBg.Render(line1)
	}
	b.WriteString(line1 + "\n")

	// Line 2: source · time
	meta := fmt.Sprintf("      %s · %s",
		StyleMuted.Render(item.SourceName),
		StyleMuted.Render(item.TimeAgo()),
	)
	b.WriteString(meta + "\n")

	return b.String()
}

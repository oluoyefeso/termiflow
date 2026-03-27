package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/oluoyefeso/termiflow/internal/db"
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
		// Guard: ignore stale responses from a previous subscription
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
				return m, func() tea.Msg {
					return OpenDetailMsg{
						Item:  item,
						Items: m.filtered,
						Index: m.cursor,
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
	// Live filter as you type
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

func (m FeedModel) View() string {
	w := m.width
	if w == 0 {
		w = 69
	}

	var b strings.Builder

	// Header
	title := strings.ToUpper(m.topic)
	countLabel := fmt.Sprintf("%d items", len(m.filtered))
	if m.unreadOnly {
		countLabel += " (unread)"
	}
	top := StyleMuted.Render(Bar("═", w))
	headerPad := w - lipgloss.Width(StyleAccent.Render(title)) - lipgloss.Width(StyleMuted.Render(countLabel)) - 4
	if headerPad < 1 {
		headerPad = 1
	}
	content := fmt.Sprintf("  %s%s%s",
		StyleAccent.Render(title),
		strings.Repeat(" ", headerPad),
		StyleMuted.Render(countLabel),
	)
	bot := StyleMuted.Render(Bar("═", w))
	b.WriteString(fmt.Sprintf("%s\n%s\n%s\n", top, content, bot))

	if m.loading {
		b.WriteString(StyleMuted.Render("\n  Loading..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(StyleError.Render(fmt.Sprintf("\n  Error: %v", m.err)))
		return b.String()
	}

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
	visibleHeight := m.height - strings.Count(b.String(), "\n") - 3 // 3 for status bar
	if visibleHeight < 3 {
		visibleHeight = 10
	}

	// Scroll window: keep cursor visible
	startIdx := 0
	if m.cursor >= visibleHeight {
		startIdx = m.cursor - visibleHeight + 1
	}
	endIdx := startIdx + visibleHeight
	if endIdx > len(m.filtered) {
		endIdx = len(m.filtered)
	}

	// Render items
	for i := startIdx; i < endIdx; i++ {
		item := m.filtered[i]
		b.WriteString(m.renderItem(item, i == m.cursor))
	}

	// Scroll indicator
	if len(m.filtered) > visibleHeight {
		b.WriteString(StyleMuted.Render(fmt.Sprintf("  ... %d/%d items\n", m.cursor+1, len(m.filtered))))
	}

	// Status bar at bottom
	hints := []string{
		StyleTitle.Render("[enter]") + " " + StyleMuted.Render("open"),
		StyleTitle.Render("[/]") + " " + StyleMuted.Render("filter"),
		StyleTitle.Render("[u]") + " " + StyleMuted.Render("unread"),
		StyleTitle.Render("[esc]") + " " + StyleMuted.Render("back"),
	}
	b.WriteString("\n" + StyleMuted.Render(Bar("─", w)) + "\n")
	b.WriteString(" " + strings.Join(hints, "  "))

	return b.String()
}

func (m FeedModel) renderItem(item *models.FeedItem, selected bool) string {
	var b strings.Builder

	// Cursor + read indicator
	cursor := "  "
	titleStyle := StyleMuted
	if !item.IsRead {
		titleStyle = StyleAccent
	}
	if selected {
		cursor = StyleSelectedIndicator.Render("▸ ")
		if !item.IsRead {
			titleStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
		} else {
			titleStyle = StyleSelected
		}
	}

	// Read indicator
	readDot := StyleSuccess.Render("●")
	if item.IsRead {
		readDot = StyleMuted.Render("○")
	}

	// Title line (rune-safe truncation for multi-byte characters)
	title := item.Title
	runes := []rune(title)
	if len(runes) > 60 {
		title = string(runes[:57]) + "..."
	}
	b.WriteString(fmt.Sprintf("  %s%s %s\n", cursor, readDot, titleStyle.Render(title)))

	// Meta line: source + time
	meta := fmt.Sprintf("      %s  %s",
		StyleMuted.Render(item.SourceName),
		StyleMuted.Render(item.TimeAgo()),
	)
	if item.RelevanceScore > 0 {
		meta += fmt.Sprintf("  %s", StyleMuted.Render(fmt.Sprintf("%.0f%%", item.RelevanceScore*100)))
	}
	b.WriteString(meta + "\n")

	return b.String()
}

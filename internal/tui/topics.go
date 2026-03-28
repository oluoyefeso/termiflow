package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/oluoyefeso/termiflow/internal/db"
	"github.com/oluoyefeso/termiflow/internal/tui/components"
	"github.com/oluoyefeso/termiflow/pkg/models"
)

// topicsSection tracks which section the cursor is in.
type topicsSection int

const (
	sectionSubscribed topicsSection = iota
	sectionAvailable
)

// TopicsModel is the topics browser screen.
type TopicsModel struct {
	subscribed  []SubInfo
	available   []AvailableCategory
	section     topicsSection
	cursor      int
	width       int
	height      int
	loading     bool
	err         error
	statusMsg   string
	freqPicking bool   // showing frequency picker
	freqCursor  int    // 0=hourly, 1=daily, 2=weekly
	freqAction  string // "subscribe" or "edit"
	freqTopic   string // topic being acted on
	confirming  bool   // showing delete confirmation
}

var frequencies = []string{"hourly", "daily", "weekly"}

func NewTopicsModel() TopicsModel {
	return TopicsModel{loading: true}
}

func loadTopics() tea.Cmd {
	return func() tea.Msg {
		subs, err := db.GetActiveSubscriptions()
		if err != nil {
			return TopicsLoadedMsg{Err: err}
		}

		var subInfos []SubInfo
		subscribedNames := make(map[string]bool)
		for _, sub := range subs {
			subscribedNames[sub.Topic] = true
			total, unread, err := db.GetSubscriptionItemCount(sub.ID)
			if err != nil {
				total, unread = 0, 0
			}
			subInfos = append(subInfos, SubInfo{Sub: sub, Total: total, Unread: unread})
		}

		var avail []AvailableCategory
		for _, cat := range models.DefaultCategories {
			if !subscribedNames[cat.Name] {
				avail = append(avail, AvailableCategory{
					Name:        cat.Name,
					DisplayName: cat.DisplayName,
					Description: cat.Description,
				})
			}
		}

		return TopicsLoadedMsg{Subscribed: subInfos, Available: avail}
	}
}

func (m TopicsModel) Init() tea.Cmd {
	return loadTopics()
}

func (m TopicsModel) Update(msg tea.Msg) (TopicsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case TopicsLoadedMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.subscribed = msg.Subscribed
		m.available = msg.Available
		m.cursor = 0
		return m, nil

	case TopicSubscribedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Failed to subscribe: %v", msg.Err)
		} else {
			m.statusMsg = fmt.Sprintf("Subscribed to %s", msg.Topic)
		}
		m.freqPicking = false
		return m, loadTopics()

	case TopicUnsubscribedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Failed to unsubscribe: %v", msg.Err)
		} else {
			m.statusMsg = fmt.Sprintf("Unsubscribed from %s", msg.Topic)
		}
		m.confirming = false
		return m, loadTopics()

	case TopicFrequencyChangedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Failed to update: %v", msg.Err)
		} else {
			m.statusMsg = fmt.Sprintf("Updated %s frequency", msg.Topic)
		}
		m.freqPicking = false
		return m, loadTopics()

	case tea.KeyMsg:
		if m.freqPicking {
			return m.updateFreqPicker(msg)
		}
		if m.confirming {
			return m.updateConfirm(msg)
		}
		return m.updateNormal(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m *TopicsModel) updateNormal(msg tea.KeyMsg) (TopicsModel, tea.Cmd) {
	switch {
	case key.Matches(msg, TopicsKeys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, TopicsKeys.Down):
		mx := m.currentListLen() - 1
		if mx < 0 {
			mx = 0
		}
		if m.cursor < mx {
			m.cursor++
		}
	case key.Matches(msg, TopicsKeys.Tab):
		if m.section == sectionSubscribed {
			m.section = sectionAvailable
		} else {
			m.section = sectionSubscribed
		}
		m.cursor = 0
		m.statusMsg = ""
	case key.Matches(msg, TopicsKeys.Enter):
		if m.section == sectionAvailable && m.cursor < len(m.available) {
			m.freqPicking = true
			m.freqCursor = 1 // default daily
			m.freqAction = "subscribe"
			m.freqTopic = m.available[m.cursor].Name
		}
	case key.Matches(msg, TopicsKeys.Delete):
		if m.section == sectionSubscribed && m.cursor < len(m.subscribed) {
			m.confirming = true
		}
	case key.Matches(msg, TopicsKeys.EditFreq):
		if m.section == sectionSubscribed && m.cursor < len(m.subscribed) {
			sub := m.subscribed[m.cursor]
			m.freqPicking = true
			m.freqAction = "edit"
			m.freqTopic = sub.Sub.Topic
			m.freqCursor = 1
			for i, f := range frequencies {
				if f == sub.Sub.Frequency {
					m.freqCursor = i
					break
				}
			}
		}
	case key.Matches(msg, TopicsKeys.Back):
		return *m, func() tea.Msg {
			return SwitchScreenMsg{Screen: ScreenDashboard}
		}
	}
	return *m, nil
}

func (m *TopicsModel) updateFreqPicker(msg tea.KeyMsg) (TopicsModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.freqCursor > 0 {
			m.freqCursor--
		}
	case "down", "j":
		if m.freqCursor < len(frequencies)-1 {
			m.freqCursor++
		}
	case "enter":
		freq := frequencies[m.freqCursor]
		topic := m.freqTopic
		if m.freqAction == "subscribe" {
			return *m, subscribeTopic(topic, freq)
		}
		return *m, changeFrequency(topic, freq)
	case "esc":
		m.freqPicking = false
	}
	return *m, nil
}

func (m *TopicsModel) updateConfirm(msg tea.KeyMsg) (TopicsModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.cursor < len(m.subscribed) {
			topic := m.subscribed[m.cursor].Sub.Topic
			m.confirming = false
			return *m, unsubscribeTopic(topic)
		}
	case "n", "N", "esc":
		m.confirming = false
	}
	return *m, nil
}

func (m TopicsModel) currentListLen() int {
	if m.section == sectionSubscribed {
		return len(m.subscribed)
	}
	return len(m.available)
}

func subscribeTopic(topic, frequency string) tea.Cmd {
	return func() tea.Msg {
		cat := models.GetCategoryByName(topic)
		sub := &models.Subscription{
			Topic:     topic,
			Frequency: frequency,
			Sources:   []string{"tavily", "rss"},
			IsActive:  true,
		}
		if cat != nil {
			sub.Category = cat.Name
		}
		err := db.CreateSubscription(sub)
		return TopicSubscribedMsg{Topic: topic, Err: err}
	}
}

func unsubscribeTopic(topic string) tea.Cmd {
	return func() tea.Msg {
		err := db.DeleteSubscription(topic)
		return TopicUnsubscribedMsg{Topic: topic, Err: err}
	}
}

func changeFrequency(topic, frequency string) tea.Cmd {
	return func() tea.Msg {
		sub, err := db.GetSubscription(topic)
		if err != nil {
			return TopicFrequencyChangedMsg{Topic: topic, Err: err}
		}
		sub.Frequency = frequency
		err = db.UpdateSubscription(sub)
		return TopicFrequencyChangedMsg{Topic: topic, Err: err}
	}
}

// Breadcrumb returns breadcrumb segments for the topics screen.
func (m TopicsModel) Breadcrumb() []string {
	return []string{"TOPICS"}
}

// StatusHints returns keybinding hints for the topics screen.
func (m TopicsModel) StatusHints() []components.KeyHint {
	if m.section == sectionSubscribed {
		return []components.KeyHint{
			{Key: "d", Desc: "unsub"},
			{Key: "e", Desc: "frequency"},
			{Key: "tab", Desc: "available"},
		}
	}
	return []components.KeyHint{
		{Key: "enter", Desc: "subscribe"},
		{Key: "tab", Desc: "subscribed"},
	}
}

// ContentView renders the topics browser content without header/footer chrome.
func (m TopicsModel) ContentView() string {
	var b strings.Builder

	if m.loading {
		b.WriteString(StyleMuted.Render("\n  Loading..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(StyleError.Render(fmt.Sprintf("\n  Error: %v", m.err)))
		return b.String()
	}

	// Frequency picker overlay
	if m.freqPicking {
		b.WriteString(m.renderFreqPicker())
		return b.String()
	}

	// Delete confirmation
	if m.confirming && m.cursor < len(m.subscribed) {
		topic := m.subscribed[m.cursor].Sub.Topic
		b.WriteString(fmt.Sprintf("\n  Unsubscribe from %s?\n\n", StyleAccent.Render(topic)))
		b.WriteString(fmt.Sprintf("  %s / %s\n",
			StyleTitle.Render("[y] yes"),
			StyleMuted.Render("[n] no"),
		))
		return b.String()
	}

	// Section tabs
	subTab := StyleMuted.Render("  SUBSCRIPTIONS")
	availTab := StyleMuted.Render("  AVAILABLE")
	if m.section == sectionSubscribed {
		subTab = StyleAccent.Render("▸ SUBSCRIPTIONS")
	} else {
		availTab = StyleAccent.Render("▸ AVAILABLE")
	}
	b.WriteString(fmt.Sprintf("\n  %s    %s\n\n", subTab, availTab))

	// Calculate column widths
	topicWidth := 20
	for _, info := range m.subscribed {
		if len(info.Sub.Topic) > topicWidth {
			topicWidth = len(info.Sub.Topic)
		}
	}
	for _, cat := range m.available {
		if len(cat.Name) > topicWidth {
			topicWidth = len(cat.Name)
		}
	}
	topicWidth += 2

	// Active section
	if m.section == sectionSubscribed {
		if len(m.subscribed) == 0 {
			b.WriteString(StyleMuted.Render("  No subscriptions yet. Press Tab to browse available topics.\n"))
		}
		for i, info := range m.subscribed {
			cursor := "  "
			nameStyle := lipgloss.NewStyle()
			if i == m.cursor {
				cursor = StyleSelectedIndicator.Render("▸ ")
				nameStyle = StyleSelected
			}

			topic := PadRight(nameStyle.Render(info.Sub.Topic), topicWidth)
			freq := PadRight(StyleMuted.Render(info.Sub.Frequency), 8)
			items := PadLeft(fmt.Sprintf("%d items", info.Total), 9)
			unread := PadLeft(fmt.Sprintf("%d unread", info.Unread), 10)

			b.WriteString(fmt.Sprintf("  %s%s %s %s  %s\n",
				cursor, topic, freq, items, unread))
		}
	} else {
		if len(m.available) == 0 {
			b.WriteString(StyleMuted.Render("  All categories subscribed!\n"))
		}
		for i, cat := range m.available {
			cursor := "  "
			nameStyle := lipgloss.NewStyle()
			if i == m.cursor {
				cursor = StyleSelectedIndicator.Render("▸ ")
				nameStyle = StyleSelected
			}

			topic := PadRight(nameStyle.Render(cat.Name), topicWidth)
			b.WriteString(fmt.Sprintf("  %s%s %s\n",
				cursor, topic,
				StyleMuted.Render(cat.DisplayName),
			))
		}
	}

	// Status message
	if m.statusMsg != "" {
		b.WriteString(fmt.Sprintf("\n  %s\n", StyleSuccess.Render("✓ "+m.statusMsg)))
	}

	return b.String()
}

func (m TopicsModel) renderFreqPicker() string {
	var b strings.Builder
	action := "Subscribe to"
	if m.freqAction == "edit" {
		action = "Set frequency for"
	}
	b.WriteString(fmt.Sprintf("\n  %s %s\n\n", action, StyleAccent.Render(m.freqTopic)))
	b.WriteString("  Choose frequency:\n\n")
	for i, f := range frequencies {
		cursor := "  "
		style := StyleMuted
		if i == m.freqCursor {
			cursor = StyleSelectedIndicator.Render("▸ ")
			style = StyleSelected
		}
		b.WriteString(fmt.Sprintf("  %s%s\n", cursor, style.Render(f)))
	}
	b.WriteString(fmt.Sprintf("\n  %s to confirm, %s to cancel\n",
		StyleTitle.Render("Enter"),
		StyleMuted.Render("Esc"),
	))
	return b.String()
}

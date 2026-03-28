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
	customAdding bool           // showing custom topic text input
	customInput  textinput.Model // text input for custom topic name
}

var frequencies = []string{"hourly", "daily", "weekly"}

func NewTopicsModel() TopicsModel {
	ti := textinput.New()
	ti.Placeholder = "e.g., nvidia chip stocks, golang concurrency..."
	ti.CharLimit = 200
	return TopicsModel{loading: true, customInput: ti}
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
			// Source subscriptions are managed in the Sources screen
			if sub.IsSourceSubscription() {
				continue
			}
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
	// Forward non-key messages to textinput when custom adding (for blink cursor)
	if m.customAdding {
		if _, ok := msg.(tea.KeyMsg); !ok {
			var cmd tea.Cmd
			m.customInput, cmd = m.customInput.Update(msg)
			return m, cmd
		}
	}

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
		if m.customAdding {
			return m.updateCustomInput(msg)
		}
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
		if m.section == sectionAvailable {
			if m.cursor == 0 {
				// "+ Custom topic..." entry
				m.customAdding = true
				m.customInput.Reset()
				m.customInput.Focus()
				return *m, textinput.Blink
			}
			// Predefined categories (offset by 1 for custom entry)
			catIdx := m.cursor - 1
			if catIdx < len(m.available) {
				m.freqPicking = true
				m.freqCursor = 1 // default daily
				m.freqAction = "subscribe"
				m.freqTopic = m.available[catIdx].Name
			}
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

func (m *TopicsModel) updateCustomInput(msg tea.KeyMsg) (TopicsModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		topic := strings.TrimSpace(m.customInput.Value())
		if topic == "" {
			return *m, nil
		}
		m.customAdding = false
		m.freqPicking = true
		m.freqCursor = 1 // default daily
		m.freqAction = "subscribe"
		m.freqTopic = topic
	case "esc":
		m.customAdding = false
	default:
		var cmd tea.Cmd
		m.customInput, cmd = m.customInput.Update(msg)
		return *m, cmd
	}
	return *m, nil
}

func (m TopicsModel) currentListLen() int {
	if m.section == sectionSubscribed {
		return len(m.subscribed)
	}
	// +1 for the "+ Custom topic..." entry at index 0
	return len(m.available) + 1
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

	// Custom topic input overlay
	if m.customAdding {
		fmt.Fprintf(&b, "\n  %s\n\n", StyleAccent.Render("Custom Topic"))
		b.WriteString("  Enter a topic to follow:\n\n")
		fmt.Fprintf(&b, "  %s\n\n", m.customInput.View())
		fmt.Fprintf(&b, "  %s to continue, %s to cancel\n",
			StyleTitle.Render("Enter"),
			StyleMuted.Render("Esc"),
		)
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
		fmt.Fprintf(&b, "\n  Unsubscribe from %s?\n\n", StyleAccent.Render(topic))
		fmt.Fprintf(&b, "  %s / %s\n",
			StyleTitle.Render("[y] yes"),
			StyleMuted.Render("[n] no"),
		)
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
	fmt.Fprintf(&b, "\n  %s    %s\n\n", subTab, availTab)

	// Calculate column widths (visual width for Unicode safety)
	topicWidth := 20
	for _, info := range m.subscribed {
		w := lipgloss.Width(info.Sub.Topic)
		if w > topicWidth {
			topicWidth = w
		}
	}
	for _, cat := range m.available {
		w := lipgloss.Width(cat.Name)
		if w > topicWidth {
			topicWidth = w
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

			displayTopic := info.Sub.Topic
			if info.Sub.DisplayName != "" {
				displayTopic = info.Sub.DisplayName
			}
			topic := PadRight(nameStyle.Render(strings.ToUpper(displayTopic)), topicWidth)

			freq := PadRight(StyleMuted.Render(info.Sub.Frequency), 8)
			items := PadLeft(fmt.Sprintf("%d items", info.Total), 9)
			unread := PadLeft(fmt.Sprintf("%d unread", info.Unread), 10)

			fmt.Fprintf(&b, "  %s%s %s %s  %s\n",
				cursor, topic, freq, items, unread)
		}
	} else {
		// "+ Custom topic..." entry at index 0
		{
			cursor := "  "
			nameStyle := StyleCyan
			if m.cursor == 0 {
				cursor = StyleSelectedIndicator.Render("▸ ")
				nameStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
			}
			fmt.Fprintf(&b, "  %s%s\n", cursor, nameStyle.Render("+ Custom topic..."))
		}

		if len(m.available) == 0 {
			b.WriteString(StyleMuted.Render("\n  All predefined categories subscribed!\n"))
		}
		for i, cat := range m.available {
			cursor := "  "
			nameStyle := lipgloss.NewStyle()
			// Offset by 1 for the custom entry
			if i+1 == m.cursor {
				cursor = StyleSelectedIndicator.Render("▸ ")
				nameStyle = StyleSelected
			}

			topic := PadRight(nameStyle.Render(strings.ToUpper(cat.Name)), topicWidth)
			fmt.Fprintf(&b, "  %s%s %s\n",
				cursor, topic,
				StyleMuted.Render(cat.DisplayName),
			)
			// Show description on next line
			if cat.Description != "" {
				fmt.Fprintf(&b, "      %s\n", StyleWarmMuted.Render(cat.Description))
			}
		}
	}

	// Status message
	if m.statusMsg != "" {
		fmt.Fprintf(&b, "\n  %s\n", StyleSuccess.Render("✓ "+m.statusMsg))
	}

	return b.String()
}

func (m TopicsModel) renderFreqPicker() string {
	var b strings.Builder
	action := "Subscribe to"
	if m.freqAction == "edit" {
		action = "Set frequency for"
	}
	fmt.Fprintf(&b, "\n  %s %s\n\n", action, StyleAccent.Render(m.freqTopic))
	b.WriteString("  Choose frequency:\n\n")
	for i, f := range frequencies {
		cursor := "  "
		style := StyleMuted
		if i == m.freqCursor {
			cursor = StyleSelectedIndicator.Render("▸ ")
			style = StyleSelected
		}
		fmt.Fprintf(&b, "  %s%s\n", cursor, style.Render(f))
	}
	fmt.Fprintf(&b, "\n  %s to confirm, %s to cancel\n",
		StyleTitle.Render("Enter"),
		StyleMuted.Render("Esc"),
	)
	return b.String()
}

package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/oluoyefeso/termiflow/internal/db"
	"github.com/oluoyefeso/termiflow/internal/tui/components"
	"github.com/oluoyefeso/termiflow/pkg/models"
)

// DetailModel shows a single article in a scrollable view.
type DetailModel struct {
	item      *models.FeedItem
	items     []*models.FeedItem // full list for n/p navigation
	index     int                // position in items
	scrollY   int                // scroll offset
	width     int
	height    int
	statusMsg string // transient status message
	topic     string // topic name for breadcrumb
}

func NewDetailModel(item *models.FeedItem, items []*models.FeedItem, index int) DetailModel {
	return DetailModel{
		item:  item,
		items: items,
		index: index,
	}
}

func markItemRead(id int64) tea.Cmd {
	return func() tea.Msg {
		err := db.MarkItemRead(id)
		return ItemMarkedReadMsg{ItemID: id, Err: err}
	}
}

func (m DetailModel) Init() tea.Cmd {
	if m.item != nil && !m.item.IsRead {
		return markItemRead(m.item.ID)
	}
	return nil
}

func (m DetailModel) Update(msg tea.Msg) (DetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case ItemMarkedReadMsg:
		if msg.Err == nil && m.item != nil && m.item.ID == msg.ItemID {
			m.item.IsRead = true
		}
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DetailKeys.Back):
			return m, func() tea.Msg {
				return SwitchScreenMsg{Screen: ScreenFeed}
			}
		case key.Matches(msg, DetailKeys.Up):
			if m.scrollY > 0 {
				m.scrollY--
			}
		case key.Matches(msg, DetailKeys.Down):
			m.scrollY++
		case key.Matches(msg, DetailKeys.Open):
			if m.item != nil && m.item.SourceURL != "" {
				if err := openBrowser(m.item.SourceURL); err != nil {
					m.statusMsg = "Could not open browser"
				} else {
					m.statusMsg = "Opened in browser"
				}
			}
		case key.Matches(msg, DetailKeys.MarkRead):
			if m.item != nil && !m.item.IsRead {
				return m, markItemRead(m.item.ID)
			}
		case key.Matches(msg, DetailKeys.Next):
			if m.index < len(m.items)-1 {
				m.index++
				m.item = m.items[m.index]
				m.scrollY = 0
				m.statusMsg = ""
				if !m.item.IsRead {
					return m, markItemRead(m.item.ID)
				}
			}
		case key.Matches(msg, DetailKeys.Prev):
			if m.index > 0 {
				m.index--
				m.item = m.items[m.index]
				m.scrollY = 0
				m.statusMsg = ""
				if !m.item.IsRead {
					return m, markItemRead(m.item.ID)
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// Breadcrumb returns breadcrumb segments for the detail screen.
func (m DetailModel) Breadcrumb() []string {
	crumbs := []string{}
	if m.topic != "" {
		crumbs = append(crumbs, strings.ToUpper(m.topic))
	}
	if m.item != nil {
		title := m.item.Title
		runes := []rune(title)
		if len(runes) > 30 {
			title = string(runes[:27]) + "..."
		}
		crumbs = append(crumbs, title)
	}
	return crumbs
}

// StatusHints returns keybinding hints for the detail screen.
func (m DetailModel) StatusHints() []components.KeyHint {
	return []components.KeyHint{
		{Key: "o", Desc: "open"},
		{Key: "m", Desc: "read"},
		{Key: "n/p", Desc: "next/prev"},
	}
}

// ContentView renders the article detail content without header/footer chrome.
func (m DetailModel) ContentView() string {
	if m.item == nil {
		return StyleMuted.Render("  No article selected")
	}

	w := m.width
	if w == 0 {
		w = 69
	}

	// Content width capped at 72 for readability
	contentWidth := w - 4
	if contentWidth > 72 {
		contentWidth = 72
	}

	// Build the full content first, then apply scroll
	var lines []string

	// Title
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("  %s", StyleAccent.Render(m.item.Title)))

	// Separator
	lines = append(lines, "  "+StyleMuted.Render(strings.Repeat("─", contentWidth)))

	// Meta: source · time · relevance bar
	meta := fmt.Sprintf("  %s · %s",
		StyleMuted.Render(m.item.SourceName),
		StyleMuted.Render(m.item.TimeAgo()),
	)
	if m.item.RelevanceScore > 0 {
		meta += fmt.Sprintf(" · %s %s",
			RelevanceBar(m.item.RelevanceScore),
			StyleMuted.Render(fmt.Sprintf("%.0f%%", m.item.RelevanceScore*100)))
	}
	lines = append(lines, meta)

	if m.item.SourceURL != "" {
		lines = append(lines, fmt.Sprintf("  %s", StyleMuted.Render(m.item.SourceURL)))
	}

	// Horizontal rule between metadata and content
	lines = append(lines, "")
	lines = append(lines, "  "+StyleMuted.Render(strings.Repeat("─", contentWidth)))
	lines = append(lines, "")

	// Summary
	if m.item.Summary != "" {
		wrapped := wrapText(m.item.Summary, contentWidth)
		for _, line := range strings.Split(wrapped, "\n") {
			lines = append(lines, "  "+line)
		}
		lines = append(lines, "")
	}

	// Content (if different from summary)
	if m.item.Content != "" && m.item.Content != m.item.Summary {
		lines = append(lines, "  "+LabeledRule("Full content", contentWidth))
		lines = append(lines, "")
		wrapped := wrapText(m.item.Content, contentWidth)
		for _, line := range strings.Split(wrapped, "\n") {
			lines = append(lines, "  "+line)
		}
		lines = append(lines, "")
	}

	// Tags with dotted separator
	if len(m.item.Tags) > 0 {
		lines = append(lines, "  "+DottedRule(contentWidth))
		lines = append(lines, "")
		var tags []string
		for _, tag := range m.item.Tags {
			tags = append(tags, StyleTag.Render("#"+tag))
		}
		lines = append(lines, "   "+strings.Join(tags, "  "))
		lines = append(lines, "")
	}

	// Status message
	if m.statusMsg != "" {
		lines = append(lines, fmt.Sprintf("  %s %s",
			StyleSuccess.Render("✓"), m.statusMsg))
		lines = append(lines, "")
	}

	// Position indicator
	lines = append(lines, fmt.Sprintf("  %s",
		StyleMuted.Render(fmt.Sprintf("%d/%d", m.index+1, len(m.items)))))

	// Clamp scroll
	totalLines := len(lines)
	viewportHeight := m.height
	if viewportHeight < 5 {
		viewportHeight = 20
	}
	maxScroll := totalLines - viewportHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrollY > maxScroll {
		m.scrollY = maxScroll
	}

	// Apply scroll window
	endIdx := m.scrollY + viewportHeight
	if endIdx > totalLines {
		endIdx = totalLines
	}
	visible := lines[m.scrollY:endIdx]

	var b strings.Builder
	b.WriteString(strings.Join(visible, "\n"))
	b.WriteString("\n")

	// Scroll indicator
	if totalLines > viewportHeight {
		pct := 0
		if maxScroll > 0 {
			pct = m.scrollY * 100 / maxScroll
		}
		b.WriteString(StyleMuted.Render(fmt.Sprintf("  ── %d%% ──", pct)))
		b.WriteString("\n")
	}

	return b.String()
}

// wrapText wraps text to the given width.
func wrapText(text string, width int) string {
	if width <= 0 {
		width = 65
	}
	var result strings.Builder
	words := strings.Fields(text)
	lineLen := 0

	for i, word := range words {
		if i > 0 {
			if lineLen+len(word)+1 > width {
				result.WriteString("\n")
				lineLen = 0
			} else {
				result.WriteString(" ")
				lineLen++
			}
		}
		result.WriteString(word)
		lineLen += len(word)
	}

	return result.String()
}

// openBrowser opens the given URL in the default browser.
// Only http/https URLs are allowed.
func openBrowser(rawURL string) error {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return fmt.Errorf("unsupported URL scheme")
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "linux":
		cmd = exec.Command("xdg-open", rawURL)
	default:
		cmd = exec.Command("open", rawURL)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

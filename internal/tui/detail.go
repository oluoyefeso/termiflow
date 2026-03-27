package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/oluoyefeso/termiflow/internal/db"
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

func (m DetailModel) View() string {
	if m.item == nil {
		return StyleMuted.Render("  No article selected")
	}

	w := m.width
	if w == 0 {
		w = 69
	}

	// Build the full content first, then apply scroll
	var lines []string

	// Header
	lines = append(lines, StyleMuted.Render(Bar("═", w)))
	lines = append(lines, fmt.Sprintf("  %s", StyleAccent.Render(m.item.Title)))
	lines = append(lines, StyleMuted.Render(Bar("─", w)))

	// Meta
	meta := fmt.Sprintf("  %s  %s",
		StyleMuted.Render(m.item.SourceName),
		StyleMuted.Render(m.item.TimeAgo()),
	)
	if m.item.RelevanceScore > 0 {
		meta += fmt.Sprintf("  %s",
			StyleTitle.Render(fmt.Sprintf("%.0f%% relevant", m.item.RelevanceScore*100)))
	}
	lines = append(lines, meta)

	if m.item.SourceURL != "" {
		lines = append(lines, fmt.Sprintf("  %s", StyleMuted.Render(m.item.SourceURL)))
	}

	lines = append(lines, "")

	// Summary
	if m.item.Summary != "" {
		wrapped := wrapText(m.item.Summary, w-4)
		for _, line := range strings.Split(wrapped, "\n") {
			lines = append(lines, "  "+line)
		}
		lines = append(lines, "")
	}

	// Content (if different from summary)
	if m.item.Content != "" && m.item.Content != m.item.Summary {
		lines = append(lines, StyleMuted.Render("  ─── Full content ───"))
		lines = append(lines, "")
		wrapped := wrapText(m.item.Content, w-4)
		for _, line := range strings.Split(wrapped, "\n") {
			lines = append(lines, "  "+line)
		}
		lines = append(lines, "")
	}

	// Tags
	if len(m.item.Tags) > 0 {
		var tags []string
		for _, tag := range m.item.Tags {
			tags = append(tags, StyleTag.Render("["+tag+"]"))
		}
		lines = append(lines, "  "+strings.Join(tags, " "))
		lines = append(lines, "")
	}

	// Clamp scroll
	totalLines := len(lines)
	viewportHeight := m.height - 3 // reserve for status bar
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

	// Status bar
	b.WriteString(StyleMuted.Render(Bar("─", w)) + "\n")

	posLabel := StyleMuted.Render(fmt.Sprintf("%d/%d", m.index+1, len(m.items)))
	hints := []string{
		StyleTitle.Render("[o]") + " " + StyleMuted.Render("open"),
		StyleTitle.Render("[m]") + " " + StyleMuted.Render("read"),
		StyleTitle.Render("[n/p]") + " " + StyleMuted.Render("next/prev"),
		StyleTitle.Render("[esc]") + " " + StyleMuted.Render("back"),
		posLabel,
	}
	if m.statusMsg != "" {
		hints = append(hints, StyleSuccess.Render(m.statusMsg))
	}
	b.WriteString(" " + strings.Join(hints, "  "))

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
	// Reap the child process in the background
	go func() { _ = cmd.Wait() }()
	return nil
}

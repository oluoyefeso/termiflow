package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

const lineWidth = 65

func Header(title string) string {
	return HeaderBoxStyle.Render(fmt.Sprintf("  %s", title))
}

func HeaderWithDate(title string) string {
	date := time.Now().Format("Jan 2, 2006 · 15:04")
	padding := lineWidth - len(title) - len(date) - 2
	if padding < 1 {
		padding = 1
	}
	content := fmt.Sprintf("  %s%s%s", title, strings.Repeat(" ", padding), MutedStyle.Render(date))
	return HeaderBoxStyle.Render(content)
}

func Section(title string, count int, countLabel string) string {
	var countStr string
	if count > 0 {
		countStr = fmt.Sprintf("%d %s", count, countLabel)
	}

	padding := lineWidth - len(title) - len(countStr) - 4
	if padding < 1 {
		padding = 1
	}

	bullet := SuccessStyle.Render("●")
	line := strings.Repeat("─", lineWidth)

	return fmt.Sprintf("\n %s %s%s%s\n %s\n",
		bullet,
		BoldStyle.Render(title),
		strings.Repeat(" ", padding),
		MutedStyle.Render(countStr),
		MutedStyle.Render(line),
	)
}

func Divider() string {
	return fmt.Sprintf("\n   %s\n", MutedStyle.Render(strings.Repeat("─ ", 33)))
}

func SmallDivider() string {
	return MutedStyle.Render(strings.Repeat("─", lineWidth))
}

func Success(message string) string {
	return fmt.Sprintf(" %s %s\n", SuccessStyle.Render("✓"), message)
}

func Error(message string) string {
	return fmt.Sprintf(" %s %s\n", ErrorStyle.Render("✗"), message)
}

func Warning(message string) string {
	return fmt.Sprintf(" %s %s\n", WarningStyle.Render("!"), message)
}

func Info(label, value string) string {
	padding := 16 - len(label)
	if padding < 1 {
		padding = 1
	}
	return fmt.Sprintf("   %s%s%s\n",
		MutedStyle.Render(label+":"),
		strings.Repeat(" ", padding),
		value,
	)
}

func Bullet(text string) string {
	return fmt.Sprintf(" → %s\n", text)
}

func Indent(text string, spaces int) string {
	indent := strings.Repeat(" ", spaces)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n")
}

func WrapText(text string, width int) string {
	if len(text) <= width {
		return text
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

func Tags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}

	var parts []string
	for _, tag := range tags {
		parts = append(parts, TagStyle.Render("#"+tag))
	}
	return strings.Join(parts, "  ")
}

func FormatFeedItem(title, source, timeAgo, summary string, tags []string) string {
	var b strings.Builder

	// Title
	b.WriteString(fmt.Sprintf("   %s\n", BoldStyle.Render(title)))

	// Source and time
	b.WriteString(fmt.Sprintf("   %s · %s\n",
		MutedStyle.Render(source),
		MutedStyle.Render(timeAgo),
	))

	// Summary
	if summary != "" {
		b.WriteString("   \n")
		wrapped := WrapText(summary, 60)
		for _, line := range strings.Split(wrapped, "\n") {
			b.WriteString(fmt.Sprintf("   %s\n", line))
		}
	}

	// Tags
	if len(tags) > 0 {
		b.WriteString("   \n")
		b.WriteString(fmt.Sprintf("   %s\n", Tags(tags)))
	}

	return b.String()
}

func SubscriptionRow(topic, frequency string, total, unread int, isCategory bool) string {
	bullet := "○"
	if isCategory {
		bullet = "●"
	}

	stats := ""
	if total > 0 {
		stats = fmt.Sprintf("%d items", total)
		if unread > 0 {
			stats += fmt.Sprintf(" (%d unread)", unread)
		}
	}

	return fmt.Sprintf("   %s %-24s %-10s %s\n",
		SuccessStyle.Render(bullet),
		topic,
		MutedStyle.Render(frequency),
		MutedStyle.Render(stats),
	)
}

func CategoryRow(name, displayName string) string {
	return fmt.Sprintf("   %s %-24s %s\n",
		MutedStyle.Render("○"),
		name,
		MutedStyle.Render(displayName),
	)
}

func Tip(text string) string {
	return fmt.Sprintf("\n %s %s\n",
		MutedStyle.Render("Tip:"),
		text,
	)
}

func Footer(items, topics int, lastUpdated string) string {
	return fmt.Sprintf("\n %s\n %s\n",
		SmallDivider(),
		MutedStyle.Render(fmt.Sprintf(" %d items · %d topics · Last updated %s", items, topics, lastUpdated)),
	)
}

func NoColor(enable bool) {
	if enable {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
}

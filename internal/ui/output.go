package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

const lineWidth = 69

// bar returns a horizontal rule using the given character
func bar(ch string, width int) string {
	return strings.Repeat(ch, width)
}

func Header(title string) string {
	top := MutedStyle.Render(bar("═", lineWidth))
	label := AccentStyle.Render("  " + strings.ToUpper(title))
	bot := MutedStyle.Render(bar("═", lineWidth))
	return fmt.Sprintf("%s\n%s\n%s", top, label, bot)
}

func HeaderWithDate(title string) string {
	date := time.Now().Format("02 Jan 06 · 15:04")
	label := strings.ToUpper(title)
	padding := lineWidth - len(label) - len(date) - 4
	if padding < 1 {
		padding = 1
	}
	top := MutedStyle.Render(bar("═", lineWidth))
	content := fmt.Sprintf("  %s%s%s",
		AccentStyle.Render(label),
		strings.Repeat(" ", padding),
		MutedStyle.Render(date),
	)
	bot := MutedStyle.Render(bar("═", lineWidth))
	return fmt.Sprintf("%s\n%s\n%s", top, content, bot)
}

func Section(title string, count int, countLabel string) string {
	label := strings.ToUpper(title)
	countStr := ""
	if count > 0 {
		countStr = fmt.Sprintf("%d %s", count, strings.ToUpper(countLabel))
	}
	dashWidth := lineWidth - len(label) - len(countStr) - 3
	if dashWidth < 1 {
		dashWidth = 1
	}
	return fmt.Sprintf("\n %s %s %s\n",
		TitleStyle.Render("▸"),
		AccentStyle.Render(label),
		MutedStyle.Render(bar("─", dashWidth)+" "+countStr),
	)
}

func Divider() string {
	return fmt.Sprintf("   %s\n", MutedStyle.Render(bar("·", lineWidth-3)))
}

func SmallDivider() string {
	return MutedStyle.Render(bar("─", lineWidth))
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
	return fmt.Sprintf(" %s %s\n", MutedStyle.Render("›"), text)
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
		parts = append(parts, TagStyle.Render("["+tag+"]"))
	}
	return strings.Join(parts, " ")
}

func FormatFeedItem(title, source, timeAgo, summary string, tags []string) string {
	var b strings.Builder

	// Title — amber, bold, no indent bloat
	b.WriteString(fmt.Sprintf("  %s\n", AccentStyle.Render(title)))

	// Source | time on same line, compact
	b.WriteString(fmt.Sprintf("  %s  %s\n",
		MutedStyle.Render(source),
		MutedStyle.Render(timeAgo),
	))

	// Summary — slightly indented, wrapped
	if summary != "" {
		wrapped := WrapText(summary, 65)
		for _, line := range strings.Split(wrapped, "\n") {
			b.WriteString(fmt.Sprintf("  %s\n", MutedStyle.Render(line)))
		}
	}

	// Tags — inline, compact
	if len(tags) > 0 {
		b.WriteString(fmt.Sprintf("  %s\n", Tags(tags)))
	}

	return b.String()
}

func SubscriptionRow(topic, frequency string, total, unread int, isCategory bool) string {
	bullet := MutedStyle.Render("○")
	if isCategory {
		bullet = TitleStyle.Render("●")
	}

	stats := ""
	if total > 0 {
		stats = fmt.Sprintf("%d items", total)
		if unread > 0 {
			stats += fmt.Sprintf(" (%d unread)", unread)
		}
	}

	return fmt.Sprintf("  %s %-26s %-10s %s\n",
		bullet,
		topic,
		MutedStyle.Render(frequency),
		MutedStyle.Render(stats),
	)
}

func CategoryRow(name, displayName string) string {
	return fmt.Sprintf("  %s %-26s %s\n",
		MutedStyle.Render("○"),
		name,
		MutedStyle.Render(displayName),
	)
}

func Tip(text string) string {
	return fmt.Sprintf("\n  %s %s\n",
		MutedStyle.Render("tip:"),
		text,
	)
}

func Footer(items, topics int, lastUpdated string) string {
	line := SmallDivider()
	stats := MutedStyle.Render(fmt.Sprintf("  %d items  %d topics  updated %s", items, topics, lastUpdated))
	return fmt.Sprintf("\n%s\n%s\n", line, stats)
}

// Banner renders a notification banner with type-based coloring.
// Types: info=cyan, warning=amber, update=blue, breaking=red, version=blue.
func Banner(bannerType string, message string) string {
	var style lipgloss.Style
	icon := "▲"

	switch bannerType {
	case "info":
		style = CyanStyle
	case "warning":
		style = WarningStyle
	case "update", "version":
		style = BlueStyle
	case "breaking":
		style = ErrorStyle
		icon = "!"
	default:
		style = MutedStyle
	}

	return fmt.Sprintf(" %s %s\n", style.Render(icon), style.Render(message))
}

func NoColor(enable bool) {
	if enable {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
}

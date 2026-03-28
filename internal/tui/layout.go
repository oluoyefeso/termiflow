package tui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"github.com/oluoyefeso/termiflow/internal/tui/components"
)

// Header lines: top bar + content + bottom bar = 3 lines
const headerLines = 3

// Footer lines: separator + hints = 2 lines
const footerLines = 2

// bannerHeight returns the number of lines a single banner card occupies.
const bannerCardHeight = 3 // top border + content + bottom border + gap handled by caller

// RenderHeader renders the persistent header bar with breadcrumb, unread badge,
// and last refresh indicator.
func RenderHeader(width int, breadcrumb []string, unreadCount int, lastRefresh time.Time) string {
	// Build breadcrumb trail
	crumbs := "TERMIFLOW"
	for _, seg := range breadcrumb {
		crumbs += " › " + seg
	}

	// Right side: unread badge + refresh indicator
	var rightParts []string

	if unreadCount > 0 {
		rightParts = append(rightParts, StyleSuccess.Render("●")+" "+StyleUnreadBadge.Render(fmt.Sprintf("%d unread", unreadCount)))
	}

	if !lastRefresh.IsZero() {
		ago := time.Since(lastRefresh)
		label := "just now"
		if ago > time.Hour {
			label = fmt.Sprintf("%dh ago", int(ago.Hours()))
		} else if ago > time.Minute {
			label = fmt.Sprintf("%dm ago", int(ago.Minutes()))
		}
		// Color the dot based on freshness
		dot := StyleSuccess.Render("⦻") // green = fresh (< 1h)
		if ago > time.Hour {
			dot = StyleWarning.Render("⦻") // amber = stale (> 1h)
		}
		rightParts = append(rightParts, dot+" "+StyleMuted.Render(label))
	}

	rightStr := strings.Join(rightParts, "  ")

	// Calculate padding
	crumbRendered := StyleAccent.Render(crumbs)
	crumbWidth := lipgloss.Width(crumbRendered)
	rightWidth := lipgloss.Width(rightStr)
	padding := width - crumbWidth - rightWidth - 2 // 2 for leading indent
	if padding < 1 {
		padding = 1
	}

	top := StyleMuted.Render(Bar("═", width))
	content := fmt.Sprintf(" %s%s%s",
		crumbRendered,
		strings.Repeat(" ", padding),
		rightStr,
	)
	bot := StyleMuted.Render(Bar("═", width))

	return fmt.Sprintf("%s\n%s\n%s", top, content, bot)
}

// RenderBanner renders a notification banner as a bordered card.
func RenderBanner(banner BannerMsg, width int) string {
	icon := "▲"
	label := strings.ToUpper(banner.Type)
	var borderStyle, textStyle lipgloss.Style
	double := false

	switch banner.Type {
	case "info":
		borderStyle = StyleCyan
		textStyle = StyleCyan
	case "warning":
		borderStyle = StyleWarning
		textStyle = StyleWarning
	case "update", "version":
		borderStyle = StyleBlue
		textStyle = StyleBlue
		label = "UPDATE"
	case "breaking":
		icon = "!"
		borderStyle = StyleError
		textStyle = StyleError
		double = true
	default:
		borderStyle = StyleMuted
		textStyle = StyleMuted
	}

	innerWidth := width - 4 // borders + padding
	if innerWidth < 10 {
		innerWidth = 10
	}

	msg := fmt.Sprintf("  %s %s", icon, banner.Message)
	// Truncate if too long
	if utf8.RuneCountInString(msg) > innerWidth {
		runes := []rune(msg)
		msg = string(runes[:innerWidth-3]) + "..."
	}
	// Pad to fill
	msgWidth := utf8.RuneCountInString(msg)
	if msgWidth < innerWidth {
		msg += strings.Repeat(" ", innerWidth-msgWidth)
	}

	labelLen := utf8.RuneCountInString(label)
	// Top border must match bottom border width (innerWidth + 4 total)
	// topLeft = "┌─ " + label + " " = 4 + labelLen chars
	// topRight = fill + "┐" = fill + 1 chars
	// Total = 4 + labelLen + fill + 1 = innerWidth + 4 → fill = innerWidth - labelLen - 1
	fillLen := innerWidth - labelLen - 1
	if fillLen < 0 {
		fillLen = 0
	}

	if double {
		topLeft := "╔═ " + label + " "
		topRight := strings.Repeat("═", fillLen) + "╗"
		top := borderStyle.Render(topLeft + topRight)
		mid := borderStyle.Render("║") + textStyle.Render(msg) + borderStyle.Render("║")
		bot := borderStyle.Render("╚" + strings.Repeat("═", innerWidth+2) + "╝")
		return fmt.Sprintf("%s\n%s\n%s", top, mid, bot)
	}

	topLeft := "┌─ " + label + " "
	topRight := strings.Repeat("─", fillLen) + "┐"
	top := borderStyle.Render(topLeft + topRight)
	mid := borderStyle.Render("│") + textStyle.Render(msg) + borderStyle.Render("│")
	bot := borderStyle.Render("└" + strings.Repeat("─", innerWidth+2) + "┘")
	return fmt.Sprintf("%s\n%s\n%s", top, mid, bot)
}

// RenderCard renders a bordered card with a title label and content lines.
func RenderCard(title string, contentLines []string, width int) string {
	innerWidth := width - 4 // borders + padding
	if innerWidth < 10 {
		innerWidth = 10
	}

	// Top border with title (must match bottom border width: innerWidth + 4)
	topLeft := "┌─ " + title + " "
	topLeftLen := 3 + utf8.RuneCountInString(title) + 1 // "┌─ " + title + " "
	remaining := (innerWidth + 4) - topLeftLen - 1       // -1 for closing "┐"
	if remaining < 0 {
		remaining = 0
	}
	topRight := strings.Repeat("─", remaining) + "┐"
	top := StyleMuted.Render(topLeft + topRight)

	// Content lines
	var lines []string
	lines = append(lines, top)
	contentArea := innerWidth - 2 // subtract 2 for left padding spaces
	if contentArea < 0 {
		contentArea = 0
	}
	for _, line := range contentLines {
		lineWidth := lipgloss.Width(line)
		pad := contentArea - lineWidth
		if pad < 0 {
			pad = 0
		}
		lines = append(lines, StyleMuted.Render("│")+"  "+line+strings.Repeat(" ", pad)+"  "+StyleMuted.Render("│"))
	}

	// Bottom border
	bot := StyleMuted.Render("└" + strings.Repeat("─", innerWidth+2) + "┘")
	lines = append(lines, bot)

	return strings.Join(lines, "\n")
}

// RelevanceBar returns a 4-character visual bar for a relevance score (0.0-1.0).
func RelevanceBar(score float64) string {
	pct := score * 100
	var bar string
	switch {
	case pct >= 90:
		bar = "████"
	case pct >= 75:
		bar = "███░"
	case pct >= 60:
		bar = "██▓░"
	case pct >= 40:
		bar = "██░░"
	case pct >= 20:
		bar = "█░░░"
	default:
		bar = "░░░░"
	}
	return StyleTitle.Render(bar)
}

// ActivityBar returns a 6-character activity bar showing unread/total ratio.
func ActivityBar(unread, total int) string {
	if total == 0 {
		return StyleMuted.Render("░░░░░░")
	}

	filled := 0
	if unread > 0 {
		filled = (unread * 6) / total
		if filled == 0 {
			filled = 1 // minimum 1 if any unread
		}
		if filled > 6 {
			filled = 6
		}
	}

	filledStr := strings.Repeat("█", filled)
	emptyStr := strings.Repeat("░", 6-filled)
	if filled > 0 {
		return StyleTitle.Render(filledStr) + StyleMuted.Render(emptyStr)
	}
	return StyleMuted.Render(emptyStr)
}

// LabeledRule renders a horizontal rule with an embedded label: ── Label ──────
func LabeledRule(label string, width int) string {
	prefix := "── " + label + " "
	remaining := width - utf8.RuneCountInString(prefix)
	if remaining < 0 {
		remaining = 0
	}
	return StyleMuted.Render(prefix + strings.Repeat("─", remaining))
}

// DottedRule renders a dotted separator line.
func DottedRule(width int) string {
	return StyleMuted.Render(strings.Repeat("╌", width))
}

// AnimatedSpinner returns the loading animation character for a given frame.
func AnimatedSpinner(frame int) string {
	frames := []string{"▁", "▂", "▃", "▄", "▃", "▂"}
	idx := frame % len(frames)
	return StyleCyan.Render(frames[idx])
}

// AnimatedDots returns animated dots for "Searching" state.
func AnimatedDots(frame int) string {
	dots := (frame / 3) % 4 // cycle through 0,1,2,3 dots
	return strings.Repeat("·", dots)
}

// PadRight pads a string to the given width with spaces.
// Uses rune-aware width measurement.
func PadRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// PadLeft right-aligns a string within the given width.
func PadLeft(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return strings.Repeat(" ", width-w) + s
}

// ContentHeight calculates available height for screen content.
func ContentHeight(totalHeight, bannerCount int) int {
	bannerLines := bannerCount * (bannerCardHeight + 1) // +1 for gap between banners
	h := totalHeight - headerLines - footerLines - bannerLines
	if h < 5 {
		h = 5
	}
	return h
}

// RenderStatusBar renders the persistent footer with inverted key style.
func RenderStatusBar(hints []components.KeyHint, width int) string {
	if len(hints) == 0 {
		return ""
	}

	var parts []string
	for _, h := range hints {
		key := StyleInvertedKey.Render(" " + h.Key + " ")
		parts = append(parts, key+" "+StyleMuted.Render(h.Desc))
	}

	content := strings.Join(parts, "  ")
	sep := StyleMuted.Render(Bar("═", width))

	return sep + "\n " + content
}

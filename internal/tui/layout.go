package tui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"github.com/oluoyefeso/termiflow/internal/config"
	"github.com/oluoyefeso/termiflow/internal/tui/components"
)

// headerLines: top bar + 3 content lines + bottom bar = 5 lines (+ 1 if breadcrumb present)
const headerLines = 6 // reserve max (with breadcrumb)

// Footer lines: separator + hints = 2 lines
const footerLines = 2

// bannerHeight returns the number of lines a single banner card occupies.
const bannerCardHeight = 3 // top border + content + bottom border + gap handled by caller

// ASCII art logo (box-drawing style, 3 lines)
var asciiLogo = []string{
	" ╔╦╗╔═╗╦═╗╔╦╗╦╔═╗╦  ╔═╗╦ ╦",
	"  ║ ║╣ ╠╦╝║║║║╠╣ ║  ║ ║║║║",
	"  ║ ╚═╝╩╚═╩ ╩╩╚  ╩═╝╚═╝╚╩╝",
}

// HeaderInfo holds pre-computed values for the header so we don't call config on every render.
type HeaderInfo struct {
	Mode      string // "Managed" or "Self-hosted"
	Endpoint  string // API URL or provider name
	Version   string
	APIHealth string // "ok", "degraded", "unknown", or "" (not yet checked)
}

// NewHeaderInfo reads config once and returns header display values.
func NewHeaderInfo(version string) HeaderInfo {
	h := HeaderInfo{Version: version}
	if config.IsManagedMode() {
		h.Mode = "Managed"
		h.Endpoint = "termiflow cloud"
	} else {
		h.Mode = "Self-hosted"
		cfg := config.Get()
		h.Endpoint = cfg.General.DefaultProvider
	}
	return h
}

// RenderHeader renders the persistent header with ASCII logo, stats, and breadcrumb.
func RenderHeader(width int, breadcrumb []string, unreadCount, subCount int, lastRefresh time.Time, info HeaderInfo) string {
	top := StyleMuted.Render(Bar("═", width))

	// Logo width (visual width of widest line)
	logoWidth := 0
	for _, l := range asciiLogo {
		if w := lipgloss.Width(l); w > logoWidth {
			logoWidth = w
		}
	}

	// Narrow terminal fallback: skip logo, use compact text header
	if width < logoWidth+20 {
		// Compact: just "TERMIFLOW v0.3.3.0" centered
		title := StyleAccent.Render("TERMIFLOW")
		if info.Version != "" {
			title += " " + StyleMuted.Render("v"+info.Version)
		}
		bot := StyleMuted.Render(Bar("═", width))
		return fmt.Sprintf("%s\n %s\n%s", top, title, bot)
	}

	// Build right-side stats (3 lines to match logo height)
	var rightLines [3]string

	// Line 1: mode + endpoint + health dot (managed only)
	rightLines[0] = StyleMuted.Render(info.Mode) + StyleMuted.Render(" · ") + StyleAccent.Render(info.Endpoint)
	if info.Mode == "Managed" {
		rightLines[0] += " " + HealthDot(info.APIHealth)
	}

	// Line 2: subs + unread + refresh
	var statParts []string
	statParts = append(statParts, StyleMuted.Render(fmt.Sprintf("%d subs", subCount)))
	if unreadCount > 0 {
		statParts = append(statParts, StyleSuccess.Render("●")+" "+StyleUnreadBadge.Render(fmt.Sprintf("%d unread", unreadCount)))
	}
	if !lastRefresh.IsZero() {
		ago := time.Since(lastRefresh)
		label := "just now"
		if ago > time.Hour {
			label = fmt.Sprintf("%dh ago", int(ago.Hours()))
		} else if ago > time.Minute {
			label = fmt.Sprintf("%dm ago", int(ago.Minutes()))
		}
		dot := StyleSuccess.Render("⦻")
		if ago > time.Hour {
			dot = StyleWarning.Render("⦻")
		}
		statParts = append(statParts, dot+" "+StyleMuted.Render(label))
	}
	rightLines[1] = strings.Join(statParts, "  ")

	// Line 3: version
	if info.Version != "" {
		rightLines[2] = StyleMuted.Render("v" + info.Version)
	}

	// Compose the 3 logo + stats lines
	var contentLines []string
	gap := 4 // space between logo and stats
	for i := 0; i < 3; i++ {
		logo := StyleAccent.Render(asciiLogo[i])
		rightStr := rightLines[i]
		rightWidth := lipgloss.Width(rightStr)

		padding := width - logoWidth - rightWidth - gap - 1 // 1 for leading space
		if padding < 1 {
			padding = 1
		}
		line := " " + logo + strings.Repeat(" ", padding) + rightStr
		contentLines = append(contentLines, line)
	}

	// Breadcrumb line (only if there's content)
	var lines []string
	lines = append(lines, top)
	lines = append(lines, contentLines...)
	if len(breadcrumb) > 0 {
		crumbs := strings.Join(breadcrumb, " › ")
		lines = append(lines, " "+StyleMuted.Render(crumbs))
	}
	bot := StyleMuted.Render(Bar("═", width))
	lines = append(lines, bot)

	return strings.Join(lines, "\n")
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
	remaining := (innerWidth + 4) - topLeftLen - 1      // -1 for closing "┐"
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
		if lineWidth > contentArea && contentArea > 3 {
			// Truncate with ellipsis to fit within card borders
			runes := []rune(line)
			for lipgloss.Width(string(runes)) > contentArea-3 && len(runes) > 0 {
				runes = runes[:len(runes)-1]
			}
			line = string(runes) + "..."
			lineWidth = lipgloss.Width(line)
		}
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

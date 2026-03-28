package ui

import "github.com/charmbracelet/glamour"

// RenderMarkdown renders markdown for terminal display using glamour.
// Falls back to raw text on any error. Width controls word-wrap.
func RenderMarkdown(text string, width int) string {
	if width <= 0 {
		width = 72
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return text
	}
	out, err := r.Render(text)
	if err != nil {
		return text
	}
	return out
}

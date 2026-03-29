package ui

import "github.com/charmbracelet/glamour"

// RenderMarkdown renders markdown for terminal display using glamour.
// Uses the themed style config if a theme has been loaded, otherwise auto-detects.
// Falls back to raw text on any error. Width controls word-wrap.
func RenderMarkdown(text string, width int) string {
	if width <= 0 {
		width = 72
	}

	opts := []glamour.TermRendererOption{
		glamour.WithWordWrap(width),
	}
	if s := GetGlamourStyle(); s != nil {
		opts = append(opts, glamour.WithStyles(*s))
	} else {
		opts = append(opts, glamour.WithAutoStyle())
	}

	r, err := glamour.NewTermRenderer(opts...)
	if err != nil {
		return text
	}
	out, err := r.Render(text)
	if err != nil {
		return text
	}
	return out
}

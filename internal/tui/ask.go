package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/oluoyefeso/termiflow/internal/config"
	"github.com/oluoyefeso/termiflow/internal/providers/llm"
	"github.com/oluoyefeso/termiflow/internal/providers/search"
	"github.com/oluoyefeso/termiflow/internal/tui/components"
)

// askPhase tracks the current state of the ask flow.
type askPhase int

const (
	askPhaseInput     askPhase = iota // waiting for question
	askPhaseSearching                 // fetching sources
	askPhaseStreaming                 // LLM streaming response
	askPhaseDone                      // response complete
)

// AskModel is the inline Q&A screen.
type AskModel struct {
	input    textinput.Model
	phase    askPhase
	question string
	answer   strings.Builder
	sources  []AskSource
	scrollY  int
	width    int
	height   int
	err      error
	cancel   context.CancelFunc
	saved    string // path if saved
}

func NewAskModel() AskModel {
	ti := textinput.New()
	ti.Placeholder = "Ask anything..."
	ti.CharLimit = 500
	ti.Focus()

	return AskModel{
		input: ti,
		phase: askPhaseInput,
	}
}

func (m AskModel) Init() tea.Cmd {
	return textinput.Blink
}

// searchAndStream is a two-phase command: search for sources, then stream LLM response.
// It sends AskSourcesLoadedMsg, then a series of AskChunkMsg.
func searchAndStream(question string, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		cfg := config.Get()

		// Phase 1: Search for sources
		var sources []AskSource
		searchProv, err := search.GetSearchProvider(cfg)
		if err == nil && searchProv.Available() {
			results, err := searchProv.Search(ctx, search.SearchRequest{
				Query:      question,
				MaxResults: 5,
				TimeRange:  "week",
			})
			if err == nil {
				for _, r := range results {
					sources = append(sources, AskSource{
						Title:  r.Title,
						URL:    r.URL,
						Domain: getDomain(r.URL),
					})
				}
			}
		}

		return AskSourcesLoadedMsg{Sources: sources}
	}
}

// streamLLM starts the LLM streaming and returns a Cmd that reads the first chunk.
func streamLLM(question string, sources []AskSource, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		cfg := config.Get()
		provider, err := llm.GetProvider(cfg.General.DefaultProvider, cfg)
		if err != nil {
			return AskChunkMsg{Err: err, Done: true}
		}

		prompt := buildAskPrompt(question, sources)
		chunks, err := provider.Stream(ctx, llm.CompletionRequest{
			Messages: []llm.Message{
				{Role: "system", Content: "You are a helpful assistant that provides accurate, well-researched answers. Use the provided sources to inform your response. Be concise but thorough."},
				{Role: "user", Content: prompt},
			},
			MaxTokens:   2048,
			Temperature: 0.7,
			Stream:      true,
		})
		if err != nil {
			return AskChunkMsg{Err: err, Done: true}
		}

		// Read first chunk and return it
		chunk, ok := <-chunks
		if !ok {
			return AskChunkMsg{Done: true}
		}
		if chunk.Error != nil {
			return AskChunkMsg{Err: chunk.Error, Done: true}
		}
		if chunk.Done {
			return AskChunkMsg{Done: true}
		}
		// Store the channel in a closure for subsequent reads
		return askStreamState{content: chunk.Content, chunks: chunks}
	}
}

// askStreamState carries the channel for subsequent reads.
// This is a tea.Msg that triggers the next read.
type askStreamState struct {
	content string
	chunks  <-chan llm.StreamChunk
}

// readNextChunk reads the next chunk from the channel.
func readNextChunk(chunks <-chan llm.StreamChunk) tea.Cmd {
	return func() tea.Msg {
		chunk, ok := <-chunks
		if !ok {
			return AskChunkMsg{Done: true}
		}
		if chunk.Error != nil {
			return AskChunkMsg{Err: chunk.Error, Done: true}
		}
		if chunk.Done {
			return AskChunkMsg{Done: true}
		}
		return askStreamState{content: chunk.Content, chunks: chunks}
	}
}

func (m AskModel) Update(msg tea.Msg) (AskModel, tea.Cmd) {
	switch msg := msg.(type) {
	case AskSourcesLoadedMsg:
		// Reject stale messages if we've already canceled or moved on
		if m.phase == askPhaseDone || m.phase == askPhaseInput {
			return m, nil
		}
		if msg.Err != nil {
			m.err = msg.Err
			m.phase = askPhaseDone
			return m, nil
		}
		m.sources = msg.Sources
		m.phase = askPhaseStreaming
		// Cancel old context before creating new one
		if m.cancel != nil {
			m.cancel()
		}
		ctx, cancel := context.WithCancel(context.Background())
		m.cancel = cancel
		return m, streamLLM(m.question, m.sources, ctx)

	case askStreamState:
		// Reject stale chunks if we're no longer streaming
		if m.phase != askPhaseStreaming {
			return m, nil
		}
		m.answer.WriteString(msg.content)
		return m, readNextChunk(msg.chunks)

	case AskChunkMsg:
		if msg.Err != nil {
			m.err = msg.Err
		}
		m.phase = askPhaseDone
		return m, nil

	case AskSavedMsg:
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.saved = msg.Path
		}
		return m, nil

	case tea.KeyMsg:
		switch m.phase {
		case askPhaseInput:
			return m.updateInput(msg)
		case askPhaseSearching, askPhaseStreaming:
			// Ctrl+C cancels the in-flight request
			if msg.String() == "ctrl+c" {
				if m.cancel != nil {
					m.cancel()
				}
				m.phase = askPhaseDone
				return m, nil
			}
		case askPhaseDone:
			return m.updateDone(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m *AskModel) updateInput(msg tea.KeyMsg) (AskModel, tea.Cmd) {
	switch {
	case key.Matches(msg, AskKeys.Submit):
		q := strings.TrimSpace(m.input.Value())
		if q == "" {
			return *m, nil
		}
		m.question = q
		m.phase = askPhaseSearching
		m.answer.Reset()
		m.sources = nil
		m.scrollY = 0
		m.err = nil
		m.saved = ""
		m.input.Blur()
		// Cancel any previous context before creating a new one
		if m.cancel != nil {
			m.cancel()
		}
		ctx, cancel := context.WithCancel(context.Background())
		m.cancel = cancel
		return *m, searchAndStream(q, ctx)
	case key.Matches(msg, AskKeys.Back):
		return *m, func() tea.Msg {
			return SwitchScreenMsg{Screen: ScreenDashboard}
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return *m, cmd
}

func (m *AskModel) updateDone(msg tea.KeyMsg) (AskModel, tea.Cmd) {
	switch {
	case key.Matches(msg, AskKeys.Back):
		// Cancel any lingering context and reset for a new question
		if m.cancel != nil {
			m.cancel()
		}
		m.phase = askPhaseInput
		m.input.SetValue("")
		m.input.Focus()
		m.answer.Reset()
		m.scrollY = 0
		m.err = nil
		m.saved = ""
		return *m, textinput.Blink
	case key.Matches(msg, AskKeys.Save):
		if m.saved == "" && m.answer.Len() > 0 {
			return *m, saveAskResult(m.question, m.answer.String(), m.sources)
		}
	case key.Matches(msg, AskKeys.Up):
		if m.scrollY > 0 {
			m.scrollY--
		}
	case key.Matches(msg, AskKeys.Down):
		// Rough upper bound based on answer length to prevent runaway scroll
		maxScroll := strings.Count(m.answer.String(), "\n") + len(m.sources) + 5
		if m.scrollY < maxScroll {
			m.scrollY++
		}
	}
	return *m, nil
}

func (m AskModel) View() string {
	w := m.width
	if w == 0 {
		w = 69
	}

	var b strings.Builder

	// Header
	top := StyleMuted.Render(Bar("═", w))
	title := StyleAccent.Render("ASK")
	bot := StyleMuted.Render(Bar("═", w))
	b.WriteString(fmt.Sprintf("%s\n  %s\n%s\n", top, title, bot))

	switch m.phase {
	case askPhaseInput:
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s %s\n", StyleTitle.Render(">"), m.input.View()))
		b.WriteString("\n")
		b.WriteString(StyleMuted.Render("  Type a question and press Enter. Esc to go back."))
		b.WriteString("\n")

	case askPhaseSearching:
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s %s\n", StyleTitle.Render("Q:"), m.question))
		b.WriteString(fmt.Sprintf("\n  %s\n", StyleCyan.Render("⟳ Searching for sources...")))

	case askPhaseStreaming, askPhaseDone:
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s %s\n", StyleTitle.Render("Q:"), m.question))
		b.WriteString("\n")

		// Build content lines
		var lines []string

		// Answer
		answerText := m.answer.String()
		if answerText != "" {
			wrapped := wrapText(answerText, w-4)
			for _, line := range strings.Split(wrapped, "\n") {
				lines = append(lines, "  "+line)
			}
		}

		if m.phase == askPhaseStreaming {
			lines = append(lines, StyleCyan.Render("  ▍"))
		}

		// Error
		if m.err != nil {
			lines = append(lines, "")
			lines = append(lines, StyleError.Render(fmt.Sprintf("  Error: %v", m.err)))
		}

		// Sources
		if len(m.sources) > 0 && m.phase == askPhaseDone {
			lines = append(lines, "")
			lines = append(lines, StyleMuted.Render("  "+Bar("─", w-4)))
			lines = append(lines, StyleTitle.Render("  Sources:"))
			for i, src := range m.sources {
				lines = append(lines, fmt.Sprintf("   [%d] %s %s",
					i+1,
					StyleMuted.Render(src.Domain),
					src.Title,
				))
			}
		}

		// Saved status
		if m.saved != "" {
			lines = append(lines, "")
			lines = append(lines, StyleSuccess.Render("  ✓ Saved to "+m.saved))
		}

		// Apply scroll
		viewportHeight := m.height - 10
		if viewportHeight < 5 {
			viewportHeight = 20
		}
		totalLines := len(lines)
		maxScroll := totalLines - viewportHeight
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.scrollY > maxScroll {
			m.scrollY = maxScroll
		}
		endIdx := m.scrollY + viewportHeight
		if endIdx > totalLines {
			endIdx = totalLines
		}
		startIdx := m.scrollY
		if startIdx < 0 {
			startIdx = 0
		}
		if startIdx < totalLines {
			visible := lines[startIdx:endIdx]
			b.WriteString(strings.Join(visible, "\n"))
			b.WriteString("\n")
		}
	}

	// Status bar
	var hints []components.KeyHint
	switch m.phase {
	case askPhaseInput:
		hints = []components.KeyHint{
			{Key: "enter", Desc: "ask"},
			{Key: "esc", Desc: "back"},
		}
	case askPhaseSearching, askPhaseStreaming:
		hints = []components.KeyHint{
			{Key: "ctrl+c", Desc: "cancel"},
		}
	case askPhaseDone:
		hints = []components.KeyHint{
			{Key: "s", Desc: "save"},
			{Key: "j/k", Desc: "scroll"},
			{Key: "esc", Desc: "new question"},
		}
	}
	b.WriteString("\n" + components.NewStatusBar(hints, w).View())

	return b.String()
}

// buildAskPrompt constructs the prompt with source context.
func buildAskPrompt(question string, sources []AskSource) string {
	var sb strings.Builder

	if len(sources) > 0 {
		sb.WriteString("Use the following sources to inform your answer:\n\n")
		for i, src := range sources {
			sb.WriteString(fmt.Sprintf("Source %d: %s\nURL: %s\n\n", i+1, src.Title, src.URL))
		}
		sb.WriteString("---\n\n")
	}

	sb.WriteString("Question: ")
	sb.WriteString(question)
	return sb.String()
}

// getDomain extracts the domain from a URL.
func getDomain(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "www.")
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return url
}

// saveAskResult writes the Q&A to a markdown file.
func saveAskResult(question, answer string, sources []AskSource) tea.Cmd {
	return func() tea.Msg {
		saveDir := getAskSaveDir()
		if err := os.MkdirAll(saveDir, 0755); err != nil {
			return AskSavedMsg{Err: err}
		}

		ts := time.Now()
		slug := askSlugify(question)
		filename := fmt.Sprintf("%s-%s.md", ts.Format("20060102-150405"), slug)
		path := filepath.Join(saveDir, filename)

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("## %s\n\n", question))
		sb.WriteString(fmt.Sprintf("*%s*\n\n", ts.Format("2006-01-02 15:04")))
		sb.WriteString(answer)
		sb.WriteString("\n")

		if len(sources) > 0 {
			sb.WriteString("\n### Sources\n\n")
			for i, src := range sources {
				sb.WriteString(fmt.Sprintf("%d. [%s](%s)\n", i+1, src.Title, src.URL))
			}
		}

		if err := os.WriteFile(path, []byte(sb.String()), 0600); err != nil {
			return AskSavedMsg{Err: err}
		}
		return AskSavedMsg{Path: path}
	}
}

func getAskSaveDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "termiflow", "saved")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".local", "share", "termiflow", "saved")
}

var askNonAlpha = regexp.MustCompile(`[^a-z0-9]+`)

func askSlugify(s string) string {
	s = strings.ToLower(s)
	s = askNonAlpha.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	runes := []rune(s)
	if len(runes) > 50 {
		s = string(runes[:50])
		s = strings.TrimRight(s, "-")
	}
	return s
}

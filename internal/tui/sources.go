package tui

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/oluoyefeso/termiflow/internal/db"
	"github.com/oluoyefeso/termiflow/internal/sources"
	tfSync "github.com/oluoyefeso/termiflow/internal/sync"
	"github.com/oluoyefeso/termiflow/internal/tui/components"
	"github.com/oluoyefeso/termiflow/pkg/models"
)

// addPhase tracks the multi-step source add flow.
type addPhase int

const (
	addPhaseURL         addPhase = iota // text input for URL
	addPhaseDiscovering                 // spinner while sources.Discover runs
	addPhaseFrequency                   // frequency picker
	addPhaseContext                     // optional context text input
)

// SourcesModel is the sources management screen.
type SourcesModel struct {
	sources   []SourceInfo
	cursor    int
	width     int
	height    int
	loading   bool
	err       error
	statusMsg string

	// Add mode
	adding         bool
	addPhase       addPhase
	urlInput       textinput.Model
	ctxInput       textinput.Model
	discovered     *sources.FeedInfo
	discoverURL    string
	discoverErr    error
	discoverCancel context.CancelFunc

	// Frequency picker (reused for add + edit)
	freqPicking bool
	freqCursor  int
	freqAction  string // "add" or "edit"
	freqSource  string // display name of source being acted on

	// Context editor
	editingCtx bool
	editSource int // index in sources slice

	// Delete confirmation
	confirming bool
}

func NewSourcesModel() SourcesModel {
	urlTi := textinput.New()
	urlTi.Placeholder = "https://example.com or feed URL..."
	urlTi.CharLimit = 500

	ctxTi := textinput.New()
	ctxTi.Placeholder = "e.g., AI and machine learning (optional, enter to skip)"
	ctxTi.CharLimit = 200

	return SourcesModel{
		loading:  true,
		urlInput: urlTi,
		ctxInput: ctxTi,
	}
}

// --- Tea commands ---

func loadSources() tea.Cmd {
	return func() tea.Msg {
		subs, err := db.GetActiveSubscriptions()
		if err != nil {
			return SourcesLoadedMsg{Err: err}
		}

		var infos []SourceInfo
		for _, sub := range subs {
			if !sub.IsSourceSubscription() {
				continue
			}
			total, unread, err := db.GetSubscriptionItemCount(sub.ID)
			if err != nil {
				total, unread = 0, 0
			}
			infos = append(infos, SourceInfo{
				Sub:    sub,
				Total:  total,
				Unread: unread,
				Domain: sourceDomainFromURL(sub.SourceURL),
			})
		}

		return SourcesLoadedMsg{Sources: infos}
	}
}

func discoverSource(ctx context.Context, rawURL string) tea.Cmd {
	return func() tea.Msg {
		info, err := sources.Discover(ctx, rawURL)
		return SourceDiscoveredMsg{Info: info, RawURL: rawURL, Err: err}
	}
}

func addSource(feedURL, displayName, sourceType, frequency, ctx string) tea.Cmd {
	return func() tea.Msg {
		sub := &models.Subscription{
			Topic:       displayName,
			Frequency:   frequency,
			Sources:     []string{sourceType},
			IsActive:    true,
			SourceURL:   feedURL,
			SourceType:  sourceType,
			DisplayName: displayName,
			Context:     ctx,
		}

		if err := db.CreateSubscription(sub); err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint") {
				return SourceAddedMsg{Name: displayName, Err: fmt.Errorf("already subscribed to %s", displayName)}
			}
			return SourceAddedMsg{Err: err}
		}

		tfSync.PushSubscription(context.Background(), sub)
		return SourceAddedMsg{Name: displayName}
	}
}

func removeSourceCmd(sub *models.Subscription) tea.Cmd {
	return func() tea.Msg {
		if err := db.DeleteSubscriptionBySourceURL(sub.SourceURL); err != nil {
			return SourceRemovedMsg{Name: sub.DisplayName, Err: err}
		}
		tfSync.DeleteSubscription(context.Background(), sub.Topic)
		return SourceRemovedMsg{Name: sub.DisplayName}
	}
}

func updateSourceFrequency(sub *models.Subscription, freq string) tea.Cmd {
	// Copy to avoid mutating shared pointer from UI goroutine
	clone := *sub
	clone.Frequency = freq
	return func() tea.Msg {
		err := db.UpdateSubscription(&clone)
		return SourceUpdatedMsg{Name: clone.DisplayName, Err: err}
	}
}

func updateSourceContext(sub *models.Subscription, ctx string) tea.Cmd {
	// Copy to avoid mutating shared pointer from UI goroutine
	clone := *sub
	clone.Context = ctx
	return func() tea.Msg {
		err := db.UpdateSubscription(&clone)
		return SourceUpdatedMsg{Name: clone.DisplayName, Err: err}
	}
}

// --- Model methods ---

func (m SourcesModel) Init() tea.Cmd {
	return loadSources()
}

func (m SourcesModel) Update(msg tea.Msg) (SourcesModel, tea.Cmd) {
	// Forward non-key messages to active textinput (for blink cursor)
	if _, isKey := msg.(tea.KeyMsg); !isKey {
		if m.adding && (m.addPhase == addPhaseURL || m.addPhase == addPhaseContext) {
			var cmd tea.Cmd
			if m.addPhase == addPhaseURL {
				m.urlInput, cmd = m.urlInput.Update(msg)
			} else {
				m.ctxInput, cmd = m.ctxInput.Update(msg)
			}
			// Still let SourceDiscoveredMsg fall through
			if _, isDiscover := msg.(SourceDiscoveredMsg); !isDiscover {
				return m, cmd
			}
		}
		if m.editingCtx {
			var cmd tea.Cmd
			m.ctxInput, cmd = m.ctxInput.Update(msg)
			return m, cmd
		}
	}

	switch msg := msg.(type) {
	case SourcesLoadedMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.sources = msg.Sources
		// Clamp cursor after reload
		if m.cursor >= len(m.sources) {
			m.cursor = len(m.sources) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		return m, nil

	case SourceDiscoveredMsg:
		// Guard against stale messages after cancel
		if !m.adding || m.addPhase != addPhaseDiscovering {
			return m, nil
		}
		if m.discoverCancel != nil {
			m.discoverCancel()
			m.discoverCancel = nil
		}
		if msg.Err != nil {
			if msg.Info != nil && msg.Info.ScrapeOnly {
				// Scrape-only fallback
				m.discovered = msg.Info
				m.discovered.Title = sourceDomainFromURL(msg.RawURL)
				m.discoverURL = msg.RawURL
				m.addPhase = addPhaseFrequency
				m.freqCursor = 1 // default daily
				m.statusMsg = "No RSS feed found. Will use web scraping."
				return m, nil
			}
			m.discoverErr = msg.Err
			m.adding = false
			m.statusMsg = fmt.Sprintf("Discovery failed: %v", msg.Err)
			return m, nil
		}
		m.discovered = msg.Info
		m.discoverURL = msg.RawURL
		m.addPhase = addPhaseFrequency
		m.freqCursor = 1 // default daily
		return m, nil

	case SourceAddedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Failed: %v", msg.Err)
		} else {
			m.statusMsg = fmt.Sprintf("Subscribed to %s", msg.Name)
		}
		m.adding = false
		return m, loadSources()

	case SourceRemovedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Failed to remove: %v", msg.Err)
		} else {
			m.statusMsg = fmt.Sprintf("Removed %s", msg.Name)
		}
		m.confirming = false
		return m, loadSources()

	case SourceUpdatedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Failed to update: %v", msg.Err)
		} else {
			m.statusMsg = fmt.Sprintf("Updated %s", msg.Name)
		}
		m.freqPicking = false
		m.editingCtx = false
		return m, loadSources()

	case tea.KeyMsg:
		if m.adding {
			return m.updateAdding(msg)
		}
		if m.freqPicking {
			return m.updateFreqPicker(msg)
		}
		if m.editingCtx {
			return m.updateCtxEditor(msg)
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

func (m *SourcesModel) updateNormal(msg tea.KeyMsg) (SourcesModel, tea.Cmd) {
	switch {
	case key.Matches(msg, SourcesKeys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, SourcesKeys.Down):
		mx := len(m.sources) - 1
		if mx < 0 {
			mx = 0
		}
		if m.cursor < mx {
			m.cursor++
		}
	case key.Matches(msg, SourcesKeys.Add):
		m.adding = true
		m.addPhase = addPhaseURL
		m.urlInput.Reset()
		m.urlInput.Focus()
		m.discovered = nil
		m.discoverErr = nil
		m.statusMsg = ""
		return *m, textinput.Blink
	case key.Matches(msg, SourcesKeys.Delete):
		if m.cursor < len(m.sources) {
			m.confirming = true
		}
	case key.Matches(msg, SourcesKeys.EditFreq):
		if m.cursor < len(m.sources) {
			sub := m.sources[m.cursor]
			m.freqPicking = true
			m.freqAction = "edit"
			m.freqSource = sub.Sub.DisplayName
			m.freqCursor = 1
			for i, f := range frequencies {
				if f == sub.Sub.Frequency {
					m.freqCursor = i
					break
				}
			}
		}
	case key.Matches(msg, SourcesKeys.EditContext):
		if m.cursor < len(m.sources) {
			sub := m.sources[m.cursor]
			m.editingCtx = true
			m.editSource = m.cursor
			m.ctxInput.Reset()
			m.ctxInput.SetValue(sub.Sub.Context)
			m.ctxInput.Focus()
			return *m, textinput.Blink
		}
	case key.Matches(msg, SourcesKeys.Back):
		return *m, func() tea.Msg {
			return SwitchScreenMsg{Screen: ScreenDashboard}
		}
	}
	return *m, nil
}

func (m *SourcesModel) updateAdding(msg tea.KeyMsg) (SourcesModel, tea.Cmd) {
	switch m.addPhase {
	case addPhaseURL:
		switch msg.String() {
		case "enter":
			rawURL := strings.TrimSpace(m.urlInput.Value())
			if rawURL == "" {
				return *m, nil
			}
			// Check for duplicate
			existing, _ := db.GetSubscriptionBySourceURL(rawURL)
			if existing != nil {
				m.adding = false
				m.statusMsg = fmt.Sprintf("Already subscribed to %s", existing.DisplayName)
				return *m, nil
			}
			// Start discovery
			m.addPhase = addPhaseDiscovering
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			m.discoverCancel = cancel
			return *m, discoverSource(ctx, rawURL)
		case "esc":
			m.adding = false
		default:
			var cmd tea.Cmd
			m.urlInput, cmd = m.urlInput.Update(msg)
			return *m, cmd
		}

	case addPhaseDiscovering:
		if msg.String() == "esc" {
			if m.discoverCancel != nil {
				m.discoverCancel()
				m.discoverCancel = nil
			}
			m.adding = false
		}

	case addPhaseFrequency:
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
			m.addPhase = addPhaseContext
			m.ctxInput.Reset()
			m.ctxInput.Focus()
			return *m, textinput.Blink
		case "esc":
			m.adding = false
		}

	case addPhaseContext:
		switch msg.String() {
		case "enter":
			freq := frequencies[m.freqCursor]
			ctxVal := strings.TrimSpace(m.ctxInput.Value())
			feedURL := ""
			displayName := ""
			sourceType := "feed"

			if m.discovered != nil {
				if m.discovered.ScrapeOnly {
					feedURL = m.discoverURL
					displayName = m.discovered.Title
					sourceType = "scrape"
				} else {
					feedURL = m.discovered.FeedURL
					displayName = m.discovered.Title
				}
			}
			// Fallback: use domain from original URL input
			if feedURL == "" {
				feedURL = strings.TrimSpace(m.urlInput.Value())
			}
			if displayName == "" {
				displayName = sourceDomainFromURL(strings.TrimSpace(m.urlInput.Value()))
			}

			// Check for duplicate by resolved feed URL
			existing, _ := db.GetSubscriptionBySourceURL(feedURL)
			if existing != nil {
				m.adding = false
				m.statusMsg = fmt.Sprintf("Already subscribed to this feed as %q", existing.DisplayName)
				return *m, nil
			}

			m.adding = false
			return *m, addSource(feedURL, displayName, sourceType, freq, ctxVal)
		case "esc":
			m.adding = false
		default:
			var cmd tea.Cmd
			m.ctxInput, cmd = m.ctxInput.Update(msg)
			return *m, cmd
		}
	}

	return *m, nil
}

func (m *SourcesModel) updateFreqPicker(msg tea.KeyMsg) (SourcesModel, tea.Cmd) {
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
		if m.freqAction == "edit" && m.cursor < len(m.sources) {
			freq := frequencies[m.freqCursor]
			sub := m.sources[m.cursor].Sub
			return *m, updateSourceFrequency(sub, freq)
		}
	case "esc":
		m.freqPicking = false
	}
	return *m, nil
}

func (m *SourcesModel) updateCtxEditor(msg tea.KeyMsg) (SourcesModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.editSource < len(m.sources) {
			ctxVal := strings.TrimSpace(m.ctxInput.Value())
			sub := m.sources[m.editSource].Sub
			return *m, updateSourceContext(sub, ctxVal)
		}
	case "esc":
		m.editingCtx = false
	default:
		var cmd tea.Cmd
		m.ctxInput, cmd = m.ctxInput.Update(msg)
		return *m, cmd
	}
	return *m, nil
}

func (m *SourcesModel) updateConfirm(msg tea.KeyMsg) (SourcesModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.cursor < len(m.sources) {
			sub := m.sources[m.cursor].Sub
			m.confirming = false
			return *m, removeSourceCmd(sub)
		}
	case "n", "N", "esc":
		m.confirming = false
	}
	return *m, nil
}

// Breadcrumb returns breadcrumb segments for the sources screen.
func (m SourcesModel) Breadcrumb() []string {
	return []string{"SOURCES"}
}

// StatusHints returns keybinding hints for the sources screen.
func (m SourcesModel) StatusHints() []components.KeyHint {
	if m.adding || m.freqPicking || m.editingCtx || m.confirming {
		return []components.KeyHint{
			{Key: "enter", Desc: "confirm"},
		}
	}
	if len(m.sources) == 0 {
		return []components.KeyHint{
			{Key: "a", Desc: "add source"},
		}
	}
	return []components.KeyHint{
		{Key: "a", Desc: "add"},
		{Key: "d", Desc: "delete"},
		{Key: "e", Desc: "frequency"},
		{Key: "c", Desc: "context"},
	}
}

// ContentView renders the sources management content without header/footer chrome.
func (m SourcesModel) ContentView() string {
	var b strings.Builder

	if m.loading {
		b.WriteString(StyleMuted.Render("\n  Loading..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(StyleError.Render(fmt.Sprintf("\n  Error: %v", m.err)))
		return b.String()
	}

	// Add flow overlays
	if m.adding {
		b.WriteString(m.renderAddFlow())
		return b.String()
	}

	// Frequency picker overlay
	if m.freqPicking {
		b.WriteString(m.renderFreqPicker())
		return b.String()
	}

	// Context editor overlay
	if m.editingCtx && m.editSource < len(m.sources) {
		name := m.sources[m.editSource].Sub.DisplayName
		fmt.Fprintf(&b, "\n  Edit context for %s\n\n", StyleAccent.Render(name))
		fmt.Fprintf(&b, "  %s\n\n", m.ctxInput.View())
		fmt.Fprintf(&b, "  %s to save, %s to cancel\n",
			StyleTitle.Render("Enter"),
			StyleMuted.Render("Esc"),
		)
		return b.String()
	}

	// Delete confirmation
	if m.confirming && m.cursor < len(m.sources) {
		name := m.sources[m.cursor].Sub.DisplayName
		fmt.Fprintf(&b, "\n  Remove source %s?\n\n", StyleAccent.Render(name))
		fmt.Fprintf(&b, "  %s / %s\n",
			StyleTitle.Render("[y] yes"),
			StyleMuted.Render("[n] no"),
		)
		return b.String()
	}

	// Section header
	b.WriteString("\n")
	fmt.Fprintf(&b, "  %s\n\n", StyleAccent.Render("▸ SOURCES"))

	// Zero state
	if len(m.sources) == 0 {
		b.WriteString(StyleMuted.Render("  No custom sources yet. Press [a] to add an RSS feed or blog.\n"))
		if m.statusMsg != "" {
			fmt.Fprintf(&b, "\n  %s\n", StyleSuccess.Render("✓ "+m.statusMsg))
		}
		return b.String()
	}

	// Calculate column widths
	nameWidth := 20
	for _, info := range m.sources {
		name := sourceDisplayName(info)
		w := lipgloss.Width(name)
		if w > nameWidth {
			nameWidth = w
		}
	}
	nameWidth += 2

	// Source list
	for i, info := range m.sources {
		cursor := "  "
		nameStyle := lipgloss.NewStyle()
		if i == m.cursor {
			cursor = StyleSelectedIndicator.Render("▸ ")
			nameStyle = StyleSelected
		}

		name := sourceDisplayName(info)
		nameCol := PadRight(nameStyle.Render(name), nameWidth)

		typeTag := StyleMuted.Render(fmt.Sprintf("[%s]", info.Sub.SourceType))
		typeTag = PadRight(typeTag, 8)

		domain := PadRight(StyleWarmMuted.Render(info.Domain), 22)
		freq := PadRight(StyleMuted.Render(info.Sub.Frequency), 8)
		items := PadLeft(fmt.Sprintf("%d items", info.Total), 9)
		unread := ""
		if info.Unread > 0 {
			unread = "  " + StyleUnreadBadge.Render(fmt.Sprintf("%d unread", info.Unread))
		}

		fmt.Fprintf(&b, "  %s%s %s %s %s %s%s\n",
			cursor, nameCol, typeTag, domain, freq, items, unread)

		// Show context if set
		if info.Sub.Context != "" {
			indent := "      "
			fmt.Fprintf(&b, "%s%s\n", indent, StyleSummary.Render("context: "+info.Sub.Context))
		}
	}

	// Status message
	if m.statusMsg != "" {
		fmt.Fprintf(&b, "\n  %s\n", StyleSuccess.Render("✓ "+m.statusMsg))
	}

	return b.String()
}

func (m SourcesModel) renderAddFlow() string {
	var b strings.Builder

	switch m.addPhase {
	case addPhaseURL:
		fmt.Fprintf(&b, "\n  %s\n\n", StyleAccent.Render("Add Source"))
		b.WriteString("  Enter a blog or feed URL:\n\n")
		fmt.Fprintf(&b, "  %s\n\n", m.urlInput.View())
		fmt.Fprintf(&b, "  %s to discover, %s to cancel\n",
			StyleTitle.Render("Enter"),
			StyleMuted.Render("Esc"),
		)

	case addPhaseDiscovering:
		fmt.Fprintf(&b, "\n  %s\n\n", StyleAccent.Render("Add Source"))
		rawURL := strings.TrimSpace(m.urlInput.Value())
		fmt.Fprintf(&b, "  Discovering feed for %s...\n\n", StyleWarmMuted.Render(rawURL))
		fmt.Fprintf(&b, "  %s to cancel\n", StyleMuted.Render("Esc"))

	case addPhaseFrequency:
		title := "Choose frequency"
		if m.discovered != nil {
			name := m.discovered.Title
			if name == "" {
				name = sourceDomainFromURL(m.discoverURL)
			}
			sourceType := "RSS"
			if m.discovered.ScrapeOnly {
				sourceType = "scrape"
			}
			fmt.Fprintf(&b, "\n  Found: %s %s\n\n", StyleAccent.Render(name), StyleMuted.Render("["+sourceType+"]"))
		}
		fmt.Fprintf(&b, "  %s:\n\n", title)
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

	case addPhaseContext:
		fmt.Fprintf(&b, "\n  %s\n\n", StyleAccent.Render("Relevance Context"))
		b.WriteString("  What topics are you interested in from this source?\n\n")
		fmt.Fprintf(&b, "  %s\n\n", m.ctxInput.View())
		fmt.Fprintf(&b, "  %s to confirm (empty to skip), %s to cancel\n",
			StyleTitle.Render("Enter"),
			StyleMuted.Render("Esc"),
		)
	}

	// Show warning from scrape-only detection
	if m.statusMsg != "" && m.addPhase == addPhaseFrequency {
		fmt.Fprintf(&b, "\n  %s\n", StyleWarning.Render("⚠ "+m.statusMsg))
	}

	return b.String()
}

func (m SourcesModel) renderFreqPicker() string {
	var b strings.Builder
	name := m.freqSource
	fmt.Fprintf(&b, "\n  Set frequency for %s\n\n", StyleAccent.Render(name))
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

// sourceDisplayName returns the best available name for a source.
func sourceDisplayName(info SourceInfo) string {
	if info.Sub.DisplayName != "" {
		return info.Sub.DisplayName
	}
	if info.Domain != "" {
		return info.Domain
	}
	if info.Sub.SourceURL != "" {
		return info.Sub.SourceURL
	}
	if info.Sub.Topic != "" {
		return info.Sub.Topic
	}
	return "(unnamed source)"
}

// sourceDomainFromURL extracts the domain from a URL, stripping www. prefix.
func sourceDomainFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	host := u.Hostname()
	host = strings.TrimPrefix(host, "www.")
	return host
}

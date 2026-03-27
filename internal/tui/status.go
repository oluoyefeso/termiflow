package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/oluoyefeso/termiflow/internal/config"
	"github.com/oluoyefeso/termiflow/internal/db"
	"github.com/oluoyefeso/termiflow/internal/tui/components"
)

// statusData holds all info for the status screen.
type statusData struct {
	Mode          string
	Provider      string
	APIKey        string
	BaseURL       string
	Subscriptions int
	TotalItems    int
	TotalUnread   int
	DBPath        string
	DBSize        string
	ConfigPath    string
}

// StatusModel is the read-only status/config info screen.
type StatusModel struct {
	data    statusData
	width   int
	height  int
	loading bool
	err     error
}

func NewStatusModel() StatusModel {
	return StatusModel{loading: true}
}

// statusLoadedMsg carries status data.
type statusLoadedMsg struct {
	data statusData
	err  error
}

func loadStatus() tea.Cmd {
	return func() tea.Msg {
		cfg := config.Get()
		data := statusData{}

		// Mode
		if config.IsManagedMode() {
			data.Mode = "Managed"
			data.APIKey = maskKey(cfg.Providers.Managed.APIKey)
			data.BaseURL = cfg.Providers.Managed.BaseURL
		} else {
			data.Mode = "Self-hosted"
			data.Provider = cfg.General.DefaultProvider
		}

		// Subscriptions
		subs, err := db.GetActiveSubscriptions()
		if err == nil {
			data.Subscriptions = len(subs)
			for _, sub := range subs {
				total, unread, err := db.GetSubscriptionItemCount(sub.ID)
				if err == nil {
					data.TotalItems += total
					data.TotalUnread += unread
				}
			}
		}

		// DB info
		dataDir := config.GetDataDir()
		dbPath := filepath.Join(dataDir, "termiflow.db")
		data.DBPath = dbPath
		if info, err := os.Stat(dbPath); err == nil {
			size := info.Size()
			if size > 1024*1024 {
				data.DBSize = fmt.Sprintf("%.1f MB", float64(size)/1024/1024)
			} else {
				data.DBSize = fmt.Sprintf("%.1f KB", float64(size)/1024)
			}
		}

		// Config path
		data.ConfigPath = config.GetConfigPath()

		return statusLoadedMsg{data: data}
	}
}

func maskKey(key string) string {
	if len(key) < 20 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func (m StatusModel) Init() tea.Cmd {
	return loadStatus()
}

func (m StatusModel) Update(msg tea.Msg) (StatusModel, tea.Cmd) {
	switch msg := msg.(type) {
	case statusLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.data = msg.data
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "esc" {
			return m, func() tea.Msg {
				return SwitchScreenMsg{Screen: ScreenDashboard}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m StatusModel) View() string {
	w := m.width
	if w == 0 {
		w = 69
	}

	var b strings.Builder

	// Header
	top := StyleMuted.Render(Bar("═", w))
	title := StyleAccent.Render("STATUS")
	bot := StyleMuted.Render(Bar("═", w))
	b.WriteString(fmt.Sprintf("%s\n  %s\n%s\n", top, title, bot))

	if m.loading {
		b.WriteString(StyleMuted.Render("\n  Loading..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(StyleError.Render(fmt.Sprintf("\n  Error: %v", m.err)))
		return b.String()
	}

	b.WriteString("\n")

	// Mode
	b.WriteString(infoRow("Mode", m.data.Mode))
	if m.data.Mode == "Managed" {
		b.WriteString(infoRow("API Key", m.data.APIKey))
		b.WriteString(infoRow("Base URL", m.data.BaseURL))
	} else {
		b.WriteString(infoRow("Provider", m.data.Provider))
	}

	b.WriteString("\n")

	// Subscriptions
	b.WriteString(infoRow("Subscriptions", fmt.Sprintf("%d active", m.data.Subscriptions)))
	b.WriteString(infoRow("Total Items", fmt.Sprintf("%d (%d unread)", m.data.TotalItems, m.data.TotalUnread)))

	b.WriteString("\n")

	// Database
	b.WriteString(infoRow("Database", m.data.DBPath))
	if m.data.DBSize != "" {
		b.WriteString(infoRow("DB Size", m.data.DBSize))
	}
	b.WriteString(infoRow("Config", m.data.ConfigPath))

	// Status bar
	b.WriteString("\n")
	hints := []components.KeyHint{
		{Key: "esc", Desc: "back"},
	}
	b.WriteString(components.NewStatusBar(hints, w).View())

	return b.String()
}

func infoRow(label, value string) string {
	padding := 16 - len(label)
	if padding < 1 {
		padding = 1
	}
	return fmt.Sprintf("   %s%s%s\n",
		StyleMuted.Render(label+":"),
		strings.Repeat(" ", padding),
		value,
	)
}

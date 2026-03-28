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

		if config.IsManagedMode() {
			data.Mode = "Managed"
			data.APIKey = maskKey(cfg.Providers.Managed.APIKey)
			data.BaseURL = cfg.Providers.Managed.BaseURL
		} else {
			data.Mode = "Self-hosted"
			data.Provider = cfg.General.DefaultProvider
		}

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

// Breadcrumb returns breadcrumb segments for the status screen.
func (m StatusModel) Breadcrumb() []string {
	return []string{"STATUS"}
}

// StatusHints returns keybinding hints for the status screen.
func (m StatusModel) StatusHints() []components.KeyHint {
	return nil // only esc/back, added by AppModel
}

// ContentView renders the status screen as bordered info cards.
func (m StatusModel) ContentView() string {
	w := m.width
	if w == 0 {
		w = 69
	}

	var b strings.Builder

	if m.loading {
		b.WriteString(StyleMuted.Render("\n  Loading..."))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(StyleError.Render(fmt.Sprintf("\n  Error: %v", m.err)))
		return b.String()
	}

	cardWidth := w - 2
	if cardWidth > 60 {
		cardWidth = 60
	}

	b.WriteString("\n")

	// CONNECTION card
	var connLines []string
	connLines = append(connLines, infoCardRow("Mode", m.data.Mode))
	if m.data.Mode == "Managed" {
		connLines = append(connLines, infoCardRow("API Key", m.data.APIKey))
		connLines = append(connLines, infoCardRow("Base URL", m.data.BaseURL))
	} else {
		connLines = append(connLines, infoCardRow("Provider", m.data.Provider))
	}
	b.WriteString("  " + RenderCard("CONNECTION", connLines, cardWidth))
	b.WriteString("\n\n")

	// DATA card
	var dataLines []string
	dataLines = append(dataLines, infoCardRow("Subscriptions", fmt.Sprintf("%d active", m.data.Subscriptions)))
	dataLines = append(dataLines, infoCardRow("Total Items", fmt.Sprintf("%d (%d unread)", m.data.TotalItems, m.data.TotalUnread)))
	b.WriteString("  " + RenderCard("DATA", dataLines, cardWidth))
	b.WriteString("\n\n")

	// SYSTEM card
	var sysLines []string
	sysLines = append(sysLines, infoCardRow("Database", m.data.DBPath))
	if m.data.DBSize != "" {
		sysLines = append(sysLines, infoCardRow("DB Size", m.data.DBSize))
	}
	sysLines = append(sysLines, infoCardRow("Config", m.data.ConfigPath))
	b.WriteString("  " + RenderCard("SYSTEM", sysLines, cardWidth))
	b.WriteString("\n")

	return b.String()
}

func infoCardRow(label, value string) string {
	padding := 16 - len(label)
	if padding < 1 {
		padding = 1
	}
	return fmt.Sprintf("%s%s%s",
		StyleMuted.Render(label+":"),
		strings.Repeat(" ", padding),
		value,
	)
}

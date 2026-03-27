package tui

import "github.com/charmbracelet/bubbles/key"

// GlobalKeyMap defines keys available on every screen.
type GlobalKeyMap struct {
	Quit key.Binding
	Help key.Binding
}

var GlobalKeys = GlobalKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}

// DashboardKeyMap defines keys for the dashboard screen.
type DashboardKeyMap struct {
	Enter   key.Binding
	Ask     key.Binding
	Topics  key.Binding
	Refresh key.Binding
	Status  key.Binding
	Up      key.Binding
	Down    key.Binding
}

var DashboardKeys = DashboardKeyMap{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "open feed"),
	),
	Ask: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "ask"),
	),
	Topics: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "topics"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Status: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "status"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("k/up", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("j/down", "down"),
	),
}

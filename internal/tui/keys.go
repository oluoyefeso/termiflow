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
	Sources key.Binding
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
	Sources: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "sources"),
	),
	Status: key.NewBinding(
		key.WithKeys("i", "I"),
		key.WithHelp("i", "info"),
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

// FeedKeyMap defines keys for the feed list screen.
type FeedKeyMap struct {
	Enter  key.Binding
	Back   key.Binding
	Filter key.Binding
	Unread key.Binding
	Up     key.Binding
	Down   key.Binding
}

var FeedKeys = FeedKeyMap{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "open article"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Filter: key.NewBinding(
		key.WithKeys("f", "/"),
		key.WithHelp("/", "filter"),
	),
	Unread: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "unread only"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("j", "down"),
	),
}

// DetailKeyMap defines keys for the article detail screen.
type DetailKeyMap struct {
	Back     key.Binding
	Open     key.Binding
	MarkRead key.Binding
	Next     key.Binding
	Prev     key.Binding
	Up       key.Binding
	Down     key.Binding
}

var DetailKeys = DetailKeyMap{
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Open: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open in browser"),
	),
	MarkRead: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "toggle read"),
	),
	Next: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "next article"),
	),
	Prev: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "prev article"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("k", "scroll up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("j", "scroll down"),
	),
}

// AskKeyMap defines keys for the ask screen.
type AskKeyMap struct {
	Submit key.Binding
	Back   key.Binding
	Save   key.Binding
	Up     key.Binding
	Down   key.Binding
}

var AskKeys = AskKeyMap{
	Submit: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "submit"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Save: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "save"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("k", "scroll up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("j", "scroll down"),
	),
}

// TopicsKeyMap defines keys for the topics browser screen.
type TopicsKeyMap struct {
	Enter    key.Binding
	Delete   key.Binding
	EditFreq key.Binding
	Tab      key.Binding
	Back     key.Binding
	Up       key.Binding
	Down     key.Binding
}

var TopicsKeys = TopicsKeyMap{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "subscribe"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "unsubscribe"),
	),
	EditFreq: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit frequency"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch section"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("j", "down"),
	),
}

// SourcesKeyMap defines keys for the sources management screen.
type SourcesKeyMap struct {
	Add         key.Binding
	Delete      key.Binding
	EditFreq    key.Binding
	EditContext key.Binding
	Back        key.Binding
	Up          key.Binding
	Down        key.Binding
}

var SourcesKeys = SourcesKeyMap{
	Add: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add source"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	EditFreq: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit frequency"),
	),
	EditContext: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "edit context"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("j", "down"),
	),
}

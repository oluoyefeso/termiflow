package tui

import "github.com/oluoyefeso/termiflow/pkg/models"

// Screen identifiers for routing.
type Screen int

const (
	ScreenDashboard Screen = iota
	ScreenFeed
	ScreenDetail
	ScreenAsk
	ScreenTopics
	ScreenStatus
)

// Navigation messages

// SwitchScreenMsg tells the app to change the active screen.
type SwitchScreenMsg struct {
	Screen Screen
	// Optional context for the target screen
	Subscription *models.Subscription
}

// Data messages

// SubscriptionsLoadedMsg carries subscription data after a DB fetch.
type SubscriptionsLoadedMsg struct {
	Subs []SubInfo
	Err  error
}

// SubInfo holds a subscription with its item counts.
type SubInfo struct {
	Sub    *models.Subscription
	Total  int
	Unread int
}

// FeedRefreshedMsg signals that a feed refresh completed.
type FeedRefreshedMsg struct {
	Topic    string
	NewItems int
	Err      error
}

// BannerMsg carries a notification banner to display.
type BannerMsg struct {
	Type    string // info, warning, update, breaking
	Message string
}

// BannersLoadedMsg carries all banners from the notification system.
type BannersLoadedMsg struct {
	Banners []BannerMsg
}

// ErrMsg carries an error to display.
type ErrMsg struct {
	Err error
}

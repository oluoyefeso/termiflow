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

// FeedRefreshedMsg signals that one subscription's refresh completed.
type FeedRefreshedMsg struct {
	Topic    string
	NewItems int
	Err      error
}

// AllRefreshDoneMsg signals that all subscriptions have been refreshed.
type AllRefreshDoneMsg struct {
	TotalNew int
	Errors   int   // count of per-topic errors
	Err      error // fatal error (provider init, DB)
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

// FeedItemsLoadedMsg carries feed items after a DB fetch.
type FeedItemsLoadedMsg struct {
	Items []*models.FeedItem
	Topic string
	Err   error
}

// ItemMarkedReadMsg signals that an item was marked as read.
type ItemMarkedReadMsg struct {
	ItemID int64
	Err    error
}

// OpenDetailMsg tells the app to open article detail for a specific item.
type OpenDetailMsg struct {
	Item  *models.FeedItem
	Items []*models.FeedItem // full list for n/p navigation
	Index int                // position in the list
	Topic string             // topic name for breadcrumb
}

// NavigateArticleMsg asks to move to the next or previous article in detail view.
type NavigateArticleMsg struct {
	Direction int // +1 for next, -1 for prev
}

// Ask screen messages

// AskSourcesLoadedMsg carries search results for the ask context.
type AskSourcesLoadedMsg struct {
	Sources []AskSource
	Err     error
}

// AskSource is a simplified search result for display.
type AskSource struct {
	Title  string
	URL    string
	Domain string
}

// AskChunkMsg carries a single streaming chunk from the LLM.
type AskChunkMsg struct {
	Content string
	Done    bool
	Err     error
}

// AskSavedMsg signals that the ask result was saved to disk.
type AskSavedMsg struct {
	Path string
	Err  error
}

// Topics screen messages

// TopicSubscribedMsg signals a new subscription was created.
type TopicSubscribedMsg struct {
	Topic string
	Err   error
}

// TopicUnsubscribedMsg signals a subscription was deleted.
type TopicUnsubscribedMsg struct {
	Topic string
	Err   error
}

// TopicFrequencyChangedMsg signals a subscription frequency was updated.
type TopicFrequencyChangedMsg struct {
	Topic string
	Err   error
}

// TopicsLoadedMsg carries topic data for the topics browser.
type TopicsLoadedMsg struct {
	Subscribed []SubInfo
	Available  []AvailableCategory
	Err        error
}

// AvailableCategory is a predefined category not yet subscribed.
type AvailableCategory struct {
	Name        string
	DisplayName string
	Description string
}

// ErrMsg carries an error to display.
type ErrMsg struct {
	Err error
}

// PerTopicRefreshMsg signals that one subscription's refresh completed (for inline status).
type PerTopicRefreshMsg struct {
	Topic    string
	NewItems int
	Err      error
}

// SaveFlashExpiredMsg signals that the save confirmation should fade to muted.
type SaveFlashExpiredMsg struct{}

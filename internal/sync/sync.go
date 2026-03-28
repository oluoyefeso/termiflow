// Package sync handles server-side data synchronization for managed-mode users.
// Self-hosted users are unaffected; all sync operations gate on IsManagedMode().
//
// Sync state machine:
//
//	CLI command start
//	    │
//	    ├── IsManagedMode()? ──no──► skip sync, use local DB only
//	    │
//	    yes
//	    │
//	    ▼
//	Pull (GET /v1/sync?since=last_sync_at)
//	    │
//	    ├── subscriptions: FULL-STATE (server sends ALL, client diffs + union merge)
//	    │     ├── server-only subs ──► create locally
//	    │     ├── local-only subs ──► push to server
//	    │     └── both have same sub ──► server-wins for metadata
//	    │
//	    ├── feed_items: INCREMENTAL (only since last_sync_at)
//	    │     └── INSERT OR IGNORE into local DB
//	    │
//	    └── pending_read_sync: replay buffered read-state changes
//	          └── PATCH /v1/feed-items/read ──► clear buffer on success
//	    │
//	    ▼
//	Update last_sync_at = server_time
//	    │
//	    ▼
//	Normal CLI operation (subscribe, refresh, read, etc.)
//	    │
//	    ▼
//	Push mutations immediately:
//	    subscribe    ──► PUT  /v1/subscriptions
//	    unsubscribe  ──► DELETE /v1/subscriptions/{topic}
//	    refresh      ──► POST /v1/feed-items (new items)
//	    mark-read    ──► PATCH /v1/feed-items/read (buffer on failure)
package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/oluoyefeso/termiflow/internal/config"
	"github.com/oluoyefeso/termiflow/internal/db"
	"github.com/oluoyefeso/termiflow/internal/providers"
	"github.com/oluoyefeso/termiflow/pkg/models"
)

const syncStaleness = 30 * time.Minute

// syncState is persisted to disk to track last successful sync.
type syncState struct {
	LastSyncAt string `json:"last_sync_at"`
}

// syncPullResponse is the JSON response from GET /v1/sync.
type syncPullResponse struct {
	Subscriptions  []serverSubscription `json:"subscriptions"`
	FeedItems      []serverFeedItem     `json:"feed_items"`
	FeedItemsPage  int                  `json:"feed_items_page"`
	FeedItemsTotal int                  `json:"feed_items_total_pages"`
	ServerTime     string               `json:"server_time"`
}

type serverSubscription struct {
	Topic     string   `json:"topic"`
	Category  string   `json:"category,omitempty"`
	Frequency string   `json:"frequency"`
	Sources   []string `json:"sources,omitempty"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
	IsActive  bool     `json:"is_active"`
}

type serverFeedItem struct {
	Topic          string  `json:"topic"`
	Title          string  `json:"title"`
	Summary        string  `json:"summary,omitempty"`
	SourceName     string  `json:"source_name,omitempty"`
	SourceURL      string  `json:"source_url"`
	PublishedAt    string  `json:"published_at,omitempty"`
	FetchedAt      string  `json:"fetched_at"`
	IsRead         bool    `json:"is_read"`
	RelevanceScore float64 `json:"relevance_score,omitempty"`
	Tags           string  `json:"tags,omitempty"`
	CreatedAt      string  `json:"created_at"`
}

type readStateItem struct {
	Topic     string `json:"topic"`
	SourceURL string `json:"source_url"`
}

// syncer is the package-level singleton. Nil when not in managed mode.
var syncer *syncClient

type syncClient struct {
	client *providers.ManagedClient
}

// Init initializes the sync package with a managed client.
// Call once from root command initialization. No-op if mc is nil.
func Init(mc *providers.ManagedClient) {
	if mc == nil {
		return
	}
	syncer = &syncClient{client: mc}
}

// PullIfStale runs a pull sync if more than 30 minutes since last sync.
// Safe to call frequently (on every command). No-op if not initialized.
func PullIfStale(ctx context.Context) {
	s := syncer
	if s == nil {
		return
	}
	state := loadSyncState()
	if state.LastSyncAt != "" {
		lastSync, err := time.Parse(time.RFC3339, state.LastSyncAt)
		if err == nil && time.Since(lastSync) < syncStaleness {
			return // fresh enough
		}
	}
	Pull(ctx)
}

// Pull fetches all server state and merges into local DB.
func Pull(ctx context.Context) {
	s := syncer
	if s == nil {
		return
	}

	// First, replay any pending read-state changes
	replayPendingReads(ctx)

	state := loadSyncState()

	// Pull from server
	path := "/v1/sync"
	if state.LastSyncAt != "" {
		path += "?since=" + url.QueryEscape(state.LastSyncAt)
	}

	var resp syncPullResponse
	if err := s.client.DoJSON(ctx, "GET", path, nil, &resp); err != nil {
		// Fail silently, use local cache
		return
	}

	// Merge into local DB (skip if DB not initialized, e.g. in tests)
	if db.IsOpen() {
		mergeSubscriptions(ctx, resp.Subscriptions)
		mergeFeedItems(resp.FeedItems)
	}

	// Pull remaining pages if paginated (reuse the same path base as first request)
	if db.IsOpen() {
		for page := 2; page <= resp.FeedItemsTotal; page++ {
			pagePath := fmt.Sprintf("%s&page=%d", path, page)
			if state.LastSyncAt == "" {
				pagePath = fmt.Sprintf("/v1/sync?page=%d", page)
			}
			var pageResp syncPullResponse
			if err := s.client.DoJSON(ctx, "GET", pagePath, nil, &pageResp); err != nil {
				break // best effort
			}
			mergeFeedItems(pageResp.FeedItems)
		}
	}

	// Update sync timestamp
	if resp.ServerTime != "" {
		saveSyncState(syncState{LastSyncAt: resp.ServerTime})
	}
}

// PushSubscription pushes a subscription upsert to the server.
func PushSubscription(ctx context.Context, sub *models.Subscription) {
	s := syncer
	if s == nil {
		return
	}
	serverSub := serverSubscription{
		Topic:     sub.Topic,
		Category:  sub.Category,
		Frequency: sub.Frequency,
		Sources:   sub.Sources,
		CreatedAt: sub.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: sub.UpdatedAt.UTC().Format(time.RFC3339),
		IsActive:  sub.IsActive,
	}
	// Fail silently, next pull catches up
	_ = s.client.DoJSON(ctx, "PUT", "/v1/subscriptions", serverSub, nil)
}

// DeleteSubscription removes a subscription from the server.
func DeleteSubscription(ctx context.Context, topic string) {
	s := syncer
	if s == nil {
		return
	}
	// DoJSON handles retry, 404 is fine (idempotent)
	_ = s.client.DoJSON(ctx, "DELETE", "/v1/subscriptions/"+url.PathEscape(topic), nil, nil)
}

// PushFeedItems pushes newly curated items to the server.
func PushFeedItems(ctx context.Context, items []*models.FeedItem) {
	s := syncer
	if s == nil || len(items) == 0 {
		return
	}

	serverItems := make([]serverFeedItem, 0, len(items))
	for _, item := range items {
		// Need topic name, resolve from subscription
		sub, err := db.GetSubscriptionByID(item.SubscriptionID)
		if err != nil {
			continue
		}
		var publishedAt string
		if item.PublishedAt != nil {
			publishedAt = item.PublishedAt.UTC().Format(time.RFC3339)
		}
		serverItems = append(serverItems, serverFeedItem{
			Topic:          sub.Topic,
			Title:          item.Title,
			Summary:        item.Summary,
			SourceName:     item.SourceName,
			SourceURL:      item.SourceURL,
			PublishedAt:    publishedAt,
			FetchedAt:      item.FetchedAt.UTC().Format(time.RFC3339),
			IsRead:         item.IsRead,
			RelevanceScore: item.RelevanceScore,
			Tags:           item.GetTagsJSON(),
			CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		})
	}

	if len(serverItems) == 0 {
		return
	}

	body := struct {
		Items []serverFeedItem `json:"items"`
	}{Items: serverItems}
	_ = s.client.DoJSON(ctx, "POST", "/v1/feed-items", body, nil)
}

// PushReadState pushes read-state changes to the server.
// On failure, buffers in pending_read_sync for later replay.
func PushReadState(ctx context.Context, items []*models.FeedItem) {
	s := syncer
	if s == nil || len(items) == 0 {
		return
	}

	readItems := make([]readStateItem, 0, len(items))
	for _, item := range items {
		sub, err := db.GetSubscriptionByID(item.SubscriptionID)
		if err != nil {
			continue
		}
		readItems = append(readItems, readStateItem{
			Topic:     sub.Topic,
			SourceURL: item.SourceURL,
		})
	}

	if len(readItems) == 0 {
		return
	}

	body := struct {
		Items []readStateItem `json:"items"`
	}{Items: readItems}

	if err := s.client.DoJSON(ctx, "PATCH", "/v1/feed-items/read", body, nil); err != nil {
		// Buffer for later replay
		for _, item := range readItems {
			db.InsertPendingReadSync(item.Topic, item.SourceURL)
		}
	}
}

// PushReadStateByIDs pushes read-state changes by feed item IDs.
func PushReadStateByIDs(ctx context.Context, ids []int64) {
	s := syncer
	if s == nil || len(ids) == 0 {
		return
	}

	readItems := db.GetReadSyncInfoByIDs(ids)
	if len(readItems) == 0 {
		return
	}

	apiItems := make([]readStateItem, len(readItems))
	for i, r := range readItems {
		apiItems[i] = readStateItem{Topic: r.Topic, SourceURL: r.SourceURL}
	}

	body := struct {
		Items []readStateItem `json:"items"`
	}{Items: apiItems}

	if err := s.client.DoJSON(ctx, "PATCH", "/v1/feed-items/read", body, nil); err != nil {
		for _, item := range apiItems {
			db.InsertPendingReadSync(item.Topic, item.SourceURL)
		}
	}
}

// mergeSubscriptions implements union merge for full-state subscription sync.
func mergeSubscriptions(ctx context.Context, serverSubs []serverSubscription) {
	localSubs, err := db.GetAllSubscriptions()
	if err != nil {
		return
	}

	// Build lookup maps
	serverMap := make(map[string]serverSubscription, len(serverSubs))
	for _, sub := range serverSubs {
		serverMap[sub.Topic] = sub
	}
	localMap := make(map[string]*models.Subscription, len(localSubs))
	for _, sub := range localSubs {
		localMap[sub.Topic] = sub
	}

	// Server-only subs: create locally
	for _, serverSub := range serverSubs {
		if _, exists := localMap[serverSub.Topic]; !exists {
			newSub := &models.Subscription{
				Topic:     serverSub.Topic,
				Category:  serverSub.Category,
				Frequency: serverSub.Frequency,
				Sources:   serverSub.Sources,
				IsActive:  serverSub.IsActive,
			}
			_ = db.CreateSubscription(newSub)
		} else {
			// Both have it: server-wins for metadata
			localSub := localMap[serverSub.Topic]
			if localSub.Frequency != serverSub.Frequency {
				localSub.Frequency = serverSub.Frequency
				_ = db.UpdateSubscription(localSub)
			}
		}
	}

	// Local-only subs: push to server
	for _, localSub := range localSubs {
		if _, exists := serverMap[localSub.Topic]; !exists {
			PushSubscription(ctx, localSub)
		}
	}

	// Note: With full-state sync, local-only subs are pushed above. On the next pull,
	// the server will include them. Deletion detection (server doesn't have a sub that
	// local does, AND it's not first sync) is deferred — the push-then-pull cycle handles it.
}

// mergeFeedItems merges server feed items into local DB.
func mergeFeedItems(items []serverFeedItem) {
	for _, item := range items {
		// Find local subscription for this topic
		sub, err := db.GetSubscription(item.Topic)
		if err != nil || sub == nil {
			continue // skip items for unknown topics
		}

		publishedAt, _ := time.Parse(time.RFC3339, item.PublishedAt)
		fetchedAt, _ := time.Parse(time.RFC3339, item.FetchedAt)

		feedItem := &models.FeedItem{
			SubscriptionID: sub.ID,
			Title:          item.Title,
			Summary:        item.Summary,
			SourceName:     item.SourceName,
			SourceURL:      item.SourceURL,
			PublishedAt:    &publishedAt,
			FetchedAt:      fetchedAt,
			IsRead:         item.IsRead,
			RelevanceScore: item.RelevanceScore,
		}
		_ = feedItem.SetTagsFromJSON(item.Tags) //nolint:errcheck

		// INSERT OR IGNORE (dedup by subscription_id + source_url)
		_ = db.CreateFeedItem(feedItem)

		// OR-merge read state: if server says read, mark local as read
		if item.IsRead {
			if feedItem.ID > 0 {
				// Newly inserted item
				_ = db.MarkItemRead(feedItem.ID)
			} else {
				// Existing item (INSERT OR IGNORE, ID stayed 0). Look up by URL.
				if existing, err := db.ItemBySubscriptionAndURL(sub.ID, item.SourceURL); err == nil && existing != nil && !existing.IsRead {
					_ = db.MarkItemRead(existing.ID)
				}
			}
		}
	}
}

// replayPendingReads sends buffered read-state changes to the server.
func replayPendingReads(ctx context.Context) {
	pending := db.GetPendingReadSync()
	if len(pending) == 0 {
		return
	}

	items := make([]readStateItem, 0, len(pending))
	for _, p := range pending {
		items = append(items, readStateItem{Topic: p.Topic, SourceURL: p.SourceURL})
	}

	body := struct {
		Items []readStateItem `json:"items"`
	}{Items: items}

	if syncer == nil {
		return
	}
	if err := syncer.client.DoJSON(ctx, "PATCH", "/v1/feed-items/read", body, nil); err == nil {
		db.ClearPendingReadSync()
	}
}

// syncStatePath returns the path to the sync state file.
// Variable for testing.
var syncStatePath = func() string {
	return filepath.Join(config.GetCacheDir(), "sync_state.json")
}

func loadSyncState() syncState {
	var state syncState
	data, err := os.ReadFile(syncStatePath())
	if err != nil {
		return state
	}
	_ = json.Unmarshal(data, &state)
	return state
}

func saveSyncState(state syncState) {
	data, err := json.Marshal(state)
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(syncStatePath()), 0o755)
	if err := os.WriteFile(syncStatePath(), data, 0o644); err != nil {
		log.Printf("sync: failed to save sync state: %v", err)
	}
}

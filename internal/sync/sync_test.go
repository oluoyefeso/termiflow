package sync

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/oluoyefeso/termiflow/internal/providers"
)

func TestPullIfStale_NilSyncer(t *testing.T) {
	syncer = nil
	PullIfStale(context.Background()) // should not panic
}

func TestPushSubscription_NilSyncer(t *testing.T) {
	syncer = nil
	PushSubscription(context.Background(), nil) // should not panic
}

func TestDeleteSubscription_NilSyncer(t *testing.T) {
	syncer = nil
	DeleteSubscription(context.Background(), "test") // should not panic
}

func TestPushFeedItems_NilSyncer(t *testing.T) {
	syncer = nil
	PushFeedItems(context.Background(), nil) // should not panic
}

func TestPushReadStateByIDs_NilSyncer(t *testing.T) {
	syncer = nil
	PushReadStateByIDs(context.Background(), nil) // should not panic
}

func TestSyncStateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	original := syncStatePath
	syncStatePath = func() string { return dir + "/sync_state.json" }
	defer func() { syncStatePath = original }()

	// Initially empty
	state := loadSyncState()
	if state.LastSyncAt != "" {
		t.Errorf("expected empty LastSyncAt, got %q", state.LastSyncAt)
	}

	// Save and reload
	now := time.Now().UTC().Format(time.RFC3339)
	saveSyncState(syncState{LastSyncAt: now})
	state = loadSyncState()
	if state.LastSyncAt != now {
		t.Errorf("expected %q, got %q", now, state.LastSyncAt)
	}
}

func TestPullCallsServer(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Path != "/v1/sync" {
			t.Errorf("expected /v1/sync, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("expected Authorization header")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(syncPullResponse{
			Subscriptions:  []serverSubscription{},
			FeedItems:      []serverFeedItem{},
			FeedItemsPage:  1,
			FeedItemsTotal: 1,
			ServerTime:     time.Now().UTC().Format(time.RFC3339),
		})
	}))
	defer srv.Close()

	mc := providers.NewManagedClient("tf_testkey", srv.URL)
	Init(mc)
	defer func() { syncer = nil }()

	// Override sync state path for test
	dir := t.TempDir()
	original := syncStatePath
	syncStatePath = func() string { return dir + "/sync_state.json" }
	defer func() { syncStatePath = original }()

	// Pull calls server (merge will be no-ops since no local DB)
	Pull(context.Background())

	if !called {
		t.Error("expected server to be called")
	}
}

func TestDeleteSubscriptionCallsServer(t *testing.T) {
	var deletedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			deletedPath = r.URL.Path
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	mc := providers.NewManagedClient("tf_testkey", srv.URL)
	Init(mc)
	defer func() { syncer = nil }()

	DeleteSubscription(context.Background(), "rust-lang")

	if deletedPath != "/v1/subscriptions/rust-lang" {
		t.Errorf("expected DELETE /v1/subscriptions/rust-lang, got %s", deletedPath)
	}
}

func TestPullIfStale_Fresh(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(syncPullResponse{})
	}))
	defer srv.Close()

	mc := providers.NewManagedClient("tf_testkey", srv.URL)
	Init(mc)
	defer func() { syncer = nil }()

	// Set sync state to recent
	dir := t.TempDir()
	original := syncStatePath
	syncStatePath = func() string { return dir + "/sync_state.json" }
	defer func() { syncStatePath = original }()

	saveSyncState(syncState{LastSyncAt: time.Now().UTC().Format(time.RFC3339)})

	PullIfStale(context.Background())

	if called {
		t.Error("expected server NOT to be called (sync is fresh)")
	}
}

func TestPullIfStale_Stale(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(syncPullResponse{
			ServerTime: time.Now().UTC().Format(time.RFC3339),
		})
	}))
	defer srv.Close()

	mc := providers.NewManagedClient("tf_testkey", srv.URL)
	Init(mc)
	defer func() { syncer = nil }()

	// Set sync state to old
	dir := t.TempDir()
	original := syncStatePath
	syncStatePath = func() string { return dir + "/sync_state.json" }
	defer func() { syncStatePath = original }()

	old := time.Now().Add(-45 * time.Minute).UTC().Format(time.RFC3339)
	saveSyncState(syncState{LastSyncAt: old})

	PullIfStale(context.Background())

	if !called {
		t.Error("expected server to be called (sync is stale)")
	}
}

func TestPullNetworkError_Silently(t *testing.T) {
	mc := providers.NewManagedClient("tf_testkey", "http://localhost:1") // invalid port
	Init(mc)
	defer func() { syncer = nil }()

	dir := t.TempDir()
	original := syncStatePath
	syncStatePath = func() string { return dir + "/sync_state.json" }
	defer func() { syncStatePath = original }()

	// Should not panic on network error
	Pull(context.Background())
}

func TestInitNil(t *testing.T) {
	syncer = nil
	Init(nil) // should not panic
	if syncer != nil {
		t.Error("expected syncer to remain nil after Init(nil)")
	}
}

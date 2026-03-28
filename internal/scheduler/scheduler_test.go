package scheduler

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/oluoyefeso/termiflow/internal/db"
	"github.com/oluoyefeso/termiflow/pkg/models"
)

func setupTestDB(t *testing.T) func() {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	return func() {
		db.Close() //nolint:errcheck
	}
}

// TestRefreshSubscriptionTransaction verifies that CreateFeedItemTx and
// UpdateLastFetchedTx work correctly within a transaction, and that the
// transaction commits atomically.
func TestRefreshSubscriptionTransaction(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create a subscription
	sub := &models.Subscription{
		Topic:     "tx-test-topic",
		Frequency: "daily",
		IsActive:  true,
	}
	if err := db.CreateSubscription(sub); err != nil {
		t.Fatalf("CreateSubscription() error = %v", err)
	}

	// Begin a transaction
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now()
	// Insert feed items within the transaction
	item1 := &models.FeedItem{
		SubscriptionID: sub.ID,
		Title:          "Test Item 1",
		Summary:        "Summary 1",
		SourceName:     "Source 1",
		SourceURL:      "https://example.com/1",
		PublishedAt:    &now,
		RelevanceScore: 0.9,
		FetchedAt:      now,
	}
	item2 := &models.FeedItem{
		SubscriptionID: sub.ID,
		Title:          "Test Item 2",
		Summary:        "Summary 2",
		SourceName:     "Source 2",
		SourceURL:      "https://example.com/2",
		PublishedAt:    &now,
		RelevanceScore: 0.8,
		FetchedAt:      now,
	}

	if err := db.CreateFeedItemTx(tx, item1); err != nil {
		t.Fatalf("CreateFeedItemTx(item1) error = %v", err)
	}
	if item1.ID == 0 {
		t.Error("item1.ID should be set after CreateFeedItemTx")
	}

	if err := db.CreateFeedItemTx(tx, item2); err != nil {
		t.Fatalf("CreateFeedItemTx(item2) error = %v", err)
	}
	if item2.ID == 0 {
		t.Error("item2.ID should be set after CreateFeedItemTx")
	}

	// Update last fetched within the transaction
	if err := db.UpdateLastFetchedTx(tx, sub.ID); err != nil {
		t.Fatalf("UpdateLastFetchedTx() error = %v", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	// Verify items were persisted
	items, err := db.GetFeedItems(db.FeedItemFilter{SubscriptionID: sub.ID})
	if err != nil {
		t.Fatalf("GetFeedItems() error = %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}

	// Verify last_fetched_at was updated
	updated, err := db.GetSubscriptionByID(sub.ID)
	if err != nil {
		t.Fatalf("GetSubscriptionByID() error = %v", err)
	}
	if updated.LastFetchedAt == nil {
		t.Error("LastFetchedAt should be set after UpdateLastFetchedTx")
	}
}

// TestRefreshSubscriptionTransactionRollback verifies that if we don't commit,
// items are not persisted (rollback behavior).
func TestRefreshSubscriptionTransactionRollback(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	sub := &models.Subscription{
		Topic:     "rollback-test",
		Frequency: "daily",
		IsActive:  true,
	}
	if err := db.CreateSubscription(sub); err != nil {
		t.Fatalf("CreateSubscription() error = %v", err)
	}

	// Begin a transaction but don't commit
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	now := time.Now()
	item := &models.FeedItem{
		SubscriptionID: sub.ID,
		Title:          "Rollback Item",
		SourceURL:      "https://example.com/rollback",
		PublishedAt:    &now,
		FetchedAt:      now,
	}
	if err := db.CreateFeedItemTx(tx, item); err != nil {
		t.Fatalf("CreateFeedItemTx() error = %v", err)
	}

	// Rollback
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}

	// Verify item was NOT persisted
	items, err := db.GetFeedItems(db.FeedItemFilter{SubscriptionID: sub.ID})
	if err != nil {
		t.Fatalf("GetFeedItems() error = %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items after rollback, got %d", len(items))
	}
}

// TestCreateFeedItemTxDuplicate verifies INSERT OR IGNORE behavior within a transaction.
func TestCreateFeedItemTxDuplicate(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	sub := &models.Subscription{
		Topic:     "dup-tx-test",
		Frequency: "daily",
		IsActive:  true,
	}
	if err := db.CreateSubscription(sub); err != nil {
		t.Fatalf("CreateSubscription() error = %v", err)
	}

	now := time.Now()
	// Insert first item
	item1 := &models.FeedItem{
		SubscriptionID: sub.ID,
		Title:          "Dup Item",
		SourceURL:      "https://example.com/dup",
		PublishedAt:    &now,
		FetchedAt:      now,
	}
	if err := db.CreateFeedItem(item1); err != nil {
		t.Fatalf("CreateFeedItem() error = %v", err)
	}

	// Try to insert duplicate in a transaction (same subscription_id + source_url)
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	item2 := &models.FeedItem{
		SubscriptionID: sub.ID,
		Title:          "Dup Item Again",
		SourceURL:      "https://example.com/dup",
		PublishedAt:    &now,
		FetchedAt:      now,
	}
	// Should not error due to INSERT OR IGNORE
	if err := db.CreateFeedItemTx(tx, item2); err != nil {
		t.Fatalf("CreateFeedItemTx() for duplicate should not error, got: %v", err)
	}
	// ID should be 0 for ignored insert
	if item2.ID != 0 {
		t.Logf("Note: duplicate item ID = %d (may be 0 depending on driver behavior)", item2.ID)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	// Verify only 1 item exists
	items, err := db.GetFeedItems(db.FeedItemFilter{SubscriptionID: sub.ID})
	if err != nil {
		t.Fatalf("GetFeedItems() error = %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item (duplicate ignored), got %d", len(items))
	}
}

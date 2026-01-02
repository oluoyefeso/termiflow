package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oluoyefeso/termiflow/pkg/models"
)

func setupTestDB(t *testing.T) func() {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	return func() {
		Close()
	}
}

func TestOpen(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer Close()

	if db == nil {
		t.Error("Open() should set db variable")
	}

	// Verify file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestGet(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	result := Get()
	if result == nil {
		t.Error("Get() returned nil")
	}
}

func TestClose(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	err = Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestCloseNil(t *testing.T) {
	db = nil
	err := Close()
	if err != nil {
		t.Errorf("Close() on nil db should not error, got: %v", err)
	}
}

func TestRunMigrations(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Verify tables exist by querying them
	tables := []string{"subscriptions", "feed_items", "query_history", "categories"}

	for _, table := range tables {
		_, err := db.Exec("SELECT 1 FROM " + table + " LIMIT 1")
		if err != nil {
			t.Errorf("Table %s does not exist: %v", table, err)
		}
	}
}

func TestSeedCategories(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Verify default categories were seeded
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count categories: %v", err)
	}

	expected := len(models.DefaultCategories)
	if count != expected {
		t.Errorf("Categories count = %d, want %d", count, expected)
	}
}

func TestCreateSubscription(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	sub := &models.Subscription{
		Topic:     "test-topic",
		Category:  "test-category",
		Frequency: "daily",
		Sources:   []string{"tavily", "rss"},
		IsActive:  true,
	}

	err := CreateSubscription(sub)
	if err != nil {
		t.Fatalf("CreateSubscription() error = %v", err)
	}

	if sub.ID == 0 {
		t.Error("CreateSubscription() should set ID")
	}
}

func TestGetSubscription(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create a subscription first
	sub := &models.Subscription{
		Topic:     "get-test-topic",
		Frequency: "weekly",
		IsActive:  true,
	}
	err := CreateSubscription(sub)
	if err != nil {
		t.Fatalf("CreateSubscription() error = %v", err)
	}

	// Get it back
	retrieved, err := GetSubscription("get-test-topic")
	if err != nil {
		t.Fatalf("GetSubscription() error = %v", err)
	}

	if retrieved.Topic != sub.Topic {
		t.Errorf("Topic = %q, want %q", retrieved.Topic, sub.Topic)
	}
	if retrieved.Frequency != sub.Frequency {
		t.Errorf("Frequency = %q, want %q", retrieved.Frequency, sub.Frequency)
	}
}

func TestGetSubscriptionNotFound(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	_, err := GetSubscription("nonexistent")
	if err == nil {
		t.Error("GetSubscription() should return error for nonexistent topic")
	}
}

func TestGetSubscriptionByID(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	sub := &models.Subscription{
		Topic:     "id-test-topic",
		Frequency: "hourly",
		IsActive:  true,
	}
	err := CreateSubscription(sub)
	if err != nil {
		t.Fatalf("CreateSubscription() error = %v", err)
	}

	retrieved, err := GetSubscriptionByID(sub.ID)
	if err != nil {
		t.Fatalf("GetSubscriptionByID() error = %v", err)
	}

	if retrieved.ID != sub.ID {
		t.Errorf("ID = %d, want %d", retrieved.ID, sub.ID)
	}
}

func TestGetActiveSubscriptions(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create active and inactive subscriptions
	active1 := &models.Subscription{Topic: "active1", Frequency: "daily", IsActive: true}
	active2 := &models.Subscription{Topic: "active2", Frequency: "daily", IsActive: true}
	inactive := &models.Subscription{Topic: "inactive", Frequency: "daily", IsActive: false}

	CreateSubscription(active1)
	CreateSubscription(active2)
	CreateSubscription(inactive)

	subs, err := GetActiveSubscriptions()
	if err != nil {
		t.Fatalf("GetActiveSubscriptions() error = %v", err)
	}

	if len(subs) != 2 {
		t.Errorf("GetActiveSubscriptions() returned %d subscriptions, want 2", len(subs))
	}

	for _, s := range subs {
		if !s.IsActive {
			t.Errorf("GetActiveSubscriptions() returned inactive subscription: %s", s.Topic)
		}
	}
}

func TestGetAllSubscriptions(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// Create subscriptions
	CreateSubscription(&models.Subscription{Topic: "all1", Frequency: "daily", IsActive: true})
	CreateSubscription(&models.Subscription{Topic: "all2", Frequency: "daily", IsActive: false})

	subs, err := GetAllSubscriptions()
	if err != nil {
		t.Fatalf("GetAllSubscriptions() error = %v", err)
	}

	if len(subs) != 2 {
		t.Errorf("GetAllSubscriptions() returned %d subscriptions, want 2", len(subs))
	}
}

func TestUpdateSubscription(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	sub := &models.Subscription{
		Topic:     "update-test",
		Frequency: "daily",
		IsActive:  true,
	}
	CreateSubscription(sub)

	// Update it
	sub.Frequency = "hourly"
	sub.IsActive = false
	err := UpdateSubscription(sub)
	if err != nil {
		t.Fatalf("UpdateSubscription() error = %v", err)
	}

	// Retrieve and verify
	updated, _ := GetSubscriptionByID(sub.ID)
	if updated.Frequency != "hourly" {
		t.Errorf("Frequency = %q, want %q", updated.Frequency, "hourly")
	}
	if updated.IsActive != false {
		t.Error("IsActive should be false")
	}
}

func TestDeleteSubscription(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	sub := &models.Subscription{
		Topic:     "delete-test",
		Frequency: "daily",
		IsActive:  true,
	}
	CreateSubscription(sub)

	err := DeleteSubscription("delete-test")
	if err != nil {
		t.Fatalf("DeleteSubscription() error = %v", err)
	}

	_, err = GetSubscription("delete-test")
	if err == nil {
		t.Error("Subscription should be deleted")
	}
}

func TestDeleteAllSubscriptions(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	CreateSubscription(&models.Subscription{Topic: "delall1", Frequency: "daily", IsActive: true})
	CreateSubscription(&models.Subscription{Topic: "delall2", Frequency: "daily", IsActive: true})

	err := DeleteAllSubscriptions()
	if err != nil {
		t.Fatalf("DeleteAllSubscriptions() error = %v", err)
	}

	subs, _ := GetAllSubscriptions()
	if len(subs) != 0 {
		t.Errorf("GetAllSubscriptions() returned %d subscriptions after delete all", len(subs))
	}
}

func TestCreateSubscriptionDuplicateTopic(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	sub1 := &models.Subscription{Topic: "duplicate", Frequency: "daily", IsActive: true}
	sub2 := &models.Subscription{Topic: "duplicate", Frequency: "weekly", IsActive: true}

	err := CreateSubscription(sub1)
	if err != nil {
		t.Fatalf("First CreateSubscription() error = %v", err)
	}

	err = CreateSubscription(sub2)
	if err == nil {
		t.Error("Second CreateSubscription() should fail for duplicate topic")
	}
}

func TestSubscriptionWithSources(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	sub := &models.Subscription{
		Topic:     "sources-test",
		Frequency: "daily",
		Sources:   []string{"tavily", "rss", "scrape"},
		IsActive:  true,
	}
	CreateSubscription(sub)

	retrieved, err := GetSubscription("sources-test")
	if err != nil {
		t.Fatalf("GetSubscription() error = %v", err)
	}

	if len(retrieved.Sources) != 3 {
		t.Errorf("Sources length = %d, want 3", len(retrieved.Sources))
	}
}

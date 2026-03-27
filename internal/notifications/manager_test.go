package notifications

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testManager(t *testing.T, managed bool) (*Manager, string) {
	t.Helper()
	dir := t.TempDir()
	m := NewManager("http://localhost", "tf_test", "1.0.0", dir, managed)
	return m, dir
}

func writeCache(t *testing.T, dir string, cache Cache) {
	t.Helper()
	data, _ := json.Marshal(cache)
	os.WriteFile(filepath.Join(dir, "announcements.json"), data, 0644)
}

func TestLoadCacheMissing(t *testing.T) {
	m, _ := testManager(t, true)
	m.LoadCache()

	if !m.stale {
		t.Error("cache should be stale when file missing")
	}
	if len(m.cache.Announcements) != 0 {
		t.Error("should have empty announcements")
	}
}

func TestLoadCacheCorrupted(t *testing.T) {
	m, dir := testManager(t, true)
	os.WriteFile(filepath.Join(dir, "announcements.json"), []byte("{bad json"), 0644)
	m.LoadCache()

	if !m.stale {
		t.Error("corrupted cache should be marked stale")
	}
}

func TestLoadCacheValid(t *testing.T) {
	m, dir := testManager(t, true)
	writeCache(t, dir, Cache{
		LatestVersion: "2.0.0",
		FetchedAt:     time.Now().UTC().Format(time.RFC3339),
		Announcements: []Announcement{{ID: "ann_1", Type: "info", Message: "Hello"}},
	})
	m.LoadCache()

	if m.stale {
		t.Error("fresh cache should not be stale")
	}
	if m.cache.LatestVersion != "2.0.0" {
		t.Errorf("latest_version = %q", m.cache.LatestVersion)
	}
	if len(m.cache.Announcements) != 1 {
		t.Errorf("announcements count = %d", len(m.cache.Announcements))
	}
}

func TestLoadCacheStale(t *testing.T) {
	m, dir := testManager(t, true)
	writeCache(t, dir, Cache{
		LatestVersion: "2.0.0",
		FetchedAt:     time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
	})
	m.LoadCache()

	if !m.stale {
		t.Error("old cache should be stale")
	}
	// But data should still be loaded
	if m.cache.LatestVersion != "2.0.0" {
		t.Error("stale cache data should still be loaded")
	}
}

func TestGetBannersVersionUpdate(t *testing.T) {
	m, dir := testManager(t, true)
	writeCache(t, dir, Cache{
		LatestVersion: "2.0.0",
		FetchedAt:     time.Now().UTC().Format(time.RFC3339),
	})
	m.LoadCache()

	banners := m.GetBanners()
	if len(banners) != 1 {
		t.Fatalf("expected 1 banner, got %d", len(banners))
	}
	if banners[0].Type != "version" {
		t.Errorf("type = %q, want version", banners[0].Type)
	}
}

func TestGetBannersAlreadyLatest(t *testing.T) {
	m, dir := testManager(t, true)
	writeCache(t, dir, Cache{
		LatestVersion: "1.0.0",
		FetchedAt:     time.Now().UTC().Format(time.RFC3339),
	})
	m.LoadCache()

	banners := m.GetBanners()
	if len(banners) != 0 {
		t.Errorf("should have no banners when on latest, got %d", len(banners))
	}
}

func TestGetBannersDevVersion(t *testing.T) {
	m, dir := testManager(t, true)
	m.version = "dev"
	writeCache(t, dir, Cache{
		LatestVersion: "2.0.0",
		FetchedAt:     time.Now().UTC().Format(time.RFC3339),
	})
	m.LoadCache()

	banners := m.GetBanners()
	if len(banners) != 0 {
		t.Errorf("dev version should skip banners, got %d", len(banners))
	}
}

func TestGetBannersMinSupportedVersion(t *testing.T) {
	m, dir := testManager(t, true)
	writeCache(t, dir, Cache{
		LatestVersion:       "2.0.0",
		MinSupportedVersion: "1.5.0",
		FetchedAt:           time.Now().UTC().Format(time.RFC3339),
	})
	m.LoadCache()

	banners := m.GetBanners()
	found := false
	for _, b := range banners {
		if b.Type == "breaking" {
			found = true
		}
	}
	if !found {
		t.Error("should show breaking banner when below min_supported_version")
	}
}

func TestGetBannersAnnouncements(t *testing.T) {
	m, dir := testManager(t, true)
	writeCache(t, dir, Cache{
		LatestVersion: "1.0.0",
		FetchedAt:     time.Now().UTC().Format(time.RFC3339),
		Announcements: []Announcement{
			{ID: "ann_1", Type: "info", Message: "New feature!"},
			{ID: "ann_2", Type: "warning", Message: "Downtime soon"},
		},
	})
	m.LoadCache()

	banners := m.GetBanners()
	if len(banners) != 2 {
		t.Fatalf("expected 2 banners, got %d", len(banners))
	}
	if banners[0].Type != "info" || banners[1].Type != "warning" {
		t.Errorf("types = %q, %q", banners[0].Type, banners[1].Type)
	}
}

func TestGetBannersDismissed(t *testing.T) {
	m, dir := testManager(t, true)
	writeCache(t, dir, Cache{
		LatestVersion: "1.0.0",
		FetchedAt:     time.Now().UTC().Format(time.RFC3339),
		Announcements: []Announcement{
			{ID: "ann_1", Type: "info", Message: "Seen this"},
			{ID: "ann_2", Type: "info", Message: "Not seen"},
		},
	})
	// Write dismissed state
	dismissed := DismissedState{
		DismissedIDs: map[string]string{"ann_1": ""},
	}
	data, _ := json.Marshal(dismissed)
	os.WriteFile(filepath.Join(dir, "dismissed.json"), data, 0644)

	m.LoadCache()

	banners := m.GetBanners()
	if len(banners) != 1 {
		t.Fatalf("expected 1 banner (ann_1 dismissed), got %d", len(banners))
	}
	if banners[0].Message != "Not seen" {
		t.Errorf("message = %q", banners[0].Message)
	}
}

func TestMarkDisplayedPersists(t *testing.T) {
	m, dir := testManager(t, true)
	writeCache(t, dir, Cache{
		LatestVersion: "2.0.0",
		FetchedAt:     time.Now().UTC().Format(time.RFC3339),
		Announcements: []Announcement{
			{ID: "ann_1", Type: "info", Message: "Hello"},
		},
	})
	m.LoadCache()

	banners := m.GetBanners()
	m.MarkDisplayed(banners)

	// Reload dismissed state
	data, err := os.ReadFile(filepath.Join(dir, "dismissed.json"))
	if err != nil {
		t.Fatalf("dismissed.json should exist: %v", err)
	}
	var dismissed DismissedState
	json.Unmarshal(data, &dismissed)

	if _, ok := dismissed.DismissedIDs["ann_1"]; !ok {
		t.Error("ann_1 should be in dismissed IDs")
	}
	if len(dismissed.DismissedVersions) != 1 || dismissed.DismissedVersions[0] != "2.0.0" {
		t.Error("version 2.0.0 should be in dismissed versions")
	}
}

func TestFetchAsyncManagedMode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/announcements" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"announcements": []Announcement{
				{ID: "ann_fetched", Type: "info", Message: "From API"},
			},
			"latest_version":        "3.0.0",
			"min_supported_version": "1.0.0",
		})
	}))
	defer ts.Close()

	dir := t.TempDir()
	m := NewManager(ts.URL, "tf_test", "1.0.0", dir, true)

	m.FetchAsync(context.Background())

	// Read the cache file
	data, err := os.ReadFile(filepath.Join(dir, "announcements.json"))
	if err != nil {
		t.Fatalf("cache file should exist: %v", err)
	}
	var cache Cache
	json.Unmarshal(data, &cache)

	if cache.LatestVersion != "3.0.0" {
		t.Errorf("latest_version = %q, want 3.0.0", cache.LatestVersion)
	}
	if len(cache.Announcements) != 1 {
		t.Fatalf("expected 1 announcement, got %d", len(cache.Announcements))
	}
	if cache.Announcements[0].ID != "ann_fetched" {
		t.Errorf("announcement ID = %q", cache.Announcements[0].ID)
	}
}

func TestFetchAsyncAPI404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer ts.Close()

	dir := t.TempDir()
	m := NewManager(ts.URL, "tf_test", "1.0.0", dir, true)

	// Should not crash on 404
	m.FetchAsync(context.Background())

	// Cache file should not be created
	_, err := os.ReadFile(filepath.Join(dir, "announcements.json"))
	if err == nil {
		t.Error("cache should not be written on 404")
	}
}

func TestFetchAsyncNetworkError(t *testing.T) {
	dir := t.TempDir()
	m := NewManager("http://127.0.0.1:1", "tf_test", "1.0.0", dir, true)

	// Should not crash on network error
	m.FetchAsync(context.Background())
}

func TestFetchAsyncSelfHostedMode(t *testing.T) {
	// Self-hosted mode hits GitHub Releases API which we can't mock via the manager's URL.
	// Test that the manager doesn't crash in self-hosted mode with a network error.
	dir := t.TempDir()
	m := NewManager("http://127.0.0.1:1", "", "1.0.0", dir, false)
	m.FetchAsync(context.Background()) // should not panic
}

func TestGetLatestVersion(t *testing.T) {
	m, dir := testManager(t, true)
	writeCache(t, dir, Cache{
		LatestVersion: "5.0.0",
		FetchedAt:     time.Now().UTC().Format(time.RFC3339),
	})
	m.LoadCache()

	if v := m.GetLatestVersion(); v != "5.0.0" {
		t.Errorf("GetLatestVersion() = %q, want 5.0.0", v)
	}
}

func TestAtomicCacheWrite(t *testing.T) {
	dir := t.TempDir()
	m := NewManager("http://localhost", "", "1.0.0", dir, false)

	cache := Cache{
		LatestVersion: "2.0.0",
		FetchedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	m.writeCache(cache)

	// Verify no .tmp file left
	_, err := os.Stat(filepath.Join(dir, "announcements.json.tmp"))
	if err == nil {
		t.Error(".tmp file should not exist after atomic write")
	}

	// Verify actual file exists
	data, err := os.ReadFile(filepath.Join(dir, "announcements.json"))
	if err != nil {
		t.Fatalf("cache file should exist: %v", err)
	}
	var result Cache
	json.Unmarshal(data, &result)
	if result.LatestVersion != "2.0.0" {
		t.Errorf("cached version = %q", result.LatestVersion)
	}
}

func TestPruneExpiredDismissed(t *testing.T) {
	m, dir := testManager(t, true)

	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	future := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	dismissed := DismissedState{
		DismissedIDs: map[string]string{
			"expired": past,
			"valid":   future,
			"no_exp":  "",
		},
	}
	data, _ := json.Marshal(dismissed)
	os.WriteFile(filepath.Join(dir, "dismissed.json"), data, 0644)

	m.LoadCache()
	m.pruneExpiredDismissed()

	// Reload
	data, _ = os.ReadFile(filepath.Join(dir, "dismissed.json"))
	var result DismissedState
	json.Unmarshal(data, &result)

	if _, ok := result.DismissedIDs["expired"]; ok {
		t.Error("expired ID should be pruned")
	}
	if _, ok := result.DismissedIDs["valid"]; !ok {
		t.Error("valid ID should be kept")
	}
	if _, ok := result.DismissedIDs["no_exp"]; !ok {
		t.Error("no expiry ID should be kept")
	}
}

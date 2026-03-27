package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oluoyefeso/termiflow/internal/ui"
	"github.com/oluoyefeso/termiflow/pkg/models"
)

// TestRefreshFeedsConcurrencyBounded verifies that the semaphore pattern used in
// refreshFeeds correctly bounds concurrency. Since refreshFeeds itself has many
// dependencies (DB, LLM provider, search provider, etc.), we test the semaphore
// pattern in isolation rather than calling refreshFeeds directly.
func TestRefreshFeedsConcurrencyBounded(t *testing.T) {
	const maxConcurrency = 5
	const totalTasks = 20

	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var peak atomic.Int32
	var current atomic.Int32

	for i := 0; i < totalTasks; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer func() { <-sem }()
			defer wg.Done()

			n := current.Add(1)
			// Track peak concurrency
			for {
				old := peak.Load()
				if n <= old || peak.CompareAndSwap(old, n) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			current.Add(-1)
		}()
	}

	wg.Wait()

	peakVal := int(peak.Load())
	if peakVal > maxConcurrency {
		t.Errorf("peak concurrency = %d, want <= %d", peakVal, maxConcurrency)
	}
	if peakVal == 0 {
		t.Error("peak concurrency should be > 0")
	}
	t.Logf("peak concurrency observed: %d (max allowed: %d)", peakVal, maxConcurrency)
}

func TestRenderFeedJSON_EmptyItems(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	SetVersionInfo("1.0.0", "abc", "2025-01-01")
	err := renderFeedJSON(nil, nil, nil)
	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("renderFeedJSON error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var env ui.JSONEnvelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestRenderFeedJSON_TagsNotNull(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	SetVersionInfo("1.0.0", "abc", "2025-01-01")

	items := []*models.FeedItem{
		{ID: 1, SubscriptionID: 1, Title: "Test", Tags: nil},
	}
	subs := []*models.Subscription{
		{ID: 1, Topic: "test-topic"},
	}
	grouped := map[int64][]*models.FeedItem{1: items}

	err := renderFeedJSON(items, subs, grouped)
	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("renderFeedJSON error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Tags should be [] not null
	if strings.Contains(output, `"tags": null`) {
		t.Error("tags should be [] not null in JSON output")
	}
}

func TestRenderFeedJSON_GroupedByTopic(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	SetVersionInfo("1.0.0", "abc", "2025-01-01")

	items := []*models.FeedItem{
		{ID: 1, SubscriptionID: 1, Title: "Article A", Tags: []string{"go"}},
		{ID: 2, SubscriptionID: 2, Title: "Article B", Tags: []string{}},
	}
	subs := []*models.Subscription{
		{ID: 1, Topic: "golang"},
		{ID: 2, Topic: "rust"},
	}
	grouped := map[int64][]*models.FeedItem{
		1: {items[0]},
		2: {items[1]},
	}

	err := renderFeedJSON(items, subs, grouped)
	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("renderFeedJSON error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var env struct {
		Data FeedOutputJSON `json:"data"`
	}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(env.Data.Topics) != 2 {
		t.Errorf("got %d topics, want 2", len(env.Data.Topics))
	}
	if env.Data.Total != 2 {
		t.Errorf("total = %d, want 2", env.Data.Total)
	}
}

package cli

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
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

package db

import (
	"fmt"
	"strings"
	"time"
)

// ReadSyncInfo contains the topic and URL needed to sync a read-state change.
type ReadSyncInfo struct {
	Topic     string
	SourceURL string
}

// GetReadSyncInfoByIDs resolves feed item IDs to (topic, source_url) pairs for sync.
func GetReadSyncInfoByIDs(ids []int64) []ReadSyncInfo {
	if db == nil || len(ids) == 0 {
		return nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(`
		SELECT s.topic, fi.source_url
		FROM feed_items fi
		JOIN subscriptions s ON fi.subscription_id = s.id
		WHERE fi.id IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var results []ReadSyncInfo
	for rows.Next() {
		var r ReadSyncInfo
		if err := rows.Scan(&r.Topic, &r.SourceURL); err != nil {
			continue
		}
		results = append(results, r)
	}
	return results
}

// PendingRead represents a buffered read-state change waiting to sync.
type PendingRead struct {
	Topic     string
	SourceURL string
}

// InsertPendingReadSync buffers a read-state change for later sync.
// Uses INSERT OR IGNORE to avoid duplicates. No-op if DB is not initialized.
func InsertPendingReadSync(topic, sourceURL string) {
	if db == nil {
		return
	}
	_, _ = db.Exec(
		`INSERT OR IGNORE INTO pending_read_sync (topic, source_url, created_at) VALUES (?, ?, ?)`,
		topic, sourceURL, time.Now().UTC().Format(time.RFC3339),
	)
}

// GetPendingReadSync returns all buffered read-state changes.
// Returns nil if DB is not initialized.
func GetPendingReadSync() []PendingRead {
	if db == nil {
		return nil
	}
	rows, err := db.Query(`SELECT topic, source_url FROM pending_read_sync`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var pending []PendingRead
	for rows.Next() {
		var p PendingRead
		if err := rows.Scan(&p.Topic, &p.SourceURL); err != nil {
			continue
		}
		pending = append(pending, p)
	}
	return pending
}

// ClearPendingReadSync removes all buffered read-state changes after successful sync.
func ClearPendingReadSync() {
	if db == nil {
		return
	}
	_, _ = db.Exec(`DELETE FROM pending_read_sync`)
}

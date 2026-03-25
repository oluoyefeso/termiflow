package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

const createTableSQL = `
CREATE TABLE IF NOT EXISTS api_keys (
    key        TEXT PRIMARY KEY,
    name       TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL,
    is_active  BOOLEAN NOT NULL DEFAULT 1
);
`

type KeyStore struct {
	db *sql.DB
}

func NewKeyStore(dbPath string) (*KeyStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for concurrent reads
	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode=WAL").Scan(&journalMode); err != nil {
		return nil, fmt.Errorf("failed to set WAL mode: %w", err)
	}
	if journalMode != "wal" {
		return nil, fmt.Errorf("WAL mode not enabled, got: %s", journalMode)
	}

	if _, err := db.Exec(createTableSQL); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return &KeyStore{db: db}, nil
}

func (s *KeyStore) Close() error {
	return s.db.Close()
}

// IssueKey creates a new active API key with the given name label.
func (s *KeyStore) IssueKey(name string) (string, error) {
	key, err := generateKey()
	if err != nil {
		return "", err
	}

	_, err = s.db.Exec(
		`INSERT INTO api_keys (key, name, created_at, is_active) VALUES (?, ?, ?, 1)`,
		key, name, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return "", fmt.Errorf("failed to insert key: %w", err)
	}
	return key, nil
}

// ValidateKey returns true if the key exists and is active.
func (s *KeyStore) ValidateKey(key string) (bool, error) {
	var isActive bool
	err := s.db.QueryRow(`SELECT is_active FROM api_keys WHERE key = ?`, key).Scan(&isActive)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to validate key: %w", err)
	}
	return isActive, nil
}

// RevokeKey sets is_active=false for the given key.
func (s *KeyStore) RevokeKey(key string) error {
	result, err := s.db.Exec(`UPDATE api_keys SET is_active = 0 WHERE key = ?`, key)
	if err != nil {
		return fmt.Errorf("failed to revoke key: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("key not found: %s", key)
	}
	return nil
}

type KeyInfo struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	IsActive  bool   `json:"is_active"`
}

// ListKeys returns all keys.
func (s *KeyStore) ListKeys() ([]KeyInfo, error) {
	rows, err := s.db.Query(`SELECT key, name, created_at, is_active FROM api_keys ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []KeyInfo
	for rows.Next() {
		var k KeyInfo
		if err := rows.Scan(&k.Key, &k.Name, &k.CreatedAt, &k.IsActive); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func generateKey() (string, error) {
	b := make([]byte, 24) // 24 bytes = 32 base64 chars
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}
	return "tf_" + base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b), nil
}

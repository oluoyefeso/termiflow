package db

import (
	"database/sql"
	"fmt"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/oluoyefeso/termiflow/internal/config"
)

var db *sql.DB

func Init() error {
	if err := config.EnsureDirectories(); err != nil {
		return err
	}

	dbPath := filepath.Join(config.GetDataDir(), "termiflow.db")
	return Open(dbPath)
}

func Open(dbPath string) error {
	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Enable WAL mode for concurrent read/write support (needed for parallel subscription refresh)
	var walMode string
	if err := db.QueryRow("PRAGMA journal_mode=WAL").Scan(&walMode); err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}
	if walMode != "wal" {
		return fmt.Errorf("WAL mode not supported (got %q) — concurrent writes may fail", walMode)
	}

	// Set busy timeout to 5 seconds (avoids "database is locked" under concurrent writes)
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Run migrations
	if err := RunMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func Get() *sql.DB {
	return db
}

// IsOpen returns true if the database has been initialized.
func IsOpen() bool {
	return db != nil
}

// Begin starts a new database transaction.
func Begin() (*sql.Tx, error) {
	return db.Begin()
}

func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

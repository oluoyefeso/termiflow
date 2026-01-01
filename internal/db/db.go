package db

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/termiflow/termiflow/internal/config"
	_ "modernc.org/sqlite"
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

	// Run migrations
	if err := RunMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func Get() *sql.DB {
	return db
}

func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

package db

import (
	"encoding/json"

	"github.com/oluoyefeso/termiflow/pkg/models"
)

func RunMigrations() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS subscriptions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			topic TEXT NOT NULL,
			category TEXT,
			frequency TEXT NOT NULL DEFAULT 'daily',
			sources TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_fetched_at DATETIME,
			is_active BOOLEAN DEFAULT 1,
			UNIQUE(topic)
		)`,

		`CREATE TABLE IF NOT EXISTS feed_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			subscription_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			summary TEXT,
			content TEXT,
			source_name TEXT,
			source_url TEXT,
			published_at DATETIME,
			fetched_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			is_read BOOLEAN DEFAULT 0,
			relevance_score REAL,
			tags TEXT,
			FOREIGN KEY (subscription_id) REFERENCES subscriptions(id) ON DELETE CASCADE
		)`,

		`CREATE TABLE IF NOT EXISTS query_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			query TEXT NOT NULL,
			response TEXT,
			provider TEXT,
			sources TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			display_name TEXT,
			description TEXT,
			default_sources TEXT,
			keywords TEXT
		)`,

		`CREATE INDEX IF NOT EXISTS idx_feed_items_subscription ON feed_items(subscription_id)`,
		`CREATE INDEX IF NOT EXISTS idx_feed_items_fetched ON feed_items(fetched_at)`,
		`CREATE INDEX IF NOT EXISTS idx_feed_items_read ON feed_items(is_read)`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_active ON subscriptions(is_active)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return err
		}
	}

	// Seed default categories
	if err := seedCategories(); err != nil {
		return err
	}

	return nil
}

func seedCategories() error {
	for _, cat := range models.DefaultCategories {
		keywords, _ := json.Marshal(cat.Keywords)
		sources, _ := json.Marshal(cat.DefaultRSS)

		_, err := db.Exec(`
			INSERT OR IGNORE INTO categories (name, display_name, description, default_sources, keywords)
			VALUES (?, ?, ?, ?, ?)
		`, cat.Name, cat.DisplayName, cat.Description, string(sources), string(keywords))

		if err != nil {
			return err
		}
	}
	return nil
}

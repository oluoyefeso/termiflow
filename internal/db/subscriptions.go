package db

import (
	"database/sql"
	"time"

	"github.com/termiflow/termiflow/pkg/models"
)

func CreateSubscription(sub *models.Subscription) error {
	result, err := db.Exec(`
		INSERT INTO subscriptions (topic, category, frequency, sources, is_active)
		VALUES (?, ?, ?, ?, ?)
	`, sub.Topic, sub.Category, sub.Frequency, sub.GetSourcesJSON(), sub.IsActive)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	sub.ID = id
	return nil
}

func GetSubscription(topic string) (*models.Subscription, error) {
	row := db.QueryRow(`
		SELECT id, topic, category, frequency, sources, created_at, updated_at, last_fetched_at, is_active
		FROM subscriptions WHERE topic = ?
	`, topic)

	return scanSubscription(row)
}

func GetSubscriptionByID(id int64) (*models.Subscription, error) {
	row := db.QueryRow(`
		SELECT id, topic, category, frequency, sources, created_at, updated_at, last_fetched_at, is_active
		FROM subscriptions WHERE id = ?
	`, id)

	return scanSubscription(row)
}

func GetActiveSubscriptions() ([]*models.Subscription, error) {
	rows, err := db.Query(`
		SELECT id, topic, category, frequency, sources, created_at, updated_at, last_fetched_at, is_active
		FROM subscriptions WHERE is_active = 1
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSubscriptions(rows)
}

func GetAllSubscriptions() ([]*models.Subscription, error) {
	rows, err := db.Query(`
		SELECT id, topic, category, frequency, sources, created_at, updated_at, last_fetched_at, is_active
		FROM subscriptions
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSubscriptions(rows)
}

func UpdateSubscription(sub *models.Subscription) error {
	sub.UpdatedAt = time.Now()
	_, err := db.Exec(`
		UPDATE subscriptions
		SET topic = ?, category = ?, frequency = ?, sources = ?, updated_at = ?, last_fetched_at = ?, is_active = ?
		WHERE id = ?
	`, sub.Topic, sub.Category, sub.Frequency, sub.GetSourcesJSON(), sub.UpdatedAt, sub.LastFetchedAt, sub.IsActive, sub.ID)
	return err
}

func DeleteSubscription(topic string) error {
	_, err := db.Exec(`DELETE FROM subscriptions WHERE topic = ?`, topic)
	return err
}

func DeleteAllSubscriptions() error {
	_, err := db.Exec(`DELETE FROM subscriptions`)
	return err
}

func UpdateLastFetched(id int64) error {
	now := time.Now()
	_, err := db.Exec(`UPDATE subscriptions SET last_fetched_at = ?, updated_at = ? WHERE id = ?`, now, now, id)
	return err
}

func scanSubscription(row *sql.Row) (*models.Subscription, error) {
	var sub models.Subscription
	var sources sql.NullString
	var category sql.NullString
	var lastFetched sql.NullTime

	err := row.Scan(
		&sub.ID,
		&sub.Topic,
		&category,
		&sub.Frequency,
		&sources,
		&sub.CreatedAt,
		&sub.UpdatedAt,
		&lastFetched,
		&sub.IsActive,
	)
	if err != nil {
		return nil, err
	}

	if category.Valid {
		sub.Category = category.String
	}
	if sources.Valid {
		_ = sub.SetSourcesFromJSON(sources.String)
	}
	if lastFetched.Valid {
		sub.LastFetchedAt = &lastFetched.Time
	}

	return &sub, nil
}

func scanSubscriptions(rows *sql.Rows) ([]*models.Subscription, error) {
	var subs []*models.Subscription

	for rows.Next() {
		var sub models.Subscription
		var sources sql.NullString
		var category sql.NullString
		var lastFetched sql.NullTime

		err := rows.Scan(
			&sub.ID,
			&sub.Topic,
			&category,
			&sub.Frequency,
			&sources,
			&sub.CreatedAt,
			&sub.UpdatedAt,
			&lastFetched,
			&sub.IsActive,
		)
		if err != nil {
			return nil, err
		}

		if category.Valid {
			sub.Category = category.String
		}
		if sources.Valid {
			_ = sub.SetSourcesFromJSON(sources.String)
		}
		if lastFetched.Valid {
			sub.LastFetchedAt = &lastFetched.Time
		}

		subs = append(subs, &sub)
	}

	return subs, rows.Err()
}

func GetSubscriptionItemCount(subID int64) (total int, unread int, err error) {
	row := db.QueryRow(`
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN is_read = 0 THEN 1 ELSE 0 END) as unread
		FROM feed_items WHERE subscription_id = ?
	`, subID)

	var unreadNull sql.NullInt64
	err = row.Scan(&total, &unreadNull)
	if err != nil {
		return 0, 0, err
	}
	if unreadNull.Valid {
		unread = int(unreadNull.Int64)
	}
	return total, unread, nil
}

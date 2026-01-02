package db

import (
	"database/sql"
	"time"

	"github.com/oluoyefeso/termiflow/pkg/models"
)

func CreateFeedItem(item *models.FeedItem) error {
	result, err := db.Exec(`
		INSERT INTO feed_items (subscription_id, title, summary, content, source_name, source_url, published_at, relevance_score, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, item.SubscriptionID, item.Title, item.Summary, item.Content, item.SourceName, item.SourceURL, item.PublishedAt, item.RelevanceScore, item.GetTagsJSON())

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	item.ID = id
	return nil
}

func CreateFeedItems(items []*models.FeedItem) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
		INSERT INTO feed_items (subscription_id, title, summary, content, source_name, source_url, published_at, relevance_score, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range items {
		result, err := stmt.Exec(
			item.SubscriptionID, item.Title, item.Summary, item.Content,
			item.SourceName, item.SourceURL, item.PublishedAt, item.RelevanceScore, item.GetTagsJSON(),
		)
		if err != nil {
			return err
		}
		id, _ := result.LastInsertId()
		item.ID = id
	}

	return tx.Commit()
}

type FeedItemFilter struct {
	SubscriptionID int64
	Topic          string
	Unread         bool
	Since          *time.Time
	Limit          int
	Offset         int
}

func GetFeedItems(filter FeedItemFilter) ([]*models.FeedItem, error) {
	query := `
		SELECT fi.id, fi.subscription_id, fi.title, fi.summary, fi.content,
			   fi.source_name, fi.source_url, fi.published_at, fi.fetched_at,
			   fi.is_read, fi.relevance_score, fi.tags
		FROM feed_items fi
		JOIN subscriptions s ON fi.subscription_id = s.id
		WHERE 1=1
	`
	args := []interface{}{}

	if filter.SubscriptionID > 0 {
		query += " AND fi.subscription_id = ?"
		args = append(args, filter.SubscriptionID)
	}

	if filter.Topic != "" {
		query += " AND s.topic = ?"
		args = append(args, filter.Topic)
	}

	if filter.Unread {
		query += " AND fi.is_read = 0"
	}

	if filter.Since != nil {
		query += " AND fi.fetched_at >= ?"
		args = append(args, *filter.Since)
	}

	query += " ORDER BY fi.relevance_score DESC, fi.published_at DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFeedItems(rows)
}

func GetFeedItemsBySubscription(subID int64, limit int, unreadOnly bool) ([]*models.FeedItem, error) {
	return GetFeedItems(FeedItemFilter{
		SubscriptionID: subID,
		Unread:         unreadOnly,
		Limit:          limit,
	})
}

func MarkItemRead(id int64) error {
	_, err := db.Exec(`UPDATE feed_items SET is_read = 1 WHERE id = ?`, id)
	return err
}

func MarkItemsRead(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`UPDATE feed_items SET is_read = 1 WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, id := range ids {
		if _, err := stmt.Exec(id); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func MarkAllReadForSubscription(subID int64) error {
	_, err := db.Exec(`UPDATE feed_items SET is_read = 1 WHERE subscription_id = ?`, subID)
	return err
}

func DeleteOldItems(olderThan time.Time) (int64, error) {
	result, err := db.Exec(`DELETE FROM feed_items WHERE fetched_at < ?`, olderThan)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func ItemExistsByURL(url string) (bool, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM feed_items WHERE source_url = ?`, url).Scan(&count)
	return count > 0, err
}

func scanFeedItems(rows *sql.Rows) ([]*models.FeedItem, error) {
	var items []*models.FeedItem

	for rows.Next() {
		var item models.FeedItem
		var summary, content, sourceName, sourceURL, tags sql.NullString
		var publishedAt sql.NullTime
		var relevanceScore sql.NullFloat64

		err := rows.Scan(
			&item.ID,
			&item.SubscriptionID,
			&item.Title,
			&summary,
			&content,
			&sourceName,
			&sourceURL,
			&publishedAt,
			&item.FetchedAt,
			&item.IsRead,
			&relevanceScore,
			&tags,
		)
		if err != nil {
			return nil, err
		}

		if summary.Valid {
			item.Summary = summary.String
		}
		if content.Valid {
			item.Content = content.String
		}
		if sourceName.Valid {
			item.SourceName = sourceName.String
		}
		if sourceURL.Valid {
			item.SourceURL = sourceURL.String
		}
		if publishedAt.Valid {
			item.PublishedAt = &publishedAt.Time
		}
		if relevanceScore.Valid {
			item.RelevanceScore = relevanceScore.Float64
		}
		if tags.Valid {
			_ = item.SetTagsFromJSON(tags.String)
		}

		items = append(items, &item)
	}

	return items, rows.Err()
}

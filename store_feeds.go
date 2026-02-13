package main

import (
	"context"
	"database/sql"
	"time"
)

const feedBaseColumns = `id, url, site_url, title, description, last_fetched_at, etag, last_modified, last_error, error_count, created_at`

func (s *Store) CreateFeed(ctx context.Context, url string) (Feed, bool, error) {
	res, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO feeds(url) VALUES (?)`, url)
	if err != nil {
		return Feed{}, false, err
	}
	inserted := false
	if n, _ := res.RowsAffected(); n > 0 {
		inserted = true
	}
	feed, err := s.GetFeedByURL(ctx, url)
	if err != nil {
		return Feed{}, false, err
	}
	return feed, inserted, nil
}

func (s *Store) GetFeedByURL(ctx context.Context, url string) (Feed, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+feedBaseColumns+` FROM feeds WHERE url = ?`, url)
	return scanFeedRow(row)
}

func (s *Store) GetFeedByID(ctx context.Context, id int64) (Feed, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+feedBaseColumns+` FROM feeds WHERE id = ?`, id)
	return scanFeedRow(row)
}

func (s *Store) DeleteFeed(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM feeds WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) ListFeedsWithCounts(ctx context.Context) ([]Feed, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			f.id,
			f.url,
			f.site_url,
			f.title,
			f.description,
			f.last_fetched_at,
			f.etag,
			f.last_modified,
			f.last_error,
			f.error_count,
			f.created_at,
			COALESCE(SUM(CASE WHEN e.id IS NOT NULL AND COALESCE(es.read, 0) = 0 THEN 1 ELSE 0 END), 0) AS unread_count,
			COUNT(e.id) AS total_count
		FROM feeds f
		LEFT JOIN entries e ON e.feed_id = f.id
		LEFT JOIN entry_status es ON es.entry_id = e.id
		GROUP BY f.id
		ORDER BY COALESCE(NULLIF(f.title, ''), f.url) COLLATE NOCASE
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	feeds := make([]Feed, 0)
	for rows.Next() {
		feed, err := scanFeedWithCountsRow(rows)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}
	return feeds, rows.Err()
}

func (s *Store) ListFeedsForFetch(ctx context.Context, id *int64) ([]Feed, error) {
	query := `SELECT ` + feedBaseColumns + ` FROM feeds`
	args := make([]any, 0, 1)
	if id != nil {
		query += ` WHERE id = ?`
		args = append(args, *id)
	}
	query += ` ORDER BY id`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	feeds := make([]Feed, 0)
	for rows.Next() {
		feed, err := scanFeedRow(rows)
		if err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if id != nil && len(feeds) == 0 {
		return nil, sql.ErrNoRows
	}
	return feeds, nil
}

func (s *Store) UpdateFeedFetchSuccess(ctx context.Context, feedID int64, title, siteURL, description, etag, lastModified string, fetchedAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE feeds
		SET
			title = CASE WHEN ? <> '' THEN ? ELSE title END,
			site_url = CASE WHEN ? <> '' THEN ? ELSE site_url END,
			description = CASE WHEN ? <> '' THEN ? ELSE description END,
			etag = ?,
			last_modified = ?,
			last_fetched_at = ?,
			last_error = NULL,
			error_count = 0
		WHERE id = ?
	`, title, title, siteURL, siteURL, description, description, etag, lastModified, fetchedAt.UTC().Format(time.RFC3339Nano), feedID)
	return err
}

func (s *Store) SetFeedError(ctx context.Context, feedID int64, errMsg string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE feeds
		SET last_error = ?, error_count = error_count + 1
		WHERE id = ?
	`, truncate(errMsg, 500), feedID)
	return err
}

func (s *Store) GetFetchStaleness(ctx context.Context, staleAfter time.Duration) (hasFeeds bool, stale bool, lastFetched *time.Time, err error) {
	var count int
	var maxFetched sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*), MAX(last_fetched_at) FROM feeds`).Scan(&count, &maxFetched); err != nil {
		return false, false, nil, err
	}
	if count == 0 {
		return false, false, nil, nil
	}
	hasFeeds = true
	if !maxFetched.Valid {
		return hasFeeds, true, nil, nil
	}

	t, parseErr := parseDBTime(maxFetched.String)
	if parseErr != nil {
		return hasFeeds, true, nil, nil
	}
	lastFetched = &t
	stale = time.Since(t) > staleAfter
	return hasFeeds, stale, lastFetched, nil
}

func (s *Store) ListFeedURLs(ctx context.Context) ([]Feed, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, url, COALESCE(NULLIF(title, ''), url), site_url
		FROM feeds
		ORDER BY COALESCE(NULLIF(title, ''), url) COLLATE NOCASE
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	feeds := make([]Feed, 0)
	for rows.Next() {
		var feed Feed
		var siteURL sql.NullString
		if err := rows.Scan(&feed.ID, &feed.URL, &feed.Title, &siteURL); err != nil {
			return nil, err
		}
		feed.SiteURL = siteURL.String
		feeds = append(feeds, feed)
	}
	return feeds, rows.Err()
}

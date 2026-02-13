package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

const entrySelectColumns = `
	e.id, e.feed_id, COALESCE(NULLIF(f.title, ''), f.url), e.guid,
	e.url, e.external_url, e.title, e.summary, e.content_html, e.content_md,
	e.author, e.published_at, e.date_modified, e.fetched_at,
	COALESCE(es.read, 0), COALESCE(es.starred, 0)
`

func (s *Store) UpsertEntry(ctx context.Context, in UpsertEntryInput) (entryID int64, inserted bool, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var existingID int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM entries WHERE feed_id = ? AND guid = ?`, in.FeedID, in.GUID).Scan(&existingID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		inserted = true
	case err != nil:
		return 0, false, err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO entries (
			feed_id, guid, url, external_url, title, summary,
			content_html, content_md, author, published_at, date_modified
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(feed_id, guid) DO UPDATE SET
			url = excluded.url,
			external_url = excluded.external_url,
			title = excluded.title,
			summary = excluded.summary,
			content_html = excluded.content_html,
			content_md = excluded.content_md,
			author = excluded.author,
			published_at = excluded.published_at,
			date_modified = excluded.date_modified,
			fetched_at = CURRENT_TIMESTAMP
	`,
		in.FeedID,
		in.GUID,
		in.URL,
		in.ExternalURL,
		in.Title,
		in.Summary,
		in.ContentHTML,
		in.ContentMD,
		in.Author,
		timeToDBString(in.PublishedAt),
		timeToDBString(in.DateModified),
	)
	if err != nil {
		return 0, false, err
	}

	if err = tx.QueryRowContext(ctx, `SELECT id FROM entries WHERE feed_id = ? AND guid = ?`, in.FeedID, in.GUID).Scan(&entryID); err != nil {
		return 0, false, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO entry_status(entry_id) VALUES (?)`, entryID); err != nil {
		return 0, false, err
	}

	if err = tx.Commit(); err != nil {
		return 0, false, err
	}
	return entryID, inserted, nil
}

func (s *Store) ListEntries(ctx context.Context, opts EntryListOptions) ([]Entry, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	status := strings.ToLower(strings.TrimSpace(opts.Status))
	if status == "" {
		status = "unread"
	}

	where := make([]string, 0, 2)
	args := make([]any, 0, 3)
	if opts.FeedID > 0 {
		where = append(where, "e.feed_id = ?")
		args = append(args, opts.FeedID)
	}
	switch status {
	case "unread":
		where = append(where, "COALESCE(es.read, 0) = 0")
	case "read":
		where = append(where, "COALESCE(es.read, 0) = 1")
	case "all":
	default:
		return nil, fmt.Errorf("%w: invalid status %q (expected unread|read|all)", ErrInvalidInput, opts.Status)
	}

	query := `SELECT ` + entrySelectColumns + `
		FROM entries e
		JOIN feeds f ON f.id = e.feed_id
		LEFT JOIN entry_status es ON es.entry_id = e.id`
	if len(where) > 0 {
		query += ` WHERE ` + strings.Join(where, " AND ")
	}
	query += ` ORDER BY CASE WHEN e.published_at IS NULL OR e.published_at = '' THEN 1 ELSE 0 END, COALESCE(e.published_at, e.fetched_at) DESC LIMIT ?`
	args = append(args, opts.Limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]Entry, 0)
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (s *Store) GetEntry(ctx context.Context, id int64) (Entry, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+entrySelectColumns+`
		FROM entries e
		JOIN feeds f ON f.id = e.feed_id
		LEFT JOIN entry_status es ON es.entry_id = e.id
		WHERE e.id = ?
	`, id)
	entry, err := scanEntry(row)
	if err != nil {
		return Entry{}, wrapNotFound("entry", err)
	}
	return entry, nil
}

func (s *Store) SearchEntries(ctx context.Context, opts SearchOptions) ([]Entry, error) {
	if strings.TrimSpace(opts.Query) == "" {
		return nil, fmt.Errorf("%w: query must not be empty", ErrInvalidInput)
	}
	if opts.Limit <= 0 {
		opts.Limit = 50
	}

	where := []string{"entries_fts MATCH ?"}
	args := []any{opts.Query}
	if opts.Feed > 0 {
		where = append(where, "e.feed_id = ?")
		args = append(args, opts.Feed)
	}

	query := `
		SELECT ` + entrySelectColumns + `
		FROM entries_fts
		JOIN entries e ON e.id = entries_fts.rowid
		JOIN feeds f ON f.id = e.feed_id
		LEFT JOIN entry_status es ON es.entry_id = e.id
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY bm25(entries_fts), COALESCE(e.published_at, e.fetched_at) DESC
		LIMIT ?
	`
	args = append(args, opts.Limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]Entry, 0)
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (s *Store) GetStats(ctx context.Context) (Stats, error) {
	var stats Stats
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM feeds`).Scan(&stats.Feeds); err != nil {
		return Stats{}, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM entries`).Scan(&stats.Total); err != nil {
		return Stats{}, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM entry_status WHERE read = 0`).Scan(&stats.Unread); err != nil {
		return Stats{}, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM entry_status WHERE starred = 1`).Scan(&stats.Starred); err != nil {
		return Stats{}, err
	}
	return stats, nil
}

func (s *Store) PruneReadEntriesOlderThan(ctx context.Context, days int) (int64, error) {
	if days <= 0 {
		return 0, nil
	}

	cutoff := timestampBeforeDays(days)
	rows, err := s.db.QueryContext(ctx, `
		SELECT e.id, COALESCE(e.published_at, e.fetched_at)
		FROM entries e
		JOIN entry_status es ON es.entry_id = e.id
		WHERE es.read = 1
		  AND COALESCE(es.starred, 0) = 0
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		var ts string
		if err := rows.Scan(&id, &ts); err != nil {
			return 0, err
		}
		t, err := parseDBTime(ts)
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			ids = append(ids, id)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}

	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}

	query := `DELETE FROM entries WHERE id IN (` + strings.Join(placeholders, ",") + `)`
	res, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

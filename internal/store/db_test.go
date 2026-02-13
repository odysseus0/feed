package store

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestOpenDB_MigrationsAreAppliedAndIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "feed.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB first: %v", err)
	}
	_ = db.Close()

	db, err = OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB second: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if count != len(migrations) {
		t.Fatalf("migration count = %d, want %d", count, len(migrations))
	}
}

func TestOpenDB_UpgradesLegacySchemaWithoutDataLoss(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.db")

	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	if _, err := raw.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		t.Fatalf("pragma foreign_keys: %v", err)
	}
	if _, err := raw.Exec(`
		CREATE TABLE feeds (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url TEXT NOT NULL UNIQUE,
			site_url TEXT,
			title TEXT,
			description TEXT,
			last_fetched_at DATETIME,
			etag TEXT,
			last_modified TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		t.Fatalf("create legacy feeds: %v", err)
	}
	if _, err := raw.Exec(`
		CREATE TABLE entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			feed_id INTEGER NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
			guid TEXT NOT NULL,
			url TEXT,
			external_url TEXT,
			title TEXT,
			summary TEXT,
			content_html TEXT,
			content_md TEXT,
			author TEXT,
			published_at DATETIME,
			date_modified DATETIME,
			fetched_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(feed_id, guid)
		);
	`); err != nil {
		t.Fatalf("create legacy entries: %v", err)
	}
	if _, err := raw.Exec(`
		CREATE TABLE entry_status (
			entry_id INTEGER PRIMARY KEY REFERENCES entries(id) ON DELETE CASCADE,
			read BOOLEAN NOT NULL DEFAULT 0,
			starred BOOLEAN NOT NULL DEFAULT 0,
			read_at DATETIME,
			starred_at DATETIME
		);
	`); err != nil {
		t.Fatalf("create legacy entry_status: %v", err)
	}
	if _, err := raw.Exec(`
		CREATE VIRTUAL TABLE entries_fts USING fts5(
			title,
			summary,
			content_md,
			content=entries,
			content_rowid=id
		);
	`); err != nil {
		t.Fatalf("create legacy entries_fts: %v", err)
	}
	if _, err := raw.Exec(`INSERT INTO feeds(url, title) VALUES ('https://example.com/feed.xml', 'Legacy Feed')`); err != nil {
		t.Fatalf("insert legacy feed: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("close legacy db: %v", err)
	}

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB upgrade: %v", err)
	}
	defer db.Close()

	var feedCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM feeds WHERE title = 'Legacy Feed'`).Scan(&feedCount); err != nil {
		t.Fatalf("count preserved feeds: %v", err)
	}
	if feedCount != 1 {
		t.Fatalf("expected preserved feed row, got %d", feedCount)
	}

	hasLastError, err := hasFeedColumnInDB(db, "last_error")
	if err != nil {
		t.Fatalf("check last_error column: %v", err)
	}
	if !hasLastError {
		t.Fatalf("expected last_error column after migration")
	}

	hasErrorCount, err := hasFeedColumnInDB(db, "error_count")
	if err != nil {
		t.Fatalf("check error_count column: %v", err)
	}
	if !hasErrorCount {
		t.Fatalf("expected error_count column after migration")
	}
}

func hasFeedColumnInDB(db *sql.DB, column string) (bool, error) {
	rows, err := db.Query(`PRAGMA table_info(feeds);`)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notNull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

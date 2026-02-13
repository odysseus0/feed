package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func openDB(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// SQLite allows one writer at a time; serialize connections to avoid busy/locked storms.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		db.Close()
		return nil, err
	}
	if err := initSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func initSchema(db *sql.DB) error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS feeds (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url TEXT NOT NULL UNIQUE,
			site_url TEXT,
			title TEXT,
			description TEXT,
			last_fetched_at DATETIME,
			etag TEXT,
			last_modified TEXT,
			last_error TEXT,
			error_count INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS entries (
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
		);`,
		`CREATE TABLE IF NOT EXISTS entry_status (
			entry_id INTEGER PRIMARY KEY REFERENCES entries(id) ON DELETE CASCADE,
			read BOOLEAN NOT NULL DEFAULT 0,
			starred BOOLEAN NOT NULL DEFAULT 0,
			read_at DATETIME,
			starred_at DATETIME
		);`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS entries_fts USING fts5(
			title,
			summary,
			content_md,
			content=entries,
			content_rowid=id
		);`,
		`CREATE TRIGGER IF NOT EXISTS entries_ai AFTER INSERT ON entries BEGIN
			INSERT INTO entries_fts(rowid, title, summary, content_md)
			VALUES (new.id, new.title, new.summary, new.content_md);
		END;`,
		`CREATE TRIGGER IF NOT EXISTS entries_ad AFTER DELETE ON entries BEGIN
			INSERT INTO entries_fts(entries_fts, rowid, title, summary, content_md)
			VALUES ('delete', old.id, old.title, old.summary, old.content_md);
		END;`,
		`CREATE TRIGGER IF NOT EXISTS entries_au AFTER UPDATE ON entries BEGIN
			INSERT INTO entries_fts(entries_fts, rowid, title, summary, content_md)
			VALUES ('delete', old.id, old.title, old.summary, old.content_md);
			INSERT INTO entries_fts(rowid, title, summary, content_md)
			VALUES (new.id, new.title, new.summary, new.content_md);
		END;`,
		`CREATE INDEX IF NOT EXISTS idx_entries_feed_published ON entries(feed_id, published_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_entry_status_read ON entry_status(read);`,
		`CREATE INDEX IF NOT EXISTS idx_entry_status_starred ON entry_status(starred);`,
	}
	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	if err := ensureFeedErrorColumns(db); err != nil {
		return err
	}
	if _, err := db.Exec(`INSERT INTO entries_fts(entries_fts) VALUES ('rebuild');`); err != nil {
		return fmt.Errorf("rebuild fts: %w", err)
	}
	return nil
}

func ensureFeedErrorColumns(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(feeds);`)
	if err != nil {
		return err
	}
	defer rows.Close()

	hasLastError := false
	hasErrorCount := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notNull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dflt, &pk); err != nil {
			return err
		}
		switch name {
		case "last_error":
			hasLastError = true
		case "error_count":
			hasErrorCount = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if !hasLastError {
		if _, err := db.Exec(`ALTER TABLE feeds ADD COLUMN last_error TEXT;`); err != nil {
			return err
		}
	}
	if !hasErrorCount {
		if _, err := db.Exec(`ALTER TABLE feeds ADD COLUMN error_count INTEGER NOT NULL DEFAULT 0;`); err != nil {
			return err
		}
	}
	return nil
}

package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type migration struct {
	name string
	run  func(tx *sql.Tx) error
}

var migrations = []migration{
	{name: "0001_initial_schema", run: migrateInitialSchema},
	{name: "0002_feed_error_columns", run: migrateFeedErrorColumns},
	{name: "0003_fts_rebuild", run: migrateFTSRebuild},
}

func OpenDB(path string) (*sql.DB, error) {
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

	for _, pragma := range []string{
		`PRAGMA foreign_keys = ON;`,
		`PRAGMA busy_timeout = 5000;`,
		`PRAGMA journal_mode = WAL;`,
	} {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, err
		}
	}

	if err := runMigrations(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func runMigrations(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name TEXT PRIMARY KEY,
			applied_at DATETIME NOT NULL
		);
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied, err := appliedMigrations(db)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if _, ok := applied[m.name]; ok {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", m.name, err)
		}

		if err := m.run(tx); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("run migration %s: %w", m.name, err)
		}

		if _, err := tx.Exec(
			`INSERT INTO schema_migrations(name, applied_at) VALUES (?, ?)`,
			m.name,
			time.Now().UTC().Format(time.RFC3339Nano),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", m.name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", m.name, err)
		}
	}

	return nil
}

func appliedMigrations(db *sql.DB) (map[string]struct{}, error) {
	rows, err := db.Query(`SELECT name FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("query schema_migrations: %w", err)
	}
	defer rows.Close()

	out := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan schema_migrations: %w", err)
		}
		out[name] = struct{}{}
	}
	return out, rows.Err()
}

func migrateInitialSchema(tx *sql.Tx) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS feeds (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url TEXT NOT NULL UNIQUE,
			site_url TEXT,
			title TEXT,
			description TEXT,
			last_fetched_at DATETIME,
			etag TEXT,
			last_modified TEXT,
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

	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func migrateFeedErrorColumns(tx *sql.Tx) error {
	hasLastError, err := hasFeedColumn(tx, "last_error")
	if err != nil {
		return err
	}
	if !hasLastError {
		if _, err := tx.Exec(`ALTER TABLE feeds ADD COLUMN last_error TEXT;`); err != nil {
			return err
		}
	}

	hasErrorCount, err := hasFeedColumn(tx, "error_count")
	if err != nil {
		return err
	}
	if !hasErrorCount {
		if _, err := tx.Exec(`ALTER TABLE feeds ADD COLUMN error_count INTEGER NOT NULL DEFAULT 0;`); err != nil {
			return err
		}
	}
	return nil
}

func hasFeedColumn(tx *sql.Tx, target string) (bool, error) {
	rows, err := tx.Query(`PRAGMA table_info(feeds);`)
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
		if name == target {
			return true, nil
		}
	}
	return false, rows.Err()
}

func migrateFTSRebuild(tx *sql.Tx) error {
	if _, err := tx.Exec(`INSERT INTO entries_fts(entries_fts) VALUES ('rebuild');`); err != nil {
		return err
	}
	return nil
}

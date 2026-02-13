package store

import (
	"context"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "feed.db")
	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return NewStore(db)
}

func mustCreateFeed(t *testing.T, store *Store, url string) Feed {
	t.Helper()
	feed, _, err := store.CreateFeed(context.Background(), url)
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}
	return feed
}

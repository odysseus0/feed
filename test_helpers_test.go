package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "feed.db")
	db, err := openDB(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return NewStore(db)
}

func newTestFetcher(store *Store) *Fetcher {
	cfg := Config{
		HTTPTimeout:      5 * time.Second,
		FetchConcurrency: 4,
		UserAgent:        "feed-test/1.0",
	}
	return NewFetcher(store, NewRenderer(), cfg)
}

func mustCreateFeed(t *testing.T, store *Store, url string) Feed {
	t.Helper()
	feed, _, err := store.CreateFeed(context.Background(), url)
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}
	return feed
}

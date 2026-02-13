package fetch

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/odysseus0/feed/internal/config"
	"github.com/odysseus0/feed/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "feed.db")
	db, err := store.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return store.NewStore(db)
}

func newTestFetcher(s *store.Store) *Fetcher {
	cfg := config.Config{
		HTTPTimeout:      5 * time.Second,
		FetchConcurrency: 4,
		UserAgent:        "feed-test/1.0",
	}
	return NewFetcher(s, NewRenderer(), cfg)
}

func mustCreateFeed(t *testing.T, s *store.Store, url string) store.Feed {
	t.Helper()
	feed, _, err := s.CreateFeed(context.Background(), url)
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}
	return feed
}

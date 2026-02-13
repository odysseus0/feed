package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestStoreGetFetchStaleness(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	hasFeeds, stale, lastFetched, err := store.GetFetchStaleness(ctx, 30*time.Minute)
	if err != nil {
		t.Fatalf("staleness empty: %v", err)
	}
	if hasFeeds || stale || lastFetched != nil {
		t.Fatalf("unexpected empty staleness: hasFeeds=%v stale=%v last=%v", hasFeeds, stale, lastFetched)
	}

	f1 := mustCreateFeed(t, store, "https://example.com/feed1.xml")
	hasFeeds, stale, lastFetched, err = store.GetFetchStaleness(ctx, 30*time.Minute)
	if err != nil {
		t.Fatalf("staleness one feed: %v", err)
	}
	if !hasFeeds || !stale || lastFetched != nil {
		t.Fatalf("expected stale with never-fetched feed")
	}

	now := time.Now().UTC()
	if err := store.UpdateFeedFetchSuccess(ctx, f1.ID, "f1", "", "", "", "", now); err != nil {
		t.Fatalf("update fetch success: %v", err)
	}
	_ = mustCreateFeed(t, store, "https://example.com/feed2.xml")

	hasFeeds, stale, lastFetched, err = store.GetFetchStaleness(ctx, time.Hour)
	if err != nil {
		t.Fatalf("staleness mixed: %v", err)
	}
	if !hasFeeds || stale || lastFetched == nil {
		t.Fatalf("expected non-stale when max(last_fetched_at) is fresh")
	}

	old := time.Now().Add(-2 * time.Hour).UTC()
	if err := store.UpdateFeedFetchSuccess(ctx, f1.ID, "f1", "", "", "", "", old); err != nil {
		t.Fatalf("update old fetch: %v", err)
	}
	_, stale, _, err = store.GetFetchStaleness(ctx, time.Hour)
	if err != nil {
		t.Fatalf("staleness old: %v", err)
	}
	if !stale {
		t.Fatalf("expected stale when latest fetch is too old")
	}
}

func TestStoreStatusAndSearch(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	feed := mustCreateFeed(t, store, "https://example.com/feed.xml")

	id1, inserted, err := store.UpsertEntry(ctx, UpsertEntryInput{FeedID: feed.ID, GUID: "g1", Title: "Rust ownership", Summary: "memory safety"})
	if err != nil || !inserted {
		t.Fatalf("upsert 1: inserted=%v err=%v", inserted, err)
	}
	id2, inserted, err := store.UpsertEntry(ctx, UpsertEntryInput{FeedID: feed.ID, GUID: "g2", Title: "Go scheduler", Summary: "goroutines"})
	if err != nil || !inserted {
		t.Fatalf("upsert 2: inserted=%v err=%v", inserted, err)
	}

	results, err := store.SearchEntries(ctx, SearchOptions{Query: "Rust", Limit: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) == 0 || results[0].ID != id1 {
		t.Fatalf("expected rust entry in search results")
	}

	if err := store.UpdateEntryRead(ctx, id1, true); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	if _, err := store.ToggleEntryStarred(ctx, id1); err != nil {
		t.Fatalf("toggle star: %v", err)
	}

	unread, err := store.ListEntries(ctx, EntryListOptions{Status: "unread", Limit: 10})
	if err != nil {
		t.Fatalf("list unread: %v", err)
	}
	if len(unread) != 1 || unread[0].ID != id2 {
		t.Fatalf("expected only second entry unread, got %#v", unread)
	}

	stats, err := store.GetStats(ctx)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.Total != 2 || stats.Unread != 1 || stats.Starred != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestStorePruneReadEntriesOlderThan(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	feed := mustCreateFeed(t, store, "https://example.com/prune.xml")

	oldReadID, _, err := store.UpsertEntry(ctx, UpsertEntryInput{
		FeedID:      feed.ID,
		GUID:        "old-read",
		Title:       "old read",
		PublishedAt: ptrTime(time.Now().Add(-72 * 24 * time.Hour)),
	})
	if err != nil {
		t.Fatalf("upsert old read: %v", err)
	}
	oldStarID, _, err := store.UpsertEntry(ctx, UpsertEntryInput{
		FeedID:      feed.ID,
		GUID:        "old-star",
		Title:       "old star",
		PublishedAt: ptrTime(time.Now().Add(-72 * 24 * time.Hour)),
	})
	if err != nil {
		t.Fatalf("upsert old starred: %v", err)
	}
	oldUnreadID, _, err := store.UpsertEntry(ctx, UpsertEntryInput{
		FeedID:      feed.ID,
		GUID:        "old-unread",
		Title:       "old unread",
		PublishedAt: ptrTime(time.Now().Add(-72 * 24 * time.Hour)),
	})
	if err != nil {
		t.Fatalf("upsert old unread: %v", err)
	}
	newReadID, _, err := store.UpsertEntry(ctx, UpsertEntryInput{
		FeedID:      feed.ID,
		GUID:        "new-read",
		Title:       "new read",
		PublishedAt: ptrTime(time.Now().Add(-2 * time.Hour)),
	})
	if err != nil {
		t.Fatalf("upsert new read: %v", err)
	}

	if err := store.UpdateEntryRead(ctx, oldReadID, true); err != nil {
		t.Fatalf("mark old read: %v", err)
	}
	if err := store.UpdateEntryRead(ctx, oldStarID, true); err != nil {
		t.Fatalf("mark old star read: %v", err)
	}
	if _, err := store.ToggleEntryStarred(ctx, oldStarID); err != nil {
		t.Fatalf("star old star: %v", err)
	}
	if err := store.UpdateEntryRead(ctx, newReadID, true); err != nil {
		t.Fatalf("mark new read: %v", err)
	}
	_ = oldUnreadID // ensure this remains unread and should not be pruned

	pruned, err := store.PruneReadEntriesOlderThan(ctx, 30)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if pruned != 1 {
		t.Fatalf("expected exactly one entry pruned, got %d", pruned)
	}

	all, err := store.ListEntries(ctx, EntryListOptions{Status: "all", Limit: 20})
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 remaining entries, got %d", len(all))
	}
}

func ptrTime(t time.Time) *time.Time {
	tt := t.UTC()
	return &tt
}

func TestStoreNotFoundErrors(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	if err := s.DeleteFeed(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeleteFeed err=%v, want ErrNotFound", err)
	}
	if _, err := s.GetEntry(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetEntry err=%v, want ErrNotFound", err)
	}
	if err := s.UpdateEntryRead(ctx, 999, true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateEntryRead err=%v, want ErrNotFound", err)
	}
}

func TestStoreInvalidInputErrors(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	if _, err := s.ListEntries(ctx, EntryListOptions{Status: "bad", Limit: 5}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("ListEntries err=%v, want ErrInvalidInput", err)
	}
	if _, err := s.SearchEntries(ctx, SearchOptions{Query: " ", Limit: 5}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("SearchEntries err=%v, want ErrInvalidInput", err)
	}
}

func TestStoreFeedQueriesAndBatchStatus(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	feedA := mustCreateFeed(t, s, "https://example.com/a.xml")
	feedB := mustCreateFeed(t, s, "https://example.com/b.xml")

	entryA1, _, err := s.UpsertEntry(ctx, UpsertEntryInput{FeedID: feedA.ID, GUID: "a1", Title: "A1"})
	if err != nil {
		t.Fatalf("upsert a1: %v", err)
	}
	entryA2, _, err := s.UpsertEntry(ctx, UpsertEntryInput{FeedID: feedA.ID, GUID: "a2", Title: "A2"})
	if err != nil {
		t.Fatalf("upsert a2: %v", err)
	}
	entryB1, _, err := s.UpsertEntry(ctx, UpsertEntryInput{FeedID: feedB.ID, GUID: "b1", Title: "B1"})
	if err != nil {
		t.Fatalf("upsert b1: %v", err)
	}

	if err := s.SetEntriesRead(ctx, []int64{entryA1, entryA2}, true); err != nil {
		t.Fatalf("SetEntriesRead true: %v", err)
	}
	if err := s.SetEntriesRead(ctx, []int64{entryA2}, false); err != nil {
		t.Fatalf("SetEntriesRead false: %v", err)
	}
	if err := s.SetEntriesStarred(ctx, []int64{entryB1}, true); err != nil {
		t.Fatalf("SetEntriesStarred true: %v", err)
	}
	if err := s.SetEntriesStarred(ctx, []int64{entryB1}, false); err != nil {
		t.Fatalf("SetEntriesStarred false: %v", err)
	}

	feeds, err := s.ListFeedsWithCounts(ctx)
	if err != nil {
		t.Fatalf("ListFeedsWithCounts: %v", err)
	}
	if len(feeds) != 2 {
		t.Fatalf("expected 2 feeds, got %d", len(feeds))
	}

	fetched, err := s.ListFeedsForFetch(ctx, &feedA.ID)
	if err != nil {
		t.Fatalf("ListFeedsForFetch by id: %v", err)
	}
	if len(fetched) != 1 || fetched[0].ID != feedA.ID {
		t.Fatalf("unexpected fetch list: %#v", fetched)
	}
	allFetch, err := s.ListFeedsForFetch(ctx, nil)
	if err != nil {
		t.Fatalf("ListFeedsForFetch all: %v", err)
	}
	if len(allFetch) != 2 {
		t.Fatalf("expected 2 feeds for fetch, got %d", len(allFetch))
	}
	missingID := int64(999)
	if _, err := s.ListFeedsForFetch(ctx, &missingID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ListFeedsForFetch missing err=%v, want ErrNotFound", err)
	}

	gotA, err := s.GetFeedByID(ctx, feedA.ID)
	if err != nil {
		t.Fatalf("GetFeedByID: %v", err)
	}
	if gotA.ID != feedA.ID {
		t.Fatalf("unexpected feed id %d", gotA.ID)
	}

	longErr := make([]byte, 800)
	for i := range longErr {
		longErr[i] = 'x'
	}
	if err := s.SetFeedError(ctx, feedA.ID, string(longErr)); err != nil {
		t.Fatalf("SetFeedError: %v", err)
	}
	gotA, err = s.GetFeedByID(ctx, feedA.ID)
	if err != nil {
		t.Fatalf("GetFeedByID after SetFeedError: %v", err)
	}
	if gotA.ErrorCount != 1 {
		t.Fatalf("expected error_count=1, got %d", gotA.ErrorCount)
	}
	if len(gotA.LastError) != 500 {
		t.Fatalf("expected truncated last_error length 500, got %d", len(gotA.LastError))
	}

	feedURLs, err := s.ListFeedURLs(ctx)
	if err != nil {
		t.Fatalf("ListFeedURLs: %v", err)
	}
	if len(feedURLs) != 2 {
		t.Fatalf("expected 2 feed urls, got %d", len(feedURLs))
	}
}

package main

import (
	"context"
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

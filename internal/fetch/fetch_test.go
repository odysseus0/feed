package fetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/odysseus0/feed/internal/config"
	"github.com/odysseus0/feed/internal/store"
)

func TestFetcher_ConditionalFetchAndSanitization(t *testing.T) {
	store := newTestStore(t)
	fetcher := newTestFetcher(store)
	ctx := context.Background()

	const etag = `"v1"`
	const feedXML = `<?xml version="1.0"?>
<rss version="2.0"><channel>
<title>Test Feed</title><link>https://example.com</link><description>desc</description>
<item>
  <guid>item-1</guid>
  <title>Entry One</title>
  <link>https://example.com/entry-1</link>
  <description><![CDATA[<p>Hello</p><script>alert(1)</script>]]></description>
</item>
</channel></rss>`

	var reqCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&reqCount, 1)
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", etag)
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(feedXML))
	}))
	defer srv.Close()

	feed := mustCreateFeed(t, store, srv.URL)

	rep1, err := fetcher.Fetch(ctx, &feed.ID)
	if err != nil {
		t.Fatalf("fetch 1: %v", err)
	}
	if len(rep1.Results) != 1 || rep1.Results[0].NewEntries != 1 {
		t.Fatalf("unexpected first fetch report: %+v", rep1)
	}

	entries, err := store.ListEntries(ctx, EntryListOptions{Status: "all", Limit: 10})
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if strings.Contains(strings.ToLower(entries[0].ContentHTML), "<script") {
		t.Fatalf("content_html was not sanitized: %q", entries[0].ContentHTML)
	}

	rep2, err := fetcher.Fetch(ctx, &feed.ID)
	if err != nil {
		t.Fatalf("fetch 2: %v", err)
	}
	if len(rep2.Results) != 1 || !rep2.Results[0].NotModified {
		t.Fatalf("expected not_modified on second fetch, got %+v", rep2.Results)
	}

	entries, err = store.ListEntries(ctx, EntryListOptions{Status: "all", Limit: 10})
	if err != nil {
		t.Fatalf("list entries after second fetch: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected dedup to keep 1 entry, got %d", len(entries))
	}
	if atomic.LoadInt32(&reqCount) < 2 {
		t.Fatalf("expected >=2 requests, got %d", reqCount)
	}

	updatedFeed, err := store.GetFeedByID(ctx, feed.ID)
	if err != nil {
		t.Fatalf("get feed by id: %v", err)
	}
	if updatedFeed.ETag != etag {
		t.Fatalf("etag not persisted: got %q want %q", updatedFeed.ETag, etag)
	}
}

func TestFetcher_GUIDFallbackFromEntryURL(t *testing.T) {
	store := newTestStore(t)
	fetcher := newTestFetcher(store)
	ctx := context.Background()

	const entryURL = "https://example.com/entry-without-guid"
	const feedXML = `<?xml version="1.0"?>
<rss version="2.0"><channel>
<title>Test Feed</title><link>https://example.com</link>
<item>
  <title>No GUID</title>
  <link>` + entryURL + `</link>
  <description>desc</description>
</item>
</channel></rss>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(feedXML))
	}))
	defer srv.Close()

	feed := mustCreateFeed(t, store, srv.URL)
	if _, err := fetcher.Fetch(ctx, &feed.ID); err != nil {
		t.Fatalf("fetch: %v", err)
	}

	entries, err := store.ListEntries(ctx, EntryListOptions{Status: "all", Limit: 1})
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(entries))
	}
	guid := entries[0].GUID
	if guid != entryURL {
		t.Fatalf("expected guid fallback to link, got %q", guid)
	}
}

func TestFetcher_GUIDFallbackFromTitleAndPublishedDate(t *testing.T) {
	store := newTestStore(t)
	fetcher := newTestFetcher(store)
	ctx := context.Background()

	// Entry has no guid and no link; dedup should fall back to title+published hash.
	const feedXML = `<?xml version="1.0"?>
<rss version="2.0"><channel>
<title>Test Feed</title><link>https://example.com</link>
<item>
  <title>No GUID or Link</title>
  <pubDate>Fri, 13 Feb 2026 00:00:00 GMT</pubDate>
  <description>desc</description>
</item>
</channel></rss>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(feedXML))
	}))
	defer srv.Close()

	feed := mustCreateFeed(t, store, srv.URL)
	if _, err := fetcher.Fetch(ctx, &feed.ID); err != nil {
		t.Fatalf("fetch 1: %v", err)
	}
	if _, err := fetcher.Fetch(ctx, &feed.ID); err != nil {
		t.Fatalf("fetch 2: %v", err)
	}

	all, err := store.ListEntries(ctx, EntryListOptions{Status: "all", Limit: 10})
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected dedup to keep one entry, got %d", len(all))
	}
	if !strings.HasPrefix(all[0].GUID, "sha1:") {
		t.Fatalf("expected sha1 guid fallback, got %q", all[0].GUID)
	}
}

func TestFetcherWithProgress_CallbackCounts(t *testing.T) {
	store := newTestStore(t)
	fetcher := newTestFetcher(store)
	ctx := context.Background()

	const feedXML = `<?xml version="1.0"?><rss version="2.0"><channel><title>T</title><link>https://example.com</link><item><guid>g1</guid><title>A</title><link>https://example.com/a</link></item></channel></rss>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(feedXML))
	}))
	defer srv.Close()

	mustCreateFeed(t, store, srv.URL+"/one")
	mustCreateFeed(t, store, srv.URL+"/two")

	callbackCount := 0
	lastTotal := 0
	_, err := fetcher.FetchWithProgress(ctx, nil, func(done, total int, result FetchResult) {
		callbackCount++
		lastTotal = total
		if done < 1 || done > total {
			t.Errorf("invalid callback done/total: %d/%d", done, total)
		}
	})
	if err != nil {
		t.Fatalf("fetch with progress: %v", err)
	}
	if callbackCount != 2 || lastTotal != 2 {
		t.Fatalf("expected 2 callbacks with total 2, got callbacks=%d total=%d", callbackCount, lastTotal)
	}
}

func TestFetcherFetchSingleFeedNotFoundSetsError(t *testing.T) {
	store := newTestStore(t)
	fetcher := newTestFetcher(store)
	ctx := context.Background()

	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	feed := mustCreateFeed(t, store, srv.URL)
	rep, err := fetcher.Fetch(ctx, &feed.ID)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(rep.Results) != 1 || rep.Results[0].Error == "" {
		t.Fatalf("expected fetch error result, got %+v", rep.Results)
	}

	updated, err := store.GetFeedByID(ctx, feed.ID)
	if err != nil {
		t.Fatalf("get feed: %v", err)
	}
	if updated.ErrorCount == 0 || updated.LastError == "" {
		t.Fatalf("expected feed error state to be populated: %+v", updated)
	}
}

func TestFetcherFetchUnknownFeedID(t *testing.T) {
	s := newTestStore(t)
	fetcher := newTestFetcher(s)
	ctx := context.Background()

	id := int64(999)
	_, err := fetcher.Fetch(ctx, &id)
	if err == nil {
		t.Fatalf("expected error for unknown feed id")
	}
	if err != store.ErrNotFound {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetcherReportsPruneWarning(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	const feedXML = `<?xml version="1.0"?><rss version="2.0"><channel><title>T</title><link>https://example.com</link></channel></rss>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(feedXML))
	}))
	defer srv.Close()

	feed := mustCreateFeed(t, s, srv.URL)
	old := time.Now().AddDate(0, 0, -40).UTC()
	entryID, _, err := s.UpsertEntry(ctx, UpsertEntryInput{
		FeedID:      feed.ID,
		GUID:        "old-guid",
		Title:       "old",
		PublishedAt: &old,
	})
	if err != nil {
		t.Fatalf("upsert old entry: %v", err)
	}
	if err := s.UpdateEntryRead(ctx, entryID, true); err != nil {
		t.Fatalf("mark old entry read: %v", err)
	}

	fetcher := NewFetcher(s, NewRenderer(), config.Config{
		HTTPTimeout:      5 * time.Second,
		FetchConcurrency: 2,
		RetentionDays:    30,
		UserAgent:        "feed-test/1.0",
	})
	rep, err := fetcher.Fetch(ctx, &feed.ID)
	if err != nil {
		t.Fatalf("fetch with retention: %v", err)
	}
	found := false
	for _, warning := range rep.Warnings {
		if strings.Contains(warning, "pruned 1 old entries") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected prune warning, got %#v", rep.Warnings)
	}
}

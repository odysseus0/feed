package fetch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mmcdole/gofeed"
)

type Fetcher struct {
	store    *Store
	renderer *Renderer
	cfg      Config
	client   *http.Client
}

type fetchProgressFn func(done, total int, result FetchResult)

func NewFetcher(store *Store, renderer *Renderer, cfg Config) *Fetcher {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,
	}

	return &Fetcher{
		store:    store,
		renderer: renderer,
		cfg:      cfg,
		client: &http.Client{
			Timeout:   cfg.HTTPTimeout,
			Transport: transport,
		},
	}
}

func (f *Fetcher) Fetch(ctx context.Context, feedID *int64) (FetchReport, error) {
	return f.FetchWithProgress(ctx, feedID, nil)
}

func (f *Fetcher) HTTPClient() *http.Client {
	return f.client
}

func (f *Fetcher) DiscoverFeedURL(ctx context.Context, rawURL string) (string, error) {
	return DiscoverFeedURL(ctx, f.client, gofeed.NewParser(), rawURL, f.cfg.UserAgent)
}

func (f *Fetcher) FetchWithProgress(ctx context.Context, feedID *int64, onResult fetchProgressFn) (FetchReport, error) {
	feeds, err := f.store.ListFeedsForFetch(ctx, feedID)
	if err != nil {
		return FetchReport{}, err
	}

	report := FetchReport{StartedAt: time.Now()}
	if len(feeds) == 0 {
		report.EndedAt = time.Now()
		return report, nil
	}

	results := f.fetchAll(ctx, feeds, onResult)
	sort.Slice(results, func(i, j int) bool { return results[i].FeedID < results[j].FeedID })
	report.Results = results

	if f.cfg.RetentionDays > 0 {
		pruned, pruneErr := f.store.PruneReadEntriesOlderThan(ctx, f.cfg.RetentionDays)
		if pruneErr != nil {
			report.Warnings = append(report.Warnings, fmt.Sprintf("prune failed: %v", pruneErr))
		} else if pruned > 0 {
			report.Warnings = append(report.Warnings, fmt.Sprintf("pruned %d old entries", pruned))
		}
	}

	report.EndedAt = time.Now()
	return report, nil
}

func (f *Fetcher) fetchAll(ctx context.Context, feeds []Feed, onResult fetchProgressFn) []FetchResult {
	results := make([]FetchResult, 0, len(feeds))
	total := len(feeds)
	if total == 1 {
		result := f.fetchSingle(ctx, feeds[0])
		if onResult != nil {
			onResult(1, 1, result)
		}
		return append(results, result)
	}

	concurrency := f.cfg.FetchConcurrency
	if concurrency < 1 {
		concurrency = 10
	}

	jobs := make(chan Feed)
	out := make(chan FetchResult, total)
	wg := sync.WaitGroup{}
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for feed := range jobs {
				out <- f.fetchSingle(ctx, feed)
			}
		}()
	}

	go func() {
		for _, feed := range feeds {
			jobs <- feed
		}
		close(jobs)
		wg.Wait()
		close(out)
	}()

	var done int64
	for result := range out {
		results = append(results, result)
		if onResult != nil {
			onResult(int(atomic.AddInt64(&done, 1)), total, result)
		}
	}
	return results
}

func (f *Fetcher) fetchSingle(ctx context.Context, feed Feed) FetchResult {
	result := FetchResult{
		FeedID:    feed.ID,
		FeedTitle: fallback(feed.Title, feed.URL),
		FeedURL:   feed.URL,
	}

	req, err := f.newFeedRequest(ctx, feed)
	if err != nil {
		return f.failFeed(ctx, feed.ID, result, err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return f.failFeed(ctx, feed.ID, result, err)
	}
	defer resp.Body.Close()

	etag, lastModified := mergeCacheHeaders(resp, feed)
	if resp.StatusCode == http.StatusNotModified {
		result.NotModified = true
		if err := f.store.UpdateFeedFetchSuccess(ctx, feed.ID, "", "", "", etag, lastModified, time.Now()); err != nil {
			return f.failFeed(ctx, feed.ID, result, err)
		}
		return result
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return f.failFeed(ctx, feed.ID, result, fmt.Errorf("http %d", resp.StatusCode))
	}

	parsed, err := parseFeedResponse(resp.Body)
	if err != nil {
		return f.failFeed(ctx, feed.ID, result, err)
	}

	newCount, updatedCount, err := f.storeFeedItems(ctx, feed.ID, parsed.Items)
	if err != nil {
		return f.failFeed(ctx, feed.ID, result, err)
	}
	result.NewEntries = newCount
	result.Updated = updatedCount

	if err := f.store.UpdateFeedFetchSuccess(ctx, feed.ID, strings.TrimSpace(parsed.Title), strings.TrimSpace(parsed.Link), strings.TrimSpace(parsed.Description), etag, lastModified, time.Now()); err != nil {
		return f.failFeed(ctx, feed.ID, result, err)
	}
	if parsed.Title != "" {
		result.FeedTitle = parsed.Title
	}
	return result
}

func (f *Fetcher) newFeedRequest(ctx context.Context, feed Feed) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feed.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", f.cfg.UserAgent)
	req.Header.Set("Accept", "application/xml, application/atom+xml, application/rss+xml, application/feed+json, text/xml, text/html, */*;q=0.8")
	if feed.ETag != "" {
		req.Header.Set("If-None-Match", feed.ETag)
	}
	if feed.LastModified != "" {
		req.Header.Set("If-Modified-Since", feed.LastModified)
	}
	return req, nil
}

func mergeCacheHeaders(resp *http.Response, feed Feed) (etag, lastModified string) {
	etag = strings.TrimSpace(resp.Header.Get("ETag"))
	if etag == "" {
		etag = feed.ETag
	}
	lastModified = strings.TrimSpace(resp.Header.Get("Last-Modified"))
	if lastModified == "" {
		lastModified = feed.LastModified
	}
	return etag, lastModified
}

func parseFeedResponse(body io.Reader) (*gofeed.Feed, error) {
	data, err := io.ReadAll(io.LimitReader(body, 16<<20))
	if err != nil {
		return nil, err
	}
	return gofeed.NewParser().Parse(bytes.NewReader(data))
}

func (f *Fetcher) storeFeedItems(ctx context.Context, feedID int64, items []*gofeed.Item) (newCount int, updatedCount int, err error) {
	for _, item := range items {
		guid := strings.TrimSpace(item.GUID)
		if guid == "" {
			guid = dedupGUID(item.Link, item.Title, item.PublishedParsed)
		}

		contentHTML := strings.TrimSpace(item.Content)
		if contentHTML == "" {
			contentHTML = strings.TrimSpace(item.Description)
		}
		contentHTML = SanitizeHTML(contentHTML)

		author := ""
		if item.Author != nil {
			author = strings.TrimSpace(item.Author.Name)
		}

		_, inserted, err := f.store.UpsertEntry(ctx, UpsertEntryInput{
			FeedID:       feedID,
			GUID:         guid,
			URL:          strings.TrimSpace(item.Link),
			ExternalURL:  "",
			Title:        strings.TrimSpace(item.Title),
			Summary:      summarize(item.Description, f.renderer),
			ContentHTML:  contentHTML,
			ContentMD:    f.renderer.HTMLToMarkdown(contentHTML),
			Author:       author,
			PublishedAt:  item.PublishedParsed,
			DateModified: item.UpdatedParsed,
		})
		if err != nil {
			return newCount, updatedCount, err
		}
		if inserted {
			newCount++
		} else {
			updatedCount++
		}
	}
	return newCount, updatedCount, nil
}

func (f *Fetcher) failFeed(ctx context.Context, feedID int64, result FetchResult, err error) FetchResult {
	result.Error = err.Error()
	if persistErr := f.store.SetFeedError(ctx, feedID, result.Error); persistErr != nil {
		result.Error = fmt.Sprintf("%s; additionally failed to persist feed error: %v", result.Error, persistErr)
	}
	return result
}

func summarize(raw string, renderer *Renderer) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = SanitizeHTML(raw)
	md := renderer.HTMLToMarkdown(raw)
	return compactText(md, 280)
}

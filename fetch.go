package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
)

type Fetcher struct {
	store    *Store
	renderer *Renderer
	cfg      Config
	client   *http.Client
}

func NewFetcher(store *Store, renderer *Renderer, cfg Config) *Fetcher {
	client := &http.Client{Timeout: cfg.HTTPTimeout}
	return &Fetcher{
		store:    store,
		renderer: renderer,
		cfg:      cfg,
		client:   client,
	}
}

func (f *Fetcher) Fetch(ctx context.Context, feedID *int64) (FetchReport, error) {
	feeds, err := f.store.ListFeedsForFetch(ctx, feedID)
	if err != nil {
		return FetchReport{}, err
	}
	report := FetchReport{StartedAt: time.Now()}
	if len(feeds) == 0 {
		report.EndedAt = time.Now()
		return report, nil
	}

	results := make([]FetchResult, 0, len(feeds))
	if len(feeds) == 1 {
		res := f.fetchSingle(ctx, feeds[0])
		results = append(results, res)
	} else {
		concurrency := f.cfg.FetchConcurrency
		if concurrency < 1 {
			concurrency = 10
		}
		jobs := make(chan Feed)
		out := make(chan FetchResult, len(feeds))
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

		for r := range out {
			results = append(results, r)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].FeedID < results[j].FeedID
	})
	report.Results = results

	if f.cfg.RetentionDays > 0 {
		_, _ = f.store.PruneReadEntriesOlderThan(ctx, f.cfg.RetentionDays)
	}

	report.EndedAt = time.Now()
	return report, nil
}

func (f *Fetcher) fetchSingle(ctx context.Context, feed Feed) FetchResult {
	result := FetchResult{
		FeedID:    feed.ID,
		FeedTitle: fallback(feed.Title, feed.URL),
		FeedURL:   feed.URL,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feed.URL, nil)
	if err != nil {
		result.Error = err.Error()
		_ = f.store.SetFeedError(ctx, feed.ID, result.Error)
		return result
	}
	req.Header.Set("User-Agent", f.cfg.UserAgent)
	if feed.ETag != "" {
		req.Header.Set("If-None-Match", feed.ETag)
	}
	if feed.LastModified != "" {
		req.Header.Set("If-Modified-Since", feed.LastModified)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		result.Error = err.Error()
		_ = f.store.SetFeedError(ctx, feed.ID, result.Error)
		return result
	}
	defer resp.Body.Close()

	etag := strings.TrimSpace(resp.Header.Get("ETag"))
	if etag == "" {
		etag = feed.ETag
	}
	lastModified := strings.TrimSpace(resp.Header.Get("Last-Modified"))
	if lastModified == "" {
		lastModified = feed.LastModified
	}

	if resp.StatusCode == http.StatusNotModified {
		result.NotModified = true
		if err := f.store.UpdateFeedFetchSuccess(ctx, feed.ID, "", "", "", etag, lastModified, time.Now()); err != nil {
			result.Error = err.Error()
			_ = f.store.SetFeedError(ctx, feed.ID, result.Error)
		}
		return result
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.Error = fmt.Sprintf("http %d", resp.StatusCode)
		_ = f.store.SetFeedError(ctx, feed.ID, result.Error)
		return result
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		result.Error = err.Error()
		_ = f.store.SetFeedError(ctx, feed.ID, result.Error)
		return result
	}

	parser := gofeed.NewParser()
	parsed, err := parser.Parse(bytes.NewReader(body))
	if err != nil {
		result.Error = err.Error()
		_ = f.store.SetFeedError(ctx, feed.ID, result.Error)
		return result
	}

	for _, item := range parsed.Items {
		published := item.PublishedParsed
		modified := item.UpdatedParsed
		guid := strings.TrimSpace(item.GUID)
		if guid == "" {
			guid = dedupGUID(item.Link, item.Title, published)
		}

		contentHTML := strings.TrimSpace(item.Content)
		if contentHTML == "" {
			contentHTML = strings.TrimSpace(item.Description)
		}
		contentMD := f.renderer.HTMLToMarkdown(contentHTML)
		summary := summarize(item.Description, f.renderer)
		author := ""
		if item.Author != nil {
			author = strings.TrimSpace(item.Author.Name)
		}

		_, inserted, upsertErr := f.store.UpsertEntry(ctx, UpsertEntryInput{
			FeedID:       feed.ID,
			GUID:         guid,
			URL:          strings.TrimSpace(item.Link),
			ExternalURL:  "",
			Title:        strings.TrimSpace(item.Title),
			Summary:      summary,
			ContentHTML:  contentHTML,
			ContentMD:    contentMD,
			Author:       author,
			PublishedAt:  published,
			DateModified: modified,
		})
		if upsertErr != nil {
			result.Error = upsertErr.Error()
			_ = f.store.SetFeedError(ctx, feed.ID, result.Error)
			return result
		}
		if inserted {
			result.NewEntries++
		} else {
			result.Updated++
		}
	}

	title := strings.TrimSpace(parsed.Title)
	siteURL := strings.TrimSpace(parsed.Link)
	description := strings.TrimSpace(parsed.Description)
	if err := f.store.UpdateFeedFetchSuccess(ctx, feed.ID, title, siteURL, description, etag, lastModified, time.Now()); err != nil {
		result.Error = err.Error()
		_ = f.store.SetFeedError(ctx, feed.ID, result.Error)
		return result
	}

	if title != "" {
		result.FeedTitle = title
	}
	return result
}

func summarize(raw string, renderer *Renderer) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	md := renderer.HTMLToMarkdown(raw)
	return compactText(md, 280)
}

func fallback(v, fb string) string {
	if strings.TrimSpace(v) == "" {
		return fb
	}
	return v
}

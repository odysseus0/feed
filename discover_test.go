package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mmcdole/gofeed"
)

func TestDiscoverFeedURL_DirectFeedViaRedirect(t *testing.T) {
	const feedXML = `<?xml version="1.0"?><rss version="2.0"><channel><title>T</title><link>https://example.com</link><item><title>A</title><link>https://example.com/a</link></item></channel></rss>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start":
			http.Redirect(w, r, "/feed.xml", http.StatusMovedPermanently)
		case "/feed.xml":
			w.Header().Set("Content-Type", "application/rss+xml")
			_, _ = w.Write([]byte(feedXML))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	got, err := DiscoverFeedURL(context.Background(), srv.Client(), gofeed.NewParser(), srv.URL+"/start", "feed-test/1.0")
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	want := srv.URL + "/feed.xml"
	if got != want {
		t.Fatalf("unexpected discovered url: got %q want %q", got, want)
	}
}

func TestDiscoverFeedURL_FromHTMLAlternateWithBase(t *testing.T) {
	const page = `<!doctype html><html><head><base href="https://example.org/blog/"><link rel="alternate" type="application/rss+xml" href="feed.xml"></head></html>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(page))
	}))
	defer srv.Close()

	got, err := DiscoverFeedURL(context.Background(), srv.Client(), gofeed.NewParser(), srv.URL, "feed-test/1.0")
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if got != "https://example.org/blog/feed.xml" {
		t.Fatalf("unexpected discovered url: %q", got)
	}
}

func TestDiscoverFeedCandidates_IgnoresWpJSONAndDedupes(t *testing.T) {
	html := `
<!doctype html>
<html><head>
<link rel="alternate" type="application/json" href="https://example.org/wp-json/wp/v2/posts/1">
<link rel="alternate stylesheet" type="application/rss+xml" href="/feed.xml">
<link rel="alternate" type="application/rss+xml" href="/feed.xml">
</head></html>`

	parsedBase := mustParseURL(t, "https://example.org/page")
	urls := discoverFeedCandidates([]byte(html), parsedBase)
	if len(urls) != 1 {
		t.Fatalf("expected 1 candidate, got %d (%v)", len(urls), urls)
	}
	if urls[0] != "https://example.org/feed.xml" {
		t.Fatalf("unexpected candidate %q", urls[0])
	}
}

func TestDiscoverFeedURL_NoFeedFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><head><title>x</title></head><body>none</body></html>"))
	}))
	defer srv.Close()

	_, err := DiscoverFeedURL(context.Background(), srv.Client(), gofeed.NewParser(), srv.URL, "feed-test/1.0")
	if err == nil {
		t.Fatalf("expected discovery error")
	}
	if !strings.Contains(err.Error(), "no feed discovered") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDiscoverFeedCandidates_TableDriven(t *testing.T) {
	tests := []struct {
		name    string
		html    string
		baseURL string
		want    []string
	}{
		{
			name: "rss absolute",
			html: `<!doctype html><html><head><link rel="alternate" type="application/rss+xml" href="https://example.org/rss"></head></html>`,
			want: []string{"https://example.org/rss"},
		},
		{
			name: "atom absolute",
			html: `<!doctype html><html><head><link rel="alternate" type="application/atom+xml" href="https://example.org/atom.xml"></head></html>`,
			want: []string{"https://example.org/atom.xml"},
		},
		{
			name: "json old mime accepted",
			html: `<!doctype html><html><head><link rel="alternate" type="application/json" href="https://example.org/feed.json"></head></html>`,
			want: []string{"https://example.org/feed.json"},
		},
		{
			name: "wp-json ignored",
			html: `<!doctype html><html><head><link rel="alternate" type="application/json" href="https://example.org/wp-json/wp/v2/posts/123"></head></html>`,
			want: nil,
		},
		{
			name: "relative href resolved",
			html: `<!doctype html><html><head><link rel="alternate" type="application/feed+json" href="/feed.json"></head></html>`,
			want: []string{"https://example.org/feed.json"},
		},
		{
			name: "alternate token in rel list",
			html: `<!doctype html><html><head><link rel="alternate stylesheet" type="application/rss+xml" href="/rss.xml"></head></html>`,
			want: []string{"https://example.org/rss.xml"},
		},
		{
			name: "dedupe same feed",
			html: `<!doctype html><html><head><link rel="alternate" type="application/rss+xml" href="/feed.xml"><link rel="alternate" type="application/rss+xml" href="/feed.xml"></head></html>`,
			want: []string{"https://example.org/feed.xml"},
		},
		{
			name: "no href no result",
			html: `<!doctype html><html><head><link rel="alternate" type="application/rss+xml"></head></html>`,
			want: nil,
		},
		{
			name:    "type omitted but feedish href",
			html:    `<!doctype html><html><head><link rel="alternate" href="/feed"></head></html>`,
			want:    []string{"https://example.org/feed"},
			baseURL: "https://example.org/blog/post",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := tt.baseURL
			if base == "" {
				base = "https://example.org/page"
			}
			urls := discoverFeedCandidates([]byte(tt.html), mustParseURL(t, base))
			if len(urls) != len(tt.want) {
				t.Fatalf("unexpected result count: got %d want %d (%v)", len(urls), len(tt.want), urls)
			}
			for i := range tt.want {
				if urls[i] != tt.want[i] {
					t.Fatalf("unexpected candidate at %d: got %q want %q", i, urls[i], tt.want[i])
				}
			}
		})
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	return u
}

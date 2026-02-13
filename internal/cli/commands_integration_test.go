package cli

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/odysseus0/feed/internal/config"
	"github.com/odysseus0/feed/internal/store"
)

func runCLI(t *testing.T, cfgPath string, args ...string) {
	t.Helper()
	cmd := NewRootCmd(testConfig(cfgPath))
	cmd.SetArgs(append([]string{"--db", cfgPath}, args...))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed (%v): %v", args, err)
	}
}

func TestCLICommandFlowSmoke(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "feed.db")

	const feedXML = `<?xml version="1.0"?>
<rss version="2.0"><channel>
<title>Test Feed</title><link>https://example.com</link><description>desc</description>
<item>
  <guid>item-1</guid>
  <title>Entry One</title>
  <link>https://example.com/entry-1</link>
  <description>hello world</description>
</item>
</channel></rss>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			_, _ = w.Write([]byte(`<html><head><link rel="alternate" type="application/rss+xml" href="/feed.xml"></head></html>`))
		case "/feed.xml":
			w.Header().Set("Content-Type", "application/rss+xml")
			_, _ = w.Write([]byte(feedXML))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	runCLI(t, dbPath, "add", "feed", srv.URL)
	runCLI(t, dbPath, "get", "feeds")
	runCLI(t, dbPath, "get", "entries", "--status=all", "--no-fetch")
	runCLI(t, dbPath, "search", "Entry")
	runCLI(t, dbPath, "fetch")

	db, err := store.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	s := store.NewStore(db)
	entries, err := s.ListEntries(context.Background(), EntryListOptions{Status: "all", Limit: 10})
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected at least one entry")
	}
	entryID := entries[0].ID
	_ = db.Close()

	runCLI(t, dbPath, "get", "entry", fmt.Sprintf("%d", entryID))
	runCLI(t, dbPath, "update", "entry", fmt.Sprintf("%d", entryID), "--read")
	runCLI(t, dbPath, "update", "entry", fmt.Sprintf("%d", entryID), "--unread")
	runCLI(t, dbPath, "update", "entries", fmt.Sprintf("%d", entryID), "--starred")
	runCLI(t, dbPath, "update", "entries", fmt.Sprintf("%d", entryID), "--unstarred")

	opmlFile := filepath.Join(t.TempDir(), "in.opml")
	if err := os.WriteFile(opmlFile, []byte(`<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline text="A" xmlUrl="`+srv.URL+`/feed.xml" />
    <outline text="Bad" xmlUrl="  " />
  </body>
</opml>`), 0o644); err != nil {
		t.Fatalf("write opml: %v", err)
	}
	runCLI(t, dbPath, "import", opmlFile, "-o", "wide")
	runCLI(t, dbPath, "export")
}

func TestExecuteHelpPath(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"feed", "--help"}
	t.Cleanup(func() {
		os.Args = oldArgs
	})
	if code := run(); code != 0 {
		t.Fatalf("expected exit code 0 for --help, got %d", code)
	}
}

func run() int {
	err := Execute()
	PrintError(err)
	return ErrorExitCode(err)
}

func testConfig(dbPath string) config.Config {
	return config.Config{
		DBPath:           dbPath,
		StaleAfter:       30 * time.Minute,
		FetchConcurrency: 2,
		HTTPTimeout:      10 * time.Second,
		UserAgent:        "feed-test/1.0",
	}
}

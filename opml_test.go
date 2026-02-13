package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadOPML_ParsesNestedAndDedupes(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "in.opml")
	content := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <body>
    <outline text="Top">
      <outline text="A" xmlUrl="https://a.example/feed.xml" />
      <outline text="B" xmlurl="https://b.example/feed.xml" />
      <outline text="A-dup" xmlUrl="https://a.example/feed.xml" />
    </outline>
  </body>
</opml>`
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		t.Fatalf("write opml: %v", err)
	}

	urls, err := ReadOPML(tmp)
	if err != nil {
		t.Fatalf("read opml: %v", err)
	}
	if len(urls) != 2 {
		t.Fatalf("expected 2 unique urls, got %d (%v)", len(urls), urls)
	}
}

func TestWriteOPML_ContainsFeedURLs(t *testing.T) {
	var b strings.Builder
	feeds := []Feed{
		{Title: "A", URL: "https://a.example/feed.xml", SiteURL: "https://a.example"},
		{Title: "B", URL: "https://b.example/feed.xml", SiteURL: "https://b.example"},
	}
	if err := WriteOPML(&b, feeds); err != nil {
		t.Fatalf("write opml: %v", err)
	}
	out := b.String()
	for _, want := range []string{"xmlUrl=\"https://a.example/feed.xml\"", "xmlUrl=\"https://b.example/feed.xml\""} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output", want)
		}
	}
}

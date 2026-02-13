package main

import (
	"strings"
	"testing"
	"time"
)

func TestParseID(t *testing.T) {
	id, err := parseID("42")
	if err != nil {
		t.Fatalf("parseID: %v", err)
	}
	if id != 42 {
		t.Fatalf("unexpected id: %d", id)
	}
	if _, err := parseID("0"); err == nil {
		t.Fatalf("expected error for zero")
	}
}

func TestDedupGUIDFallbacks(t *testing.T) {
	now := time.Date(2026, 2, 13, 0, 0, 0, 0, time.UTC)
	if got := dedupGUID(" https://example.com/a ", "x", &now); got != "https://example.com/a" {
		t.Fatalf("expected link guid, got %q", got)
	}
	got := dedupGUID("", "title", &now)
	if !strings.HasPrefix(got, "sha1:") {
		t.Fatalf("expected sha1 fallback, got %q", got)
	}

	got2 := dedupGUID("", "title", &now)
	if got != got2 {
		t.Fatalf("expected deterministic hash fallback, got %q and %q", got, got2)
	}

	later := now.Add(1 * time.Hour)
	got3 := dedupGUID("", "title", &later)
	if got3 == got {
		t.Fatalf("expected time component to affect fallback hash")
	}
}

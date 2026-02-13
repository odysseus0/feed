package cli

import (
	"testing"
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

func TestFallback(t *testing.T) {
	if got := fallback("value", "x"); got != "value" {
		t.Fatalf("fallback non-empty: %q", got)
	}
	if got := fallback("   ", "x"); got != "x" {
		t.Fatalf("fallback empty: %q", got)
	}
}

package main

import (
	"os"
	"testing"
)

func TestRunHelp(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"feed", "--help"}
	t.Cleanup(func() {
		os.Args = oldArgs
	})
	if code := run(); code != 0 {
		t.Fatalf("run() code = %d, want 0", code)
	}
}

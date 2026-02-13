package cli

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/odysseus0/feed/internal/model"
	"github.com/odysseus0/feed/internal/store"
)

func seedEntry(t *testing.T, dbPath string) int64 {
	t.Helper()
	db, err := store.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	s := store.NewStore(db)
	feed, _, err := s.CreateFeed(context.Background(), "https://example.com/feed.xml")
	if err != nil {
		t.Fatalf("create feed: %v", err)
	}
	id, _, err := s.UpsertEntry(context.Background(), model.UpsertEntryInput{
		FeedID: feed.ID,
		GUID:   "entry-guid-1",
		Title:  "Entry",
	})
	if err != nil {
		t.Fatalf("upsert entry: %v", err)
	}
	return id
}

func loadEntry(t *testing.T, dbPath string, entryID int64) model.Entry {
	t.Helper()
	db, err := store.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	s := store.NewStore(db)
	e, err := s.GetEntry(context.Background(), entryID)
	if err != nil {
		t.Fatalf("get entry: %v", err)
	}
	return e
}

func TestUpdateEntrySupportsExplicitStarSetAndUnset(t *testing.T) {
	dbPath := t.TempDir() + "/feed.db"
	entryID := seedEntry(t, dbPath)
	idArg := fmt.Sprintf("%d", entryID)

	root := NewRootCmd(testConfig(dbPath))
	root.SetArgs([]string{"--db", dbPath, "update", "entry", idArg, "--starred"})
	if err := root.Execute(); err != nil {
		t.Fatalf("set starred: %v", err)
	}
	e := loadEntry(t, dbPath, entryID)
	if !e.Starred {
		t.Fatalf("expected starred=true after --starred")
	}

	root = NewRootCmd(testConfig(dbPath))
	root.SetArgs([]string{"--db", dbPath, "update", "entry", idArg, "--unstarred"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unset starred: %v", err)
	}
	e = loadEntry(t, dbPath, entryID)
	if e.Starred {
		t.Fatalf("expected starred=false after --unstarred")
	}
}

func TestUpdateEntriesSupportsUnreadAndUnstarred(t *testing.T) {
	dbPath := t.TempDir() + "/feed.db"
	entryID := seedEntry(t, dbPath)
	idArg := fmt.Sprintf("%d", entryID)

	db, err := store.OpenDB(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	s := store.NewStore(db)
	if err := s.UpdateEntryRead(context.Background(), entryID, true); err != nil {
		t.Fatalf("mark read seed: %v", err)
	}
	if err := s.SetEntriesStarred(context.Background(), []int64{entryID}, true); err != nil {
		t.Fatalf("mark starred seed: %v", err)
	}
	_ = db.Close()

	root := NewRootCmd(testConfig(dbPath))
	root.SetArgs([]string{"--db", dbPath, "update", "entries", idArg, "--unread"})
	if err := root.Execute(); err != nil {
		t.Fatalf("mark unread: %v", err)
	}
	root = NewRootCmd(testConfig(dbPath))
	root.SetArgs([]string{"--db", dbPath, "update", "entries", idArg, "--unstarred"})
	if err := root.Execute(); err != nil {
		t.Fatalf("mark unstarred: %v", err)
	}

	e := loadEntry(t, dbPath, entryID)
	if e.Read {
		t.Fatalf("expected read=false after --unread")
	}
	if e.Starred {
		t.Fatalf("expected starred=false after --unstarred")
	}
}

func TestNotFoundMapsToStableErrorCodeAndMessage(t *testing.T) {
	dbPath := t.TempDir() + "/feed.db"

	root := NewRootCmd(testConfig(dbPath))
	root.SetArgs([]string{"--db", dbPath, "remove", "feed", "999"})
	err := root.Execute()
	if err == nil {
		t.Fatalf("expected error")
	}
	if code := ErrorExitCode(err); code != 3 {
		t.Fatalf("expected exit code 3, got %d", code)
	}
	formatted := FormatError(err)
	if !strings.Contains(formatted, "[not-found]") {
		t.Fatalf("unexpected formatted error: %s", formatted)
	}
}

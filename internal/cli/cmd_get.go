package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/odysseus0/feed/internal/store"
)

func newGetCmd(getApp func() *App, getOutput func() OutputFormat) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get feeds, entries, and stats",
	}

	cmd.AddCommand(newGetEntriesCmd(getApp, getOutput))
	cmd.AddCommand(newGetEntryCmd(getApp, getOutput))
	cmd.AddCommand(newGetFeedsCmd(getApp, getOutput))
	cmd.AddCommand(newGetStatsCmd(getApp, getOutput))
	return cmd
}

func newGetEntriesCmd(getApp func() *App, getOutput func() OutputFormat) *cobra.Command {
	var status string
	var feedID int64
	var limit int
	var noFetch bool

	cmd := &cobra.Command{
		Use:   "entries",
		Short: "List entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(getApp)
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			if !noFetch {
				hasFeeds, stale, lastFetched, err := app.store.GetFetchStaleness(ctx, app.cfg.StaleAfter)
				if err != nil {
					return fmt.Errorf("check fetch staleness: %w", err)
				}
				if hasFeeds && stale {
					msg := "Fetching feeds"
					if lastFetched != nil {
						msg += fmt.Sprintf(" (last fetch: %s)", humanAgo(lastFetched))
					} else {
						msg += " (last fetch: never)"
					}
					fmt.Fprintln(os.Stderr, msg+"...")
					rep, err := app.fetcher.Fetch(ctx, nil)
					if err != nil {
						return fmt.Errorf("fetch feeds: %w", err)
					}
					for _, warning := range rep.Warnings {
						fmt.Fprintf(os.Stderr, "warning: %s\n", warning)
					}
					errCount := 0
					for _, r := range rep.Results {
						if strings.TrimSpace(r.Error) != "" {
							errCount++
						}
					}
					if errCount > 0 {
						fmt.Fprintf(os.Stderr, "Fetch completed with %d error(s). Run `feed fetch -o wide` for details.\n", errCount)
					}
				}
			}

			entries, err := app.store.ListEntries(ctx, EntryListOptions{
				Status: status,
				FeedID: feedID,
				Limit:  limit,
			})
			if err != nil {
				return fmt.Errorf("list entries: %w", err)
			}

			switch getOutput() {
			case OutputJSON:
				return writeJSON(os.Stdout, entries)
			case OutputWide:
				writeEntriesTable(os.Stdout, entries, true)
			default:
				writeEntriesTable(os.Stdout, entries, false)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&status, "status", "unread", "Entry status: unread, read, all")
	cmd.Flags().Int64Var(&feedID, "feed", 0, "Filter by feed ID")
	cmd.Flags().IntVar(&limit, "limit", 50, "Result limit")
	cmd.Flags().BoolVar(&noFetch, "no-fetch", false, "Skip staleness auto-fetch")
	return cmd
}

func newGetEntryCmd(getApp func() *App, getOutput func() OutputFormat) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "entry <id>",
		Short: "Get full entry content",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(getApp)
			if err != nil {
				return err
			}
			id, err := parseID(args[0])
			if err != nil {
				return fmt.Errorf("%w: %v", store.ErrInvalidInput, err)
			}
			entry, err := app.store.GetEntry(cmd.Context(), id)
			if err != nil {
				return fmt.Errorf("get entry: %w", err)
			}

			if getOutput() == OutputJSON {
				return writeJSON(os.Stdout, entry)
			}

			title := strings.TrimSpace(entry.Title)
			if title == "" {
				title = fallback(entry.URL, "(untitled)")
			}
			date := formatDate(entry.PublishedAt)
			url := fallback(entry.URL, "-")
			fmt.Fprintf(os.Stdout, "# %s\n", title)
			fmt.Fprintf(os.Stdout, "source: %s | date: %s | url: %s\n\n", entry.FeedTitle, date, url)

			content := strings.TrimSpace(entry.ContentMD)
			if content == "" {
				content = strings.TrimSpace(entry.Summary)
			}
			if content == "" {
				content = url
			}
			fmt.Fprintln(os.Stdout, content)
			return nil
		},
	}
	return cmd
}

func newGetFeedsCmd(getApp func() *App, getOutput func() OutputFormat) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feeds",
		Short: "List subscribed feeds",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(getApp)
			if err != nil {
				return err
			}
			feeds, err := app.store.ListFeedsWithCounts(cmd.Context())
			if err != nil {
				return fmt.Errorf("list feeds: %w", err)
			}
			switch getOutput() {
			case OutputJSON:
				return writeJSON(os.Stdout, feeds)
			case OutputWide:
				writeFeedsTable(os.Stdout, feeds, true)
			default:
				writeFeedsTable(os.Stdout, feeds, false)
			}
			return nil
		},
	}
	return cmd
}

func newGetStatsCmd(getApp func() *App, getOutput func() OutputFormat) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Get aggregate stats",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(getApp)
			if err != nil {
				return err
			}
			stats, err := app.store.GetStats(cmd.Context())
			if err != nil {
				return fmt.Errorf("get stats: %w", err)
			}
			if getOutput() == OutputJSON {
				return writeJSON(os.Stdout, stats)
			}
			writeStatsTable(os.Stdout, stats)
			return nil
		},
	}
	return cmd
}

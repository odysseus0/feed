package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tengjizhang/feed/internal/store"
)

func newFetchCmd(getApp func() *App, getOutput func() OutputFormat) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fetch [id]",
		Short: "Fetch all feeds or one feed by ID",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(getApp)
			if err != nil {
				return err
			}
			var id *int64
			if len(args) == 1 {
				v, err := parseID(args[0])
				if err != nil {
					return fmt.Errorf("%w: %v", store.ErrInvalidInput, err)
				}
				id = &v
			}

			rep, err := app.fetcher.FetchWithProgress(cmd.Context(), id, func(done, total int, result FetchResult) {
				label := fallback(result.FeedTitle, result.FeedURL)
				if result.Error != "" {
					fmt.Fprintf(os.Stderr, "[%d/%d] %s -> error: %s\n", done, total, label, result.Error)
					return
				}
				if result.NotModified {
					fmt.Fprintf(os.Stderr, "[%d/%d] %s -> not modified\n", done, total, label)
					return
				}
				fmt.Fprintf(os.Stderr, "[%d/%d] %s -> %d new, %d updated\n", done, total, label, result.NewEntries, result.Updated)
			})
			if err != nil {
				return fmt.Errorf("fetch feeds: %w", err)
			}
			for _, warning := range rep.Warnings {
				fmt.Fprintf(os.Stderr, "warning: %s\n", warning)
			}
			switch getOutput() {
			case OutputJSON:
				return writeJSON(os.Stdout, rep)
			case OutputWide:
				writeFetchReportTable(os.Stdout, rep)
			default:
				writeFetchReportTable(os.Stdout, rep)
			}
			return nil
		},
	}
	return cmd
}

func newSearchCmd(getApp func() *App, getOutput func() OutputFormat) *cobra.Command {
	var feedID int64
	var limit int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search entries with full-text search",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(getApp)
			if err != nil {
				return err
			}
			entries, err := app.store.SearchEntries(cmd.Context(), SearchOptions{
				Query: args[0],
				Feed:  feedID,
				Limit: limit,
			})
			if err != nil {
				return fmt.Errorf("search entries: %w", err)
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
	cmd.Flags().Int64Var(&feedID, "feed", 0, "Filter by feed ID")
	cmd.Flags().IntVar(&limit, "limit", 50, "Result limit")
	return cmd
}

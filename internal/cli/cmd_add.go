package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newAddCmd(getApp func() *App, getOutput func() OutputFormat) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add resources",
	}
	cmd.AddCommand(newAddFeedCmd(getApp, getOutput))
	return cmd
}

func newAddFeedCmd(getApp func() *App, getOutput func() OutputFormat) *cobra.Command {
	return &cobra.Command{
		Use:   "feed <url>",
		Short: "Add a feed URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(getApp)
			if err != nil {
				return err
			}

			discovered, err := app.fetcher.DiscoverFeedURL(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("discover feed url: %w", err)
			}
			if discovered != args[0] {
				fmt.Fprintf(os.Stderr, "Discovered feed URL: %s\n", discovered)
			}

			feed, inserted, err := app.store.CreateFeed(cmd.Context(), discovered)
			if err != nil {
				return fmt.Errorf("create feed: %w", err)
			}
			report, err := app.fetcher.Fetch(cmd.Context(), &feed.ID)
			if err != nil {
				return fmt.Errorf("initial fetch: %w", err)
			}

			feed, _ = app.store.GetFeedByID(cmd.Context(), feed.ID)
			if getOutput() == OutputJSON {
				return writeJSON(os.Stdout, AddFeedResponse{
					Feed:          feed,
					Inserted:      inserted,
					DiscoveredURL: discovered,
					FetchReport:   report,
				})
			}

			if inserted {
				fmt.Fprintf(os.Stdout, "Added feed %d: %s\n", feed.ID, fallback(feed.Title, feed.URL))
			} else {
				fmt.Fprintf(os.Stdout, "Skipped existing feed (%d): %s\n", feed.ID, fallback(feed.Title, feed.URL))
			}
			if len(report.Results) > 0 {
				result := report.Results[0]
				if result.Error != "" {
					fmt.Fprintf(os.Stderr, "Initial fetch failed: %s\n", result.Error)
				} else {
					fmt.Fprintf(os.Stdout, "Fetched: %d new, %d updated\n", result.NewEntries, result.Updated)
				}
			}
			return nil
		},
	}
}

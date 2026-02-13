package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type ImportResult struct {
	InputURL      string `json:"input_url"`
	NormalizedURL string `json:"normalized_url,omitempty"`
	FeedID        int64  `json:"feed_id,omitempty"`
	Added         bool   `json:"added"`
	Error         string `json:"error,omitempty"`
}

type ImportReport struct {
	File     string         `json:"file"`
	Total    int            `json:"total"`
	Added    int            `json:"added"`
	Existing int            `json:"existing"`
	Failed   int            `json:"failed"`
	Results  []ImportResult `json:"results"`
}

func newImportCmd(getApp func() *App, getOutput func() OutputFormat) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <file.opml>",
		Short: "Import feeds from OPML",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(getApp)
			if err != nil {
				return err
			}
			urls, err := ReadOPML(args[0])
			if err != nil {
				return err
			}
			report := ImportReport{File: args[0], Total: len(urls), Results: make([]ImportResult, 0, len(urls))}

			for _, u := range urls {
				item := ImportResult{InputURL: u}
				normalized, normalizeErr := normalizeURL(u)
				if normalizeErr != nil {
					item.Error = normalizeErr.Error()
					report.Failed++
					report.Results = append(report.Results, item)
					continue
				}
				item.NormalizedURL = normalized

				feed, added, createErr := app.store.CreateFeed(cmd.Context(), normalized)
				if createErr != nil {
					item.Error = createErr.Error()
					report.Failed++
					report.Results = append(report.Results, item)
					continue
				}
				item.Added = added
				item.FeedID = feed.ID
				if added {
					report.Added++
				} else {
					report.Existing++
				}

				report.Results = append(report.Results, item)
			}

			if getOutput() == OutputJSON {
				return writeJSON(os.Stdout, report)
			}

			fmt.Fprintf(os.Stdout, "Imported %d feeds from %s\n", report.Total, report.File)
			fmt.Fprintf(os.Stdout, "Added: %d, Existing: %d, Failed: %d\n", report.Added, report.Existing, report.Failed)
			if getOutput() == OutputWide {
				for _, r := range report.Results {
					if r.Error != "" {
						fmt.Fprintf(os.Stdout, "- %s -> error: %s\n", r.InputURL, r.Error)
					}
				}
			}
			return nil
		},
	}
	return cmd
}

func newExportCmd(getApp func() *App, getOutput func() OutputFormat) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export feeds as OPML to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(getApp)
			if err != nil {
				return err
			}
			feeds, err := app.store.ListFeedURLs(cmd.Context())
			if err != nil {
				return err
			}
			return WriteOPML(os.Stdout, feeds)
		},
	}
	return cmd
}

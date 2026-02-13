package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newRemoveCmd(getApp func() *App, getOutput func() OutputFormat) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove resources",
	}
	cmd.AddCommand(newRemoveFeedCmd(getApp, getOutput))
	return cmd
}

func newRemoveFeedCmd(getApp func() *App, getOutput func() OutputFormat) *cobra.Command {
	return &cobra.Command{
		Use:   "feed <id>",
		Short: "Remove a feed by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(getApp)
			if err != nil {
				return err
			}

			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			if err := app.store.DeleteFeed(cmd.Context(), id); err != nil {
				return err
			}
			if getOutput() == OutputJSON {
				return writeJSON(os.Stdout, RemoveFeedResponse{RemovedFeedID: id})
			}
			fmt.Fprintf(os.Stdout, "Removed feed %d\n", id)
			return nil
		},
	}
}

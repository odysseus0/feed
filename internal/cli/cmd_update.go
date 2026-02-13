package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/odysseus0/feed/internal/store"
)

func newUpdateCmd(getApp func() *App, getOutput func() OutputFormat) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update resources",
	}
	cmd.AddCommand(newUpdateEntryCmd(getApp, getOutput))
	cmd.AddCommand(newUpdateEntriesCmd(getApp, getOutput))
	return cmd
}

func newUpdateEntryCmd(getApp func() *App, getOutput func() OutputFormat) *cobra.Command {
	var markRead bool
	var markUnread bool
	var markStarred bool
	var markUnstarred bool
	var toggleStar bool

	cmd := &cobra.Command{
		Use:   "entry <id>",
		Short: "Update one entry status",
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
			selected := 0
			if markRead {
				selected++
			}
			if markUnread {
				selected++
			}
			if markStarred {
				selected++
			}
			if markUnstarred {
				selected++
			}
			if toggleStar {
				selected++
			}
			if selected != 1 {
				return fmt.Errorf("%w: choose exactly one of --read, --unread, --starred, --unstarred, --toggle-starred", store.ErrInvalidInput)
			}

			resp := UpdateEntryResponse{EntryID: id}
			switch {
			case markRead:
				err = app.store.UpdateEntryRead(cmd.Context(), id, true)
				v := true
				resp.Read = &v
			case markUnread:
				err = app.store.UpdateEntryRead(cmd.Context(), id, false)
				v := false
				resp.Read = &v
			case markStarred:
				err = app.store.SetEntriesStarred(cmd.Context(), []int64{id}, true)
				v := true
				resp.Starred = &v
			case markUnstarred:
				err = app.store.SetEntriesStarred(cmd.Context(), []int64{id}, false)
				v := false
				resp.Starred = &v
			case toggleStar:
				var starred bool
				starred, err = app.store.ToggleEntryStarred(cmd.Context(), id)
				resp.Starred = &starred
				fmt.Fprintln(os.Stderr, "warning: --toggle-starred is deprecated; use --starred or --unstarred")
			}
			if err != nil {
				return fmt.Errorf("update entry: %w", err)
			}

			if getOutput() == OutputJSON {
				return writeJSON(os.Stdout, resp)
			}
			if resp.Read != nil {
				fmt.Fprintf(os.Stdout, "Entry %d read=%v\n", id, *resp.Read)
			} else if resp.Starred != nil {
				fmt.Fprintf(os.Stdout, "Entry %d starred=%v\n", id, *resp.Starred)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&markRead, "read", false, "Mark entry as read")
	cmd.Flags().BoolVar(&markUnread, "unread", false, "Mark entry as unread")
	cmd.Flags().BoolVar(&markStarred, "starred", false, "Mark entry as starred")
	cmd.Flags().BoolVar(&markUnstarred, "unstarred", false, "Mark entry as unstarred")
	cmd.Flags().BoolVar(&toggleStar, "toggle-starred", false, "Toggle starred (deprecated)")
	return cmd
}

func newUpdateEntriesCmd(getApp func() *App, getOutput func() OutputFormat) *cobra.Command {
	var markRead bool
	var markUnread bool
	var markStarred bool
	var markUnstarred bool

	cmd := &cobra.Command{
		Use:   "entries [id] [id...]",
		Short: "Batch update entries",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := requireApp(getApp)
			if err != nil {
				return err
			}
			selected := 0
			if markRead {
				selected++
			}
			if markUnread {
				selected++
			}
			if markStarred {
				selected++
			}
			if markUnstarred {
				selected++
			}
			if selected != 1 {
				return fmt.Errorf("%w: choose exactly one of --read, --unread, --starred, --unstarred", store.ErrInvalidInput)
			}

			ids, err := parseIDs(args)
			if err != nil {
				return fmt.Errorf("%w: %v", store.ErrInvalidInput, err)
			}

			if markRead {
				if err := app.store.SetEntriesRead(cmd.Context(), ids, true); err != nil {
					return fmt.Errorf("update entries read: %w", err)
				}
				if getOutput() == OutputJSON {
					v := true
					return writeJSON(os.Stdout, BatchUpdateEntriesResponse{
						Updated: len(ids),
						IDs:     ids,
						Read:    &v,
					})
				}
				fmt.Fprintf(os.Stdout, "Marked %d entries as read\n", len(ids))
				return nil
			}

			if markUnread {
				if err := app.store.SetEntriesRead(cmd.Context(), ids, false); err != nil {
					return fmt.Errorf("update entries read: %w", err)
				}
				if getOutput() == OutputJSON {
					v := false
					return writeJSON(os.Stdout, BatchUpdateEntriesResponse{
						Updated: len(ids),
						IDs:     ids,
						Read:    &v,
					})
				}
				fmt.Fprintf(os.Stdout, "Marked %d entries as unread\n", len(ids))
				return nil
			}

			if markStarred {
				if err := app.store.SetEntriesStarred(cmd.Context(), ids, true); err != nil {
					return fmt.Errorf("update entries starred: %w", err)
				}
				if getOutput() == OutputJSON {
					v := true
					return writeJSON(os.Stdout, BatchUpdateEntriesResponse{
						Updated: len(ids),
						IDs:     ids,
						Starred: &v,
					})
				}
				fmt.Fprintf(os.Stdout, "Marked %d entries as starred\n", len(ids))
				return nil
			}

			if err := app.store.SetEntriesStarred(cmd.Context(), ids, false); err != nil {
				return fmt.Errorf("update entries starred: %w", err)
			}
			if getOutput() == OutputJSON {
				v := false
				return writeJSON(os.Stdout, BatchUpdateEntriesResponse{
					Updated: len(ids),
					IDs:     ids,
					Starred: &v,
				})
			}
			fmt.Fprintf(os.Stdout, "Marked %d entries as unstarred\n", len(ids))
			return nil
		},
	}

	cmd.Flags().BoolVar(&markRead, "read", false, "Mark all provided IDs as read")
	cmd.Flags().BoolVar(&markUnread, "unread", false, "Mark all provided IDs as unread")
	cmd.Flags().BoolVar(&markStarred, "starred", false, "Mark all provided IDs as starred")
	cmd.Flags().BoolVar(&markUnstarred, "unstarred", false, "Mark all provided IDs as unstarred")
	return cmd
}

func parseIDs(args []string) ([]int64, error) {
	ids := make([]int64, 0, len(args))
	for _, arg := range args {
		id, err := parseID(arg)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

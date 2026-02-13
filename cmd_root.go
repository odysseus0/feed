package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func NewRootCmd(cfg Config) *cobra.Command {
	var dbPath string
	var output string
	var outFmt OutputFormat
	var app *App

	dbPath = cfg.DBPath
	output = string(OutputTable)

	getApp := func() *App { return app }
	getOutput := func() OutputFormat { return outFmt }

	cmd := &cobra.Command{
		Use:           "feed",
		Short:         "Local-first RSS CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			parsedFmt, err := parseOutputFormat(output)
			if err != nil {
				return err
			}
			outFmt = parsedFmt
			if !requiresApp(cmd) {
				return nil
			}
			if app != nil {
				return nil
			}
			a, err := NewApp(cfg, dbPath)
			if err != nil {
				return err
			}
			app = a
			return nil
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if app != nil {
				_ = app.Close()
				app = nil
			}
		},
	}

	cmd.PersistentFlags().StringVar(&dbPath, "db", dbPath, "SQLite database path")
	cmd.PersistentFlags().StringVarP(&output, "output", "o", output, "Output format: table, json, wide")

	cmd.AddCommand(newGetCmd(getApp, getOutput))
	cmd.AddCommand(newAddCmd(getApp, getOutput))
	cmd.AddCommand(newRemoveCmd(getApp, getOutput))
	cmd.AddCommand(newUpdateCmd(getApp, getOutput))
	cmd.AddCommand(newFetchCmd(getApp, getOutput))
	cmd.AddCommand(newImportCmd(getApp, getOutput))
	cmd.AddCommand(newExportCmd(getApp, getOutput))
	cmd.AddCommand(newSearchCmd(getApp, getOutput))

	return cmd
}

func parseOutputFormat(raw string) (OutputFormat, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch OutputFormat(s) {
	case OutputTable, OutputJSON, OutputWide:
		return OutputFormat(s), nil
	default:
		return "", fmt.Errorf("invalid output format %q (expected table|json|wide)", raw)
	}
}

func printCmdError(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
}

func requiresApp(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		name := c.Name()
		if name == "help" || name == "completion" {
			return false
		}
	}
	return true
}

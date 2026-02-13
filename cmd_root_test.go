package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRequiresApp(t *testing.T) {
	root := &cobra.Command{Use: "feed"}
	getCmd := &cobra.Command{Use: "get"}
	root.AddCommand(getCmd)

	helpCmd := &cobra.Command{Use: "help"}
	root.AddCommand(helpCmd)

	completionCmd := &cobra.Command{Use: "completion"}
	bashCmd := &cobra.Command{Use: "bash"}
	completionCmd.AddCommand(bashCmd)
	root.AddCommand(completionCmd)

	if !requiresApp(getCmd) {
		t.Fatalf("regular command should require app")
	}
	if requiresApp(helpCmd) {
		t.Fatalf("help command should not require app")
	}
	if requiresApp(bashCmd) {
		t.Fatalf("completion subcommand should not require app")
	}
}

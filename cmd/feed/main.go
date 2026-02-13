package main

import (
	"os"

	"github.com/odysseus0/feed/internal/cli"
)

func main() {
	os.Exit(run())
}

func run() int {
	err := cli.Execute()
	cli.PrintError(err)
	return cli.ErrorExitCode(err)
}

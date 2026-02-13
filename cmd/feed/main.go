package main

import (
	"os"

	"github.com/tengjizhang/feed/internal/cli"
)

func main() {
	os.Exit(run())
}

func run() int {
	err := cli.Execute()
	cli.PrintError(err)
	return cli.ErrorExitCode(err)
}

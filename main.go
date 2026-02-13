package main

import "os"

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		printCmdError(err)
		os.Exit(1)
	}
	cmd := NewRootCmd(cfg)
	if err := cmd.Execute(); err != nil {
		printCmdError(err)
		os.Exit(1)
	}
}

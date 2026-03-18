package main

import (
	"os"

	"github.com/kkkqkx123/mihomo-cli/cmd"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	if err := cmd.Execute(version, commit); err != nil {
		os.Exit(1)
	}
}

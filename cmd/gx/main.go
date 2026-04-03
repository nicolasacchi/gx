package main

import (
	"fmt"
	"os"

	"github.com/nicolasacchi/gx/internal/commands"
)

var version = "dev"

func main() {
	commands.SetVersion(version)
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Command clibo is the entry point of the clibo CLI.
//
// All logic lives under internal/. This main only delegates to cli.Execute,
// which is responsible for parsing argv, dispatching to subcommands, and
// returning a non-nil error on any failure.
package main

import (
	"os"

	"github.com/rm-info/online_clipboard_cli/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		// cobra already prints the error to stderr.
		os.Exit(1)
	}
}

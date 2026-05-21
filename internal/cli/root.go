// Package cli wires every clibo subcommand together via cobra.
//
// Each subcommand lives in its own file under this package and exposes a
// newXxxCmd() constructor that returns a *cobra.Command. The root command
// registers them explicitly (no init() side effects) so the call graph is
// readable top-to-bottom.
package cli

import (
	"github.com/spf13/cobra"
)

// GlobalFlags holds flag values that apply to every subcommand. The struct
// is populated by cobra during root flag parsing and read by subcommands
// through the package-level Flags variable.
type GlobalFlags struct {
	Server     string // --server: overrides config.server_url for this invocation
	Password   string // -p/--password: footgun, leaks via ps/shell history
	NoPassword bool   // --no-password: skip prompt, use sentinel password
}

// Flags is the singleton populated by the root command. Subcommands read
// from it; tests can mutate it directly.
var Flags GlobalFlags

// Execute parses os.Args and dispatches to the requested subcommand.
// Returns the error from cobra so main() can set a non-zero exit code.
func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clibo",
		Short: "End-to-end encrypted clipboard CLI",
		Long: `clibo is a single-binary CLI client for the online_clipboard E2EE
clipboard service. All cryptography happens client-side: the server
stores only opaque ciphertext, can never read your data, and never sees
your password.

Quick start:
  clibo config set server_url https://clipboard.lab.rm-info.fr
  echo "hello" | clibo copy new          # create a session
  clibo paste                            # read the last entry`,
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	pf := cmd.PersistentFlags()
	pf.StringVar(&Flags.Server, "server", "", "override server URL (config used if unset)")
	pf.StringVarP(&Flags.Password, "password", "p", "", "session password (leaks via ps and shell history — prefer the interactive prompt)")
	pf.BoolVar(&Flags.NoPassword, "no-password", false, "skip password prompt and use sentinel (for scripts)")

	cmd.AddCommand(
		newConfigCmd(),
		newLastCmd(),
		newStatusCmd(),
		newCopyCmd(),
		newPasteCmd(),
	)

	return cmd
}

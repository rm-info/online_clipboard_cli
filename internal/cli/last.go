package cli

import (
	"errors"
	"fmt"

	"github.com/rm-info/online_clipboard_cli/internal/state"
	"github.com/spf13/cobra"
)

func newLastCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "last",
		Short: "Print the most recently used session ID",
		Long: `Print the session ID stored as "last" — useful for scripting:

  clibo paste $(clibo last)
  clibo del $(clibo last) 3

Exits non-zero if no last session is known.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := state.Load()
			if err != nil {
				return err
			}
			if s.LastSID == "" {
				return errors.New("no last session known (run `clibo copy new ...` first)")
			}
			fmt.Fprintln(cmd.OutOrStdout(), s.LastSID)
			return nil
		},
	}
}

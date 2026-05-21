package cli

import (
	"fmt"
	"time"

	"github.com/rm-info/online_clipboard_cli/internal/config"
	"github.com/rm-info/online_clipboard_cli/internal/state"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show local clibo context (server, last session, auth cache)",
		Long: `Show the configured server, the last session ID, and whether
clibo has a cached cookie for it. No network calls are made — the
cookie's expiry is shown per the local clock and may be stale.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			s, err := state.Load()
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()

			if srv, err := config.ResolveServerURL(cfg, Flags.Server); err == nil {
				fmt.Fprintf(out, "server: %s\n", srv)
			} else {
				fmt.Fprintln(out, "server: (not configured — run `clibo config set server_url <URL>`)")
			}

			if s.LastSID == "" {
				fmt.Fprintln(out, "last:   (none)")
				return nil
			}
			fmt.Fprintf(out, "last:   %s\n", s.LastSID)

			c, ok := s.GetCookie(s.LastSID)
			if !ok {
				fmt.Fprintln(out, "        no cached auth (next op will re-auth)")
				return nil
			}
			remaining := time.Until(time.Unix(c.ExpiresAt, 0))
			if remaining > 0 {
				fmt.Fprintf(out, "        auth cached, expires in %s (local clock)\n", formatDuration(remaining))
			} else {
				fmt.Fprintf(out, "        auth cached but expired %s ago (local clock)\n", formatDuration(-remaining))
			}
			return nil
		},
	}
}

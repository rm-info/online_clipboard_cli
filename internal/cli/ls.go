package cli

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/rm-info/online_clipboard_cli/internal/flow"
	"github.com/rm-info/online_clipboard_cli/internal/proto"
	"github.com/rm-info/online_clipboard_cli/internal/state"
	"github.com/rm-info/online_clipboard_cli/internal/ui"
	"github.com/spf13/cobra"
)

func newLsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls [SID|last]",
		Short: "List a session's entries (text previews, file names, sizes)",
		Long: `List every entry in a session in the order it was added. Text
entries show a one-line preview; file entries show the decrypted
filename. Size and relative age accompany each row.

With -s, only the session status line is shown (sid, cookie expiry
from the local cache, total entry count, total file bytes).`,
		Args: cobra.MaximumNArgs(1),
		RunE: runLs,
	}
	cmd.Flags().BoolP("status", "s", false, "show only the session status line")
	return cmd
}

func runLs(cmd *cobra.Command, args []string) error {
	ctx, err := newCLIContext(context.Background())
	if err != nil {
		return err
	}
	defer ctx.Save()

	sidArg := ""
	if len(args) == 1 {
		sidArg = args[0]
	}
	sid, err := ui.ResolveSID(sidArg, ctx.State)
	if err != nil {
		return err
	}

	entries, fileBytes, err := ctx.Auth.List(ctx.Ctx, sid)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	printSessionStatus(out, sid, len(entries), fileBytes, ctx.State)

	statusOnly, _ := cmd.Flags().GetBool("status")
	if statusOnly {
		return nil
	}
	if len(entries) == 0 {
		fmt.Fprintln(out, "(no entries)")
		return nil
	}
	printEntries(out, entries)
	return nil
}

func printSessionStatus(w io.Writer, sid string, itemCount int, fileBytes int64, st *state.State) {
	expires := "expiry unknown"
	if c, ok := st.GetCookie(sid); ok {
		remaining := time.Until(time.Unix(c.ExpiresAt, 0))
		if remaining > 0 {
			expires = "expires in " + formatDuration(remaining)
		} else {
			expires = "expired " + formatDuration(-remaining) + " ago"
		}
	}
	fmt.Fprintf(w, "%s  %s  •  %d %s  •  %s used\n",
		sid, expires, itemCount, pluralize("entry", "entries", itemCount), humanBytes(fileBytes))
}

func printEntries(w io.Writer, entries []flow.DecryptedEntry) {
	// Compute indices and column widths so the listing aligns.
	idxWidth := len(strconv.Itoa(len(entries)))
	for _, e := range entries {
		ts := time.Unix(e.Timestamp, 0)
		switch e.Type {
		case proto.EntryText:
			label := textPreview(e.Text, 50)
			if e.Secret {
				label = "(secret) " + textPreview(e.Text, 40)
			}
			fmt.Fprintf(w, "%*d  text  %-52s  %7s  %s\n",
				idxWidth, e.Index, quote(label), humanBytes(e.Size), relativeTime(ts))
		case proto.EntryFile:
			marker := ""
			if e.HasThumb {
				marker = " 🖼"
			}
			fmt.Fprintf(w, "%*d  file  %-52s  %7s  %s%s\n",
				idxWidth, e.Index, textPreview(e.Filename, 50), humanBytes(e.Size), relativeTime(ts), marker)
		}
	}
}

func quote(s string) string {
	return "\"" + s + "\""
}

func pluralize(singular, plural string, n int) string {
	if n == 1 {
		return singular
	}
	return plural
}

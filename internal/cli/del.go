package cli

import (
	"context"
	"fmt"

	"github.com/rm-info/online_clipboard_cli/internal/flow"
	"github.com/rm-info/online_clipboard_cli/internal/proto"
	"github.com/rm-info/online_clipboard_cli/internal/ui"
	"github.com/spf13/cobra"
)

func newDelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "del <SID|last> <N|last>",
		Short: "Delete one entry from a session (text or file)",
		Long: `Delete a single entry. Both positionals are required — destructive
commands don't accept defaults. A summary of the target entry is
printed before deletion; pass -y to skip the confirmation prompt.`,
		Args: cobra.ExactArgs(2),
		RunE: runDel,
	}
	cmd.Flags().BoolP("yes", "y", false, "skip confirmation")
	return cmd
}

func runDel(cmd *cobra.Command, args []string) error {
	ctx, err := newCLIContext(context.Background())
	if err != nil {
		return err
	}
	defer ctx.Save()

	sid, err := ui.ResolveSID(args[0], ctx.State)
	if err != nil {
		return err
	}
	spec, err := ui.ParseEntryIndex(args[1])
	if err != nil {
		return err
	}

	entries, _, err := ctx.Auth.List(ctx.Ctx, sid)
	if err != nil {
		return err
	}
	idx, err := flow.ResolveIndex(spec, len(entries))
	if err != nil {
		return err
	}
	target := entries[idx]

	force, _ := cmd.Flags().GetBool("yes")
	if !force {
		if err := confirmYesNo(fmt.Sprintf("Delete entry %d from %s (%s)?", target.Index, sid, describeEntry(target))); err != nil {
			return err
		}
	}
	return ctx.Auth.DeleteEntry(ctx.Ctx, sid, &target)
}

// describeEntry returns a short human label like 'text "hello world"'
// or "file report.pdf (2.4 MB)" used in confirm prompts.
func describeEntry(e flow.DecryptedEntry) string {
	switch e.Type {
	case proto.EntryText:
		return fmt.Sprintf("text \"%s\"", textPreview(e.Text, 40))
	case proto.EntryFile:
		return fmt.Sprintf("file %s, %s", textPreview(e.Filename, 40), humanBytes(e.Size))
	}
	return string(e.Type)
}

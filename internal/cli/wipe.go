package cli

import (
	"context"
	"fmt"

	"github.com/rm-info/online_clipboard_cli/internal/ui"
	"github.com/spf13/cobra"
)

func newWipeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wipe <SID|last>",
		Short: "Delete a session and every entry it holds",
		Long: `Destroy a session server-side. No default — destructive commands
require an explicit target. Shows the entry count before confirming;
pass -y to skip the prompt.`,
		Args: cobra.ExactArgs(1),
		RunE: runWipe,
	}
	cmd.Flags().BoolP("yes", "y", false, "skip confirmation")
	return cmd
}

func runWipe(cmd *cobra.Command, args []string) error {
	ctx, err := newCLIContext(context.Background())
	if err != nil {
		return err
	}
	defer ctx.Save()

	sid, err := ui.ResolveSID(args[0], ctx.State)
	if err != nil {
		return err
	}

	force, _ := cmd.Flags().GetBool("yes")
	if !force {
		res, err := ctx.Auth.List(ctx.Ctx, sid)
		if err != nil {
			return err
		}
		if err := confirmYesNo(fmt.Sprintf("Wipe session %s (%d %s)?", sid, len(res.Entries), pluralize("entry", "entries", len(res.Entries)))); err != nil {
			return err
		}
	}
	return ctx.Auth.Wipe(ctx.Ctx, sid)
}

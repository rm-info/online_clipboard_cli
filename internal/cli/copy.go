package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/rm-info/online_clipboard_cli/internal/ui"
	"github.com/spf13/cobra"
)

func newCopyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "copy <SID|last> [TEXT]",
		Short: "Append text or a file to an existing session",
		Long: `Append a new entry to a session whose ID is provided positionally.

The first positional must be "last" (resolved against the locally
stored last session) or a literal session ID. The second positional
is the content — text by default, a file path when -f is set. With
no second positional, content is read from stdin (must be a pipe).

Examples:
  echo "log line" | clibo copy last
  journalctl -u nginx | clibo copy abc12
  clibo copy last -f /tmp/report.pdf`,
		Args: cobra.RangeArgs(1, 2),
		RunE: runCopyAppend,
	}
	cmd.Flags().BoolP("file", "f", false, "interpret the positional argument as a file path to upload")
	cmd.AddCommand(newCopyNewCmd())
	return cmd
}

func newCopyNewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new [TEXT]",
		Short: "Create a new session with TEXT (or stdin) as the first entry",
		Long: `Create a fresh session. The first entry is required — text from the
positional argument or, when none is given, the contents of stdin.
Prints the new session ID on stdout.

The first entry must be text; uploading a file as the very first
entry isn't supported by the server. Add files with ` + "`clibo copy <SID> -f FILE`" + ` after creation.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runCopyNew,
	}
	cmd.Flags().Bool("secure", false, "use a 50-char secure session ID")
	cmd.Flags().Bool("secret", false, "mark the first entry as secret (shoulder-surfing protection)")
	return cmd
}

func runCopyAppend(cmd *cobra.Command, args []string) error {
	ctx, err := newCLIContext(context.Background())
	if err != nil {
		return err
	}
	defer ctx.Save()

	sid, err := ui.ResolveSID(args[0], ctx.State)
	if err != nil {
		return err
	}

	asFile, _ := cmd.Flags().GetBool("file")
	if asFile {
		if len(args) < 2 {
			return errors.New("-f requires a file path argument")
		}
		out, err := ctx.Auth.AddFile(ctx.Ctx, sid, args[1])
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "uploaded (id=%s, %d bytes)\n", out.ID, out.Size)
		return nil
	}

	positional := ""
	if len(args) >= 2 {
		positional = args[1]
	}
	text, err := readTextContent(positional)
	if err != nil {
		return err
	}
	return ctx.Auth.AddText(ctx.Ctx, sid, text, false)
}

func runCopyNew(cmd *cobra.Command, args []string) error {
	ctx, err := newCLIContext(context.Background())
	if err != nil {
		return err
	}
	defer ctx.Save()

	positional := ""
	if len(args) == 1 {
		positional = args[0]
	}
	text, err := readTextContent(positional)
	if err != nil {
		return err
	}

	password, err := newSessionPassword()
	if err != nil {
		return err
	}

	secure, _ := cmd.Flags().GetBool("secure")
	secret, _ := cmd.Flags().GetBool("secret")

	sid, err := ctx.Auth.CreateSession(ctx.Ctx, text, password, secure, secret)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), sid)
	return nil
}

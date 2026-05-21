package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rm-info/online_clipboard_cli/internal/flow"
	"github.com/rm-info/online_clipboard_cli/internal/ui"
	"github.com/spf13/cobra"
)

func newPasteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "paste [SID|last] [N|last]",
		Short: "Read an entry from a session (text → stdout, file → disk)",
		Long: `Fetch and decrypt one entry from a session.

Positionals are order-independent and detected by shape: a 5- or 50-
character string is the session ID, an integer is the 1-based entry
index, and the literal "last" fills whichever slot is still free.
Both default to "last" when omitted.

Output rules:
  text entry, no -f       → stdout
  text entry, -f PATH     → write to PATH (confirms overwrite)
  file entry, no -f       → write to $PWD/<decrypted-filename>
  file entry, -f PATH     → write to PATH
  any case, -f -          → force stdout (binary may corrupt your terminal)
  any case, -y            → skip overwrite confirmation

Examples:
  clibo paste                       # last session, last entry
  clibo paste 3                     # last session, entry 3
  clibo paste abc12                 # session abc12, last entry
  clibo paste abc12 2 -f /tmp/x     # session abc12, entry 2 → /tmp/x`,
		Args: cobra.MaximumNArgs(2),
		RunE: runPaste,
	}
	cmd.Flags().StringP("file", "f", "", "write to PATH instead of the default (use '-' for stdout)")
	cmd.Flags().BoolP("yes", "y", false, "skip overwrite confirmation")
	return cmd
}

func runPaste(cmd *cobra.Command, args []string) error {
	parsed, err := parsePasteArgs(args)
	if err != nil {
		return err
	}

	ctx, err := newCLIContext(context.Background())
	if err != nil {
		return err
	}
	defer ctx.Save()

	sid, err := ui.ResolveSID(parsed.SIDArg, ctx.State)
	if err != nil {
		return err
	}
	spec, err := ui.ParseEntryIndex(parsed.NArg)
	if err != nil {
		return err
	}

	payload, err := ctx.Auth.GetEntry(ctx.Ctx, sid, spec)
	if err != nil {
		return err
	}

	filePath, _ := cmd.Flags().GetString("file")
	force, _ := cmd.Flags().GetBool("yes")
	return emitPayload(payload, filePath, force, cmd)
}

func emitPayload(p *flow.PastePayload, flagPath string, force bool, cmd *cobra.Command) error {
	if p.IsFile {
		dest := flagPath
		if dest == "" {
			dest = filepath.Join(".", p.Filename)
		}
		// Refuse cramming binary into an interactive terminal unless the
		// user explicitly asked for stdout via "-".
		if dest == "-" && ui.IsStdoutTTY() {
			return errors.New("refusing to write binary file to a terminal (redirect to a file or pipe)")
		}
		if dest == "-" {
			_, err := os.Stdout.Write(p.Bytes)
			return err
		}
		return writeOutput(dest, p.Bytes, force)
	}

	// Text entry.
	if flagPath == "" || flagPath == "-" {
		_, err := fmt.Fprint(cmd.OutOrStdout(), p.Text)
		return err
	}
	return writeOutput(flagPath, []byte(p.Text), force)
}

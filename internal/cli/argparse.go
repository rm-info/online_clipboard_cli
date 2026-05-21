package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rm-info/online_clipboard_cli/internal/ui"
)

// pasteArgs is the resolved form of `paste`'s positional arguments.
// Both fields default to "last"; the actual SID and entry index are
// computed later (state.LastSID for SID, len(entries) for "last" N).
type pasteArgs struct {
	SIDArg string
	NArg   string
}

// parsePasteArgs disambiguates up to 2 positional arguments into SID
// vs entry-index by inspecting each independently. The detection is
// length-based for SIDs (5 or 50 chars) and Atoi-based for indices.
// "last" is a wildcard that fills whichever slot is still empty,
// preferring the SID slot if both are free.
func parsePasteArgs(args []string) (pasteArgs, error) {
	out := pasteArgs{}
	for _, a := range args {
		switch {
		case ui.IsSessionIDShape(a):
			if out.SIDArg != "" {
				return out, fmt.Errorf("two session-id-shaped arguments: %q and %q", out.SIDArg, a)
			}
			out.SIDArg = a
		case isInteger(a):
			if out.NArg != "" {
				return out, fmt.Errorf("two integer arguments: %q and %q", out.NArg, a)
			}
			out.NArg = a
		case a == "last":
			switch {
			case out.SIDArg == "":
				out.SIDArg = "last"
			case out.NArg == "":
				out.NArg = "last"
			default:
				return out, errors.New("too many 'last' arguments")
			}
		default:
			return out, fmt.Errorf("invalid argument %q (expected session ID, integer, or 'last')", a)
		}
	}
	if out.SIDArg == "" {
		out.SIDArg = "last"
	}
	if out.NArg == "" {
		out.NArg = "last"
	}
	return out, nil
}

func isInteger(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

// readTextContent returns the text to copy: explicit positional arg
// wins, otherwise stdin is slurped if it's piped. Empty result is
// rejected (the server refuses empty items anyway).
func readTextContent(positional string) (string, error) {
	if positional != "" {
		return positional, nil
	}
	if !ui.IsStdinPipe() {
		return "", errors.New("no text provided (pass it as an argument or pipe it on stdin)")
	}
	text, err := ui.ReadAllStdin()
	if err != nil {
		return "", err
	}
	if text == "" {
		return "", errors.New("stdin was empty")
	}
	return text, nil
}

// confirmOverwrite asks the user whether to overwrite path on stderr
// and reads y/n from stdin. force=true bypasses the prompt. If stdin
// is not a TTY we fail (callers can pass force from a -y flag).
func confirmOverwrite(path string, force bool) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if force {
		return nil
	}
	if ui.IsStdinPipe() {
		return fmt.Errorf("%s already exists; pass -y to overwrite (stdin is not a TTY for confirmation)", path)
	}
	fmt.Fprintf(os.Stderr, "Overwrite %s? [y/N] ", path)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	ans := strings.TrimSpace(strings.ToLower(line))
	if ans != "y" && ans != "yes" {
		return errors.New("aborted by user")
	}
	return nil
}

// writeOutput dispatches between stdout (path == "-") and a file
// write, applying confirmOverwrite for existing files.
func writeOutput(path string, data []byte, force bool) error {
	if path == "-" {
		_, err := os.Stdout.Write(data)
		return err
	}
	if err := confirmOverwrite(path, force); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// confirmYesNo prints prompt to stderr and waits for y/yes (case-
// insensitive) on stdin. Any other answer is treated as "no" and the
// function returns an "aborted" error. Errors out without prompting
// when stdin isn't a TTY — destructive commands should accept -y to
// skip this path entirely.
func confirmYesNo(prompt string) error {
	if ui.IsStdinPipe() {
		return fmt.Errorf("%s — pass -y to confirm (stdin is not a TTY)", prompt)
	}
	fmt.Fprintf(os.Stderr, "%s [y/N] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	ans := strings.TrimSpace(strings.ToLower(line))
	if ans != "y" && ans != "yes" {
		return errors.New("aborted by user")
	}
	return nil
}

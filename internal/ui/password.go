package ui

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"
)

// ErrNoTTYForPrompt is returned when PromptPassword is invoked but
// stdin is not a terminal. Scripts should pass --password, set
// CLIP_PASSWORD, or use --no-password instead.
var ErrNoTTYForPrompt = errors.New("cannot prompt for password (stdin is not a terminal); pass --password, set CLIP_PASSWORD, or use --no-password")

// PromptPassword reads a password from /dev/tty (via stdin) with echo
// disabled. The prompt is written to stderr so it doesn't pollute
// stdout pipes. Returns ErrNoTTYForPrompt if stdin is not a terminal.
func PromptPassword(prompt string) (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", ErrNoTTYForPrompt
	}
	fmt.Fprint(os.Stderr, prompt)
	b, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

package ui

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

// IsStdinPipe reports whether stdin is reading from a pipe or file
// (as opposed to a terminal). Used by `clibo copy` to decide whether
// to slurp stdin without an explicit positional arg.
func IsStdinPipe() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) == 0
}

// IsStdoutTTY reports whether stdout is going to a terminal. Used by
// `paste` to refuse dumping binary into the user's terminal by accident.
func IsStdoutTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// ReadAllStdin slurps stdin into a string.
func ReadAllStdin() (string, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	return string(data), nil
}

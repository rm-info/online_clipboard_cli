// Package ui holds the human-facing helpers: TTY-aware password
// prompting, parsing of positional session/entry arguments, and
// stdin/stdout pipe detection. Subcommands compose these primitives;
// the package itself stays free of CLI-framework dependencies.
package ui

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/rm-info/online_clipboard_cli/internal/state"
)

// Session IDs are 5 chars in normal mode and 50 chars in secure mode.
// Disambiguating SID from entry index by length lets us avoid baking
// the server's alphabet into the client.
const (
	SIDNormalLength = 5
	SIDSecureLength = 50
)

// ErrNoLastSession is returned when an argument resolves (implicitly
// or via the literal "last") to state.LastSID and that field is empty.
var ErrNoLastSession = errors.New("no last session known (run `clibo copy new ...` first)")

// IsSessionIDShape reports whether s could plausibly be a session ID
// based on length alone. Anything else is treated as an entry index.
func IsSessionIDShape(s string) bool {
	return len(s) == SIDNormalLength || len(s) == SIDSecureLength
}

// ResolveSID interprets a positional SID argument. Empty string and
// the literal "last" both fall back to state.LastSID. Anything else
// passes through unchanged — the server tells us if it's bogus.
func ResolveSID(arg string, st *state.State) (string, error) {
	if arg == "" || arg == "last" {
		if st == nil || st.LastSID == "" {
			return "", ErrNoLastSession
		}
		return st.LastSID, nil
	}
	return arg, nil
}

// EntryIndexSpec is the parsed form of paste/get/del's N argument.
// Resolution of Last to an actual entry index requires fetching
// /contents — that's the caller's job.
type EntryIndexSpec struct {
	Last  bool
	Index int // 1-based, valid when !Last
}

// ParseEntryIndex parses s as an entry-index argument:
//   - "last"        → {Last: true}
//   - "1".."N"      → {Index: N}
//   - anything else → error
func ParseEntryIndex(s string) (EntryIndexSpec, error) {
	if s == "" {
		return EntryIndexSpec{}, errors.New("entry index required")
	}
	if s == "last" {
		return EntryIndexSpec{Last: true}, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return EntryIndexSpec{}, fmt.Errorf("invalid entry index %q (expected integer or \"last\")", s)
	}
	if n < 1 {
		return EntryIndexSpec{}, fmt.Errorf("entry index must be >= 1, got %d", n)
	}
	return EntryIndexSpec{Index: n}, nil
}

// Package version holds the build-time identifiers stamped into the
// binary via -ldflags. Defaults are useful for `go run`/`go install`
// invocations that bypass the Makefile.
package version

var (
	// Version is the human-readable release tag, e.g. "v0.1.0".
	Version = "dev"
	// Commit is the short git SHA the binary was built from.
	Commit = "unknown"
)

// String renders "<version> (<commit>)" for --version output.
func String() string {
	return Version + " (" + Commit + ")"
}

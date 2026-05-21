package cli

import (
	"fmt"
	"strings"
	"time"
)

// humanBytes renders an integer byte count with a unit. KB/MB/GB are
// powers of 1024 (matching the server's quota math).
func humanBytes(n int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case n < kb:
		return fmt.Sprintf("%d B", n)
	case n < mb:
		return fmt.Sprintf("%.1f KB", float64(n)/kb)
	case n < gb:
		return fmt.Sprintf("%.1f MB", float64(n)/mb)
	default:
		return fmt.Sprintf("%.2f GB", float64(n)/gb)
	}
}

// relativeTime returns a compact "5m ago" / "2h ago" / "just now"
// description of how long ago t was.
func relativeTime(t time.Time) string {
	if t.IsZero() {
		return "?"
	}
	d := time.Since(t)
	if d < 0 {
		d = -d
	}
	if d < 30*time.Second {
		return "just now"
	}
	return formatDuration(d) + " ago"
}

// textPreview shortens s to maxRunes runes, replacing newlines so the
// preview fits on one line. Empty result is rendered as a placeholder
// so the column doesn't collapse.
func textPreview(s string, maxRunes int) string {
	s = strings.ReplaceAll(s, "\n", "↵")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	if s == "" {
		return "(empty)"
	}
	runes := []rune(s)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes-1]) + "…"
	}
	return s
}

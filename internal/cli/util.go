package cli

import (
	"fmt"
	"time"
)

// formatDuration prints d compactly: "23s", "47min", "1h 47min", "2h".
// Callers pass a positive duration; sign handling is up to them.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dmin", int(d.Minutes()))
	}
	h := int(d.Hours())
	m := int((d - time.Duration(h)*time.Hour).Minutes())
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh %dmin", h, m)
}

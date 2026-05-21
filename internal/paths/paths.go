// Package paths resolves the canonical XDG locations for clibo's
// config and state files, and provides a shared atomic-write helper.
//
// On Linux, config and state live in different XDG roots
// (XDG_CONFIG_HOME vs XDG_DATA_HOME) so user-editable preferences and
// runtime-managed state can be reasoned about independently. On macOS
// and Windows they share the platform's standard application directory.
package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const appName = "clibo"

// ConfigFile returns the path to clibo's config.toml.
func ConfigFile() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appName, "config.toml"), nil
}

// StateFile returns the path to clibo's state.json.
func StateFile() (string, error) {
	if runtime.GOOS == "linux" {
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			return filepath.Join(xdg, appName, "state.json"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share", appName, "state.json"), nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appName, "state.json"), nil
}

// AtomicWriteFile writes data to path via a temp file in the same
// directory followed by an atomic rename. A crash mid-write leaves
// either the old file intact or the new file fully written — never a
// truncated mess.
func AtomicWriteFile(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp.*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	// Always try to remove the temp; if the rename succeeded this is a no-op.
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

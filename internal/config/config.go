// Package config holds clibo's user-editable preferences.
//
// The config is a small TOML file under XDG_CONFIG_HOME. For v0.1 it
// has a single key (server_url). New keys are added by appending to
// allowedKeys and adding a matching field on Config.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/rm-info/online_clipboard_cli/internal/paths"
)

// Config holds every persisted preference. Field tags drive both
// TOML (de)serialisation and the generic Get/Set/Unset API.
type Config struct {
	ServerURL string `toml:"server_url,omitempty"`
}

// allowedKeys lists every recognised TOML key, in the order `list`
// should print them. Adding a key means appending here AND adding a
// field above with the same toml tag.
var allowedKeys = []string{"server_url"}

// ErrNoServerURL is returned by ResolveServerURL when no server URL
// is configured anywhere (flag, env, file).
var ErrNoServerURL = errors.New("no server URL configured (run: clibo config set server_url <URL>)")

// KeyValue is one entry of a config listing.
type KeyValue struct {
	Key   string
	Value string
}

// Load reads the config from its canonical XDG location. A missing
// file yields an empty Config and no error — first run is normal.
func Load() (*Config, error) {
	p, err := paths.ConfigFile()
	if err != nil {
		return nil, err
	}
	return LoadFile(p)
}

// LoadFile reads the config from an explicit path. Used by tests and
// by Load.
func LoadFile(path string) (*Config, error) {
	cfg := &Config{}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// Save writes the config to its canonical XDG location.
func Save(cfg *Config) error {
	p, err := paths.ConfigFile()
	if err != nil {
		return err
	}
	return SaveFile(p, cfg)
}

// SaveFile writes the config to an explicit path. The file always
// begins with an editor-friendly header comment.
func SaveFile(path string, cfg *Config) error {
	var b strings.Builder
	b.WriteString("# clibo configuration\n")
	b.WriteString("# Edit by hand or via `clibo config set <key> <value>`.\n\n")
	if err := toml.NewEncoder(&b).Encode(cfg); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	return paths.AtomicWriteFile(path, []byte(b.String()), 0o644)
}

// Get returns the value of a config key.
func (c *Config) Get(key string) (string, error) {
	switch key {
	case "server_url":
		return c.ServerURL, nil
	default:
		return "", unknownKey(key)
	}
}

// Set updates a config key. An empty value is equivalent to Unset.
func (c *Config) Set(key, value string) error {
	switch key {
	case "server_url":
		c.ServerURL = value
		return nil
	default:
		return unknownKey(key)
	}
}

// Unset clears a config key (reverts it to its zero value).
func (c *Config) Unset(key string) error {
	return c.Set(key, "")
}

// List returns every recognised key with its current value, in a
// stable order suitable for display.
func (c *Config) List() []KeyValue {
	out := make([]KeyValue, 0, len(allowedKeys))
	for _, k := range allowedKeys {
		v, _ := c.Get(k)
		out = append(out, KeyValue{Key: k, Value: v})
	}
	return out
}

// ResolveServerURL returns the effective server URL by checking, in
// order: flagValue, CLIP_SERVER env var, cfg.ServerURL. Returns
// ErrNoServerURL if none of those are set.
func ResolveServerURL(cfg *Config, flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if env := os.Getenv("CLIP_SERVER"); env != "" {
		return env, nil
	}
	if cfg != nil && cfg.ServerURL != "" {
		return cfg.ServerURL, nil
	}
	return "", ErrNoServerURL
}

func unknownKey(key string) error {
	return fmt.Errorf("unknown config key %q (allowed: %s)", key, strings.Join(allowedKeys, ", "))
}

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingFileReturnsEmptyConfig(t *testing.T) {
	cfg, err := LoadFile(filepath.Join(t.TempDir(), "does-not-exist.toml"))
	if err != nil {
		t.Fatalf("LoadFile on missing path: %v", err)
	}
	if cfg.ServerURL != "" {
		t.Errorf("expected empty ServerURL, got %q", cfg.ServerURL)
	}
}

func TestSaveAndLoadRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	original := &Config{ServerURL: "https://example.test"}

	if err := SaveFile(path, original); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}
	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if loaded.ServerURL != original.ServerURL {
		t.Errorf("roundtrip lost ServerURL: got %q want %q", loaded.ServerURL, original.ServerURL)
	}
}

func TestSavedFileHasHeaderComment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := SaveFile(path, &Config{ServerURL: "https://a"}); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !strings.HasPrefix(string(data), "# clibo configuration") {
		t.Errorf("expected header comment, got:\n%s", data)
	}
}

func TestEmptyServerURLOmittedFromFile(t *testing.T) {
	// omitempty means an unset key should not appear in the saved file.
	// This is what makes `unset` work without leaving stale lines behind.
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := SaveFile(path, &Config{}); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "server_url") {
		t.Errorf("expected no server_url line for empty value, got:\n%s", data)
	}
}

func TestGetSetUnset(t *testing.T) {
	cfg := &Config{}

	if err := cfg.Set("server_url", "https://foo"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	v, err := cfg.Get("server_url")
	if err != nil || v != "https://foo" {
		t.Errorf("Get after Set: got %q err=%v", v, err)
	}

	if err := cfg.Unset("server_url"); err != nil {
		t.Fatalf("Unset: %v", err)
	}
	v, _ = cfg.Get("server_url")
	if v != "" {
		t.Errorf("Get after Unset: got %q want empty", v)
	}
}

func TestUnknownKeyReturnsError(t *testing.T) {
	cfg := &Config{}
	if _, err := cfg.Get("bogus"); err == nil {
		t.Error("Get(bogus): expected error, got nil")
	}
	if err := cfg.Set("bogus", "x"); err == nil {
		t.Error("Set(bogus): expected error, got nil")
	}
}

func TestResolveServerURLLayering(t *testing.T) {
	cfg := &Config{ServerURL: "https://from-file"}

	// CLIP_SERVER must not leak in from the surrounding environment.
	t.Setenv("CLIP_SERVER", "")

	// 1. flag wins over everything.
	got, err := ResolveServerURL(cfg, "https://from-flag")
	if err != nil || got != "https://from-flag" {
		t.Errorf("flag layer: got %q err=%v", got, err)
	}

	// 2. env wins over file.
	t.Setenv("CLIP_SERVER", "https://from-env")
	got, err = ResolveServerURL(cfg, "")
	if err != nil || got != "https://from-env" {
		t.Errorf("env layer: got %q err=%v", got, err)
	}

	// 3. file used when nothing else is set.
	t.Setenv("CLIP_SERVER", "")
	got, err = ResolveServerURL(cfg, "")
	if err != nil || got != "https://from-file" {
		t.Errorf("file layer: got %q err=%v", got, err)
	}

	// 4. error when nothing is configured.
	if _, err := ResolveServerURL(&Config{}, ""); err == nil {
		t.Error("expected ErrNoServerURL when nothing is set, got nil")
	}
}

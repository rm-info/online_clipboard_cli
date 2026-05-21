package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFileReturnsEmptyState(t *testing.T) {
	s, err := LoadFile(filepath.Join(t.TempDir(), "no-state.json"))
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if s.LastSID != "" || len(s.Cookies) != 0 {
		t.Errorf("expected empty state, got %+v", s)
	}
}

func TestSaveAndLoadRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	orig := &State{
		LastSID: "abc12",
		Cookies: map[string]Cookie{
			"abc12": {Token: "tok1", ExpiresAt: 1700000000},
			"def34": {Token: "tok2", ExpiresAt: 1700001000},
		},
	}
	if err := SaveFile(path, orig); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}

	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if loaded.LastSID != orig.LastSID {
		t.Errorf("LastSID: got %q want %q", loaded.LastSID, orig.LastSID)
	}
	for k, want := range orig.Cookies {
		if got := loaded.Cookies[k]; got != want {
			t.Errorf("cookie[%q]: got %+v want %+v", k, got, want)
		}
	}
}

func TestSaveFileMode0600(t *testing.T) {
	// Cookies authorise writes to a session — they belong with mode 0600
	// regardless of what umask says.
	path := filepath.Join(t.TempDir(), "state.json")
	if err := SaveFile(path, &State{LastSID: "abc12"}); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode: got %#o want 0600", perm)
	}
}

func TestSetGetCookie(t *testing.T) {
	s := &State{}
	s.SetCookie("abc12", "tok", 12345)
	c, ok := s.GetCookie("abc12")
	if !ok || c.Token != "tok" || c.ExpiresAt != 12345 {
		t.Errorf("GetCookie: got %+v ok=%v", c, ok)
	}
	if _, ok := s.GetCookie("nope"); ok {
		t.Error("GetCookie on unknown sid: expected ok=false")
	}
}

func TestForgetSession(t *testing.T) {
	s := &State{
		LastSID: "abc12",
		Cookies: map[string]Cookie{
			"abc12": {Token: "t1"},
			"def34": {Token: "t2"},
		},
	}
	s.ForgetSession("abc12")
	if _, ok := s.GetCookie("abc12"); ok {
		t.Error("cookie for abc12 should be gone")
	}
	if s.LastSID != "" {
		t.Errorf("LastSID should be cleared, got %q", s.LastSID)
	}
	// Other sessions untouched.
	if _, ok := s.GetCookie("def34"); !ok {
		t.Error("cookie for def34 should remain")
	}
}

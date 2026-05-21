package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/rm-info/online_clipboard_cli/internal/state"
)

func TestIsSessionIDShape(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"abc12", true},                              // 5 chars: normal SID
		{strings.Repeat("a", 50), true},              // 50 chars: secure SID
		{"abc", false},                               // too short
		{strings.Repeat("a", 49), false},             // off-by-one (not 5, not 50)
		{strings.Repeat("a", 51), false},             // off-by-one (not 5, not 50)
		{"", false},                                  // empty
		{"12345", true},                              // numeric-but-5-chars is still a SID
	}
	for _, c := range cases {
		if got := IsSessionIDShape(c.in); got != c.want {
			t.Errorf("IsSessionIDShape(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestResolveSID(t *testing.T) {
	stWith := &state.State{LastSID: "abc12"}
	stEmpty := &state.State{}

	t.Run("explicit sid passes through", func(t *testing.T) {
		got, err := ResolveSID("def34", stWith)
		if err != nil || got != "def34" {
			t.Errorf("got %q err=%v", got, err)
		}
	})
	t.Run("empty arg falls back to last", func(t *testing.T) {
		got, err := ResolveSID("", stWith)
		if err != nil || got != "abc12" {
			t.Errorf("got %q err=%v", got, err)
		}
	})
	t.Run("literal 'last' falls back to last", func(t *testing.T) {
		got, err := ResolveSID("last", stWith)
		if err != nil || got != "abc12" {
			t.Errorf("got %q err=%v", got, err)
		}
	})
	t.Run("'last' with no last returns ErrNoLastSession", func(t *testing.T) {
		_, err := ResolveSID("last", stEmpty)
		if !errors.Is(err, ErrNoLastSession) {
			t.Errorf("got err=%v, want ErrNoLastSession", err)
		}
	})
	t.Run("empty arg with no last returns ErrNoLastSession", func(t *testing.T) {
		_, err := ResolveSID("", stEmpty)
		if !errors.Is(err, ErrNoLastSession) {
			t.Errorf("got err=%v, want ErrNoLastSession", err)
		}
	})
	t.Run("nil state with explicit sid still works", func(t *testing.T) {
		got, err := ResolveSID("def34", nil)
		if err != nil || got != "def34" {
			t.Errorf("got %q err=%v", got, err)
		}
	})
}

func TestParseEntryIndex(t *testing.T) {
	t.Run("last", func(t *testing.T) {
		spec, err := ParseEntryIndex("last")
		if err != nil {
			t.Fatalf("ParseEntryIndex(last): %v", err)
		}
		if !spec.Last {
			t.Errorf("expected Last=true")
		}
	})
	t.Run("integer", func(t *testing.T) {
		spec, err := ParseEntryIndex("3")
		if err != nil {
			t.Fatalf("ParseEntryIndex(3): %v", err)
		}
		if spec.Last || spec.Index != 3 {
			t.Errorf("got %+v", spec)
		}
	})
	t.Run("zero rejected", func(t *testing.T) {
		if _, err := ParseEntryIndex("0"); err == nil {
			t.Error("expected error for 0")
		}
	})
	t.Run("negative rejected", func(t *testing.T) {
		if _, err := ParseEntryIndex("-1"); err == nil {
			t.Error("expected error for -1")
		}
	})
	t.Run("non-integer rejected", func(t *testing.T) {
		if _, err := ParseEntryIndex("abc"); err == nil {
			t.Error("expected error for abc")
		}
	})
	t.Run("empty rejected", func(t *testing.T) {
		if _, err := ParseEntryIndex(""); err == nil {
			t.Error("expected error for empty string")
		}
	})
}

// Package state holds clibo's runtime state: the last session ID and
// the per-sid HMAC cookie cache.
//
// state.json is auto-managed; users should not edit it by hand. The
// cookies it stores are HMAC-signed write tokens issued by the server.
// They are not key material, but they authorise writes to the session
// — so the file is written with mode 0600.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/rm-info/online_clipboard_cli/internal/paths"
)

// Cookie is a cached server-issued auth cookie with its expiry.
type Cookie struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"` // unix seconds, per the server's Set-Cookie max-age
}

// State is the on-disk runtime state.
type State struct {
	LastSID string            `json:"last_sid,omitempty"`
	Cookies map[string]Cookie `json:"cookies,omitempty"`
}

// Load reads state from its canonical XDG location. A missing file
// yields an empty State and no error — first run is normal.
func Load() (*State, error) {
	p, err := paths.StateFile()
	if err != nil {
		return nil, err
	}
	return LoadFile(p)
}

// LoadFile reads state from an explicit path.
func LoadFile(path string) (*State, error) {
	s := &State{Cookies: map[string]Cookie{}}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state: %w", err)
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	if s.Cookies == nil {
		s.Cookies = map[string]Cookie{}
	}
	return s, nil
}

// Save writes state to its canonical XDG location with mode 0600.
func Save(s *State) error {
	p, err := paths.StateFile()
	if err != nil {
		return err
	}
	return SaveFile(p, s)
}

// SaveFile writes state to an explicit path with mode 0600.
func SaveFile(path string, s *State) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	return paths.AtomicWriteFile(path, data, 0o600)
}

// SetCookie stores (or replaces) the cookie for sid and updates LastSID.
func (s *State) SetCookie(sid, token string, expiresAt int64) {
	if s.Cookies == nil {
		s.Cookies = map[string]Cookie{}
	}
	s.Cookies[sid] = Cookie{Token: token, ExpiresAt: expiresAt}
}

// GetCookie returns the cookie for sid (and a present-ok flag).
func (s *State) GetCookie(sid string) (Cookie, bool) {
	c, ok := s.Cookies[sid]
	return c, ok
}

// ForgetSession removes the cookie for sid and clears LastSID if it
// pointed there. Call this on wipe success or on a 404 from the server.
func (s *State) ForgetSession(sid string) {
	delete(s.Cookies, sid)
	if s.LastSID == sid {
		s.LastSID = ""
	}
}

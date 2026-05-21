// Package flow orchestrates the CLI commands: it composes crypto,
// proto and state into higher-level operations (create session,
// list entries, append text, upload a file, etc.). Subcommand
// handlers call into this package; they do not call crypto or proto
// directly.
package flow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rm-info/online_clipboard_cli/internal/crypto"
	"github.com/rm-info/online_clipboard_cli/internal/proto"
	"github.com/rm-info/online_clipboard_cli/internal/state"
)

// PasswordFunc is invoked lazily when a password is needed. The
// hasPassword flag comes from /verifier so the closure can short-
// circuit (return "" for sentinel) on passwordless sessions.
type PasswordFunc func(hasPassword bool) (string, error)

// Auth handles authentication for a single session across the lifetime
// of one CLI invocation. It memoises the verifier blob, derived key
// and cookie so multiple flow operations on the same session don't
// repeat work.
type Auth struct {
	Client      *proto.Client
	State       *state.State
	GetPassword PasswordFunc

	verifier *proto.VerifierResponse
	key      []byte
	cookie   string
	sid      string
}

// Cookie returns a valid clip_token_<sid> for sid, doing the full
// password handshake (verifier → key → proof → PoW → /auth) if no
// cached cookie applies. Updates state in-memory; caller saves state.
func (a *Auth) Cookie(ctx context.Context, sid string) (string, error) {
	if a.cookie != "" && a.sid == sid {
		return a.cookie, nil
	}
	if c, ok := a.State.GetCookie(sid); ok && c.ExpiresAt > time.Now().Unix() {
		a.cookie = c.Token
		a.sid = sid
		return a.cookie, nil
	}
	if err := a.handshake(ctx, sid); err != nil {
		return "", err
	}
	return a.cookie, nil
}

// Key returns the AES-256 key derived from the password for sid.
// Forces a /verifier fetch + Argon2id on first call (we never persist
// keys on disk).
func (a *Auth) Key(ctx context.Context, sid string) ([]byte, error) {
	if a.key != nil && a.sid == sid {
		return a.key, nil
	}
	if err := a.ensureVerifier(ctx, sid); err != nil {
		return nil, err
	}
	pw, err := a.GetPassword(a.verifier.HasPassword)
	if err != nil {
		return nil, err
	}
	key, err := crypto.DeriveKey(pw, a.verifier.Salt)
	if err != nil {
		return nil, fmt.Errorf("derive key: %w", err)
	}
	a.key = key
	a.sid = sid
	return key, nil
}

func (a *Auth) ensureVerifier(ctx context.Context, sid string) error {
	if a.verifier != nil && a.sid == sid {
		return nil
	}
	v, err := a.Client.GetVerifier(ctx, sid)
	if err != nil {
		// Stale cached state for a gone session: clean up so the next
		// run doesn't trip on it.
		if errors.Is(err, proto.ErrSessionNotFound) || errors.Is(err, proto.ErrSessionLocked) {
			a.State.ForgetSession(sid)
		}
		return err
	}
	a.verifier = v
	return nil
}

func (a *Auth) handshake(ctx context.Context, sid string) error {
	key, err := a.Key(ctx, sid)
	if err != nil {
		return err
	}
	proof, err := crypto.AuthProof(a.verifier.Verifier, key)
	if err != nil {
		// AEAD failure here is the wrong-password signal.
		return ErrWrongPassword
	}
	ch, err := a.Client.FetchChallenge(ctx)
	if err != nil {
		return fmt.Errorf("fetch PoW challenge: %w", err)
	}
	nonce, err := crypto.SolvePoW(ch.Challenge, ch.Difficulty)
	if err != nil {
		return fmt.Errorf("solve PoW: %w", err)
	}
	resp, err := a.Client.Authenticate(ctx, sid, proof, ch.Challenge, nonce)
	if err != nil {
		if errors.Is(err, proto.ErrSessionNotFound) || errors.Is(err, proto.ErrSessionLocked) {
			a.State.ForgetSession(sid)
		}
		return err
	}
	a.cookie = resp.Cookie
	a.State.SetCookie(sid, resp.Cookie, resp.ExpiresAt)
	a.State.LastSID = sid
	return nil
}

// ErrWrongPassword surfaces a verifier-decrypt failure (i.e. a wrong
// password) as a flow-level sentinel.
var ErrWrongPassword = errors.New("wrong password")

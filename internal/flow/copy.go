package flow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rm-info/online_clipboard_cli/internal/crypto"
	"github.com/rm-info/online_clipboard_cli/internal/proto"
)

// CreateSession creates a brand-new session whose first item is the
// given text. Returns the new session ID; updates state.LastSID and
// caches the cookie. On a fresh session GetPassword is called once
// with hasPassword=true if and only if password is non-empty — the
// CLI layer encodes that intent through the password it provides.
//
// The text must be non-empty (the server rejects empty first items).
func (a *Auth) CreateSession(ctx context.Context, text string, password string, secureMode, secret bool) (sid string, err error) {
	if text == "" {
		return "", errors.New("first item cannot be empty")
	}
	key, saltB64, err := crypto.NewSaltedKey(password)
	if err != nil {
		return "", err
	}
	verifier, err := crypto.NewVerifier(key)
	if err != nil {
		return "", err
	}
	firstItemCT, err := crypto.EncryptText(text, key)
	if err != nil {
		return "", fmt.Errorf("encrypt first item: %w", err)
	}
	ch, err := a.Client.FetchChallenge(ctx)
	if err != nil {
		return "", fmt.Errorf("fetch PoW challenge: %w", err)
	}
	nonce, err := crypto.SolvePoW(ch.Challenge, ch.Difficulty)
	if err != nil {
		return "", fmt.Errorf("solve PoW: %w", err)
	}

	resp, err := a.Client.CreateSession(ctx, &proto.CreateSessionRequest{
		FirstItemCT:   firstItemCT,
		FirstItemSize: len(text),
		HasPassword:   password != "",
		Salt:          saltB64,
		VerifierBlob:  verifier.Blob,
		AuthAnchor:    verifier.AuthAnchor,
		POWChallenge:  ch.Challenge,
		POWNonce:      nonce,
		SecureMode:    secureMode,
		Secret:        secret,
	})
	if err != nil {
		return "", err
	}

	// Cache everything for any follow-up operation in the same process.
	a.sid = resp.SID
	a.key = key
	a.verifier = &proto.VerifierResponse{
		Verifier:    verifier.Blob,
		Salt:        saltB64,
		HasPassword: password != "",
	}
	a.cookie = resp.Cookie
	a.State.SetCookie(resp.SID, resp.Cookie, resp.ExpiresAt)
	a.State.LastSID = resp.SID

	return resp.SID, nil
}

// AddText encrypts text under the session's key and appends it.
// Authenticates lazily if no fresh cookie is cached.
func (a *Auth) AddText(ctx context.Context, sid, text string, secret bool) error {
	if text == "" {
		return errors.New("text cannot be empty")
	}
	key, err := a.Key(ctx, sid)
	if err != nil {
		return err
	}
	cookie, err := a.Cookie(ctx, sid)
	if err != nil {
		return err
	}
	ct, err := crypto.EncryptText(text, key)
	if err != nil {
		return fmt.Errorf("encrypt text: %w", err)
	}
	if err := a.Client.AddItem(ctx, sid, cookie, ct, len(text), secret); err != nil {
		return a.handleSessionErr(sid, err)
	}
	a.State.LastSID = sid
	return nil
}

// AddFile reads filePath, encrypts both the file body and the
// filename under the session's key, and uploads them.
func (a *Auth) AddFile(ctx context.Context, sid, filePath string) (*proto.UploadedFile, error) {
	contents, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filePath, err)
	}
	key, err := a.Key(ctx, sid)
	if err != nil {
		return nil, err
	}
	cookie, err := a.Cookie(ctx, sid)
	if err != nil {
		return nil, err
	}
	encryptedBody, err := crypto.EncryptBytes(contents, key)
	if err != nil {
		return nil, fmt.Errorf("encrypt file: %w", err)
	}
	encryptedName, err := crypto.EncryptText(filepath.Base(filePath), key)
	if err != nil {
		return nil, fmt.Errorf("encrypt filename: %w", err)
	}
	out, err := a.Client.UploadFile(ctx, sid, cookie, encryptedName, len(contents), encryptedBody, nil)
	if err != nil {
		return nil, a.handleSessionErr(sid, err)
	}
	a.State.LastSID = sid
	return out, nil
}

// handleSessionErr is the common cleanup: if the server says the
// session is gone, drop our stale cookie + last_sid so the next run
// doesn't loop.
func (a *Auth) handleSessionErr(sid string, err error) error {
	if errors.Is(err, proto.ErrSessionNotFound) || errors.Is(err, proto.ErrSessionLocked) {
		a.State.ForgetSession(sid)
	}
	return err
}

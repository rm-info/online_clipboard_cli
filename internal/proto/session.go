package proto

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// CreateSessionRequest groups every form field of POST /. The crypto
// layer produces FirstItemCT, Salt, VerifierBlob and AuthAnchor; the
// PoW solver produces POWChallenge and POWNonce.
type CreateSessionRequest struct {
	FirstItemCT   string
	FirstItemSize int
	HasPassword   bool
	Salt          string
	VerifierBlob  string
	AuthAnchor    string
	POWChallenge  string
	POWNonce      string
	SecureMode    bool
	Secret        bool
}

// CreateSessionResponse carries the new session ID and the cookie
// (with its absolute expiry, unix seconds) that authenticates further
// writes.
type CreateSessionResponse struct {
	SID       string
	Cookie    string
	ExpiresAt int64
}

func (c *Client) CreateSession(ctx context.Context, in *CreateSessionRequest) (*CreateSessionResponse, error) {
	form := url.Values{
		"first_item_ct":   {in.FirstItemCT},
		"first_item_size": {strconv.Itoa(in.FirstItemSize)},
		"has_password":    {boolStr(in.HasPassword)},
		"salt":            {in.Salt},
		"verifier_blob":   {in.VerifierBlob},
		"auth_anchor":     {in.AuthAnchor},
		"pow_challenge":   {in.POWChallenge},
		"pow_nonce":       {in.POWNonce},
		"secure_mode":     {boolStr(in.SecureMode)},
		"secret":          {boolStr(in.Secret)},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/"), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST /: %w", err)
	}
	defer resp.Body.Close()
	if err := standardStatusErrors(resp); err != nil {
		return nil, err
	}

	var body struct {
		OK       bool   `json:"ok"`
		SID      string `json:"sid"`
		Redirect string `json:"redirect"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode create-session response: %w", err)
	}
	if !body.OK || body.SID == "" {
		return nil, errors.New("server returned ok=false on session creation")
	}

	cookie, expiresAt := extractCookie(resp, body.SID)
	return &CreateSessionResponse{
		SID:       body.SID,
		Cookie:    cookie,
		ExpiresAt: expiresAt,
	}, nil
}

// VerifierResponse is the public auth blob served by GET /{sid}/verifier.
type VerifierResponse struct {
	Verifier    string `json:"verifier"`
	Salt        string `json:"salt"`
	HasPassword bool   `json:"has_password"`
}

func (c *Client) GetVerifier(ctx context.Context, sid string) (*VerifierResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url("/"+sid+"/verifier"), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET /%s/verifier: %w", sid, err)
	}
	defer resp.Body.Close()
	if err := standardStatusErrors(resp); err != nil {
		return nil, err
	}
	v := &VerifierResponse{}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return nil, fmt.Errorf("decode verifier: %w", err)
	}
	return v, nil
}

// AuthResponse is what Authenticate returns on success.
type AuthResponse struct {
	Cookie    string
	ExpiresAt int64
}

// Authenticate POSTs the auth_proof + PoW to /{sid}/auth.
// On 401 the error is ErrWrongPassword. The server tracks failure
// counts and will lock the session after SESSION_MAX_FAILED_ATTEMPTS;
// that surfaces as ErrSessionLocked here.
func (c *Client) Authenticate(ctx context.Context, sid, authProof, powChallenge, powNonce string) (*AuthResponse, error) {
	form := url.Values{
		"auth_proof":    {authProof},
		"pow_challenge": {powChallenge},
		"pow_nonce":     {powNonce},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/"+sid+"/auth"), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST /%s/auth: %w", sid, err)
	}
	defer resp.Body.Close()

	// 401 here means "wrong password" specifically (not "no cookie") —
	// /auth doesn't gate on a cookie, only on the auth_proof matching.
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrWrongPassword
	}
	if err := standardStatusErrors(resp); err != nil {
		return nil, err
	}
	cookie, expiresAt := extractCookie(resp, sid)
	return &AuthResponse{Cookie: cookie, ExpiresAt: expiresAt}, nil
}

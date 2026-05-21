// Package proto is the thin HTTP client for online_clipboard. It
// reads and writes opaque ciphertext: no method here knows what a key
// is, what a password is, or how Argon2id works. The crypto layer
// composes with this one via the flow package.
package proto

import (
	"net/http"
	"strings"
	"time"
)

// Client speaks the online_clipboard HTTP wire protocol. It is
// stateless apart from the base URL and the embedded http.Client;
// each authenticated method takes the per-sid cookie as an argument.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient builds a Client targeting baseURL. The HTTP client is
// configured to NOT follow redirects: the only redirecting route is
// POST /{sid}/wipe (303 → /) and we'd rather see the 303 ourselves
// than chase it.
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (c *Client) url(path string) string {
	return c.BaseURL + path
}

// setCookie attaches the per-sid HMAC write token to req. No-op if
// value is empty so unauthenticated routes can share the helper.
func setCookie(req *http.Request, sid, value string) {
	if value == "" {
		return
	}
	req.AddCookie(&http.Cookie{Name: "clip_token_" + sid, Value: value})
}

// extractCookie pulls the clip_token_<sid> Set-Cookie out of resp and
// computes its absolute expiry time (unix seconds). Returns ("", 0)
// if no cookie was set.
func extractCookie(resp *http.Response, sid string) (value string, expiresAt int64) {
	name := "clip_token_" + sid
	for _, c := range resp.Cookies() {
		if c.Name != name {
			continue
		}
		value = c.Value
		if !c.Expires.IsZero() {
			expiresAt = c.Expires.Unix()
		} else if c.MaxAge > 0 {
			expiresAt = time.Now().Add(time.Duration(c.MaxAge) * time.Second).Unix()
		}
		return value, expiresAt
	}
	return "", 0
}

// boolStr renders a bool as the server expects ("true" / "false").
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

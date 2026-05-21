package proto

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// EntryType is the discriminator returned by /contents for each entry.
type EntryType string

const (
	EntryText EntryType = "text"
	EntryFile EntryType = "file"
)

// Entry is one item in the session's ordered list. Ciphertext is set
// for text entries; EncryptedName is set for files. Both are AES-GCM
// "n:c" tokens the caller decrypts locally.
type Entry struct {
	ID            string    `json:"id"`
	Type          EntryType `json:"type"`
	Ciphertext    string    `json:"ciphertext,omitempty"`
	EncryptedName string    `json:"encrypted_name,omitempty"`
	Size          int64     `json:"size"`
	CreatedAt     int64     `json:"created_at,omitempty"`
	UploadedAt    int64     `json:"uploaded_at,omitempty"`
	Secret        bool      `json:"secret,omitempty"`
	HasThumb      bool      `json:"has_thumb,omitempty"`
}

// Contents is the response of GET /{sid}/contents.
//
// ExpiresAt is the unix-second timestamp at which the session will expire
// on the server, sliding on writes only. Distinct from the cookie's expiry
// (which slides on every authenticated request) — display this for "session
// expires in X", not the cookie expiry. Zero if the server is older than
// 2.4.3 and didn't include the field.
type Contents struct {
	Entries   []Entry `json:"entries"`
	FileBytes int64   `json:"file_bytes"`
	ExpiresAt int64   `json:"expires_at,omitempty"`
}

func (c *Client) GetContents(ctx context.Context, sid, cookie string) (*Contents, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url("/"+sid+"/contents"), nil)
	if err != nil {
		return nil, err
	}
	setCookie(req, sid, cookie)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET /%s/contents: %w", sid, err)
	}
	defer resp.Body.Close()
	if err := standardStatusErrors(resp); err != nil {
		return nil, err
	}
	out := &Contents{}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, fmt.Errorf("decode contents: %w", err)
	}
	return out, nil
}

func (c *Client) AddItem(ctx context.Context, sid, cookie, ciphertext string, plainSize int, secret bool) error {
	form := url.Values{
		"ciphertext": {ciphertext},
		"plain_size": {strconv.Itoa(plainSize)},
		"secret":     {boolStr(secret)},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/"+sid+"/items"), strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	setCookie(req, sid, cookie)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST /%s/items: %w", sid, err)
	}
	defer resp.Body.Close()
	return standardStatusErrors(resp)
}

func (c *Client) DeleteItem(ctx context.Context, sid, cookie, itemID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/"+sid+"/items/"+itemID+"/delete"), nil)
	if err != nil {
		return err
	}
	setCookie(req, sid, cookie)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST /%s/items/%s/delete: %w", sid, itemID, err)
	}
	defer resp.Body.Close()
	return standardStatusErrors(resp)
}

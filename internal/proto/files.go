package proto

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
)

// UploadedFile is the per-file metadata returned by POST /{sid}/upload.
type UploadedFile struct {
	ID            string `json:"id"`
	EncryptedName string `json:"encrypted_name"`
	Size          int64  `json:"size"`
	UploadedAt    int64  `json:"uploaded_at"`
	HasThumb      bool   `json:"has_thumb"`
}

// UploadFile sends a ciphertext file (nonce||ct) + the AES-GCM token
// of the original filename. Thumb is optional and may be nil; clibo
// v0.1 never generates thumbnails so this is here for API
// completeness rather than current use.
func (c *Client) UploadFile(ctx context.Context, sid, cookie, encryptedName string, plainSize int, ciphertext, thumb []byte) (*UploadedFile, error) {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	if err := w.WriteField("encrypted_name", encryptedName); err != nil {
		return nil, err
	}
	if err := w.WriteField("plain_size", strconv.Itoa(plainSize)); err != nil {
		return nil, err
	}
	filePart, err := w.CreateFormFile("file", "file.bin")
	if err != nil {
		return nil, err
	}
	if _, err := filePart.Write(ciphertext); err != nil {
		return nil, err
	}
	if thumb != nil {
		thumbPart, err := w.CreateFormFile("thumb", "thumb.bin")
		if err != nil {
			return nil, err
		}
		if _, err := thumbPart.Write(thumb); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/"+sid+"/upload"), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	setCookie(req, sid, cookie)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST /%s/upload: %w", sid, err)
	}
	defer resp.Body.Close()
	if err := standardStatusErrors(resp); err != nil {
		return nil, err
	}

	var envelope struct {
		OK   bool          `json:"ok"`
		File *UploadedFile `json:"file"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode upload response: %w", err)
	}
	if !envelope.OK || envelope.File == nil {
		return nil, errors.New("server returned ok=false on upload")
	}
	return envelope.File, nil
}

// DownloadFile fetches the encrypted file body and the encrypted
// filename token. The filename is URL-safe base64 of the AES-GCM
// token (see main.py's safe_name_header); we decode it here so the
// caller gets the same shape it gave to UploadFile.
func (c *Client) DownloadFile(ctx context.Context, sid, cookie, fileID string) (encryptedName string, ciphertext []byte, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url("/"+sid+"/files/"+fileID), nil)
	if err != nil {
		return "", nil, err
	}
	setCookie(req, sid, cookie)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("GET /%s/files/%s: %w", sid, fileID, err)
	}
	defer resp.Body.Close()
	if err := standardStatusErrors(resp); err != nil {
		return "", nil, err
	}

	header := resp.Header.Get("X-Clip-Encrypted-Name")
	if header == "" {
		return "", nil, errors.New("missing X-Clip-Encrypted-Name header")
	}
	decoded, err := base64.URLEncoding.DecodeString(header)
	if err != nil {
		return "", nil, fmt.Errorf("decode X-Clip-Encrypted-Name: %w", err)
	}
	encryptedName = string(decoded)

	ciphertext, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("read file body: %w", err)
	}
	return encryptedName, ciphertext, nil
}

func (c *Client) DeleteFile(ctx context.Context, sid, cookie, fileID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/"+sid+"/files/"+fileID+"/delete"), nil)
	if err != nil {
		return err
	}
	setCookie(req, sid, cookie)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST /%s/files/%s/delete: %w", sid, fileID, err)
	}
	defer resp.Body.Close()
	return standardStatusErrors(resp)
}

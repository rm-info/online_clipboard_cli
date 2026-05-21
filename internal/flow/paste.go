package flow

import (
	"context"
	"errors"
	"fmt"

	"github.com/rm-info/online_clipboard_cli/internal/crypto"
	"github.com/rm-info/online_clipboard_cli/internal/proto"
	"github.com/rm-info/online_clipboard_cli/internal/ui"
)

// DecryptedEntry is one item of a session's contents with its
// ciphertext fields decrypted. Index is 1-based in the order returned
// by GET /contents.
type DecryptedEntry struct {
	Index     int
	ID        string
	Type      proto.EntryType
	Text      string // populated when Type == EntryText
	Filename  string // populated when Type == EntryFile (decrypted name)
	Size      int64
	Timestamp int64 // CreatedAt for text, UploadedAt for file
	Secret    bool
	HasThumb  bool
}

// ListResult bundles the session-level data returned by List alongside the
// decrypted entries. ExpiresAt is the server-authoritative session expiry
// (unix seconds); zero if the server didn't include it (pre-2.4.3).
type ListResult struct {
	Entries   []DecryptedEntry
	FileBytes int64
	ExpiresAt int64
}

// List fetches and decrypts every entry in the session.
func (a *Auth) List(ctx context.Context, sid string) (*ListResult, error) {
	key, err := a.Key(ctx, sid)
	if err != nil {
		return nil, err
	}
	cookie, err := a.Cookie(ctx, sid)
	if err != nil {
		return nil, err
	}
	contents, err := a.Client.GetContents(ctx, sid, cookie)
	if err != nil {
		return nil, a.handleSessionErr(sid, err)
	}
	a.State.LastSID = sid

	out := make([]DecryptedEntry, 0, len(contents.Entries))
	for i, e := range contents.Entries {
		de := DecryptedEntry{
			Index:    i + 1,
			ID:       e.ID,
			Type:     e.Type,
			Size:     e.Size,
			Secret:   e.Secret,
			HasThumb: e.HasThumb,
		}
		switch e.Type {
		case proto.EntryText:
			de.Timestamp = e.CreatedAt
			pt, err := crypto.DecryptText(e.Ciphertext, key)
			if err != nil {
				return nil, fmt.Errorf("decrypt entry %d: %w", i+1, err)
			}
			de.Text = pt
		case proto.EntryFile:
			de.Timestamp = e.UploadedAt
			name, err := crypto.DecryptText(e.EncryptedName, key)
			if err != nil {
				return nil, fmt.Errorf("decrypt filename for entry %d: %w", i+1, err)
			}
			de.Filename = name
		}
		out = append(out, de)
	}
	return &ListResult{
		Entries:   out,
		FileBytes: contents.FileBytes,
		ExpiresAt: contents.ExpiresAt,
	}, nil
}

// PastePayload is what GetEntry returns: either decrypted text or a
// decrypted file blob plus its original filename.
type PastePayload struct {
	IsFile   bool
	Text     string // populated if !IsFile
	Filename string // populated if IsFile
	Bytes    []byte // populated if IsFile
	Secret   bool
}

// GetEntry resolves spec against the session's contents (fetching them
// once if needed) and returns the decrypted payload. For file entries
// it also performs the binary download.
//
// Returns ErrEntryNotFound if the index is out of range or the
// session has no entries.
func (a *Auth) GetEntry(ctx context.Context, sid string, spec ui.EntryIndexSpec) (*PastePayload, error) {
	res, err := a.List(ctx, sid)
	if err != nil {
		return nil, err
	}
	if len(res.Entries) == 0 {
		return nil, ErrEntryNotFound
	}

	idx, err := ResolveIndex(spec, len(res.Entries))
	if err != nil {
		return nil, err
	}
	target := res.Entries[idx]

	switch target.Type {
	case proto.EntryText:
		return &PastePayload{
			IsFile: false,
			Text:   target.Text,
			Secret: target.Secret,
		}, nil
	case proto.EntryFile:
		key, err := a.Key(ctx, sid)
		if err != nil {
			return nil, err
		}
		cookie, err := a.Cookie(ctx, sid)
		if err != nil {
			return nil, err
		}
		_, ciphertext, err := a.Client.DownloadFile(ctx, sid, cookie, target.ID)
		if err != nil {
			return nil, a.handleSessionErr(sid, err)
		}
		pt, err := crypto.DecryptBytes(ciphertext, key)
		if err != nil {
			return nil, fmt.Errorf("decrypt file: %w", err)
		}
		return &PastePayload{
			IsFile:   true,
			Filename: target.Filename,
			Bytes:    pt,
		}, nil
	}
	return nil, fmt.Errorf("unknown entry type %q", target.Type)
}

// ErrEntryNotFound is returned by GetEntry when the requested entry
// index falls outside the session's current contents.
var ErrEntryNotFound = errors.New("entry not found")

// ResolveIndex turns an EntryIndexSpec into a 0-based array index
// against a list of `length` entries. Returns ErrEntryNotFound for
// out-of-range inputs.
func ResolveIndex(spec ui.EntryIndexSpec, length int) (int, error) {
	if length == 0 {
		return 0, ErrEntryNotFound
	}
	if spec.Last {
		return length - 1, nil
	}
	if spec.Index < 1 || spec.Index > length {
		return 0, ErrEntryNotFound
	}
	return spec.Index - 1, nil
}

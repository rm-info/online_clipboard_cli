package flow

import (
	"context"

	"github.com/rm-info/online_clipboard_cli/internal/proto"
)

// DeleteEntry removes a single already-resolved entry. The caller is
// responsible for selecting the right one (via List + ResolveIndex);
// taking the entry rather than a spec lets CLI commands show a
// pre-delete confirmation without paying for a second /contents call.
func (a *Auth) DeleteEntry(ctx context.Context, sid string, e *DecryptedEntry) error {
	cookie, err := a.Cookie(ctx, sid)
	if err != nil {
		return err
	}
	switch e.Type {
	case proto.EntryText:
		return a.handleSessionErr(sid, a.Client.DeleteItem(ctx, sid, cookie, e.ID))
	case proto.EntryFile:
		return a.handleSessionErr(sid, a.Client.DeleteFile(ctx, sid, cookie, e.ID))
	}
	return nil
}

// Wipe destroys the session server-side and clears its cached state.
// Cookie-only: no key derivation needed.
func (a *Auth) Wipe(ctx context.Context, sid string) error {
	cookie, err := a.Cookie(ctx, sid)
	if err != nil {
		return err
	}
	if err := a.Client.Wipe(ctx, sid, cookie); err != nil {
		return a.handleSessionErr(sid, err)
	}
	a.State.ForgetSession(sid)
	return nil
}

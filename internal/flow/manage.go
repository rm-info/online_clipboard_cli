package flow

import (
	"context"

	"github.com/rm-info/online_clipboard_cli/internal/proto"
	"github.com/rm-info/online_clipboard_cli/internal/ui"
)

// DeleteEntry removes the entry pointed to by spec. Text vs file is
// resolved by looking up the entry in /contents first.
func (a *Auth) DeleteEntry(ctx context.Context, sid string, spec ui.EntryIndexSpec) error {
	entries, _, err := a.List(ctx, sid)
	if err != nil {
		return err
	}
	idx, err := resolveIndex(spec, len(entries))
	if err != nil {
		return err
	}
	target := entries[idx]

	cookie, err := a.Cookie(ctx, sid)
	if err != nil {
		return err
	}
	switch target.Type {
	case proto.EntryText:
		return a.handleSessionErr(sid, a.Client.DeleteItem(ctx, sid, cookie, target.ID))
	case proto.EntryFile:
		return a.handleSessionErr(sid, a.Client.DeleteFile(ctx, sid, cookie, target.ID))
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

package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/rm-info/online_clipboard_cli/internal/config"
	"github.com/rm-info/online_clipboard_cli/internal/flow"
	"github.com/rm-info/online_clipboard_cli/internal/proto"
	"github.com/rm-info/online_clipboard_cli/internal/state"
	"github.com/rm-info/online_clipboard_cli/internal/ui"
)

// cliContext bundles everything a subcommand needs to run a single
// authenticated operation: ctx, server client, loaded state, and an
// Auth handle wired to read passwords through the right resolver.
type cliContext struct {
	Ctx    context.Context
	Client *proto.Client
	State  *state.State
	Auth   *flow.Auth
}

// newCLIContext loads config + state, resolves the server URL, and
// returns a ready-to-use context. Callers should defer ctx.Save() so
// any cookie or last_sid mutation is persisted.
func newCLIContext(ctx context.Context) (*cliContext, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	serverURL, err := config.ResolveServerURL(cfg, Flags.Server)
	if err != nil {
		return nil, err
	}
	st, err := state.Load()
	if err != nil {
		return nil, err
	}
	client := proto.NewClient(serverURL)
	return &cliContext{
		Ctx:    ctx,
		Client: client,
		State:  st,
		Auth: &flow.Auth{
			Client:      client,
			State:       st,
			GetPassword: existingSessionPasswordFunc(),
		},
	}, nil
}

// Save persists any state mutations made during the command.
func (c *cliContext) Save() error {
	return state.Save(c.State)
}

// existingSessionPasswordFunc returns a closure used when the flow
// layer asks "the user needs to provide the session's password". It
// short-circuits to the sentinel for passwordless sessions.
func existingSessionPasswordFunc() flow.PasswordFunc {
	return func(hasPassword bool) (string, error) {
		if !hasPassword || Flags.NoPassword {
			return "", nil
		}
		if Flags.Password != "" {
			return Flags.Password, nil
		}
		if env := os.Getenv("CLIP_PASSWORD"); env != "" {
			return env, nil
		}
		return ui.PromptPassword("Password: ")
	}
}

// newSessionPassword resolves the password for a brand-new session.
// Unlike the existing-session resolver this one always prompts the
// user (empty answer = no password) unless an explicit flag/env
// short-circuits it.
func newSessionPassword() (string, error) {
	if Flags.NoPassword {
		return "", nil
	}
	if Flags.Password != "" {
		return Flags.Password, nil
	}
	if env := os.Getenv("CLIP_PASSWORD"); env != "" {
		return env, nil
	}
	return ui.PromptPassword("New session password (empty for none): ")
}

// shortMsg renders a flow/proto error as a user-friendly one-liner.
// Currently a thin wrapper over err.Error(); kept as a single
// chokepoint so we can polish messages without hunting through all
// the subcommands later.
func shortMsg(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", err)
}

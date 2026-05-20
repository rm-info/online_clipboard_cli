# online_clipboard CLI — brief for the implementation session

This is a fresh repo whose only purpose is to ship a small, single-binary CLI
client for the existing **online_clipboard** end-to-end-encrypted clipboard
service. The implementation hasn't started; this file captures the context
that led here so a new session can pick up cleanly.

## Why a CLI exists

The web app at `https://clipboard.lab.rm-info.fr` (deployed on belzebold,
source at `~/homelab/projects/online_clipboard/`, repo
`git@github.com:rm-info/online_clipboard.git`) is fully E2EE in v2: keys
are derived in the browser via Argon2id, payloads are AES-256-GCM, the server
stores opaque ciphertext only. The browser is the universal client — but for
ops-on-server workflows ("I SSH'd into a box, I want to ship the output to my
laptop") the browser is in the way.

Goal: enable shell-pipe usage from any server or laptop.
`journalctl … | clip send` on one host, `clip recv <url> | grep ERROR` on
another, without a browser in the middle.

## Why a binary and not a curl-friendly bash script

curl can do the HTTP (POSTs, multipart, cookies) but not the crypto.
Argon2id and AES-GCM aren't in shell builtins. OpenSSL 3.2+ has Argon2id
but isn't universal (LibreSSL on macOS, older Linux distros). PBKDF2
would be universal but fragments the protocol vs the browser and weakens
GPU resistance.

Decision (validated with the user in the prior session): ship a **single
static binary** per platform, hosted at `https://clipboard.lab.rm-info.fr/cli/<arch>`
(plus eventually GitHub releases). User does:
```sh
curl -L https://clipboard.lab.rm-info.fr/cli/linux-amd64 -o /usr/local/bin/clip && chmod +x /usr/local/bin/clip
```
Two commands, no package manager, no runtime, then `clip send` works.

## Why CLI, not TUI

TUI breaks pipes. The 90% use case here is one-shot send/receive in shell
pipelines and scripts. A full-screen browser (`clip browse <url>`) might
land later as an additive subcommand if usage warrants — not before.

## Tentative subcommand sketch (open to revision)

```
clip send  [-p PWD] [TEXT]       # stdin or arg → POST → print URL
clip recv  URL [-p PWD]          # all text entries → stdout, newline-separated
clip ls    URL [-p PWD]          # numbered listing (text + files)
clip get   URL N [-p PWD]        # fetch entry N → stdout
clip put   URL FILE [-p PWD]     # upload file
clip pull  URL N -o FILE         # download + decrypt file
clip last                        # print the last sid created on this host
```

The user has reserved the right to refine this set during the
implementation session. Don't treat it as locked.

## What the binary must do (protocol contract)

The CLI must reproduce the exact crypto + handshake the browser performs.
Source of truth lives in the online_clipboard repo:

- **Crypto**: `~/homelab/projects/online_clipboard/app/static/js/clip-crypto.js`
  - Argon2id params: `t=2, m=64 MiB, p=2, hashLength=32` (must match exactly)
  - AES-256-GCM, 12-byte nonce, base64-encoded token format `"<nonce_b64>:<ct_b64>"`
  - **Empty-password sentinel**: when password is empty, substitute `"\x00"`
    before feeding Argon2id (hash-wasm rejects empty strings — see fix in
    v2.1.3). Server doesn't care about the sentinel value as long as the
    client and browser agree.
  - PoW: SHA-256 with `POW_DIFFICULTY_BITS` (currently 18) leading zero bits
    over `challenge + ":" + nonce`

- **Routes**: `~/homelab/projects/online_clipboard/app/main.py`
  - `GET  /pow/challenge` → `{challenge, difficulty}` (single-use)
  - `POST /` → create session (form: `first_item_ct`, `first_item_size`,
    `has_password`, `salt`, `verifier_blob`, `auth_anchor`,
    `pow_challenge`, `pow_nonce`, `secure_mode`, `secret`). Returns JSON
    `{ok, sid, redirect}` + sets `clip_token_<sid>` httponly cookie.
  - `GET  /{sid}/verifier` → `{verifier, salt, has_password}` (public)
  - `POST /{sid}/auth` → form: `auth_proof`, `pow_challenge`, `pow_nonce`.
    Returns JSON + sets `clip_token_<sid>` cookie on success.
  - `GET  /{sid}/contents` → JSON `{entries: [...], file_bytes: N}` (auth required)
  - `POST /{sid}/items` → add ciphertext (form: `ciphertext`, `plain_size`, `secret`)
  - `POST /{sid}/upload` → multipart: `file` (ciphertext bytes), `encrypted_name`,
    `plain_size`, optional `thumb` (ciphertext bytes)
  - `GET  /{sid}/files/{file_id}` → binary ciphertext + `X-Clip-Encrypted-Name` header
  - `POST /{sid}/items/{item_id}/delete`, `POST /{sid}/files/{file_id}/delete`
  - `POST /{sid}/wipe`

- **Data model**: `~/homelab/projects/online_clipboard/app/session.py`
  - `entries` items have shape `{id, type: "text"|"file", ciphertext|encrypted_name, size, created_at|uploaded_at, secret, has_thumb}`
  - Text entry: decrypt `entry.ciphertext` with key
  - File entry: decrypt `entry.encrypted_name` for display; GET `/{sid}/files/{id}` for content

- **Auth handshake** (matches the browser path):
  1. Fetch `/{sid}/verifier` → get `{verifier, salt}`
  2. `key = Argon2id(password || "\x00", salt)`
  3. Decrypt `verifier` with key → `verifier_plaintext`
  4. `auth_proof = sha256_hex(verifier_plaintext)`
  5. Solve a PoW from `/pow/challenge`
  6. `POST /{sid}/auth` with `auth_proof + pow_challenge + pow_nonce`
  7. Server returns JSON + sets cookie

- **Verifier blob format** (for new sessions):
  - Generate 16 random bytes, hex-encode them → `verifier_plaintext`
  - `verifier_blob = AES-GCM-encrypt(verifier_plaintext, key)`
  - `auth_anchor = sha256_hex(verifier_plaintext)`
  - Salt is `base64(16 random bytes)`

- **README + security model**: `~/homelab/projects/online_clipboard/README.md`
  is the canonical reference for the security boundaries; the CLI should keep
  those promises (no IP logging, ciphertext-only server state, etc.).

## Open implementation questions for the new session

- **Language**: Go is the front-runner — first-class cross-compile, easy
  CGO-free Argon2id via `golang.org/x/crypto/argon2`, small binaries. Rust
  works too but more friction. Decide first thing.
- **Local state**: where to store `clip last` history. Suggestion:
  `~/.config/clip/state.json` (XDG-compliant), simple JSON list of recent
  sids + URLs.
- **Cookie persistence**: between subcommands, the signed token cookie has
  to survive. Same `~/.config/clip/cookies.json` or per-sid file.
- **Server URL**: hard-coded `https://clipboard.lab.rm-info.fr` for now, or
  configurable via env (`CLIP_SERVER`) / flag (`--server`)? Default
  config-file backed, fallback to env, fallback to flag is the conventional
  layering.
- **Password handling**: `-p PWD` on the command line leaks via shell history
  and `ps`. Better: `CLIP_PASSWORD` env var, or interactive prompt via TTY
  read. Treat the `-p` flag as a footgun unless really wanted.

## Distribution plan (consensus from prior session, revisable)

1. `Makefile` with cross-compile targets: `make build` produces
   `dist/clip-{linux,darwin,windows}-{amd64,arm64}` binaries
2. Upload binaries to GitHub releases on tag push (manual at first, GH
   Actions later if release cadence justifies)
3. Add a `/cli/<platform>` route to the existing `online_clipboard` server
   that redirects (or proxies) to the latest GitHub release artifact for
   the requested architecture. So `curl -L clipboard.lab.rm-info.fr/cli/linux-amd64`
   always gets the latest.
4. Bonus: install script at `clipboard.lab.rm-info.fr/install.sh` that
   detects the platform and drops `clip` into `~/.local/bin/`.

## Reading order for the implementation session

1. **This file** (`BRIEF.md`)
2. **`~/homelab/projects/online_clipboard/README.md`** — security model + endpoint list
3. **`~/homelab/projects/online_clipboard/app/static/js/clip-crypto.js`** — exact crypto primitives + PoW algorithm
4. **`~/homelab/projects/online_clipboard/app/main.py`** — route signatures + form fields
5. **`~/homelab/projects/online_clipboard/app/session.py`** — data shapes returned by `/contents`

The server is at `v2.1.4` as of this brief (commit `3587aa6` on `main`).
Verify with `curl https://clipboard.lab.rm-info.fr/healthz` before
starting if it matters — protocol changes would require updating this
file.

## Out of scope for v0.1.0

- TUI browser (`clip browse`)
- Update mechanism (`clip self-update`)
- Multi-server config (one server URL is enough for now)
- Image thumb regeneration locally (the CLI won't generate thumbs; uploads
  go without them, matching the v1 → v2 spirit where thumbs are nice but
  optional)

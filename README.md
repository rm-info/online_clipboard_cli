# clibo

End-to-end encrypted clipboard CLI for the
[online_clipboard](https://github.com/rm-info/online_clipboard) service.

`clibo` is a single static binary that lets you share text and files
between machines over the same E2EE protocol the browser client speaks.
The server stores opaque ciphertext only; passwords and keys never leave
your local machine.

```
$ journalctl -u nginx -n 50 | clibo copy new
abc12
# … on another machine …
$ clibo paste abc12 | grep ERROR
```

## Why this exists

The web app at `https://clipboard.lab.rm-info.fr` is fully end-to-end
encrypted: keys are derived in your browser via Argon2id, payloads
are AES-256-GCM, and the server holds only ciphertext. That's great
for human-to-human transfers but breaks down for ops workflows where
your data is already at a terminal — `clibo` adds the same protocol
to your shell, with the same security guarantees.

## Install

### One-liner (Linux & macOS)

```sh
curl -sSL https://clipboard.lab.rm-info.fr/install.sh | sh
```

Detects your platform, drops the binary in `/usr/local/bin/clibo`. Override
the target with `CLIBO_BIN=~/.local/bin/clibo` to avoid `sudo`, or point at
a self-hosted instance with `CLIBO_BASE=https://your.clipboard.tld`.

### Prebuilt binary (manual)

The server hosts a redirect that always points to the latest release for
your platform:

```sh
curl -L https://clipboard.lab.rm-info.fr/cli/linux-amd64 -o /usr/local/bin/clibo
chmod +x /usr/local/bin/clibo
```

Supported targets: `linux-amd64`, `linux-arm64`, `darwin-amd64`,
`darwin-arm64`, `windows-amd64.exe`. The discovery page at
[`/cli`](https://clipboard.lab.rm-info.fr/cli) lists them all with direct
download links.

### From source

Requires Go ≥ 1.25 (the toolchain auto-downloads it if missing).

```sh
git clone https://github.com/rm-info/online_clipboard_cli
cd online_clipboard_cli
make install                       # → ~/.local/bin/clibo
# or: make dist                    # → dist/clibo-{os}-{arch} for every release target
# or: go install ./cmd/clibo       # → $(go env GOPATH)/bin/clibo
```

## Quick start

```sh
# One-time setup
clibo config set server_url https://clipboard.lab.rm-info.fr

# Create a session from a pipe (and remember it locally as "last")
echo "hello from machine A" | clibo copy new
# → AbC12  (prints just the session ID on stdout)

# On the receiving side, paste either by SID or by 'last'
clibo paste AbC12
# → hello from machine A
```

`clibo last` prints the session ID of the most recent operation, so you
can chain commands without copy-pasting:

```sh
echo "more logs" | clibo copy last     # append to the session you just made
clibo ls                                # see what's in it
clibo paste last 2                      # entry 2 of that session
```

## Subcommand reference

Global flags (apply to every subcommand):

| Flag | Purpose |
|---|---|
| `--server URL` | Override the configured server for this invocation only |
| `-p, --password PWD` | Pass the password inline (leaks via `ps` and shell history — prefer the prompt) |
| `--no-password` | Skip the prompt and use the empty-password sentinel (for scripts) |
| `--version` | Print version + commit |

### `clibo copy new [TEXT]`

Create a new session. The first entry is `TEXT` if given, otherwise
stdin (which must be a pipe). Prints the new session ID on stdout.

```sh
clibo copy new "one-liner"
echo "from pipe" | clibo copy new
```

Flags:

- `--secure` — use a 50-character session ID (brute-force-proof URL)
- `--secret` — mark the first entry as secret (shoulder-surfing protection in the web UI)

When no `--password` / `--no-password` / `CLIP_PASSWORD` is provided,
`clibo` prompts on the TTY for a password (empty answer = passwordless).

### `clibo copy <SID|last> [TEXT]`

Append text (or, with `-f`, a file) to an existing session.

```sh
echo "log line" | clibo copy last
clibo copy abc12 "shorter to type than piping"
clibo copy last -f /tmp/report.pdf      # upload a file
```

Flags:

- `-f, --file` — treat the positional as a file path (upload), not text

### `clibo paste [SID|last] [N|last]`

Fetch and decrypt a single entry. Positionals are detected by shape:

- A 5- or 50-character string is the **session ID**.
- An integer is the **1-based entry index**.
- The literal `last` fills whichever slot is still empty (SID first, then N).
- Either position defaults to `last` if omitted.

```sh
clibo paste                       # last session, last entry
clibo paste 3                     # last session, entry 3
clibo paste abc12                 # session abc12, last entry
clibo paste abc12 2               # session abc12, entry 2
clibo paste 2 abc12               # same as above — order doesn't matter
```

Output rules:

| Case | Behaviour |
|---|---|
| Text entry, no `-f` | → stdout |
| Text entry, `-f PATH` | → write to `PATH` (confirms overwrite unless `-y`) |
| File entry, no `-f` | → write to `$PWD/<original-filename>` |
| File entry, `-f PATH` | → write to `PATH` |
| Any case, `-f -` | → force stdout (binary may corrupt your terminal) |

Flags:

- `-f, --file PATH` — output target (or `-` for stdout)
- `-y, --yes` — skip the overwrite confirmation

### `clibo ls [SID|last]`

List every entry in the session with a one-line text preview or the
decrypted filename, plus size and relative age. The header line shows
the session ID, cookie expiry (from the local cache, no network
roundtrip), and the total file bytes used.

```sh
clibo ls                # last session
clibo ls abc12          # explicit
clibo ls -s             # header only — useful in scripts
```

### `clibo del <SID|last> <N|last> [-y]`

Delete a single entry. Both positionals are required — destructive
commands accept no defaults. The target is summarised before
confirmation; pass `-y` to skip the prompt.

```sh
clibo del abc12 3        # delete entry 3 from abc12 (with confirmation)
clibo del last last -y   # delete the last entry of the last session, no confirm
```

### `clibo wipe <SID|last> [-y]`

Destroy the session entirely server-side. Same destructive-confirm
convention as `del`. Clears the cached cookie and `last_sid` locally.

```sh
clibo wipe last -y
```

### `clibo config <sub>`

Manage persistent configuration.

```sh
clibo config set <key> <value>      # set a key
clibo config get <key>              # read a key
clibo config unset <key>            # clear a key
clibo config list                   # show every recognised key
clibo config path                   # print where the config file lives
clibo config edit                   # open the config file in $EDITOR (fallback: vi)
```

For v0.1 the only configurable key is `server_url`.

### `clibo last`

Print the session ID stored as `last_sid`. Exits non-zero if no last
session is known. Useful for scripting:

```sh
SID=$(clibo last)
clibo del $SID 3
```

### `clibo status`

Print the local context (server, last session, cached cookie expiry).
Makes no network calls.

## Configuration

Two files, each in their canonical XDG location.

### `~/.config/clibo/config.toml` (Linux)

User-editable. Currently a single key:

```toml
# clibo configuration
# Edit by hand or via `clibo config set <key> <value>`.

server_url = "https://clipboard.lab.rm-info.fr"
```

On macOS the same file lives under `~/Library/Application Support/clibo/`;
on Windows under `%AppData%\clibo\`.

### `~/.local/share/clibo/state.json` (Linux)

Auto-managed. Holds `last_sid` and the per-session HMAC cookie cache.
Mode `0600` — the cookies are write tokens for sessions you've
authenticated to. Do not edit by hand.

On macOS / Windows this file shares the directory with `config.toml`.

### Layering for the server URL

Resolved in this order:

1. `--server URL` flag (one-off override)
2. `CLIP_SERVER` environment variable
3. `server_url` in `config.toml`
4. error: *"no server URL configured (run: clibo config set server_url <URL>)"*

## Security model

`clibo` reproduces the
[online_clipboard server's security model](https://github.com/rm-info/online_clipboard#security-model-v2--e2ee)
verbatim on the client side:

- **Argon2id** with `t=2, m=64 MiB, p=2, hashLength=32` derives the AES-256 key
  from your password and a per-session salt.
- **AES-256-GCM** with a fresh 12-byte nonce encrypts every text item, every
  file, and every filename. The server only ever holds the ciphertext.
- **Empty passwords** are substituted with the byte `\x00` before key
  derivation (matching the browser's hash-wasm fix). A "passwordless"
  session is one whose URL alone is enough to derive the key.
- **Auth handshake**: re-authentication submits `sha256(decrypt(verifier))`,
  which only a holder of the right key can compute. The key never crosses
  the wire.
- **Proof-of-work** (SHA-256 with 18 leading zero bits, ~50–200ms locally)
  is required on session creation and re-auth, matching the server.
- **Cookies on disk**: the HMAC-signed write tokens the server issues are
  cached at `state.json` mode `0600`. They carry no key material; they
  authorise writes to a specific session for its TTL.
- **No password on disk**: clibo never writes your password anywhere.
  Within one invocation the derived key sits in memory; between
  invocations only the cookie is preserved.

### What clibo does **not** protect against

- **A compromised local machine.** If something has code-execution access
  to your user account, it can read the cached cookies and prompt you for
  the password. Same threat model as `gpg`, `ssh-agent`, etc.
- **A compromised binary.** Static binaries are easier to verify than
  served JavaScript but are still trusted code. Build from source if it
  matters; the toolchain is reproducible (`-trimpath` is enabled in the
  Makefile).
- **Lost passwords.** The server cannot reset what it cannot read. Use a
  real password manager.

## Limits

Inherited from the server (see its README for the full list):

- Max text entry: **500 KB**
- Max file per upload: **100 MiB**
- Max files per session: **1 GiB**
- Session TTL: **2 hours of inactivity** (any write extends it)
- Failed-auth lockout: 20 wrong attempts → session locked forever

## Examples

### One-shot transfer

```sh
# Machine A
journalctl -u nginx -n 200 | clibo --no-password copy new
# → "abC12"
```

```sh
# Machine B
clibo --no-password paste abC12 | grep -i error
```

### Authenticated multi-step workflow

```sh
clibo -p hunter2 copy new "incident timeline:"           # creates session
clibo -p hunter2 copy last "10:42 — first alert"          # appends
clibo -p hunter2 copy last -f /tmp/heap-dump.bin          # uploads file
clibo -p hunter2 ls                                       # check what's there
clibo -p hunter2 paste last 1                             # read entry 1
clibo -p hunter2 wipe last -y                             # done — destroy
```

(With prompt-driven password entry instead of `-p hunter2`, the password
is asked once per invocation when the cached cookie is gone — i.e. roughly
once every 2 hours.)

### Pipe a file through

```sh
clibo paste abc12 2 -f - | sha256sum     # binary file → stdout → checksum
```

### Status check

```sh
clibo status
# server: https://clipboard.lab.rm-info.fr
# last:   abc12
#         auth cached, expires in 1h 47min (local clock)
```

## Troubleshooting

**"no server URL configured"** — run `clibo config set server_url <URL>` once.

**"wrong password"** — the password didn't decrypt the verifier blob.
clibo validates locally before hitting the server, so this never burns
a server-side failed-auth counter slot. Just retry.

**"session not found"** — the session expired (2h inactivity) or was
wiped. clibo cleans the corresponding cookie from `state.json`
automatically when it sees this from the server.

**"session locked"** — too many failed-auth attempts (20 by default).
The session is permanently locked; you cannot recover it.

**`clibo` prompts for a password in a script** — pass `--password`,
set `CLIP_PASSWORD`, or use `--no-password` for sessions that
shouldn't need one.

**Binary file dumped into my terminal** — `paste` defaults to
`$PWD/<filename>` for files; use `-f -` to force stdout intentionally.
If you redirected stdout to a file or pipe, no harm done.

## Project structure

```
cmd/clibo/                          # main: 5-line entry point
internal/
├── cli/                            # cobra command tree (one file per subcommand)
├── config/                         # TOML config (load/save/get/set/unset/list)
├── state/                          # JSON runtime state (last_sid + cookie cache)
├── paths/                          # XDG locations + AtomicWriteFile helper
├── crypto/                         # Argon2id, AES-GCM, SHA-256 PoW, verifier
├── proto/                          # HTTP client — ciphertext only, no crypto here
├── flow/                           # Orchestration: crypto + proto + state
├── ui/                             # Password prompt, pipe/TTY detection, arg parsing
└── version/                        # Build-time -ldflags target
```

Layer boundary: `cli/` calls `flow/`; `flow/` composes `crypto/` + `proto/` + `state/`;
`proto/` never sees a key, `crypto/` never sees a network. That's the
property that makes the whole thing testable.


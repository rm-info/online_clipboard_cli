// Package crypto holds the wire-format primitives clibo uses to talk
// to the online_clipboard server: Argon2id key derivation, AES-256-GCM
// envelopes (both text-token "n:c" form and raw nonce||ct binary
// form), SHA-256 proof-of-work, and the verifier handshake.
//
// All values match clip-crypto.js byte-for-byte. Changing any
// parameter here breaks compatibility with the browser client and
// with already-created sessions.
package crypto

import (
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

const (
	argon2Iterations  = 2
	argon2MemoryKB    = 64 * 1024
	argon2Parallelism = 2
	argon2HashLength  = 32

	// EmptyPasswordSentinel is the fixed byte clip-crypto.js substitutes
	// for a missing password (hash-wasm refuses empty inputs). Any client
	// that wants to interoperate with a passwordless session must apply
	// the same substitution before calling Argon2id. See clip-crypto.js
	// v2.1.3 for the original fix.
	EmptyPasswordSentinel = "\x00"
)

// DeriveKey runs Argon2id over (password OR sentinel) and the
// server-provided salt and returns the 32-byte AES-256 key.
func DeriveKey(password string, saltB64 string) ([]byte, error) {
	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return nil, fmt.Errorf("decode salt: %w", err)
	}
	if len(salt) < 8 {
		return nil, errors.New("salt too short (need >= 8 bytes)")
	}
	pw := password
	if pw == "" {
		pw = EmptyPasswordSentinel
	}
	return argon2.IDKey(
		[]byte(pw),
		salt,
		argon2Iterations,
		argon2MemoryKB,
		argon2Parallelism,
		argon2HashLength,
	), nil
}

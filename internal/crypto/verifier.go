package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// Verifier is what POST / consumes: an encrypted blob plus the
// SHA-256 hex of its plaintext. The server stores both and returns
// the blob (+ salt) on GET /{sid}/verifier so a returning client can
// re-derive the key and prove possession by decrypting locally and
// comparing the hash.
type Verifier struct {
	Blob       string // AES-GCM(verifier_plaintext, key) in "n:c" text format
	AuthAnchor string // sha256_hex(verifier_plaintext)
}

// NewSaltedKey generates a fresh 16-byte salt (base64) and derives
// the AES-256 key from (password OR sentinel) and that salt via
// Argon2id. Returned salt is submitted at session creation; returned
// key encrypts the session payload.
func NewSaltedKey(password string) (key []byte, saltB64 string, err error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, "", fmt.Errorf("salt: %w", err)
	}
	saltB64 = base64.StdEncoding.EncodeToString(salt)
	key, err = DeriveKey(password, saltB64)
	if err != nil {
		return nil, "", err
	}
	return key, saltB64, nil
}

// NewVerifier produces a fresh random verifier plaintext (16 random
// bytes, hex-encoded), encrypts it under key, and returns the blob
// plus the SHA-256 anchor. The browser client uses the exact same
// shape — that's why interop works.
func NewVerifier(key []byte) (*Verifier, error) {
	vpBytes := make([]byte, 16)
	if _, err := rand.Read(vpBytes); err != nil {
		return nil, fmt.Errorf("verifier plaintext: %w", err)
	}
	vp := hex.EncodeToString(vpBytes)

	blob, err := EncryptText(vp, key)
	if err != nil {
		return nil, fmt.Errorf("encrypt verifier: %w", err)
	}
	sum := sha256.Sum256([]byte(vp))
	return &Verifier{
		Blob:       blob,
		AuthAnchor: hex.EncodeToString(sum[:]),
	}, nil
}

// AuthProof decrypts blob with key and returns sha256_hex of the
// resulting plaintext. AEAD failure here is the password-check
// primitive — if the user typed the wrong password, the Open call
// fails and the caller surfaces that as "wrong password".
func AuthProof(blob string, key []byte) (string, error) {
	vp, err := DecryptText(blob, key)
	if err != nil {
		return "", fmt.Errorf("verifier did not decrypt (likely wrong password): %w", err)
	}
	sum := sha256.Sum256([]byte(vp))
	return hex.EncodeToString(sum[:]), nil
}

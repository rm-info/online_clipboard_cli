package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// NonceBytes is the AES-GCM nonce length used by the server protocol.
// 12 bytes is the GCM standard and is what clip-crypto.js produces.
const NonceBytes = 12

// EncryptText encrypts a UTF-8 string into the server's text token
// format: "<nonce_b64>:<ct_b64>". Used for first_item_ct, items
// ciphertext, encrypted_name, and verifier blobs.
func EncryptText(plaintext string, key []byte) (string, error) {
	nonce, ct, err := sealRaw([]byte(plaintext), key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(nonce) + ":" +
		base64.StdEncoding.EncodeToString(ct), nil
}

// DecryptText reverses EncryptText. Returns an error on malformed
// tokens, base64 errors, or AEAD authentication failure (wrong key).
func DecryptText(token string, key []byte) (string, error) {
	sep := strings.IndexByte(token, ':')
	if sep < 0 {
		return "", errors.New("invalid token format (missing ':')")
	}
	nonce, err := base64.StdEncoding.DecodeString(token[:sep])
	if err != nil {
		return "", fmt.Errorf("decode nonce: %w", err)
	}
	ct, err := base64.StdEncoding.DecodeString(token[sep+1:])
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	pt, err := openRaw(nonce, ct, key)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

// EncryptBytes encrypts a binary payload as nonce || ciphertext. This
// is the on-the-wire shape of file uploads and downloads.
func EncryptBytes(plaintext, key []byte) ([]byte, error) {
	nonce, ct, err := sealRaw(plaintext, key)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(nonce)+len(ct))
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

// DecryptBytes splits the nonce||ct blob and recovers the plaintext.
func DecryptBytes(blob, key []byte) ([]byte, error) {
	if len(blob) <= NonceBytes {
		return nil, errors.New("binary blob too short to contain a nonce")
	}
	return openRaw(blob[:NonceBytes], blob[NonceBytes:], key)
}

func sealRaw(pt, key []byte) (nonce, ct []byte, err error) {
	aead, err := newGCM(key)
	if err != nil {
		return nil, nil, err
	}
	nonce = make([]byte, NonceBytes)
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("nonce: %w", err)
	}
	ct = aead.Seal(nil, nonce, pt, nil)
	return nonce, ct, nil
}

func openRaw(nonce, ct, key []byte) ([]byte, error) {
	aead, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(nonce) != NonceBytes {
		return nil, fmt.Errorf("nonce must be %d bytes, got %d", NonceBytes, len(nonce))
	}
	return aead.Open(nil, nonce, ct, nil)
}

func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	return cipher.NewGCM(block)
}

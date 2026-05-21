package crypto

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------
// DeriveKey
// ---------------------------------------------------------------------

func TestDeriveKey_Deterministic(t *testing.T) {
	salt := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef"))
	k1, err := DeriveKey("hunter2", salt)
	if err != nil {
		t.Fatalf("DeriveKey #1: %v", err)
	}
	k2, err := DeriveKey("hunter2", salt)
	if err != nil {
		t.Fatalf("DeriveKey #2: %v", err)
	}
	if !bytes.Equal(k1, k2) {
		t.Errorf("same password+salt should yield same key")
	}
	if len(k1) != 32 {
		t.Errorf("key length: got %d want 32", len(k1))
	}
}

func TestDeriveKey_DifferentSaltsDifferKey(t *testing.T) {
	s1 := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef"))
	s2 := base64.StdEncoding.EncodeToString([]byte("fedcba9876543210"))
	k1, _ := DeriveKey("pw", s1)
	k2, _ := DeriveKey("pw", s2)
	if bytes.Equal(k1, k2) {
		t.Errorf("different salts should yield different keys")
	}
}

func TestDeriveKey_EmptyPasswordUsesSentinel(t *testing.T) {
	salt := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef"))
	kEmpty, err := DeriveKey("", salt)
	if err != nil {
		t.Fatalf("DeriveKey empty: %v", err)
	}
	kSentinel, err := DeriveKey(EmptyPasswordSentinel, salt)
	if err != nil {
		t.Fatalf("DeriveKey sentinel: %v", err)
	}
	if !bytes.Equal(kEmpty, kSentinel) {
		t.Errorf("empty password should produce same key as sentinel")
	}
}

func TestDeriveKey_RejectsShortSalt(t *testing.T) {
	if _, err := DeriveKey("pw", base64.StdEncoding.EncodeToString([]byte("short"))); err == nil {
		t.Error("expected error on short salt, got nil")
	}
}

// ---------------------------------------------------------------------
// AEAD round-trips
// ---------------------------------------------------------------------

func freshKey(t *testing.T) []byte {
	t.Helper()
	salt := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef"))
	k, err := DeriveKey("test-pw", salt)
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}
	return k
}

func TestEncryptDecryptText_Roundtrip(t *testing.T) {
	key := freshKey(t)
	for _, plain := range []string{"hello world", "", "with utf-8: 日本語 ✨", strings.Repeat("a", 4096)} {
		tok, err := EncryptText(plain, key)
		if err != nil {
			t.Fatalf("EncryptText(%q): %v", plain, err)
		}
		if !strings.Contains(tok, ":") {
			t.Errorf("token missing ':' separator: %q", tok)
		}
		got, err := DecryptText(tok, key)
		if err != nil {
			t.Fatalf("DecryptText: %v", err)
		}
		if got != plain {
			t.Errorf("round-trip mismatch: got %q want %q", got, plain)
		}
	}
}

func TestDecryptText_WrongKeyFails(t *testing.T) {
	key := freshKey(t)
	tok, _ := EncryptText("secret", key)

	// Modify the key slightly.
	bad := append([]byte{}, key...)
	bad[0] ^= 0xFF
	if _, err := DecryptText(tok, bad); err == nil {
		t.Error("DecryptText with wrong key: expected error, got nil")
	}
}

func TestDecryptText_MalformedToken(t *testing.T) {
	key := freshKey(t)
	if _, err := DecryptText("no-colon", key); err == nil {
		t.Error("expected error on token without ':'")
	}
	if _, err := DecryptText("not-base64!:foo", key); err == nil {
		t.Error("expected error on bad base64")
	}
}

func TestEncryptDecryptBytes_Roundtrip(t *testing.T) {
	key := freshKey(t)
	payload := bytes.Repeat([]byte{0xAB, 0xCD}, 1024)

	blob, err := EncryptBytes(payload, key)
	if err != nil {
		t.Fatalf("EncryptBytes: %v", err)
	}
	if len(blob) < NonceBytes+len(payload) {
		t.Errorf("blob too short: got %d want >= %d", len(blob), NonceBytes+len(payload))
	}

	got, err := DecryptBytes(blob, key)
	if err != nil {
		t.Fatalf("DecryptBytes: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("binary roundtrip mismatch")
	}
}

func TestDecryptBytes_ShortBlobFails(t *testing.T) {
	if _, err := DecryptBytes([]byte("short"), make([]byte, 32)); err == nil {
		t.Error("expected error on too-short blob")
	}
}

// ---------------------------------------------------------------------
// PoW
// ---------------------------------------------------------------------

func TestSolvePoW_ResultMeetsDifficulty(t *testing.T) {
	// 12 bits ~ 4096 attempts on average — fast and reliable for tests.
	const difficulty = 12
	const challenge = "test-challenge-abcdef"

	nonce, err := SolvePoW(challenge, difficulty)
	if err != nil {
		t.Fatalf("SolvePoW: %v", err)
	}
	sum := sha256.Sum256([]byte(challenge + ":" + nonce))
	if got := leadingZeroBits(sum[:]); got < difficulty {
		t.Errorf("SolvePoW result has only %d leading zero bits, want >= %d (nonce=%q)", got, difficulty, nonce)
	}
}

func TestSolvePoW_RejectsInvalidDifficulty(t *testing.T) {
	if _, err := SolvePoW("c", 0); err == nil {
		t.Error("expected error on difficulty=0")
	}
	if _, err := SolvePoW("c", 257); err == nil {
		t.Error("expected error on difficulty>256")
	}
}

func TestLeadingZeroBits(t *testing.T) {
	cases := []struct {
		in   []byte
		want int
	}{
		{[]byte{0xFF}, 0},
		{[]byte{0x7F}, 1},
		{[]byte{0x00, 0xFF}, 8},
		{[]byte{0x00, 0x00, 0x10}, 19},
		{[]byte{0x00, 0x00, 0x00, 0x00}, 32},
	}
	for _, c := range cases {
		if got := leadingZeroBits(c.in); got != c.want {
			t.Errorf("leadingZeroBits(%x) = %d, want %d", c.in, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------
// Verifier handshake
// ---------------------------------------------------------------------

func TestNewSaltedKey_ProducesDifferentSaltsAcrossCalls(t *testing.T) {
	_, s1, err := NewSaltedKey("pw")
	if err != nil {
		t.Fatalf("NewSaltedKey #1: %v", err)
	}
	_, s2, err := NewSaltedKey("pw")
	if err != nil {
		t.Fatalf("NewSaltedKey #2: %v", err)
	}
	if s1 == s2 {
		t.Errorf("two NewSaltedKey calls produced identical salts: %s", s1)
	}
}

func TestVerifier_RoundtripMatchesAnchor(t *testing.T) {
	key, _, err := NewSaltedKey("pw")
	if err != nil {
		t.Fatalf("NewSaltedKey: %v", err)
	}
	v, err := NewVerifier(key)
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}

	proof, err := AuthProof(v.Blob, key)
	if err != nil {
		t.Fatalf("AuthProof: %v", err)
	}
	if proof != v.AuthAnchor {
		t.Errorf("auth proof != anchor:\n  proof  = %s\n  anchor = %s", proof, v.AuthAnchor)
	}

	// Anchor is sha256 hex of the verifier plaintext — sanity-check the shape.
	if len(v.AuthAnchor) != 64 {
		t.Errorf("anchor length: got %d want 64 (sha256 hex)", len(v.AuthAnchor))
	}
	if _, err := hex.DecodeString(v.AuthAnchor); err != nil {
		t.Errorf("anchor is not valid hex: %v", err)
	}
}

func TestAuthProof_WrongKeyFails(t *testing.T) {
	key, _, _ := NewSaltedKey("pw")
	v, _ := NewVerifier(key)

	wrongKey, _, _ := NewSaltedKey("other-pw")
	if _, err := AuthProof(v.Blob, wrongKey); err == nil {
		t.Error("AuthProof with wrong key: expected error, got nil")
	}
}

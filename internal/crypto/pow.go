package crypto

import (
	"crypto/sha256"
	"fmt"
	"strconv"
)

// SolvePoW finds a nonce (decimal-encoded counter, starting at 0)
// such that SHA-256(challenge || ":" || nonce) has at least
// `difficulty` leading zero bits. Returns the nonce as a string so it
// can be posted verbatim — the server compares the raw bytes.
//
// At the server's default difficulty (18 bits), this takes ~50–200ms
// single-threaded on a modern CPU. Higher values scale exponentially.
func SolvePoW(challenge string, difficulty int) (string, error) {
	if difficulty <= 0 || difficulty > 256 {
		return "", fmt.Errorf("invalid difficulty %d (must be in 1..256)", difficulty)
	}

	prefix := []byte(challenge + ":")
	// One growable buffer reused across iterations so we don't allocate
	// in the hot loop (262k iterations at 18 bits adds up otherwise).
	buf := make([]byte, 0, len(prefix)+24)

	for nonce := uint64(0); ; nonce++ {
		buf = append(buf[:0], prefix...)
		buf = strconv.AppendUint(buf, nonce, 10)
		sum := sha256.Sum256(buf)
		if leadingZeroBits(sum[:]) >= difficulty {
			return strconv.FormatUint(nonce, 10), nil
		}
	}
}

func leadingZeroBits(b []byte) int {
	bits := 0
	for _, byt := range b {
		if byt == 0 {
			bits += 8
			continue
		}
		for mask := byte(0x80); mask != 0; mask >>= 1 {
			if byt&mask != 0 {
				return bits
			}
			bits++
		}
		return bits
	}
	return bits
}

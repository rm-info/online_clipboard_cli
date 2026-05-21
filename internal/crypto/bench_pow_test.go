package crypto

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkSolvePoW_18Bits exercises the server-default difficulty so
// we get a feel for how slow the auth handshake is on the host CPU.
// Skipped by default (no `-bench`); run with:  go test -bench=PoW ./internal/crypto
func BenchmarkSolvePoW_18Bits(b *testing.B) {
	for i := 0; i < b.N; i++ {
		nonce, err := SolvePoW(fmt.Sprintf("bench-%d", i), 18)
		if err != nil {
			b.Fatal(err)
		}
		_ = nonce
	}
}

// TestSolvePoW_18BitsTiming logs the wall-clock cost at 18 bits so we
// catch a regression if someone accidentally pessimises the inner
// loop. Bounded at 2 seconds — generous given local measurements
// come in well under that.
func TestSolvePoW_18BitsTiming(t *testing.T) {
	if testing.Short() {
		t.Skip("18-bit PoW takes too long for -short")
	}
	start := time.Now()
	nonce, err := SolvePoW("timing-test", 18)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("SolvePoW: %v", err)
	}
	t.Logf("PoW 18 bits: %v (nonce=%s)", elapsed, nonce)
	if elapsed > 2*time.Second {
		t.Errorf("PoW 18 bits took %v, want < 2s", elapsed)
	}
}

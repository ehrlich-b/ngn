package main

import (
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestQueueClockArithmetic uses the real watchdog and the harness's float64
// parsing, with unbounded references for every integer/duration intermediate.
// It launches no engines. Python RecipeTests pin the emitted -maxmoves to 400.
func TestQueueClockArithmetic(t *testing.T) {
	const maxMoves = 400
	// Pin the production parsing and UCI expressions as well as the watchdog
	// call below: changes to clock semantics must update this proof explicitly.
	for path, snippets := range map[string][]string{
		"main.go": {"strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)", "tcTimeMs = int(base * 1000)",
			"strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)", "tcIncMs = int(inc * 1000)"},
		"../../internal/uci/uci.go": {"whiteMs, blackMs := cfg.TCTimeMs, cfg.TCTimeMs",
			"*rem -= int(time.Since(moveStart).Milliseconds())", "*rem += cfg.TCIncMs",
			"timeout = time.Duration(myMs)*time.Millisecond + 30*time.Second",
			"stopAfter = time.Duration(myMs)*time.Millisecond + tcFlagGrace",
			"const tcFlagGrace = 300 * time.Millisecond"},
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, snippet := range snippets {
			if !strings.Contains(string(data), snippet) {
				t.Fatalf("%s: clock semantics changed: missing %q", path, snippet)
			}
		}
	}
	integer := func(n int64) *big.Int { return big.NewInt(n) }
	add := func(a, b *big.Int) *big.Int { return new(big.Int).Add(a, b) }
	mul := func(a *big.Int, b int64) *big.Int { return new(big.Int).Mul(a, integer(b)) }
	div := func(a *big.Int, b int64) *big.Int { return new(big.Int).Quo(a, integer(b)) }
	check := func(t *testing.T, name string, native int64, want *big.Int, bits uint) {
		t.Helper()
		limit := new(big.Int).Sub(new(big.Int).Lsh(integer(1), bits-1), integer(1))
		if want.Sign() < 0 || want.Cmp(limit) > 0 || integer(native).Cmp(want) != 0 {
			t.Fatalf("%s: native=%d reference=%s signed-%d limit=%s", name, native, want, bits, limit)
		}
	}
	parse := func(text string) (int, *big.Int) {
		seconds, err := strconv.ParseFloat(text, 64)
		if err != nil {
			t.Fatal(err)
		}
		// Capture the actual float64 product, then truncate it without a native
		// integer bound. This preserves Go rounding, including fractional ms.
		ms := seconds * 1000
		ref, _ := new(big.Float).SetFloat64(ms).Int(nil)
		return int(ms), ref
	}
	for _, base := range []string{"0.001", "0.00100000000000000000001", "0.0019", "1", "60", "86399.999999999999", "86400"} {
		for _, inc := range []string{"0", "0.000000000000001", "0.0001", "0.1", "0.6", "3599.999999999999", "3600"} {
			t.Run(base+"+"+inc, func(t *testing.T) {
				baseMs, baseRef := parse(base)
				incMs, incRef := parse(inc)
				check(t, "baseMs", int64(baseMs), baseRef, 32)
				check(t, "incMs", int64(incMs), incRef, 32)
				product := incMs * maxMoves
				productRef := mul(incRef, maxMoves)
				check(t, "incMs*400", int64(product), productRef, 32)
				check(t, "incMs*400/2", int64(product/2), div(productRef, 2), 32)
				perSide := baseMs + product/2
				perSideRef := add(baseRef, div(productRef, 2))
				check(t, "perSideMs", int64(perSide), perSideRef, 32)
				check(t, "2*perSideMs", int64(2*perSide), mul(perSideRef, 2), 32)
				game := time.Duration(2*perSide) * time.Millisecond
				gameRef := mul(perSideRef, 2*int64(time.Millisecond))
				check(t, "game ns", int64(game), gameRef, 64)
				check(t, "4*game ns", int64(4*game), mul(gameRef, 4), 64)
				watchRef := add(mul(gameRef, 4), integer(int64(time.Minute)))
				check(t, "4*game+minute ns", int64(4*game+time.Minute), watchRef, 64)
				if watchRef.Cmp(integer(int64(3*time.Minute))) < 0 {
					watchRef = integer(int64(3 * time.Minute))
				}
				watch := watchdogDeadline(runConfig{tcTimeMs: baseMs, tcIncMs: incMs, maxMoves: maxMoves})
				check(t, "real watchdog ns", int64(watch), watchRef, 64)
				// Zero elapsed time maximizes the UCI bank. Any completed move
				// deducts nonnegative elapsed ms before its Fischer increment.
				bank := baseMs
				bankRef := new(big.Int).Set(baseRef)
				for move := 0; move <= maxMoves/2; move++ {
					check(t, "UCI bank ms", int64(bank), bankRef, 32)
					check(t, "UCI bank duration", int64(time.Duration(bank)*time.Millisecond), mul(bankRef, int64(time.Millisecond)), 64)
					for _, grace := range []time.Duration{300 * time.Millisecond, 30 * time.Second} {
						check(t, "UCI timeout/grace ns", int64(time.Duration(bank)*time.Millisecond+grace),
							add(mul(bankRef, int64(time.Millisecond)), integer(int64(grace))), 64)
					}
					if move < maxMoves/2 {
						bank += incMs
						bankRef = add(bankRef, incRef)
					}
				}
				if base == "86400" && inc == "3600" {
					if product != 1440000000 || perSide != 806400000 || 2*perSide != 1612800000 || watch != 6451260000000000 {
						t.Fatal("simultaneous supported maximum changed")
					}
					t.Logf("maximum: incMs*400=%d perSideMs=%d 2*perSideMs=%d watchdog=%d ns", product, perSide, 2*perSide, watch)
				}
				if base == "0.001" && inc == "0" && watch != 3*time.Minute {
					t.Fatal("three-minute floor changed")
				}
			})
		}
	}
}

package engine

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// Texel tuning: fit the static evaluation's parameters (the piece-square tables)
// to real game outcomes by minimizing the mean squared error between each
// position's game result and a logistic of its static eval. evaluateUnsafe is a
// pure white-perspective score (the property the symmetry test guards), so we
// score it directly against results stored from White's point of view.

// TexelSample is one labeled position: a board parsed once into memory, and the
// game result from White's perspective (1.0 = White won, 0.5 = draw, 0.0 = lost).
type TexelSample struct {
	Board  Bitboard
	Result float64
}

// texelSigmoid maps a white-perspective centipawn eval to an expected score in
// (0,1) via the standard base-10 logistic with scaling constant K.
func texelSigmoid(eval, k float64) float64 {
	return 1.0 / (1.0 + math.Pow(10, -k*eval/400.0))
}

// texelMSE is the mean squared error between game results and the sigmoid of the
// static (white-perspective) eval over the dataset, at scaling constant K. It is
// parallelized across CPUs and only READS the eval parameters, so callers must
// not mutate the PSTs concurrently (coordinate descent mutates between calls).
func texelMSE(samples []TexelSample, k float64) float64 {
	n := len(samples)
	if n == 0 {
		return 0
	}
	workers := runtime.GOMAXPROCS(0)
	if workers > n {
		workers = n
	}
	chunk := (n + workers - 1) / workers
	partial := make([]float64, workers)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		lo := w * chunk
		if lo >= n {
			break
		}
		hi := lo + chunk
		if hi > n {
			hi = n
		}
		wg.Add(1)
		go func(idx, lo, hi int) {
			defer wg.Done()
			var sum float64
			for i := lo; i < hi; i++ {
				e := float64(evaluateUnsafe(&samples[i].Board))
				d := samples[i].Result - texelSigmoid(e, k)
				sum += d * d
			}
			partial[idx] = sum
		}(w, lo, hi)
	}
	wg.Wait()
	var total float64
	for _, s := range partial {
		total += s
	}
	return total / float64(n)
}

// TexelFindK ternary-searches for the scaling constant K that minimizes MSE with
// the CURRENT eval parameters — the one-time calibration before tuning so the
// logistic is scaled to the eval's centipawn units.
func TexelFindK(samples []TexelSample) float64 {
	lo, hi := 0.0, 2.0
	for iter := 0; iter < 50; iter++ {
		m1 := lo + (hi-lo)/3
		m2 := hi - (hi-lo)/3
		if texelMSE(samples, m1) < texelMSE(samples, m2) {
			hi = m2
		} else {
			lo = m1
		}
	}
	return (lo + hi) / 2
}

// TexelWeight is a named tunable evaluation weight — a pointer into one of the
// eval's package vars, with its name so a run can both optimize and print it.
type TexelWeight struct {
	Name string
	Ptr  *int
}

// TexelWeights returns the hand-set eval-term weights exposed for tuning (the
// untuned scalars complementing the already-tuned PeSTO core).
func TexelWeights() []TexelWeight {
	return []TexelWeight{
		{"passedPawnBonus", &passedPawnBonus},
		{"doubledPawnPenalty", &doubledPawnPenalty},
		{"isolatedPawnPenalty", &isolatedPawnPenalty},
		{"pawnChainWeight", &pawnChainWeight},
		{"bishopPairWeight", &bishopPairWeight},
		{"mobilityWeight", &mobilityWeight},
		{"rookOpenWeight", &rookOpenWeight},
		{"outpostWeight", &outpostWeight},
		{"kingSafetyWeight", &kingSafetyWeight},
		{"kingActivityWeight", &kingActivityWeight},
	}
}

// FormatTexelWeights renders the given weights as Go var assignments, ready to
// paste back into eval.go's tunable-weights block after a tuning run.
func FormatTexelWeights(weights []TexelWeight) string {
	var b strings.Builder
	for _, w := range weights {
		fmt.Fprintf(&b, "\t%s = %d\n", w.Name, *w.Ptr)
	}
	return b.String()
}

// FormatPSTTables renders the 12 raw piece-square tables as Go var declarations
// matching eval_pesto.go's layout (8 values per row), so a tuned set can be
// pasted back in to replace the originals.
func FormatPSTTables() string {
	named := []struct {
		name string
		tbl  *[64]int
	}{
		{"mgPawnTable", &mgPawnTable}, {"mgKnightTable", &mgKnightTable}, {"mgBishopTable", &mgBishopTable},
		{"mgRookTable", &mgRookTable}, {"mgQueenTable", &mgQueenTable}, {"mgKingTable", &mgKingTable},
		{"egPawnTable", &egPawnTable}, {"egKnightTable", &egKnightTable}, {"egBishopTable", &egBishopTable},
		{"egRookTable", &egRookTable}, {"egQueenTable", &egQueenTable}, {"egKingTable", &egKingTable},
	}
	var b strings.Builder
	for _, t := range named {
		fmt.Fprintf(&b, "var %s = [64]int{\n", t.name)
		for sq := 0; sq < 64; sq++ {
			if sq%8 == 0 {
				b.WriteByte('\t')
			}
			fmt.Fprintf(&b, "%4d,", t.tbl[sq])
			if sq%8 == 7 {
				b.WriteByte('\n')
			} else {
				b.WriteByte(' ')
			}
		}
		b.WriteString("}\n\n")
	}
	return b.String()
}

// TexelTune runs coordinate descent on the piece-square tables to minimize MSE
// over the dataset at scaling K. Each pass line-searches every parameter (steps
// in the improving direction while MSE keeps falling), keeping all gains; it
// repeats until a pass yields no improvement or maxPasses is reached. progress
// (may be nil) is called once per pass with the pass index, current MSE, and how
// many parameters moved. The PST package vars are left holding the tuned values
// (black mirrors rebuilt). Returns the final MSE.
// rebuild (may be nil) is invoked after every parameter mutation — used when a
// tuned param feeds a derived table (e.g. the PeSTO mgPST/egPST) that must be
// recomputed before the next eval. For directly-used weights, pass nil.
func TexelTune(samples []TexelSample, params []*int, k float64, step, maxPasses int, rebuild func(), progress func(pass int, mse float64, moved int)) float64 {
	best := texelMSE(samples, k)

	// lineSearch steps *p by delta while MSE strictly improves, leaving *p at the
	// best value found; returns true if it moved at all.
	lineSearch := func(p *int, delta int) bool {
		moved := false
		for {
			*p += delta
			if rebuild != nil {
				rebuild()
			}
			m := texelMSE(samples, k)
			if m < best {
				best = m
				moved = true
				continue
			}
			*p -= delta // overshot: undo the last (non-improving) step
			if rebuild != nil {
				rebuild()
			}
			return moved
		}
	}

	for pass := 0; pass < maxPasses; pass++ {
		moved := 0
		for _, p := range params {
			if lineSearch(p, step) {
				moved++
				continue
			}
			if lineSearch(p, -step) {
				moved++
			}
		}
		if progress != nil {
			progress(pass, best, moved)
		}
		if moved == 0 {
			break
		}
	}
	return best
}

// LoadTexelSamples reads a dataset of "<FEN> <result>" lines (result one of 1.0 /
// 0.5 / 0.0, White's perspective), parsing each FEN once into an in-memory board.
// Blank lines and lines starting with '#' are skipped; unparseable lines error.
func LoadTexelSamples(filename string) ([]TexelSample, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var samples []TexelSample
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// EPD datasets (e.g. Zurichess quiet-labeled.epd) label the result with a
		// `c9 "1-0"` opcode rather than a trailing float, and give a 4-field FEN
		// (no move counters). Translate those to a board + White-POV result.
		if i := strings.Index(line, `c9 "`); i >= 0 {
			fields := strings.Fields(line)
			if len(fields) < 4 {
				return nil, fmt.Errorf("line %d: short EPD: %q", lineNo, line)
			}
			fen := strings.Join(fields[:4], " ") + " 0 1"
			r := line[i+len(`c9 "`):]
			j := strings.IndexByte(r, '"')
			if j < 0 {
				return nil, fmt.Errorf("line %d: unterminated c9 opcode: %q", lineNo, line)
			}
			var result float64
			switch r[:j] {
			case "1-0":
				result = 1.0
			case "0-1":
				result = 0.0
			case "1/2-1/2":
				result = 0.5
			default:
				return nil, fmt.Errorf("line %d: unknown c9 result %q", lineNo, r[:j])
			}
			pos, err := ParseFEN(fen)
			if err != nil {
				return nil, fmt.Errorf("line %d: bad FEN %q: %w", lineNo, fen, err)
			}
			samples = append(samples, TexelSample{Board: pos.Board, Result: result})
			continue
		}
		// Result is the final whitespace-separated field; the FEN is the rest.
		idx := strings.LastIndexByte(line, ' ')
		if idx < 0 {
			return nil, fmt.Errorf("line %d: no result field: %q", lineNo, line)
		}
		fen := strings.TrimSpace(line[:idx])
		result, err := strconv.ParseFloat(strings.TrimSpace(line[idx+1:]), 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: bad result: %w", lineNo, err)
		}
		pos, err := ParseFEN(fen)
		if err != nil {
			return nil, fmt.Errorf("line %d: bad FEN %q: %w", lineNo, fen, err)
		}
		samples = append(samples, TexelSample{Board: pos.Board, Result: result})
	}
	return samples, sc.Err()
}

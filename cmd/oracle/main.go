// cmd/oracle — fast deterministic best-move-agreement pre-filter.
//
// Runs an engine over a labeled corpus (EPD with UCI `bm`) at a FIXED node (or
// depth) budget and reports the fraction of positions where it plays a labeled
// best move. Because the budget is fixed nodes/depth — not movetime — the score
// is DETERMINISTIC: zero sampling variance, exactly reproducible, unlike a
// self-play SPRT that needs hundreds of games to average out clock noise.
//
// A/B mode (-base) runs two binaries over the same corpus and prints the
// agreement delta plus the positions that flipped. It SEES search changes
// (unlike an eval-error oracle) because better search -> better move choice, so
// it works on pruning/extension/ordering changes, not just eval.
//
// Role: a seconds-fast pre-filter that sits in FRONT of the SPRT lane — kill
// obviously-bad changes immediately, spend the multi-hour SPRT only on the
// survivors. It is NOT a strength oracle on its own: "plays the labelled move"
// correlates with but is not identical to ELO (multiple moves can be fine), so
// it filters and ranks; the SPRT/gauntlet still confirms before shipping.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ehrlich-b/ngn/internal/uci"
)

type oraclePos struct {
	FEN string
	BM  []string // acceptable best moves, UCI long-algebraic
	ID  string
}

// parseEPD reads "FEN bm MOVE[ MOVE...]; comment" lines (UCI moves). Blank and
// '#'-comment lines are skipped; multiple space-separated bm moves all count.
func parseEPD(path string) ([]oraclePos, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []oraclePos
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, " bm ")
		if idx < 0 {
			continue
		}
		fen := strings.TrimSpace(line[:idx])
		rest := line[idx+len(" bm "):]
		bmField := rest
		id := ""
		if s := strings.Index(rest, ";"); s >= 0 {
			bmField = rest[:s]
			id = strings.TrimSpace(rest[s+1:])
		}
		bms := strings.Fields(bmField)
		if len(bms) == 0 {
			continue
		}
		out = append(out, oraclePos{FEN: fen, BM: bms, ID: id})
	}
	return out, sc.Err()
}

func isHit(mv string, bm []string) bool {
	for _, b := range bm {
		if mv == b {
			return true
		}
	}
	return false
}

// scoreEngine searches every position at the fixed budget and records the move
// played. Reset() (ucinewgame) clears the TT/history between positions so the
// result is independent of corpus order and fully reproducible.
func scoreEngine(path string, positions []oraclePos, goCmd string, timeout time.Duration, lowPower bool) (hits int, moves []string) {
	e := uci.Start(path, "oracle", lowPower)
	if e == nil {
		fmt.Fprintf(os.Stderr, "oracle: failed to start engine %q\n", path)
		os.Exit(1)
	}
	defer uci.Stop(e)
	moves = make([]string, len(positions))
	for i, p := range positions {
		uci.Reset(e)
		uci.Send(e, "position fen "+p.FEN)
		mv := uci.GetMove(e, goCmd, timeout, 0) // stopAfter=0: fixed nodes/depth self-terminates
		moves[i] = mv
		if isHit(mv, p.BM) {
			hits++
		}
	}
	return hits, moves
}

func main() {
	var (
		corpus   = flag.String("corpus", "validated_tactical_positions.epd", "labeled EPD with UCI `bm`")
		newBin   = flag.String("new", "build/ngn", "engine binary to score")
		baseBin  = flag.String("base", "", "second binary for A/B comparison (optional)")
		nodes    = flag.Int("nodes", 200000, "fixed node budget per position")
		depth    = flag.Int("depth", 0, "fixed depth per position (overrides -nodes when >0)")
		limit    = flag.Int("n", 0, "cap number of positions (0 = all)")
		lowPower = flag.Bool("lowpower", true, "route engines to E-cores (taskpolicy -b)")
		verbose  = flag.Bool("v", false, "list the flipped positions in A/B mode")
	)
	flag.Parse()

	positions, err := parseEPD(*corpus)
	if err != nil {
		fmt.Fprintf(os.Stderr, "oracle: load corpus: %v\n", err)
		os.Exit(1)
	}
	if *limit > 0 && *limit < len(positions) {
		positions = positions[:*limit]
	}
	if len(positions) == 0 {
		fmt.Fprintln(os.Stderr, "oracle: no labeled positions in corpus")
		os.Exit(1)
	}

	goCmd := fmt.Sprintf("go nodes %d", *nodes)
	budget := fmt.Sprintf("nodes %d", *nodes)
	if *depth > 0 {
		goCmd = fmt.Sprintf("go depth %d", *depth)
		budget = fmt.Sprintf("depth %d", *depth)
	}
	timeout := 60 * time.Second
	pct := func(h int) float64 { return 100 * float64(h) / float64(len(positions)) }

	fmt.Printf("oracle: %s  (%d positions, %s)%s\n", *corpus, len(positions), budget, uci.PowerNote(*lowPower))

	newHits, newMoves := scoreEngine(*newBin, positions, goCmd, timeout, *lowPower)
	fmt.Printf("  new   %-24s %5.1f%%  (%d/%d)\n", *newBin, pct(newHits), newHits, len(positions))

	if *baseBin == "" {
		return
	}
	baseHits, baseMoves := scoreEngine(*baseBin, positions, goCmd, timeout, *lowPower)
	fmt.Printf("  base  %-24s %5.1f%%  (%d/%d)\n", *baseBin, pct(baseHits), baseHits, len(positions))

	gained, lost := 0, 0
	for i := range positions {
		nh, bh := isHit(newMoves[i], positions[i].BM), isHit(baseMoves[i], positions[i].BM)
		switch {
		case nh && !bh:
			gained++
		case bh && !nh:
			lost++
		}
	}
	fmt.Printf("  Δ %+.1f%%  (%+d solved; +%d gained / -%d lost)\n",
		pct(newHits)-pct(baseHits), newHits-baseHits, gained, lost)

	if *verbose {
		fmt.Println("  flips (bm | base -> new):")
		for i := range positions {
			nh, bh := isHit(newMoves[i], positions[i].BM), isHit(baseMoves[i], positions[i].BM)
			if nh == bh {
				continue
			}
			tag := "GAINED"
			if bh {
				tag = "LOST  "
			}
			fmt.Printf("    %s #%d  bm=%s  base=%s new=%s  %s\n",
				tag, i+1, strings.Join(positions[i].BM, "/"), baseMoves[i], newMoves[i], positions[i].ID)
		}
	}
}

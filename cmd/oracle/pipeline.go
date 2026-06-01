package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ehrlich-b/ngn/engine"
	"github.com/ehrlich-b/ngn/internal/uci"
)

// startPosition returns a fresh game-start Position (mirrors cmd/pgn_to_fens).
func startPosition() *engine.Position {
	return &engine.Position{
		Board: engine.StartingBoard(),
		Tag: engine.WhiteCanCastleKingSide | engine.WhiteCanCastleQueenSide |
			engine.BlackCanCastleKingSide | engine.BlackCanCastleQueenSide | engine.WhiteToMove,
		EnPassant: engine.NoSquare,
	}
}

// applyUCI parses uciMove in pos and makes it; false on illegal/parse error.
func applyUCI(pos *engine.Position, uciMove string) bool {
	mv, err := engine.ParseUCIMove(pos, uciMove)
	if err != nil {
		return false
	}
	pos.MakeMove(mv)
	return true
}

func badMove(m string) bool { return m == "" || m == "(none)" || m == "0000" }

// readLines reads non-empty, non-'#' lines.
func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		ln := strings.TrimSpace(sc.Text())
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		out = append(out, ln)
	}
	return out, sc.Err()
}

// ---------- gencorpus: self-play a diverse, realistic position set ----------

func runGenCorpus(engPath, openingsPath, outPath string, nOpenings, plies, every int, goCmd string, timeout time.Duration, lowPower bool) {
	openings, err := uci.LoadOpenings(openingsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "oracle gencorpus: load openings: %v\n", err)
		os.Exit(1)
	}
	if nOpenings > 0 && nOpenings < len(openings) {
		openings = openings[:nOpenings]
	}
	e := uci.Start(engPath, "gen", lowPower)
	if e == nil {
		fmt.Fprintf(os.Stderr, "oracle gencorpus: cannot start %q\n", engPath)
		os.Exit(1)
	}
	defer uci.Stop(e)
	out, err := os.Create(outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "oracle gencorpus: create %s: %v\n", outPath, err)
		os.Exit(1)
	}
	defer out.Close()
	w := bufio.NewWriter(out)
	defer w.Flush()

	count := 0
	for oi, om := range openings {
		pos := startPosition()
		ok := true
		for _, m := range om {
			if !applyUCI(pos, m) {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		for p := 0; p < plies; p++ {
			if len(engine.GenerateLegalMoves(pos)) == 0 {
				break // checkmate / stalemate
			}
			uci.Send(e, "position fen "+engine.GenerateFEN(pos))
			mv := uci.GetMove(e, goCmd, timeout, 0)
			if badMove(mv) || !applyUCI(pos, mv) {
				break
			}
			if p%every == 0 {
				fmt.Fprintln(w, engine.GenerateFEN(pos))
				count++
			}
		}
		if (oi+1)%25 == 0 {
			w.Flush()
			fmt.Printf("  ... %d openings, %d positions\n", oi+1, count)
		}
	}
	fmt.Printf("gencorpus: wrote %d positions to %s\n", count, outPath)
}

// ---------- label: Stockfish MultiPV eval-per-move ----------

type moveEval struct {
	move string
	cp   int // from side-to-move's perspective; mate mapped to +-(100000-|n|)
}

// labelPosition asks SF for the eval of each of its top moves (deepest wins).
func labelPosition(sf *uci.Engine, fen string, depth int) []moveEval {
	uci.Send(sf, "position fen "+fen)
	lines := uci.Analyze(sf, fmt.Sprintf("go depth %d", depth), 60*time.Second)
	byRank := map[int]moveEval{}
	for _, ln := range lines {
		if !strings.Contains(ln, " multipv ") {
			continue
		}
		f := strings.Fields(ln)
		rank, cp, mv, haveCp := 0, 0, "", false
		for i := 0; i+1 < len(f); i++ {
			switch f[i] {
			case "multipv":
				rank, _ = strconv.Atoi(f[i+1])
			case "cp":
				cp, _ = strconv.Atoi(f[i+1])
				haveCp = true
			case "mate":
				m, _ := strconv.Atoi(f[i+1])
				sign := 1
				if m < 0 {
					sign, m = -1, -m
				}
				cp, haveCp = sign*(100000-m), true
			case "pv":
				mv = f[i+1]
			}
		}
		if rank > 0 && mv != "" && haveCp {
			byRank[rank] = moveEval{mv, cp}
		}
	}
	if len(byRank) == 0 {
		return nil
	}
	ranks := make([]int, 0, len(byRank))
	for r := range byRank {
		ranks = append(ranks, r)
	}
	sort.Ints(ranks)
	out := make([]moveEval, 0, len(ranks))
	for _, r := range ranks {
		out = append(out, byRank[r])
	}
	return out
}

func runLabel(sfPath, fensPath, outPath string, multipv, depth int, lowPower bool) {
	fens, err := readLines(fensPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "oracle label: read fens: %v\n", err)
		os.Exit(1)
	}
	sf := uci.Start(sfPath, "stockfish", lowPower)
	if sf == nil {
		fmt.Fprintf(os.Stderr, "oracle label: cannot start stockfish %q\n", sfPath)
		os.Exit(1)
	}
	defer uci.Stop(sf)
	uci.Send(sf, fmt.Sprintf("setoption name MultiPV value %d", multipv))
	uci.Send(sf, "isready")
	uci.WaitFor(sf, "readyok", 5*time.Second)

	out, err := os.Create(outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "oracle label: create %s: %v\n", outPath, err)
		os.Exit(1)
	}
	defer out.Close()
	w := bufio.NewWriter(out)
	defer w.Flush()

	labeled := 0
	for i, fen := range fens {
		evs := labelPosition(sf, fen, depth)
		if len(evs) == 0 {
			continue
		}
		var sb strings.Builder
		sb.WriteString(fen)
		sb.WriteString(" |")
		for _, e := range evs {
			fmt.Fprintf(&sb, " %s:%d", e.move, e.cp)
		}
		fmt.Fprintln(w, sb.String())
		labeled++
		if (i+1)%50 == 0 {
			w.Flush()
			fmt.Printf("  ... labeled %d/%d\n", labeled, len(fens))
		}
	}
	fmt.Printf("label: wrote %d labeled positions to %s (multipv %d, depth %d)\n", labeled, outPath, multipv, depth)
}

// ---------- acpl: average centipawn loss vs SF labels ----------

type labeledPos struct {
	fen   string
	evals []moveEval // best-first
}

func parseLabels(path string) ([]labeledPos, error) {
	lines, err := readLines(path)
	if err != nil {
		return nil, err
	}
	var out []labeledPos
	for _, ln := range lines {
		i := strings.Index(ln, " |")
		if i < 0 {
			continue
		}
		fen := strings.TrimSpace(ln[:i])
		var evs []moveEval
		for _, tok := range strings.Fields(ln[i+2:]) {
			c := strings.LastIndex(tok, ":")
			if c < 0 {
				continue
			}
			cp, err := strconv.Atoi(tok[c+1:])
			if err != nil {
				continue
			}
			evs = append(evs, moveEval{tok[:c], cp})
		}
		if len(evs) > 0 {
			out = append(out, labeledPos{fen, evs})
		}
	}
	return out, nil
}

// acplEngine returns one binary's average centipawn loss vs the SF labels.
// A move outside SF's labeled top-N is floored at the worst labeled eval (a
// conservative LOWER bound on its true loss).
func acplEngine(engPath string, labels []labeledPos, goCmd string, timeout time.Duration, lowPower bool, capCp int) float64 {
	e := uci.Start(engPath, "acpl", lowPower)
	if e == nil {
		fmt.Fprintf(os.Stderr, "oracle acpl: cannot start %q\n", engPath)
		os.Exit(1)
	}
	defer uci.Stop(e)
	var total float64
	for _, lp := range labels {
		uci.Reset(e)
		uci.Send(e, "position fen "+lp.fen)
		mv := uci.GetMove(e, goCmd, timeout, 0)
		bestCp := lp.evals[0].cp
		moveCp, found := lp.evals[len(lp.evals)-1].cp, false
		for _, me := range lp.evals {
			if me.move == mv {
				moveCp, found = me.cp, true
				break
			}
		}
		_ = found
		loss := float64(bestCp - moveCp)
		if loss < 0 {
			loss = 0
		}
		if loss > float64(capCp) {
			loss = float64(capCp)
		}
		total += loss
	}
	return total / float64(len(labels))
}

func runACPL(newBin, baseBin, labelFile, goCmd, budget string, timeout time.Duration, lowPower bool, capCp int) {
	labels, err := parseLabels(labelFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "oracle acpl: %v\n", err)
		os.Exit(1)
	}
	if len(labels) == 0 {
		fmt.Fprintln(os.Stderr, "oracle acpl: no labeled positions")
		os.Exit(1)
	}
	fmt.Printf("acpl: %s  (%d positions, %s, cap %dcp)%s\n", labelFile, len(labels), budget, capCp, uci.PowerNote(lowPower))
	newACPL := acplEngine(newBin, labels, goCmd, timeout, lowPower, capCp)
	fmt.Printf("  new   %-24s ACPL %6.1f\n", newBin, newACPL)
	if baseBin == "" {
		return
	}
	baseACPL := acplEngine(baseBin, labels, goCmd, timeout, lowPower, capCp)
	fmt.Printf("  base  %-24s ACPL %6.1f\n", baseBin, baseACPL)
	fmt.Printf("  Δ %+.1f cp/move  (lower ACPL = better; positive Δ means NEW gives up more eval = WORSE)\n", newACPL-baseACPL)
}

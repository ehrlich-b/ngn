// Command blunder-finder surfaces positions where NGN plays an objectively bad
// move. For each input position (NGN to move) it runs NGN's own search, then
// uses full-strength Stockfish as an oracle: the "drop" is how much worse NGN's
// chosen move is than the position's value (centipawn loss). Positions whose
// drop exceeds a threshold are written to a JSONL corpus, sorted worst-first,
// for later eval/search step-through.
//
// Input is either a file of positions (-in; any line containing a FEN) or a UCI
// log (-log; reconstructs FENs from "position [startpos|fen ...] moves ..." lines).
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ehrlich-b/ngn/engine"
)

const startposFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

// Blunder is one oracle-confirmed bad move.
type Blunder struct {
	FEN      string `json:"fen"`       // position with NGN to move
	NGNMove  string `json:"ngn_move"`  // move NGN chose
	NGNScore int    `json:"ngn_score"` // NGN's own search score (cp, NGN POV)
	NGNDepth int    `json:"ngn_depth"` // depth NGN reached
	SFBest   string `json:"sf_best"`   // Stockfish's best move
	SFBefore int    `json:"sf_before"` // SF eval of position (cp, NGN POV)
	SFAfter  int    `json:"sf_after"`  // SF eval after NGN's move (cp, NGN POV)
	Drop     int    `json:"drop"`      // SFBefore - SFAfter = centipawn loss
}

// fenRe matches a full 6-field FEN anywhere in a line.
var fenRe = regexp.MustCompile(`([pnbrqkPNBRQK1-8]+(?:/[pnbrqkPNBRQK1-8]+){7} [wb] \S+ \S+ \d+ \d+)`)

func main() {
	inFile := flag.String("in", "", "input file: one position per line (any line containing a FEN)")
	logFile := flag.String("log", "", "UCI log to reconstruct positions from 'position ... moves ...' lines")
	sfPath := flag.String("sf", "./stockfish/stockfish-ubuntu-x86-64-avx2", "Stockfish binary path")
	movetime := flag.Int("movetime", 1000, "NGN movetime per position (ms)")
	sfMovetime := flag.Int("sf-movetime", 300, "Stockfish oracle movetime per eval (ms)")
	threshold := flag.Int("threshold", 150, "min centipawn loss to count as a blunder")
	limit := flag.Int("limit", 0, "max positions to scan (0 = all)")
	stride := flag.Int("stride", 1, "scan every Nth position (spread sampling across input)")
	outFile := flag.String("out", "output/blunders.jsonl", "output JSONL path")
	topN := flag.Int("top", 15, "print this many worst blunders to stdout")
	lowPower := flag.Bool("lowpower", true, "route Stockfish to E-cores via taskpolicy -b")
	flag.Parse()

	var fens []string
	switch {
	case *logFile != "":
		fens = fensFromLog(*logFile)
	case *inFile != "":
		fens = fensFromFile(*inFile)
	default:
		fmt.Fprintln(os.Stderr, "need -in <fenfile> or -log <ucilog>")
		os.Exit(1)
	}
	fens = sampleAndLimit(dedup(fens), *stride, *limit)
	if len(fens) == 0 {
		fmt.Fprintln(os.Stderr, "no positions parsed from input")
		os.Exit(1)
	}

	var sf *engine.StockfishClient
	var err error
	if *lowPower {
		sf, err = engine.NewStockfishClientArgs("taskpolicy", "-b", *sfPath)
	} else {
		sf, err = engine.NewStockfishClientArgs(*sfPath)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "stockfish: %v\n", err)
		os.Exit(1)
	}
	defer sf.Close()

	engine.TranspositionTable = engine.NewCache(engine.DEFAULT_CACHE_SIZE)

	fmt.Printf("Scanning %d positions (NGN %dms, SF oracle %dms, blunder threshold %dcp)\n",
		len(fens), *movetime, *sfMovetime, *threshold)

	var blunders []Blunder
	scanned, errs := 0, 0
	for i, fen := range fens {
		b, ok := analyze(sf, fen, *movetime, *sfMovetime)
		if !ok {
			errs++
			continue
		}
		scanned++
		if b.Drop >= *threshold {
			blunders = append(blunders, b)
		}
		if (i+1)%10 == 0 || i+1 == len(fens) {
			fmt.Printf("\r  %d/%d scanned, %d blunders found, %d skipped", i+1, len(fens), len(blunders), errs)
		}
	}
	fmt.Println()

	sort.Slice(blunders, func(a, b int) bool { return blunders[a].Drop > blunders[b].Drop })
	writeJSONL(*outFile, blunders)

	rate := 0.0
	if scanned > 0 {
		rate = 100 * float64(len(blunders)) / float64(scanned)
	}
	fmt.Printf("\n=== %d blunders / %d positions (%.1f%%), wrote %s ===\n",
		len(blunders), scanned, rate, *outFile)

	n := *topN
	if n > len(blunders) {
		n = len(blunders)
	}
	for i := 0; i < n; i++ {
		b := blunders[i]
		fmt.Printf("\n[%d] drop %dcp  NGN played %s (d%d, NGN eval %+d)  SF best %s (eval %+d -> %+d)\n     %s\n",
			i+1, b.Drop, b.NGNMove, b.NGNDepth, b.NGNScore, b.SFBest, b.SFBefore, b.SFAfter, b.FEN)
	}
}

// analyze runs NGN on a position and oracle-scores its move. ok=false on any error
// (illegal FEN, search produced no move, stockfish failure).
func analyze(sf *engine.StockfishClient, fen string, movetime, sfMovetime int) (Blunder, bool) {
	pos, err := engine.ParseFEN(fen)
	if err != nil {
		return Blunder{}, false
	}

	// Fresh per-position move-ordering state so the search is reproducible.
	engine.ClearHistoryTable()
	engine.ClearKillerMoves()
	engine.ClearCounterMoves()
	engine.ClearStop()

	tm := engine.NewTimeManager()
	tm.SetTimeControl(engine.SearchParams{MoveTime: movetime}, pos.Turn() == engine.White)
	info := engine.SearchIterativeDeepening(pos, engine.MaximumDepth, tm)
	if info.BestMove == engine.EmptyMove {
		return Blunder{}, false
	}

	sfBefore, sfBest, err := sf.GetEval(fen, sfMovetime)
	if err != nil {
		return Blunder{}, false
	}

	// Apply NGN's move and evaluate the resulting position from NGN's POV.
	pos2, _ := engine.ParseFEN(fen)
	mv, err := engine.ParseUCIMove(pos2, info.BestMove.ToString())
	if err != nil {
		return Blunder{}, false
	}
	pos2.MakeMove(mv)

	var ngnAfter int
	if len(engine.GenerateLegalMoves(pos2)) == 0 {
		if pos2.IsInCheck() {
			ngnAfter = 30000 // NGN delivered checkmate
		} else {
			ngnAfter = 0 // stalemate
		}
	} else {
		raw, _, err := sf.GetEval(engine.GenerateFEN(pos2), sfMovetime)
		if err != nil {
			return Blunder{}, false
		}
		ngnAfter = -raw // flip to NGN's POV (opponent is to move after NGN's move)
	}

	return Blunder{
		FEN:      fen,
		NGNMove:  info.BestMove.ToString(),
		NGNScore: info.BestScore,
		NGNDepth: info.Depth,
		SFBest:   sfBest,
		SFBefore: sfBefore,
		SFAfter:  ngnAfter,
		Drop:     sfBefore - ngnAfter,
	}, true
}

// fensFromFile extracts the first FEN found on each line.
func fensFromFile(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open %s: %v\n", path, err)
		os.Exit(1)
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		if m := fenRe.FindString(sc.Text()); m != "" {
			out = append(out, m)
		}
	}
	return out
}

// fensFromLog reconstructs the position after each "position ... moves ..." line.
func fensFromLog(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open %s: %v\n", path, err)
		os.Exit(1)
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		idx := strings.Index(line, "position ")
		if idx < 0 {
			continue
		}
		if fen := fenFromPositionCmd(line[idx:]); fen != "" {
			out = append(out, fen)
		}
	}
	return out
}

// fenFromPositionCmd turns a "position [startpos|fen <fen>] moves m1 m2 ..." into
// the resulting FEN by replaying the moves.
func fenFromPositionCmd(cmd string) string {
	toks := strings.Fields(cmd)
	if len(toks) < 2 || toks[0] != "position" {
		return ""
	}
	var base string
	var rest []string
	switch toks[1] {
	case "startpos":
		base = startposFEN
		rest = toks[2:]
	case "fen":
		if len(toks) < 8 {
			return ""
		}
		base = strings.Join(toks[2:8], " ")
		rest = toks[8:]
	default:
		return ""
	}

	pos, err := engine.ParseFEN(base)
	if err != nil {
		return ""
	}
	if len(rest) == 0 || rest[0] != "moves" {
		return engine.GenerateFEN(pos)
	}
	for _, mvStr := range rest[1:] {
		mv, err := engine.ParseUCIMove(pos, mvStr)
		if err != nil {
			return "" // malformed move list; skip this line
		}
		pos.MakeMove(mv)
	}
	return engine.GenerateFEN(pos)
}

func dedup(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := in[:0]
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func sampleAndLimit(in []string, stride, limit int) []string {
	if stride < 1 {
		stride = 1
	}
	var out []string
	for i := 0; i < len(in); i += stride {
		out = append(out, in[i])
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func writeJSONL(path string, blunders []Blunder) {
	if dir := filepath.Dir(path); dir != "" {
		os.MkdirAll(dir, 0o755)
	}
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create %s: %v\n", path, err)
		return
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, b := range blunders {
		enc.Encode(b)
	}
}

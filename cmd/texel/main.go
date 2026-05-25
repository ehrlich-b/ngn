// Command texel runs Texel tuning of NGN's piece-square tables.
//
//	texel -mode gendata -games 1500 -nodes 8000 -out data.txt
//	texel -mode tune    -data data.txt -out tuned_psts.go
//
// gendata self-plays games in-process (single-threaded — Search shares global
// TT/history state, so it cannot run concurrently in one process; run several
// instances under `taskpolicy -b` and concatenate for more data) and records
// quiet, non-check positions labeled with the game's final result from White's
// perspective. tune loads that dataset, calibrates K, runs coordinate descent
// over the PSTs, and dumps the tuned tables as Go source to paste into eval.go.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/ehrlich-b/ngn/engine"
)

const startposFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

func main() {
	mode := flag.String("mode", "", "gendata | tune")
	// gendata flags
	games := flag.Int("games", 1500, "gendata: number of self-play games")
	nodes := flag.Int("nodes", 8000, "gendata: fixed node budget per move")
	prefix := flag.Int("prefix", 8, "gendata: random plies before recording (opening diversity)")
	maxPlies := flag.Int("maxplies", 240, "gendata: max plies before scoring the game a draw")
	seed := flag.Int64("seed", 0, "gendata: RNG seed (0 = time-based)")
	// tune flags
	data := flag.String("data", "", "tune: dataset file of '<FEN> <result>' lines")
	passes := flag.Int("passes", 8, "tune: max coordinate-descent passes per step size")
	// shared
	out := flag.String("out", "", "output file (default stdout)")
	flag.Parse()

	switch *mode {
	case "gendata":
		genData(*games, *nodes, *prefix, *maxPlies, *seed, *out)
	case "tune":
		tune(*data, *passes, *out)
	default:
		fmt.Fprintln(os.Stderr, "specify -mode gendata or -mode tune")
		flag.Usage()
		os.Exit(2)
	}
}

// genData self-plays games and writes labeled quiet positions.
func genData(games, nodes, prefix, maxPlies int, seed int64, out string) {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	rng := rand.New(rand.NewSource(seed))

	w := os.Stdout
	if out != "" {
		f, err := os.Create(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, "create:", err)
			os.Exit(1)
		}
		defer f.Close()
		w = f
	}
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	start := time.Now()
	totalPos := 0
	for g := 0; g < games; g++ {
		fens, result := playGame(nodes, prefix, maxPlies, rng)
		for _, fen := range fens {
			fmt.Fprintf(bw, "%s %.1f\n", fen, result)
		}
		totalPos += len(fens)
		if (g+1)%50 == 0 {
			bw.Flush()
			fmt.Fprintf(os.Stderr, "  %d/%d games, %d positions, %s elapsed\n",
				g+1, games, totalPos, time.Since(start).Round(time.Second))
		}
	}
	fmt.Fprintf(os.Stderr, "done: %d games, %d positions in %s\n",
		games, totalPos, time.Since(start).Round(time.Second))
}

// playGame plays one self-play game at fixed nodes/move after a random opening
// prefix, returning the recorded quiet (non-check, post-prefix) position FENs and
// the game result from White's perspective (1.0 / 0.5 / 0.0).
func playGame(nodes, prefix, maxPlies int, rng *rand.Rand) ([]string, float64) {
	// Fresh per-game search state (ucinewgame semantics).
	engine.TranspositionTable = engine.NewCache(engine.DEFAULT_CACHE_SIZE)
	engine.ClearHistoryTable()
	engine.ClearKillerMoves()
	engine.ClearCounterMoves()
	engine.SetLastMovePlayed(engine.EmptyMove)

	pos, _ := engine.ParseFEN(startposFEN)
	var fens []string
	result := 0.5 // draw if the game reaches maxPlies

	for ply := 0; ply < maxPlies; ply++ {
		legal := engine.GenerateLegalMoves(pos)
		if len(legal) == 0 {
			if pos.IsInCheck() {
				// Side to move is checkmated and loses.
				if pos.Turn() == engine.White {
					result = 0.0
				} else {
					result = 1.0
				}
			} // else stalemate -> 0.5
			break
		}
		if pos.IsFIDEDrawRule() {
			break // threefold / 50-move -> 0.5
		}

		if ply < prefix {
			// Random opening move for diversity.
			pos.GameMakeMove(legal[rng.Intn(len(legal))])
			continue
		}

		// Record the position before moving (quiet = not in check).
		if !pos.IsInCheck() {
			fens = append(fens, engine.GenerateFEN(pos))
		}

		engine.SetMaxNodes(uint64(nodes))
		info := engine.Search(pos, engine.MaximumDepth)
		engine.SetMaxNodes(0)
		if info.BestMove == engine.EmptyMove {
			break
		}
		pos.GameMakeMove(info.BestMove)
	}
	return fens, result
}

// tune loads the dataset, calibrates K, runs a decreasing-step coordinate-descent
// schedule, and writes the tuned PSTs as Go source.
func tune(data string, passes int, out string) {
	if data == "" {
		fmt.Fprintln(os.Stderr, "tune: -data is required")
		os.Exit(2)
	}
	fmt.Fprintf(os.Stderr, "loading %s ...\n", data)
	samples, err := engine.LoadTexelSamples(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "loaded %d positions\n", len(samples))
	if len(samples) == 0 {
		os.Exit(1)
	}

	k := engine.TexelFindK(samples)
	fmt.Fprintf(os.Stderr, "calibrated K = %.4f\n", k)

	weights := engine.TexelWeights()
	ptrs := make([]*int, len(weights))
	for i := range weights {
		ptrs[i] = weights[i].Ptr
	}
	fmt.Fprintf(os.Stderr, "weights before:\n%s", engine.FormatTexelWeights(weights))

	start := time.Now()
	for _, step := range []int{8, 4, 2, 1} {
		fmt.Fprintf(os.Stderr, "--- coordinate descent step=%d ---\n", step)
		engine.TexelTune(samples, ptrs, k, step, passes, nil, func(pass int, mse float64, moved int) {
			fmt.Fprintf(os.Stderr, "  step=%d pass=%d MSE=%.6f moved=%d  (%s)\n",
				step, pass, mse, moved, time.Since(start).Round(time.Second))
		})
	}
	fmt.Fprintf(os.Stderr, "weights after:\n%s", engine.FormatTexelWeights(weights))

	w := os.Stdout
	if out != "" {
		f, err := os.Create(out)
		if err != nil {
			fmt.Fprintln(os.Stderr, "create:", err)
			os.Exit(1)
		}
		defer f.Close()
		w = f
	}
	fmt.Fprint(w, engine.FormatTexelWeights(weights))
	fmt.Fprintf(os.Stderr, "tuning done in %s\n", time.Since(start).Round(time.Second))
}

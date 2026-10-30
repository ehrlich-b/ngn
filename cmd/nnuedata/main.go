// Command nnuedata generates game-grouped HCE search and outcome labels.
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

type sample struct {
	fen   string
	score int
	white bool
}

func main() {
	games := flag.Int("games", 1000, "number of games")
	nodes := flag.Int("nodes", 100, "search nodes per move")
	seed := flag.Int64("seed", 1, "random seed")
	prefix := flag.Int("prefix", 8, "random opening plies")
	cap := flag.Int("maxplies", 200, "game ply cap, drawn")
	out := flag.String("out", "", "output TSV path")
	flag.Parse()
	if *games <= 0 || *nodes <= 0 || *prefix < 0 || *cap <= *prefix || *out == "" {
		fmt.Fprintln(os.Stderr, "invalid arguments")
		os.Exit(2)
	}
	f, err := os.Create(*out)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()
	rng := rand.New(rand.NewSource(*seed))
	start := time.Now()
	count := 0
	aborted := 0
	for g := 0; g < *games; g++ {
		engine.ClearHash()
		engine.ClearHistoryTable()
		engine.ClearKillerMoves()
		engine.ClearCounterMoves()
		engine.SetLastMovePlayed(engine.EmptyMove)
		pos, err := engine.ParseFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
		if err != nil {
			panic(err)
		}
		var samples []sample
		result := 0.5
		failed := false
		for ply := 0; ply < *cap; ply++ {
			legal := engine.GenerateLegalMoves(pos)
			if len(legal) == 0 {
				if pos.IsInCheck() {
					if pos.Turn() == engine.White {
						result = 0
					} else {
						result = 1
					}
				}
				break
			}
			if pos.IsFIDEDrawRule() {
				break
			}
			if ply < *prefix {
				pos.GameMakeMove(legal[rng.Intn(len(legal))])
				continue
			}
			fen := engine.GenerateFEN(pos)
			white := pos.Turn() == engine.White
			engine.SetMaxNodes(uint64(*nodes))
			info := engine.Search(pos, engine.MaximumDepth)
			engine.SetMaxNodes(0)
			if info.BestMove == engine.EmptyMove {
				failed = true
				break
			}
			samples = append(samples, sample{fen, info.BestScore, white})
			pos.GameMakeMove(info.BestMove)
		}
		if failed {
			aborted++
			continue
		}
		for _, s := range samples {
			outcome := result
			if !s.white {
				outcome = 1 - result
			}
			fmt.Fprintf(w, "%d\t%s\t%d\t%.1f\n", g, s.fen, s.score, outcome)
		}
		count += len(samples)
	}
	if err := w.Flush(); err != nil {
		panic(err)
	}
	elapsed := time.Since(start).Seconds()
	fmt.Fprintf(os.Stderr, "games=%d positions=%d seconds=%.3f positions_per_hour=%.1f seed=%d nodes=%d aborted=%d\n", *games, count, elapsed, float64(count)*3600/elapsed, *seed, *nodes, aborted)
}

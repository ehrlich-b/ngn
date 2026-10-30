// Command nnuegames generates NNUE self-play with independent HCE labels.
package main

import (
	"bufio"
	"compress/gzip"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/ehrlich-b/ngn/engine"
	"github.com/ehrlich-b/ngn/nnue"
)

type sample struct {
	fen   string
	score int
	white bool
}

func game(play, label *engine.SearchEngine, rng *rand.Rand, prefix, cap int, deadline time.Time) ([]sample, float64, bool) {
	play.ClearHash()
	play.ClearHistoryTable()
	play.ClearKillerMoves()
	play.ClearCounterMoves()
	play.SetLastMovePlayed(engine.EmptyMove)
	label.ClearHash()
	label.ClearHistoryTable()
	label.ClearKillerMoves()
	label.ClearCounterMoves()
	label.SetLastMovePlayed(engine.EmptyMove)
	pos, err := engine.ParseFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if err != nil {
		panic(err)
	}
	var samples []sample
	for ply := 0; ply < cap; ply++ {
		if time.Now().After(deadline) {
			return nil, 0, false
		}
		legal := engine.GenerateLegalMoves(pos)
		if len(legal) == 0 {
			return samples, result(pos), true
		}
		if pos.IsFIDEDrawRule() {
			return samples, .5, true
		}
		if ply < prefix {
			pos.GameMakeMove(legal[rng.Intn(len(legal))])
			continue
		}
		info := play.Search(pos, engine.MaximumDepth)
		if info.BestMove == engine.EmptyMove {
			return nil, 0, false
		}
		score := label.Search(pos, engine.MaximumDepth)
		if score.BestMove == engine.EmptyMove {
			return nil, 0, false
		}
		samples = append(samples, sample{engine.GenerateFEN(pos), score.BestScore, pos.Turn() == engine.White})
		pos.GameMakeMove(info.BestMove)
	}
	if len(engine.GenerateLegalMoves(pos)) == 0 {
		return samples, result(pos), true
	}
	return samples, .5, true
}
func trainingGameID(games int) int { return games + 1 + games/49 }

func result(pos *engine.Position) float64 {
	if !pos.IsInCheck() {
		return .5
	}
	if pos.Turn() == engine.White {
		return 0
	}
	return 1
}
func run() error {
	net := flag.String("net", "", "self-play NGN-v1 network")
	out := flag.String("out", "", "new gzip shard (must not exist)")
	seconds := flag.Int("seconds", 2700, "maximum generation seconds")
	seed := flag.Int64("seed", 103004, "opening seed")
	nodes := flag.Int("nodes", 1000, "self-play nodes")
	labels := flag.Int("label-nodes", 4000, "HCE labeling nodes")
	flag.Parse()
	if *net == "" || *out == "" || *seconds <= 0 || *seconds > 2700 || *nodes < 1 || *labels < 4000 {
		return fmt.Errorf("invalid arguments: require net, output, 1..2700 seconds and at least 4000 label nodes")
	}
	f, err := os.Open(*net)
	if err != nil {
		return err
	}
	model, err := nnue.Load(f)
	f.Close()
	if err != nil {
		return err
	}
	play := engine.NewSearchEngine()
	if err := play.SelectNGNV1Evaluator(model); err != nil {
		return err
	}
	play.SetMaxNodes(uint64(*nodes))
	label := engine.NewSearchEngine()
	label.SetMaxNodes(uint64(*labels))
	dst, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	gz := gzip.NewWriter(dst)
	w := bufio.NewWriter(gz)
	start := time.Now()
	rng := rand.New(rand.NewSource(*seed))
	games, positions, aborted := 0, 0, 0
	for time.Since(start) < time.Duration(*seconds)*time.Second {
		rows, outcome, ok := game(play, label, rng, 8, 200, start.Add(time.Duration(*seconds)*time.Second))
		if !ok {
			aborted++
			continue
		}
		// New games are training only; preserve stage 3's exact holdout.
		id := trainingGameID(games)
		for _, s := range rows {
			v := outcome
			if !s.white {
				v = 1 - outcome
			}
			if _, err = fmt.Fprintf(w, "%d\t%s\t%d\t%.1f\n", id, s.fen, s.score, v); err != nil {
				return err
			}
		}
		games++
		positions += len(rows)
	}
	if err = w.Flush(); err != nil {
		return err
	}
	if err = gz.Close(); err != nil {
		return err
	}
	if err = dst.Close(); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "games=%d positions=%d aborted=%d seconds=%.3f nodes=%d label_nodes=%d seed=%d\n", games, positions, aborted, time.Since(start).Seconds(), *nodes, *labels, *seed)
	if positions == 0 {
		return fmt.Errorf("no positions generated")
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

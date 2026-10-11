// Command pairmatch runs a bounded external match and finishes admitted pairs.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ehrlich-b/ngn/internal/rating"
	"github.com/ehrlich-b/ngn/internal/uci"
)

type options []string

func (o *options) String() string { return strings.Join(*o, ";") }
func (o *options) Set(v string) error {
	n, s, ok := strings.Cut(v, "=")
	if !ok || strings.TrimSpace(n) == "" || strings.ContainsAny(v, "\r\n") {
		return fmt.Errorf("want NAME=VALUE")
	}
	*o = append(*o, "setoption name "+n+" value "+s)
	return nil
}

type game struct {
	White  bool              `json:"ngn_white"`
	Result uci.GameResult    `json:"result"`
	Moves  []string          `json:"moves"`
	Clocks []uci.ClockSample `json:"clocks"`
}
type pair struct {
	ID      int    `json:"pair_id"`
	Opening string `json:"opening"`
	Games   []game `json:"games"`
	Error   string `json:"error,omitempty"`
}

// pairQueue admits each opening once and never exceeds a fixed pair count.
type pairQueue struct {
	mu          sync.Mutex
	next, limit int
	cutoff      time.Time
	invalid     bool
}

func (q *pairQueue) admit(now time.Time) (int, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.invalid || (!q.cutoff.IsZero() && !now.Before(q.cutoff)) || q.next >= q.limit {
		return 0, false
	}
	id := q.next
	q.next++
	return id, true
}

func (q *pairQueue) fail() { q.mu.Lock(); q.invalid = true; q.mu.Unlock() }

func pairLimit(games, openings int, deadline time.Time) (int, error) {
	if games < 0 || games%2 != 0 {
		return 0, fmt.Errorf("games must be a positive even count, or zero for deadline mode")
	}
	if games == 0 {
		if deadline.IsZero() {
			return 0, fmt.Errorf("games or until required")
		}
		return openings, nil
	}
	if games/2 > openings {
		return 0, fmt.Errorf("need %d distinct openings, have %d", games/2, openings)
	}
	return games / 2, nil
}

func validResult(r uci.GameResult) bool {
	if r.Res != uci.Win && r.Res != uci.Draw && r.Res != uci.Loss {
		return false
	}
	switch r.Reason {
	case "checkmate", "stalemate", "draw-rule", "max-moves":
		return true
	default:
		return false
	}
}

type workerReceipt struct {
	Worker   int             `json:"worker"`
	NGN      uci.Diagnostics `json:"ngn"`
	Opponent uci.Diagnostics `json:"opponent"`
}

func played(a, b *uci.Engine, op []string, white bool) (game, bool) {
	done := make(chan game, 1)
	go func() {
		var moves []string
		var clocks []uci.ClockSample
		r := uci.PlayGame(a, b, op, white, uci.GameConfig{
			TCTimeMs: 120000, TCIncMs: 1000, MaxMoves: 400, MovesOut: &moves, ClockOut: &clocks,
		})
		done <- game{White: white, Result: r, Moves: moves, Clocks: clocks}
	}()
	select {
	case g := <-done:
		return g, true
	case <-time.After(12 * time.Minute):
		uci.Stop(a)
		uci.Stop(b)
		return game{}, false
	}
}

func main() {
	aPath := flag.String("ngn", "", "accepted NGN")
	bPath := flag.String("opponent", "", "external opponent")
	book := flag.String("openings", "", "standard opening file")
	until := flag.String("until", "", "RFC3339 final deadline")
	games := flag.Int("games", 0, "exact even game count; incomplete samples fail")
	conc := flag.Int("concurrency", 6, "parallel pair workers")
	out := flag.String("out", "", "append-only raw pair JSONL")
	var ao, bo options
	flag.Var(&ao, "ngnoption", "NAME=VALUE")
	flag.Var(&bo, "opponentoption", "NAME=VALUE")
	flag.Parse()
	var deadline time.Time
	var err error
	if *until != "" {
		deadline, err = time.Parse(time.RFC3339, *until)
		if err != nil {
			log.Fatal("valid RFC3339 deadline required")
		}
	}
	if *conc < 1 || *conc > 6 {
		log.Fatal("concurrency 1..6 required")
	}
	openings, err := uci.LoadOpenings(*book)
	if err != nil || len(openings) == 0 {
		log.Fatal("opening load failed: ", err)
	}
	limit, err := pairLimit(*games, len(openings), deadline)
	if err != nil {
		log.Fatal(err)
	}
	f, err := os.OpenFile(*out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	// Two 12-minute watchdogs plus one minute of cleanup fit every admitted pair.
	var cutoff time.Time
	if !deadline.IsZero() {
		cutoff = deadline.Add(-25 * time.Minute)
	}
	queue := pairQueue{limit: limit, cutoff: cutoff}
	results := make(chan pair, *conc)
	workers := make(chan workerReceipt, *conc)
	var wg sync.WaitGroup
	for w := 0; w < *conc; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			a := uci.StartWithOptions(*aPath, "NGN", false, ao)
			b := uci.StartWithOptions(*bPath, "opponent", false, bo)
			defer func() {
				uci.Stop(a)
				uci.Stop(b)
				r := workerReceipt{Worker: worker}
				if a != nil {
					r.NGN = a.Diagnostics()
				}
				if b != nil {
					r.Opponent = b.Diagnostics()
				}
				workers <- r
			}()
			if a == nil || b == nil || a.Diagnostics().StartupFailures > 0 || b.Diagnostics().StartupFailures > 0 {
				queue.fail()
				results <- pair{ID: -1, Error: "UCI handshake/warmup failed or required retry"}
				return
			}
			for {
				id, ok := queue.admit(time.Now())
				if !ok {
					return
				}
				p := pair{ID: id, Opening: strings.Join(openings[id], " ")}
				for _, white := range []bool{true, false} {
					g, ok := played(a, b, openings[id], white)
					if !ok || !validResult(g.Result) {
						if ok {
							p.Games = append(p.Games, g)
							p.Error = "operational game result: " + g.Result.Reason
						} else {
							p.Error = "watchdog; incomplete sample"
						}
						queue.fail()
						break
					}
					p.Games = append(p.Games, g)
				}
				results <- p
				if p.Error != "" {
					return
				}
			}
		}(w)
	}
	go func() { wg.Wait(); close(results); close(workers) }()
	var w, d, l, pairs, ngnFlags, opponentFlags, faults int
	var penta [5]int
	for p := range results {
		if err := enc.Encode(p); err != nil {
			log.Fatal(err)
		}
		if err := f.Sync(); err != nil {
			log.Fatal(err)
		}
		for _, g := range p.Games {
			if g.Result.Reason == "time-forfeit" {
				if g.Result.Res == uci.Loss {
					ngnFlags++
				} else {
					opponentFlags++
				}
			}
		}
		if p.Error != "" || len(p.Games) != 2 {
			faults++
			continue
		}
		points := 0
		for _, g := range p.Games {
			switch g.Result.Res {
			case uci.Win:
				w++
				points += 2
			case uci.Draw:
				d++
				points++
			case uci.Loss:
				l++
			}
			if strings.Contains(g.Result.Reason, "no-move") || strings.Contains(g.Result.Reason, "illegal") || strings.Contains(g.Result.Reason, "crash") {
				faults++
			}
		}
		penta[points]++
		pairs++
	}
	var receipts []workerReceipt
	for r := range workers {
		receipts = append(receipts, r)
		if r.NGN.ExitError != "" || r.Opponent.ExitError != "" || r.NGN.StartupFailures > 0 || r.Opponent.StartupFailures > 0 {
			faults++
		}
	}
	complete := *games == 0 || (pairs == *games/2 && w+d+l == *games && faults == 0)
	n := float64(pairs)
	mean, second := 0.0, 0.0
	for k, count := range penta {
		x := float64(k) / 4
		mean += x * float64(count)
		second += x * x * float64(count)
	}
	elo, lo, hi, score := 0.0, -800.0, 800.0, 0.0
	if n > 0 {
		mean /= n
		second /= n
		se := math.Sqrt(math.Max(second-mean*mean, .001) / n)
		elo, lo, hi = rating.ScoreToEloDiff(mean), rating.ScoreToEloDiff(mean-1.96*se), rating.ScoreToEloDiff(mean+1.96*se)
		score = mean
	}
	json.NewEncoder(os.Stdout).Encode(map[string]any{
		"games": w + d + l, "pairs": pairs, "wins": w, "draws": d, "losses": l,
		"score": score, "penta": penta, "elo": elo, "elo_lower95": lo, "elo_upper95": hi,
		"ngn_flag_outs": ngnFlags, "opponent_flag_outs": opponentFlags, "operational_faults": faults,
		"requested_games": *games, "complete": complete, "worker_receipts": receipts,
		"minimum160_met": w+d+l >= 160, "opponent_label": 3532,
		"implied_performance": 3532 + elo, "implied_lower95": 3532 + lo, "implied_upper95": 3532 + hi,
		"limitation": "one opponent at one time control; not a rating or campaign claim",
	})
	if faults > 0 || !complete || ngnFlags > 0 || opponentFlags > 0 {
		os.Exit(1)
	}
}

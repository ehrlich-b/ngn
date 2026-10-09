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
	White  bool           `json:"ngn_white"`
	Result uci.GameResult `json:"result"`
	Moves  []string       `json:"moves"`
}
type pair struct {
	ID      int    `json:"pair_id"`
	Opening string `json:"opening"`
	Games   []game `json:"games"`
	Error   string `json:"error,omitempty"`
}

func played(a, b *uci.Engine, op []string, white bool) (game, bool) {
	done := make(chan game, 1)
	go func() {
		var moves []string
		r := uci.PlayGame(a, b, op, white, uci.GameConfig{
			TCTimeMs: 120000, TCIncMs: 1000, MaxMoves: 400, MovesOut: &moves,
		})
		done <- game{white, r, moves}
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
	conc := flag.Int("concurrency", 6, "parallel pair workers")
	out := flag.String("out", "", "append-only raw pair JSONL")
	var ao, bo options
	flag.Var(&ao, "ngnoption", "NAME=VALUE")
	flag.Var(&bo, "opponentoption", "NAME=VALUE")
	flag.Parse()
	deadline, err := time.Parse(time.RFC3339, *until)
	if err != nil || *conc < 1 || *conc > 6 {
		log.Fatal("valid deadline and concurrency 1..6 required")
	}
	openings, err := uci.LoadOpenings(*book)
	if err != nil || len(openings) == 0 {
		log.Fatal("opening load failed: ", err)
	}
	f, err := os.OpenFile(*out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	// Two 12-minute watchdogs plus one minute of cleanup fit every admitted pair.
	cutoff := deadline.Add(-25 * time.Minute)
	var mu sync.Mutex
	next := 0
	invalid := false
	results := make(chan pair, *conc)
	var wg sync.WaitGroup
	for w := 0; w < *conc; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a := uci.StartWithOptions(*aPath, "NGN", false, ao)
			b := uci.StartWithOptions(*bPath, "opponent", false, bo)
			defer uci.Stop(a)
			defer uci.Stop(b)
			if a == nil || b == nil {
				mu.Lock()
				invalid = true
				mu.Unlock()
				results <- pair{ID: -1, Error: "UCI handshake/warmup failed"}
				return
			}
			for {
				mu.Lock()
				if invalid || !time.Now().Before(cutoff) || next >= len(openings) {
					mu.Unlock()
					return
				}
				id := next
				next++
				mu.Unlock()
				p := pair{ID: id, Opening: strings.Join(openings[id], " ")}
				for _, white := range []bool{true, false} {
					g, ok := played(a, b, openings[id], white)
					if !ok {
						p.Error = "watchdog; incomplete pair excluded"
						mu.Lock()
						invalid = true
						mu.Unlock()
						break
					}
					p.Games = append(p.Games, g)
				}
				results <- p
				if p.Error != "" {
					return
				}
			}
		}()
	}
	go func() { wg.Wait(); close(results) }()
	var w, d, l, pairs, ngnFlags, opponentFlags, faults int
	var penta [5]int
	for p := range results {
		if err := enc.Encode(p); err != nil {
			log.Fatal(err)
		}
		if err := f.Sync(); err != nil {
			log.Fatal(err)
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
			if g.Result.Reason == "time-forfeit" {
				if g.Result.Res == uci.Loss {
					ngnFlags++
				} else {
					opponentFlags++
				}
			}
			if strings.Contains(g.Result.Reason, "no-move") || strings.Contains(g.Result.Reason, "illegal") || strings.Contains(g.Result.Reason, "crash") {
				faults++
			}
		}
		penta[points]++
		pairs++
	}
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
		"minimum160_met": w+d+l >= 160, "opponent_label": 3532,
		"implied_performance": 3532 + elo, "implied_lower95": 3532 + lo, "implied_upper95": 3532 + hi,
		"limitation": "one opponent at one time control; not a rating or campaign claim",
	})
	if faults > 0 {
		os.Exit(1)
	}
}

// blackboxmatch diagnoses an external engine without inspecting its code or net.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ehrlich-b/ngn/engine"
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

type event struct {
	UTC       string `json:"utc"`
	ElapsedNS int64  `json:"elapsed_ns"`
	Kind      string `json:"kind"`
	Text      string `json:"text"`
}
type recorder struct {
	mu    sync.Mutex
	enc   *json.Encoder
	start time.Time
	fault bool
}

func (r *recorder) trace(kind, text string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	must(r.enc.Encode(event{time.Now().UTC().Format(time.RFC3339Nano), time.Since(r.start).Nanoseconds(), kind, text}))
	if kind == "start-error" || kind == "stdout-error" || kind == "forced-kill" || kind == "exit" && text != "0" {
		r.fault = true
	}
}
func start(path string, opts options, log string) (*uci.Engine, *os.File, *recorder) {
	f, err := os.OpenFile(log, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	must(err)
	r := &recorder{enc: json.NewEncoder(f), start: time.Now()}
	e := uci.StartRecorded(path, opts, r.trace)
	if e == nil {
		f.Close()
		panic("recorded UCI start failed: " + log)
	}
	return e, f, r
}
func (r *recorder) healthy() bool { r.mu.Lock(); defer r.mu.Unlock(); return !r.fault }

type playedGame struct {
	White       bool           `json:"ngn_white"`
	Result      uci.GameResult `json:"result"`
	Moves       []string       `json:"moves"`
	WallSeconds float64        `json:"wall_seconds"`
}
type pair struct {
	ID      int          `json:"pair_id"`
	Opening string       `json:"opening"`
	Games   []playedGame `json:"games"`
	Fault   bool         `json:"fault"`
}

func gameFault(g playedGame) bool {
	switch g.Result.Reason {
	case "checkmate", "stalemate", "draw-rule", "max-moves":
		return false
	}
	return true
}
func summary(pairs []pair) map[string]any {
	var penta [5]int
	w, d, l, faults := 0, 0, 0, 0
	for _, p := range pairs {
		if p.Fault || len(p.Games) != 2 {
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
		}
		penta[points]++
	}
	n := 0
	sum, second := 0., 0.
	for k, c := range penta {
		n += c
		x := float64(k) / 4
		sum += x * float64(c)
		second += x * x * float64(c)
	}
	mean, lo, hi := 0., 0., 0.
	if n > 0 {
		mean = sum / float64(n)
		se := math.Sqrt(math.Max(second/float64(n)-mean*mean, .001) / float64(n))
		lo = rating.ScoreToEloDiff(mean - 1.96*se)
		hi = rating.ScoreToEloDiff(mean + 1.96*se)
	}
	return map[string]any{"games": w + d + l, "pairs": n, "wins": w, "draws": d, "losses": l, "penta": penta, "score": mean, "elo": rating.ScoreToEloDiff(mean), "elo_lower95": lo, "elo_upper95": hi, "faulty_pairs": faults, "purpose": "black-box diagnosis only; no engine adoption or absolute rating"}
}

func main() {
	mode := flag.String("mode", "match", "match, probe, or compliance")
	ngn := flag.String("ngn", "", "frozen NGN binary")
	opponent := flag.String("opponent", "", "existing external binary")
	openingsFile := flag.String("openings", "", "frozen opening file")
	positionsFile := flag.String("positions", "", "frozen FENs, one per line")
	count := flag.Int("pairs", 100, "exact complete reversed pairs")
	conc := flag.Int("concurrency", 2, "parallel pair workers (1..6)")
	nodes := flag.Int("nodes", 0, "common per-move node limit")
	movetime := flag.Int("movetime", 0, "common per-move milliseconds")
	bank := flag.Int("bank", 0, "initial clock milliseconds")
	inc := flag.Int("increment", 0, "clock increment milliseconds")
	reps := flag.Int("repetitions", 3, "alternating probe repetitions")
	out := flag.String("out", "", "new output directory")
	var ao, bo options
	flag.Var(&ao, "ngnoption", "NAME=VALUE")
	flag.Var(&bo, "opponentoption", "NAME=VALUE")
	flag.Parse()
	if *ngn == "" || *opponent == "" || *out == "" || *conc < 1 || *conc > 6 || *count < 1 || *inc < 0 || *reps < 1 {
		panic("invalid arguments")
	}
	if *mode != "probe" && *mode != "match" && *mode != "compliance" {
		panic("invalid mode")
	}
	budgets := 0
	for _, b := range []int{*nodes, *movetime, *bank} {
		if b < 0 {
			panic("negative budget")
		}
		if b > 0 {
			budgets++
		}
	}
	if *mode == "match" && budgets != 1 {
		panic("select exactly one budget")
	}
	must(os.Mkdir(*out, 0700))
	cfg := uci.GameConfig{Nodes: *nodes, Movetime: *movetime, TCTimeMs: *bank, TCIncMs: *inc, MaxMoves: 400}
	meta, err := os.Create(*out + "/argv.json")
	must(err)
	must(json.NewEncoder(meta).Encode(map[string]any{"argv": os.Args, "config": cfg, "GOMAXPROCS": os.Getenv("GOMAXPROCS")}))
	must(meta.Close())
	if *mode == "probe" || *mode == "compliance" {
		if *mode == "compliance" && *nodes <= 0 {
			panic("compliance requires node budget")
		}
		probe(*ngn, *opponent, ao, bo, *positionsFile, *out, *reps, *nodes, *mode == "compliance")
		return
	}
	openings, err := uci.LoadOpenings(*openingsFile)
	must(err)
	if len(openings) < *count {
		panic("not enough frozen openings")
	}
	raw, err := os.OpenFile(*out+"/pairs.jsonl", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	must(err)
	defer raw.Close()
	var mu sync.Mutex
	next := 0
	failed := false
	results := make(chan pair, *conc)
	var wg sync.WaitGroup
	for worker := 0; worker < *conc; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for {
				mu.Lock()
				if failed || next >= *count {
					mu.Unlock()
					return
				}
				id := next
				next++
				mu.Unlock()
				a, af, ar := start(*ngn, ao, fmt.Sprintf("%s/pair-%03d-ngn.jsonl", *out, id))
				b, bf, br := start(*opponent, bo, fmt.Sprintf("%s/pair-%03d-opponent.jsonl", *out, id))
				p := pair{ID: id, Opening: strings.Join(openings[id], " ")}
				for _, white := range []bool{true, false} {
					ar.trace("game", fmt.Sprintf("pair=%d ngn_white=%v", id, white))
					br.trace("game", fmt.Sprintf("pair=%d ngn_white=%v", id, white))
					var moves []string
					c := cfg
					c.MovesOut = &moves
					started := time.Now()
					g := playedGame{White: white, Result: uci.PlayGame(a, b, openings[id], white, c)}
					g.Moves = moves
					g.WallSeconds = time.Since(started).Seconds()
					p.Games = append(p.Games, g)
					if gameFault(g) {
						p.Fault = true
						break
					}
				}
				uci.Stop(a)
				uci.Stop(b)
				p.Fault = p.Fault || !ar.healthy() || !br.healthy()
				must(af.Close())
				must(bf.Close())
				if p.Fault {
					mu.Lock()
					failed = true
					mu.Unlock()
				}
				results <- p
			}
		}(worker)
	}
	go func() { wg.Wait(); close(results) }()
	var pairs []pair
	enc := json.NewEncoder(raw)
	for p := range results {
		must(enc.Encode(p))
		must(raw.Sync())
		pairs = append(pairs, p)
	}
	s := summary(pairs)
	s["requested_pairs"] = *count
	s["complete_sample"] = s["pairs"].(int) == *count && s["faulty_pairs"].(int) == 0
	f, err := os.Create(*out + "/summary.json")
	must(err)
	must(json.NewEncoder(f).Encode(s))
	must(f.Close())
	must(json.NewEncoder(os.Stdout).Encode(s))
	if !s["complete_sample"].(bool) {
		os.Exit(1)
	}
}

func infoInt(lines []string, key string) int64 {
	var value int64
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] != "info" {
			continue
		}
		for i := 1; i+1 < len(fields); i++ {
			if fields[i] == "pv" || fields[i] == "string" {
				break
			}
			if fields[i] == key {
				var v int64
				if _, err := fmt.Sscan(fields[i+1], &v); err == nil {
					value = v
				}
			}
		}
	}
	return value
}
func probe(a, b string, ao, bo options, file, out string, reps, nodeLimit int, nodeOnly bool) {
	data, err := os.ReadFile(file)
	must(err)
	var fens []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p, err := engine.ParseFEN(line)
		must(err)
		if len(engine.GenerateLegalMoves(p)) == 0 {
			panic("terminal probe FEN")
		}
		fens = append(fens, line)
	}
	if len(fens) != 32 {
		panic("freeze exactly32 nonterminal positions")
	}
	f, err := os.OpenFile(out+"/probes.jsonl", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	must(err)
	defer f.Close()
	enc := json.NewEncoder(f)
	for rep := 0; rep < reps; rep++ {
		paths := []string{a, b}
		opts := []options{ao, bo}
		names := []string{"ngn", "opponent"}
		if rep%2 == 1 {
			paths[0], paths[1] = paths[1], paths[0]
			opts[0], opts[1] = opts[1], opts[0]
			names[0], names[1] = names[1], names[0]
		}
		// Alternating per-position order keeps thermal/runtime drift paired.
		for index, fen := range fens {
			for side, path := range paths {
				func() {
					e, log, r := start(path, opts[side], fmt.Sprintf("%s/probe-%d-%02d-%s.jsonl", out, rep, index, names[side]))
					defer log.Close()
					defer uci.Stop(e)
					budgets := []int{1000, 5000}
					if nodeOnly {
						budgets = nil
					}
					for _, ms := range budgets {
						uci.Reset(e)
						uci.Send(e, "position fen "+fen)
						started := time.Now()
						lines := uci.Analyze(e, fmt.Sprintf("go movetime %d", ms), time.Duration(ms)*time.Millisecond+5*time.Second)
						wall := time.Since(started).Seconds()
						last := ""
						if len(lines) > 0 {
							last = lines[len(lines)-1]
						}
						if !legalBestmove(fen, last) {
							panic("probe missing/illegal bestmove")
						}
						must(enc.Encode(map[string]any{"repetition": rep, "position": index, "fen": fen, "engine": names[side], "movetime_ms": ms, "wall_seconds": wall, "nodes": infoInt(lines, "nodes"), "depth": infoInt(lines, "depth"), "uci_time_ms": infoInt(lines, "time"), "bestmove": last, "natural_budget_compliance": wall <= float64(ms)/1000+.2}))
						if wall > float64(ms)/1000+.2 {
							uci.Stop(e)
							log.Close()
							panic("movetime overshoot; preserve failure and stop diagnostic")
						}
					}
					if nodeLimit > 0 {
						uci.Reset(e)
						uci.Send(e, "position fen "+fen)
						started := time.Now()
						lines := uci.Analyze(e, fmt.Sprintf("go nodes %d", nodeLimit), 60*time.Second)
						wall := time.Since(started).Seconds()
						reported := infoInt(lines, "nodes")
						last := ""
						if len(lines) > 0 {
							last = lines[len(lines)-1]
						}
						must(enc.Encode(map[string]any{"repetition": rep, "position": index, "engine": names[side], "node_budget": nodeLimit, "nodes": reported, "wall_seconds": wall, "bestmove": last}))
						if !legalBestmove(fen, last) || reported <= 0 || reported > int64(nodeLimit)+max(4096, int64(nodeLimit)/100) {
							uci.Stop(e)
							log.Close()
							panic("node compliance failed")
						}
					}
					uci.Stop(e)
					must(log.Close())
					if !r.healthy() {
						panic("probe process failure")
					}
				}()
			}
		}
	}
}
func must(err error) {
	if err != nil {
		panic(err)
	}
}

func legalBestmove(fen, line string) bool {
	fields := strings.Fields(line)
	if len(fields) < 2 || fields[0] != "bestmove" {
		return false
	}
	p, err := engine.ParseFEN(fen)
	if err != nil {
		return false
	}
	m, err := engine.ParseUCIMove(p, fields[1])
	if err != nil {
		return false
	}
	_, _, _, ok := p.GameMakeMove(m)
	return ok
}

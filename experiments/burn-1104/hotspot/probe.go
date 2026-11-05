// Command probe compares cold fixed searches using the embedded network or HCE.
package main

import (
	"encoding/json"
	"flag"
	"os"
	"runtime/pprof"
	"time"

	"github.com/ehrlich-b/ngn/engine"
)

type result struct {
	Position int     `json:"position"`
	FEN      string  `json:"fen"`
	Nodes    uint64  `json:"nodes"`
	Score    int     `json:"score"`
	Move     string  `json:"move"`
	Seconds  float64 `json:"seconds"`
	Threads  int     `json:"threads"`
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	corpus := flag.String("corpus", "", "JSON array of FENs")
	depth := flag.Int("depth", 4, "fixed search depth")
	position := flag.Int("position", -1, "one position, or all")
	profile := flag.String("profile", "", "CPU profile path")
	backend := flag.String("backend", "nnue", "nnue (embedded) or hce")
	flag.Parse()
	if *depth < 1 || *depth > 32 || (*backend != "nnue" && *backend != "hce") {
		panic("invalid probe options")
	}
	data, err := os.ReadFile(*corpus)
	must(err)
	var positions []string
	must(json.Unmarshal(data, &positions))
	if *position < -1 || *position >= len(positions) {
		panic("invalid position")
	}
	// Decode once before profiling; the search uses the actual embedded bytes.
	var network *engine.NGNN1Network
	if *backend == "nnue" {
		network, err = engine.LoadDefaultNet()
		must(err)
	}
	if *profile != "" {
		f, err := os.Create(*profile)
		must(err)
		must(pprof.StartCPUProfile(f))
		defer f.Close()
		defer pprof.StopCPUProfile()
	}
	enc := json.NewEncoder(os.Stdout)
	for i, fen := range positions {
		if *position >= 0 && i != *position {
			continue
		}
		e, err := engine.NewSearchEngineWithHash(16)
		must(err)
		if network != nil {
			must(e.SelectNGNN1Evaluator(network))
		}
		p, err := engine.ParseFEN(fen)
		must(err)
		start := time.Now()
		info := e.SearchFixed(p, *depth, nil)
		must(enc.Encode(result{i, fen, info.Nodes, info.BestScore, info.BestMove.ToString(), time.Since(start).Seconds(), info.EffectiveThreads}))
	}
}

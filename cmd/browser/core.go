package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/ehrlich-b/ngn/engine"
)

const startFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

type request struct {
	FEN    string   `json:"fen"`
	Moves  []string `json:"moves"`
	Move   string   `json:"move"`
	Depth  int      `json:"depth"`
	Millis int      `json:"millis"`
}

func execute(data []byte) []byte {
	var r request
	if err := json.Unmarshal(data, &r); err != nil {
		return []byte(`{"error":"invalid request"}`)
	}
	if r.FEN == "" {
		r.FEN = startFEN
	}
	p, err := engine.ParseFEN(r.FEN)
	if err != nil {
		return result(map[string]any{"error": err.Error()})
	}
	for _, m := range append(r.Moves, r.Move) {
		if m == "" {
			continue
		}
		found := false
		for _, legal := range engine.GenerateLegalMoves(p) {
			if legal.ToString() == m {
				p.GameMakeMove(legal)
				found = true
				break
			}
		}
		if !found {
			return result(map[string]any{"error": "illegal move"})
		}
	}
	legal := engine.GenerateLegalMoves(p)
	moves := make([]string, 0, len(legal))
	for _, m := range legal {
		moves = append(moves, m.ToString())
	}
	status := ""
	if len(legal) == 0 {
		if p.IsInCheck() {
			status = "checkmate"
		} else {
			status = "stalemate"
		}
	} else if p.HalfMoveClock >= 100 {
		status = "50-move rule"
	} else if p.Positions[p.Hash()] >= 3 {
		status = "repetition"
	} else if p.IsDraw() {
		status = "insufficient material"
	}
	out := map[string]any{"fen": engine.GenerateFEN(p), "legal": moves, "status": status, "network": fmt.Sprintf("%x", sha256.Sum256(engine.DefaultNetBytes()))}
	if (r.Depth > 0 || r.Millis > 0) && status == "" {
		s, e := engine.NewSearchEngineWithHash(16)
		if e != nil {
			return result(map[string]any{"error": e.Error()})
		}
		net, e := engine.LoadDefaultNet()
		if e != nil {
			return result(map[string]any{"error": e.Error()})
		}
		if e = s.SelectNGNN1Evaluator(net); e != nil {
			return result(map[string]any{"error": e.Error()})
		}
		var info *engine.SearchInfo
		if r.Depth > 0 {
			info = s.SearchIterativeDeepening(p, r.Depth, nil)
		} else {
			tm := engine.NewTimeManager()
			tm.SetTimeControl(engine.SearchParams{MoveTime: r.Millis}, p.Turn() == engine.White)
			info = s.SearchIterativeDeepening(p, engine.MaximumDepth, tm)
		}
		out["bestmove"] = info.BestMove.ToString()
		out["score"] = info.BestScore
		out["nodes"] = info.Nodes
	}
	return result(out)
}
func result(v any) []byte { b, _ := json.Marshal(v); return b }

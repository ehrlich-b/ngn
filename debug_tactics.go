package main

import (
	"fmt"
	"github.com/ehrlich-b/ngn/engine"
)

func main() {
	// Test the basic fork position that's failing
	fen := "rnbqkb1r/pppp1ppp/5n2/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3"
	
	pos, err := engine.ParseFEN(fen)
	if err != nil {
		fmt.Printf("Failed to parse FEN: %v\n", err)
		return
	}
	
	fmt.Printf("=== TACTICAL DEBUG: Basic Fork Position ===\n")
	fmt.Printf("FEN: %s\n", fen)
	fmt.Printf("Expected: Ng5 (knight fork)\n")
	fmt.Printf("Engine choice at different depths:\n\n")
	
	// Test at various depths
	for depth := 1; depth <= 6; depth++ {
		result := engine.Search(pos, depth)
		if result != nil {
			move := result.BestMove.ToString()
			score := result.BestScore
			nodes := result.Nodes
			fmt.Printf("Depth %d: %s (score: %d, nodes: %d)\n", depth, move, score, nodes)
		} else {
			fmt.Printf("Depth %d: No result\n", depth)
		}
	}
	
	// Show all legal moves and their static evaluations
	fmt.Printf("\n=== All Legal Moves (Static Evaluation) ===\n")
	moves := engine.GenerateLegalMoves(pos)
	
	type moveEval struct {
		move string
		score int
	}
	
	var moveList []moveEval
	
	for _, move := range moves {
		ep, tag, hc, _ := pos.MakeMove(move)
		score := engine.EvaluateForPlayer(&pos.Board, engine.White)
		pos.UnMakeMove(move, tag, ep, hc)
		
		moveList = append(moveList, moveEval{
			move: move.ToString(),
			score: score,
		})
	}
	
	// Sort by score (simple bubble sort for small list)
	for i := 0; i < len(moveList)-1; i++ {
		for j := 0; j < len(moveList)-i-1; j++ {
			if moveList[j].score < moveList[j+1].score {
				moveList[j], moveList[j+1] = moveList[j+1], moveList[j]
			}
		}
	}
	
	for i, me := range moveList {
		if i < 10 { // Top 10 moves
			fmt.Printf("%2d. %s -> %d\n", i+1, me.move, me.score)
		}
	}
}
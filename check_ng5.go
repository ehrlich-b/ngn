package main

import (
	"fmt"
	"github.com/ehrlich-b/ngn/engine"
)

func main() {
	// Test if Ng5 is a legal move in the fork position
	fen := "rnbqkb1r/pppp1ppp/5n2/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3"
	
	pos, err := engine.ParseFEN(fen)
	if err != nil {
		fmt.Printf("Failed to parse FEN: %v\n", err)
		return
	}
	
	fmt.Printf("=== CHECKING IF Ng5 IS LEGAL ===\n")
	fmt.Printf("FEN: %s\n", fen)
	
	moves := engine.GenerateLegalMoves(pos)
	fmt.Printf("Total legal moves: %d\n\n", len(moves))
	
	foundNg5 := false
	ng5Score := 0
	
	// Check all moves and specifically look for Ng5
	for _, move := range moves {
		moveStr := move.ToString()
		if moveStr == "f3g5" {  // Ng5 in coordinate notation
			foundNg5 = true
			
			// Evaluate this move
			ep, tag, hc, _ := pos.MakeMove(move)
			ng5Score = engine.EvaluateForPlayer(&pos.Board, engine.White)
			pos.UnMakeMove(move, tag, ep, hc)
			
			fmt.Printf("✅ FOUND Ng5 (f3g5): Static score = %d\n", ng5Score)
		}
		
		// Show all knight moves from f3
		if moveStr[:2] == "f3" {
			ep, tag, hc, _ := pos.MakeMove(move)
			score := engine.EvaluateForPlayer(&pos.Board, engine.White)
			pos.UnMakeMove(move, tag, ep, hc)
			
			fmt.Printf("Knight move: %s -> %d\n", moveStr, score)
		}
	}
	
	if !foundNg5 {
		fmt.Printf("❌ Ng5 (f3g5) NOT FOUND in legal moves!\n")
	}
	
	// Now let's see what happens if we search with Ng5 as the first move
	if foundNg5 {
		fmt.Printf("\n=== ANALYZING Ng5 POSITION ===\n")
		
		// Make Ng5 move
		ng5Move := engine.NewMove(engine.F3, engine.G5, engine.WhiteKnight, engine.NoPiece, engine.NoType, 0)
		ep, tag, hc, _ := pos.MakeMove(ng5Move)
		
		// Evaluate the position after Ng5
		afterNg5 := engine.EvaluateForPlayer(&pos.Board, engine.Black) // Black's turn now
		fmt.Printf("Position after Ng5 (black perspective): %d\n", afterNg5)
		fmt.Printf("Position after Ng5 (white perspective): %d\n", -afterNg5)
		
		// See what black's best response is
		fmt.Printf("Black's best responses to Ng5:\n")
		blackMoves := engine.GenerateLegalMoves(pos)
		fmt.Printf("Black has %d legal moves\n", len(blackMoves))
		
		// Evaluate a few of black's options
		for i, move := range blackMoves {
			if i < 5 { // Just top 5
				ep2, tag2, hc2, _ := pos.MakeMove(move)
				score := engine.EvaluateForPlayer(&pos.Board, engine.Black)
				pos.UnMakeMove(move, tag2, ep2, hc2)
				
				fmt.Printf("  %s -> %d (black perspective)\n", move.ToString(), score)
			}
		}
		
		pos.UnMakeMove(ng5Move, tag, ep, hc)
	}
}
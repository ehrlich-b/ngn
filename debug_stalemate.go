package main

import (
	"fmt"
	"github.com/ehrlich-b/ngn/engine"
)

func main() {
	// Stalemate position
	fen := "k7/P7/K7/8/8/8/8/8 b - - 0 1"
	pos, err := engine.ParseFEN(fen)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Stalemate analysis\n")
	fmt.Printf("Position: %s\n", fen)
	
	// Show the position
	fmt.Printf("\nBoard layout:\n")
	fmt.Printf("  Black king on a8: %v\n", pos.Board.PieceAt(engine.A8))
	fmt.Printf("  White pawn on a7: %v\n", pos.Board.PieceAt(engine.A7))  
	fmt.Printf("  White king on a6: %v\n", pos.Board.PieceAt(engine.A6))
	
	// Check if Black is in check
	blackInCheck := engine.IsInCheckDirect(pos, engine.Black)
	fmt.Printf("  Black king in check: %v\n", blackInCheck)
	
	// Our move generation
	moves := engine.GenerateLegalMoves(pos)
	fmt.Printf("\nOur legal moves for Black: %d\n", len(moves))
	for i, move := range moves {
		fmt.Printf("  %d: %s\n", i+1, move.ToString())
	}
	
	// Analyze the a8b8 move
	if len(moves) > 0 && moves[0].ToString() == "a8b8" {
		fmt.Printf("\nAnalyzing a8b8 move:\n")
		
		// Make the move
		ep, tag, hc, _ := pos.MakeMove(moves[0])
		
		fmt.Printf("  After a8b8:\n")
		fmt.Printf("  Black king now on b8: %v\n", pos.Board.PieceAt(engine.B8))
		fmt.Printf("  Is black king in check after move: %v\n", engine.IsInCheckDirect(pos, engine.Black))
		
		// Check what can attack b8
		fmt.Printf("  What attacks b8:\n")
		fmt.Printf("    White king on a6 attacks b7, b6, b5 (not b8)\n") 
		fmt.Printf("    White pawn on a7 attacks b8! (diagonal)\n")
		
		// Unmake the move  
		pos.UnMakeMove(moves[0], tag, ep, hc)
		
		fmt.Printf("\nThe bug: a8b8 moves into check (attacked by pawn on a7)\n")
		fmt.Printf("Our IsLegalMove should detect this and reject the move\n")
	}
}
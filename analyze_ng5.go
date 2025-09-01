package main

import (
	"fmt"
	"github.com/ehrlich-b/ngn/engine"
)

func main() {
	// Analyze what happens after Ng5 with deeper search
	fen := "rnbqkb1r/pppp1ppp/5n2/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3"
	
	pos, err := engine.ParseFEN(fen)
	if err != nil {
		fmt.Printf("Failed to parse FEN: %v\n", err)
		return
	}
	
	fmt.Printf("=== DEEP ANALYSIS OF Ng5 FORK ===\n")
	
	// Make Ng5 move
	ng5Move := engine.NewMove(engine.F3, engine.G5, engine.WhiteKnight, engine.NoPiece, engine.NoType, 0)
	ep, tag, hc, _ := pos.MakeMove(ng5Move)
	
	fmt.Printf("Position after Ng5:\n")
	fmt.Printf("Now it's Black's turn. Let's see what search thinks...\n\n")
	
	// Search from black's perspective (deeper to see the tactical consequences)
	for depth := 1; depth <= 4; depth++ {
		result := engine.Search(pos, depth)
		if result != nil {
			blackMove := result.BestMove.ToString()
			blackScore := result.BestScore  // This is from black's perspective
			whiteScore := -blackScore       // Convert to white's perspective
			
			fmt.Printf("Depth %d: Black plays %s (white gets %d)\n", depth, blackMove, whiteScore)
		}
	}
	
	pos.UnMakeMove(ng5Move, tag, ep, hc)
	
	// Let's check if the position has any checks or attacks after Ng5
	fmt.Printf("\n=== CHECKING FOR TACTICAL MOTIFS ===\n")
	
	// Make Ng5 again for analysis
	pos.MakeMove(ng5Move)
	
	// Check if the knight on g5 attacks anything important
	fmt.Printf("Knight on g5 attacks:\n")
	
	// Knight on g5 attacks: e4, e6, f3, f7, h3, h7
	// Let's see if any of these squares have important pieces
	
	// Check f7 square - often a weak point
	f7Piece := pos.Board.GetPiece(engine.F7)
	if f7Piece != engine.NoPiece {
		fmt.Printf("- f7: %s (potential fork target!)\n", f7Piece.Name())
	}
	
	// Check e6 square
	e6Piece := pos.Board.GetPiece(engine.E6)
	if e6Piece != engine.NoPiece {
		fmt.Printf("- e6: %s\n", e6Piece.Name())
	}
	
	// Check if there's a king on e8 that would create a royal fork
	e8Piece := pos.Board.GetPiece(engine.E8)
	if e8Piece != engine.NoPiece {
		fmt.Printf("- Can attack e8 in some variations: %s\n", e8Piece.Name())
	}
	
	pos.UnMakeMove(ng5Move, tag, ep, hc)
	
	fmt.Printf("\n=== COMPARING IMMEDIATE VS TACTICAL ===\n")
	fmt.Printf("f3xe5: Wins pawn immediately (+233)\n")
	fmt.Printf("Ng5: Potentially attacks f7 pawn and threatens tactical sequences\n")
	fmt.Printf("The engine chooses the immediate material gain over the tactical setup\n")
}
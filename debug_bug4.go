package main

import (
	"fmt"
	"github.com/ehrlich-b/ngn/engine"
)

func main() {
	// BUG #4 position: Bishop Color Bind
	fen := "8/8/8/8/8/8/8/B6b w - - 0 1"
	pos, err := engine.ParseFEN(fen)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Debugging BUG #4 - Bishop illegal moves\n")
	fmt.Printf("Position: %s\n", fen)
	
	// Check position
	fmt.Printf("\nPosition analysis:\n")
	fmt.Printf("  White bishop on a1: %v\n", pos.Board.PieceAt(engine.A1))
	fmt.Printf("  Black bishop on h1: %v\n", pos.Board.PieceAt(engine.H1))
	fmt.Printf("  Turn: %v\n", pos.Turn())
	
	// Check if kings exist (since we had that bug before)
	whiteKingFound := false
	blackKingFound := false
	for sq := 0; sq < 64; sq++ {
		piece := pos.Board.PieceAt(engine.Square(sq))
		if piece == engine.WhiteKing {
			whiteKingFound = true
			fmt.Printf("  White king at: %s\n", engine.Square(sq).Name())
		}
		if piece == engine.BlackKing {
			blackKingFound = true
			fmt.Printf("  Black king at: %s\n", engine.Square(sq).Name())
		}
	}
	
	if !whiteKingFound {
		fmt.Printf("  *** NO WHITE KING FOUND! ***\n")
	}
	if !blackKingFound {
		fmt.Printf("  *** NO BLACK KING FOUND! ***\n")
	}
	
	// Generate moves
	moves := engine.GenerateLegalMoves(pos)
	fmt.Printf("\nOur legal moves: %d\n", len(moves))
	for i, move := range moves {
		fmt.Printf("  %d: %s\n", i+1, move.ToString())
	}
	
	fmt.Printf("\nExpected: Stockfish says 0 legal moves\n")
	if len(moves) > 0 {
		fmt.Printf("BUG CONFIRMED: We generate moves when Stockfish says none\n")
	} else {
		fmt.Printf("BUG FIXED: We also generate 0 moves\n")
	}
}
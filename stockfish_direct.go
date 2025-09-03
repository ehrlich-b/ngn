package main

import (
	"fmt"
	"github.com/ehrlich-b/ngn/engine"
)

func main() {
	// Test BUG #2 position directly with Stockfish
	fen := "4k3/8/8/8/3Qp3/8/8/8 w - - 0 1"
	
	client, err := engine.NewStockfishClient("stockfish")
	if err != nil {
		fmt.Printf("Error creating Stockfish client: %v\n", err)
		return
	}
	defer client.Close()
	
	fmt.Printf("Testing position: %s\n", fen)
	
	// Get legal moves from Stockfish
	moves, err := client.GetLegalMovesWithTime(fen, 100)
	if err != nil {
		fmt.Printf("Error getting moves from Stockfish: %v\n", err)
		return
	}
	
	fmt.Printf("Stockfish legal moves: %d\n", len(moves))
	for i, move := range moves {
		fmt.Printf("  %d: %s\n", i+1, move)
	}
	
	// Try to get best move
	bestMove, err := client.GetBestMove(fen, 100)
	if err != nil {
		fmt.Printf("Error getting best move: %v\n", err)
	} else {
		fmt.Printf("Stockfish best move: %s\n", bestMove)
	}
	
	// Additional analysis
	fmt.Printf("\nChecking position validity...\n")
	valid, err := client.ValidatePosition(fen)
	if err != nil {
		fmt.Printf("Error validating position: %v\n", err)
	} else {
		fmt.Printf("Position valid: %v\n", valid)
	}
}
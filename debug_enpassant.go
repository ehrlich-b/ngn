package main

import (
	"fmt"
	"github.com/ehrlich-b/ngn/engine"
)

func main() {
	// BUG #1 position
	fen := "rnbqkbnr/pp2pppp/8/2pP4/8/8/PPP2PPP/RNBQK1NR b KQkq c6 0 3"
	pos, err := engine.ParseFEN(fen)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Debugging BUG #1 - En passant diagonal move\n")
	fmt.Printf("Position: %s\n", fen)
	
	// Check the position state
	fmt.Printf("\nPosition analysis:\n")
	fmt.Printf("  En passant target: %s\n", pos.EnPassant.Name())
	fmt.Printf("  Piece on b7: %v\n", pos.Board.PieceAt(engine.B7))
	fmt.Printf("  Piece on c6: %v\n", pos.Board.PieceAt(engine.C6))
	fmt.Printf("  Piece on c5: %v\n", pos.Board.PieceAt(engine.C5))
	fmt.Printf("  Piece on d5: %v\n", pos.Board.PieceAt(engine.D5))
	
	// Generate moves
	moves := engine.GenerateLegalMoves(pos)
	fmt.Printf("\nLegal moves generated: %d\n", len(moves))
	
	// Find the b7c6 move
	var illegalMove engine.Move
	for _, move := range moves {
		if move.ToString() == "b7c6" {
			illegalMove = move
			break
		}
	}
	
	if illegalMove != 0 {
		fmt.Printf("\nFound illegal move b7c6:\n")
		fmt.Printf("  From: %s\n", illegalMove.Source().Name())
		fmt.Printf("  To: %s\n", illegalMove.Destination().Name())
		fmt.Printf("  Moving piece: %v\n", illegalMove.MovingPiece())
		fmt.Printf("  Captured piece: %v\n", illegalMove.CapturedPiece())
		fmt.Printf("  Is capture: %v\n", illegalMove.IsCapture())
		fmt.Printf("  Is en passant: %v\n", illegalMove.IsEnPassant())
		
		fmt.Printf("\nThis move violates chess rules - pawns can't move diagonally to empty squares!\n")
	} else {
		fmt.Printf("\nb7c6 move not found in our generation\n")
	}
}
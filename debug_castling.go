package main

import (
	"fmt"
	"github.com/ehrlich-b/ngn/engine"
)

func main() {
	// Castling bug position
	fen := "r4rk1/8/8/8/8/8/8/R2QK2R w KQ - 1 1"
	pos, err := engine.ParseFEN(fen)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Position: %s\n", fen)
	
	// Find the castling move e1g1
	moves := engine.GenerateLegalMoves(pos)
	var castleMove engine.Move
	for _, move := range moves {
		if move.ToString() == "e1g1" {
			castleMove = move
			break
		}
	}
	
	if castleMove == 0 {
		fmt.Printf("e1g1 castling move not found in our legal moves!\n")
		return
	}
	
	fmt.Printf("Found castling move: %s\n", castleMove.ToString())
	fmt.Printf("Is castle: %v\n", castleMove.IsCastle())
	fmt.Printf("Is kingside castle: %v\n", castleMove.IsKingSideCastle())
	
	// Check castling legality step by step
	fmt.Printf("\nCastling legality check:\n")
	
	// 1. King in check before castling?
	whiteInCheck := engine.IsInCheckDirect(pos, engine.White)
	fmt.Printf("1. White king in check before castling: %v\n", whiteInCheck)
	
	// 2. Check if f1 and g1 are attacked by Black
	fmt.Printf("2. Checking if castling path is attacked:\n")
	
	// I'll need to add this function to check square attacks
	// For now, let me manually check the position
	fmt.Printf("   Position analysis:\n")
	fmt.Printf("   f1 piece: %v\n", pos.Board.PieceAt(engine.F1))
	fmt.Printf("   g1 piece: %v\n", pos.Board.PieceAt(engine.G1))
	fmt.Printf("   f8 piece: %v\n", pos.Board.PieceAt(engine.F8))
	fmt.Printf("   g8 piece: %v\n", pos.Board.PieceAt(engine.G8))
	
	fmt.Printf("\nThe black rook on f8 attacks the f-file\n")
	fmt.Printf("If f1 is attacked by the rook, castling should be illegal\n")
	
	// Test our square attack detection
	fmt.Printf("\n3. Testing square attack detection:\n")
	
	// Get f1 square constant
	f1Square := engine.Square(5) // f1 = file f (5) + rank 1 (0) = 0*8 + 5 = 5
	g1Square := engine.Square(6) // g1 = file g (6) + rank 1 (0) = 0*8 + 6 = 6
	
	fmt.Printf("   f1 square constant: %d\n", f1Square)
	fmt.Printf("   g1 square constant: %d\n", g1Square)
	
	// Test if our IsLegalMove is checking these squares correctly
	fmt.Printf("\n4. Manual castling legality test:\n")
	isLegal := engine.IsLegalMove(pos, castleMove)
	fmt.Printf("   Our IsLegalMove says: %v\n", isLegal)
	fmt.Printf("   But Stockfish says it's illegal - there's the bug!\n")
}
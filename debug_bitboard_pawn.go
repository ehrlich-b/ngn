package main

import (
	"fmt"
	"github.com/ehrlich-b/ngn/engine"
)

func main() {
	// Test bitboard pawn attack calculation
	fen := "k7/P7/K7/8/8/8/8/8 b - - 0 1"
	pos, err := engine.ParseFEN(fen)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Debugging bitboard pawn attack calculation\n")
	fmt.Printf("Position: %s\n", fen)
	
	// After a8b8 move - black king on b8
	moves := engine.GenerateLegalMoves(pos)
	if len(moves) > 0 {
		ep, tag, hc, _ := pos.MakeMove(moves[0]) // a8b8
		
		fmt.Printf("After a8b8: Black king on b8\n")
		
		kingSquare := engine.B8  // b8 = 57
		fmt.Printf("King square (b8): %d\n", kingSquare)
		
		// Manual calculation from isInCheck function:
		// rightAttack := (SquareMask[kingSquare] >> 9) & ^FileMasks[FileA]
		
		fmt.Printf("\nTracing bitboard calculation:\n")
		fmt.Printf("kingSquare: %d\n", kingSquare)
		
		// Check what SquareMask[57] looks like
		kingMask := engine.SquareMask[kingSquare]
		fmt.Printf("SquareMask[%d]: %064b\n", kingSquare, kingMask)
		
		// Shift right by 9
		shiftedRight := kingMask >> 9
		fmt.Printf("Shifted >> 9: %064b\n", shiftedRight)
		fmt.Printf("Shifted >> 9 (decimal): %d\n", shiftedRight)
		
		// Apply file mask
		fileAMask := engine.FileMasks[engine.FileA]
		fmt.Printf("FileMasks[FileA]: %064b\n", fileAMask)
		
		notFileAMask := ^fileAMask
		fmt.Printf("^FileMasks[FileA]: %064b\n", notFileAMask)
		
		rightAttack := shiftedRight & notFileAMask
		fmt.Printf("Final rightAttack: %064b\n", rightAttack)
		fmt.Printf("Final rightAttack (decimal): %d\n", rightAttack)
		
		// Check if white pawn bitboard intersects
		whitePawnBB := pos.Board.GetBitboardOf(engine.WhitePawn)
		fmt.Printf("White pawn bitboard: %064b\n", whitePawnBB)
		
		intersection := rightAttack & whitePawnBB
		fmt.Printf("Intersection: %064b\n", intersection)
		fmt.Printf("Attack detected: %v\n", intersection != 0)
		
		pos.UnMakeMove(moves[0], tag, ep, hc)
	}
}
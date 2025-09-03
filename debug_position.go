package main

import (
	"fmt"
	"github.com/ehrlich-b/ngn/engine"
)

func main() {
	// BUG #2 position: Queen generates 24 illegal moves
	fen := "4k3/8/8/8/3Qp3/8/8/8 w - - 0 1"
	
	pos, err := engine.ParseFEN(fen)
	if err != nil {
		fmt.Printf("Error parsing FEN: %v\n", err)
		return
	}
	
	fmt.Printf("Position: %s\n", fen)
	fmt.Printf("Turn: %v\n", pos.Turn())
	fmt.Printf("King in check: %v\n", engine.IsInCheckDirect(pos, pos.Turn()))
	
	// Generate pseudo-legal moves
	pseudoMoves := engine.GenerateMoves(pos)
	fmt.Printf("Pseudo-legal moves: %d\n", len(pseudoMoves))
	for i, move := range pseudoMoves {
		fmt.Printf("  %d: %s\n", i+1, move.ToString())
	}
	
	// Generate legal moves
	legalMoves := engine.GenerateLegalMoves(pos)
	fmt.Printf("Legal moves: %d\n", len(legalMoves))
	for i, move := range legalMoves {
		fmt.Printf("  %d: %s\n", i+1, move.ToString())
	}
	
	// Manual analysis: what pieces are on the board?
	fmt.Printf("\nBoard analysis:\n")
	fmt.Printf("White queen on d4: %v\n", pos.Board.PieceAt(engine.D4))
	fmt.Printf("Black pawn on e4: %v\n", pos.Board.PieceAt(engine.E4))
	fmt.Printf("Black king on e8: %v\n", pos.Board.PieceAt(engine.E8))
	
	// Analysis: would Black have any legal moves after white queen moves?
	fmt.Printf("\nStalemate analysis:\n")
	
	// Create a copy and switch turns to see black's moves
	posCopy := *pos
	if pos.Turn() == engine.White {
		posCopy.SetTag(engine.BlackToMove)
		posCopy.ClearTag(engine.WhiteToMove)
	} else {
		posCopy.SetTag(engine.WhiteToMove)  
		posCopy.ClearTag(engine.BlackToMove)
	}
	
	blackMoves := engine.GenerateLegalMoves(&posCopy)
	fmt.Printf("Black legal moves: %d\n", len(blackMoves))
	for i, move := range blackMoves {
		fmt.Printf("  %d: %s\n", i+1, move.ToString())
	}
	
	fmt.Printf("Black king in check: %v\n", engine.IsInCheckDirect(&posCopy, engine.Black))
	
	// If Black has 0 moves and is not in check, this is stalemate
	if len(blackMoves) == 0 && !engine.IsInCheckDirect(&posCopy, engine.Black) {
		fmt.Printf("STALEMATE POSITION: Black has no moves and is not in check\n")
		fmt.Printf("Therefore, all white moves should be ILLEGAL (cause stalemate)\n")
	} else {
		fmt.Printf("NOT stalemate - Black has moves available\n")
	}
	
	// Test what happens when we make one of our "legal" moves
	if len(legalMoves) > 0 {
		fmt.Printf("\nTesting move: %s\n", legalMoves[0].ToString())
		
		// Make the move
		ep, tag, hc, _ := pos.MakeMove(legalMoves[0])
		
		// Check the resulting position
		fmt.Printf("After move - Turn: %v\n", pos.Turn())
		fmt.Printf("After move - Black in check: %v\n", engine.IsInCheckDirect(pos, engine.Black))
		
		// Check Black's moves after this move
		blackMovesAfter := engine.GenerateLegalMoves(pos)
		fmt.Printf("Black moves after white move: %d\n", len(blackMovesAfter))
		
		if len(blackMovesAfter) == 0 {
			if engine.IsInCheckDirect(pos, engine.Black) {
				fmt.Printf("CHECKMATE! Black is in check with no legal moves\n")
			} else {
				fmt.Printf("STALEMATE! Black has no legal moves but is not in check\n")
			}
		}
		
		// Unmake the move
		pos.UnMakeMove(legalMoves[0], tag, ep, hc)
	}
}
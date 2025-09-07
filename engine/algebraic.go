package engine

import (
	"fmt"
	"strings"
)

// ParseAlgebraicMove parses a move in algebraic notation (e.g., "e2e4", "e7e8q")
// and returns the corresponding Move for the given position
func ParseAlgebraicMove(moveStr string, pos *Position) (Move, error) {
	moveStr = strings.ToLower(moveStr)

	if len(moveStr) < 4 {
		return EmptyMove, fmt.Errorf("move string too short: %s", moveStr)
	}

	// Parse source square
	fromSquare, err := ParseSquare(moveStr[0:2])
	if err != nil {
		return EmptyMove, fmt.Errorf("invalid from square: %v", err)
	}

	// Parse destination square
	toSquare, err := ParseSquare(moveStr[2:4])
	if err != nil {
		return EmptyMove, fmt.Errorf("invalid to square: %v", err)
	}

	// Get the piece at the source square
	movingPiece := pos.Board.PieceAt(fromSquare)
	if movingPiece == NoPiece {
		return EmptyMove, fmt.Errorf("no piece at square %s", moveStr[0:2])
	}

	// Check if it's the right color's turn
	if movingPiece.Color() != pos.Turn() {
		return EmptyMove, fmt.Errorf("wrong color piece at %s", moveStr[0:2])
	}

	// Get captured piece (if any)
	capturedPiece := pos.Board.PieceAt(toSquare)

	// Parse promotion (if present)
	var promoType PieceType = NoType
	if len(moveStr) == 5 {
		switch moveStr[4] {
		case 'q':
			promoType = Queen
		case 'r':
			promoType = Rook
		case 'b':
			promoType = Bishop
		case 'n':
			promoType = Knight
		default:
			return EmptyMove, fmt.Errorf("invalid promotion piece: %c", moveStr[4])
		}
	}

	// Determine move flags
	var moveTag MoveTag = 0

	// Check for castling
	if movingPiece.Type() == King {
		fromFile := int(fromSquare % 8)
		toFile := int(toSquare % 8)

		if fromFile == 4 && toFile == 6 { // King side castle
			moveTag |= KingSideCastle
		} else if fromFile == 4 && toFile == 2 { // Queen side castle
			moveTag |= QueenSideCastle
		}
	}

	// Check for en passant
	if movingPiece.Type() == Pawn && capturedPiece == NoPiece && toSquare == pos.EnPassant {
		moveTag |= EnPassant
		// In en passant, we capture the pawn behind the target square
		if pos.Turn() == White {
			capturedPiece = BlackPawn
		} else {
			capturedPiece = WhitePawn
		}
	}

	// Check for regular capture
	if capturedPiece != NoPiece {
		moveTag |= Capture
	}

	// Create the move
	move := NewMove(fromSquare, toSquare, movingPiece, capturedPiece, promoType, moveTag)

	// Validate that this move is legal
	legalMoves := GenerateLegalMoves(pos)
	for _, legalMove := range legalMoves {
		if move == legalMove {
			return move, nil
		}
	}

	return EmptyMove, fmt.Errorf("illegal move: %s", moveStr)
}

// MoveToAlgebraic converts a Move to algebraic notation
func MoveToAlgebraic(move Move) string {
	result := move.Source().Name() + move.Destination().Name()

	// Add promotion piece if any
	if move.PromoType() != NoType {
		switch move.PromoType() {
		case Queen:
			result += "q"
		case Rook:
			result += "r"
		case Bishop:
			result += "b"
		case Knight:
			result += "n"
		}
	}

	return result
}

// ParseLongAlgebraicNotation parses moves in long algebraic notation
// This is an alias for ParseAlgebraicMove for clarity
func ParseLongAlgebraicNotation(moveStr string, pos *Position) (Move, error) {
	return ParseAlgebraicMove(moveStr, pos)
}

// ValidateAlgebraicMove checks if an algebraic move string is valid for the position
func ValidateAlgebraicMove(moveStr string, pos *Position) error {
	_, err := ParseAlgebraicMove(moveStr, pos)
	return err
}

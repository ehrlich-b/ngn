package engine

import (
	"fmt"
	"math/bits"

	"github.com/ehrlich-b/ngn/countereval"
	"github.com/ehrlich-b/ngn/nnue"
)

// counterBoardFromPosition uses the engine's maintained per-piece bitboards.
// It performs no FEN conversion or mailbox scan on search transitions.
func counterBoardFromPosition(pos *Position) (countereval.Board, error) {
	if pos == nil {
		return countereval.Board{}, fmt.Errorf("%w: nil Counter position", errWorkerEvaluator)
	}
	var board countereval.Board
	for piece := WhitePawn; piece <= BlackKing; piece++ {
		board[piece-WhitePawn] = pos.Board.GetBitboardOf(piece)
	}
	return board, nil
}

// counterMoveDelta translates the already-established NGN semantic move bridge
// into Counter's plane/square representation without duplicating move rules.
func counterMoveDelta(pos *Position, move Move) (countereval.MoveDelta, error) {
	source, err := nnueMoveDelta(pos, move)
	if err != nil {
		return countereval.MoveDelta{}, err
	}
	movingPlane, err := counterPlane(source.Removed[0])
	if err != nil {
		return countereval.MoveDelta{}, err
	}
	result := countereval.MoveDelta{
		MovingPlane: uint8(movingPlane),
		From:        uint8(source.Removed[0].Square),
		To:          uint8(source.Added[0].Square),
	}
	switch source.Kind {
	case nnue.MoveNormal:
	case nnue.MoveCapture, nnue.MoveEnPassant:
		capturedPlane, planeErr := counterPlane(source.Removed[1])
		if planeErr != nil {
			return countereval.MoveDelta{}, planeErr
		}
		result.HasCapture = true
		result.CapturedPlane = uint8(capturedPlane)
		result.CaptureSquare = uint8(source.Removed[1].Square)
	case nnue.MovePromotion, nnue.MovePromotionCapture:
		promotionPlane, planeErr := counterPlane(source.Added[0])
		if planeErr != nil {
			return countereval.MoveDelta{}, planeErr
		}
		result.HasPromotion = true
		result.PromotionPlane = uint8(promotionPlane)
		if source.Kind == nnue.MovePromotionCapture {
			capturedPlane, capturedErr := counterPlane(source.Removed[1])
			if capturedErr != nil {
				return countereval.MoveDelta{}, capturedErr
			}
			result.HasCapture = true
			result.CapturedPlane = uint8(capturedPlane)
			result.CaptureSquare = uint8(source.Removed[1].Square)
		}
	case nnue.MoveCastle:
		rookBefore, planeErr := counterPlane(source.Removed[1])
		if planeErr != nil {
			return countereval.MoveDelta{}, planeErr
		}
		rookAfter, afterErr := counterPlane(source.Added[1])
		if afterErr != nil || rookAfter != rookBefore {
			return countereval.MoveDelta{}, fmt.Errorf("%w: malformed Counter castle rook", errWorkerEvaluator)
		}
		result.HasCastleRook = true
		result.CastleRookFrom = uint8(source.Removed[1].Square)
		result.CastleRookTo = uint8(source.Added[1].Square)
	default:
		return countereval.MoveDelta{}, fmt.Errorf("%w: unknown Counter move kind %d", errWorkerEvaluator, source.Kind)
	}
	return result, nil
}

func counterPlane(piece nnue.PieceOnSquare) (int, error) {
	if piece.Color > nnue.Black || piece.Piece > nnue.King || piece.Square >= 64 {
		return 0, fmt.Errorf("%w: invalid Counter piece %+v", errWorkerEvaluator, piece)
	}
	return int(piece.Color)*6 + int(piece.Piece), nil
}

func validateCounterMoveAfter(pos *Position, delta countereval.MoveDelta) (countereval.Board, error) {
	if pos == nil {
		return countereval.Board{}, fmt.Errorf("%w: nil Counter post-move position", errWorkerEvaluator)
	}
	movingWhite := delta.MovingPlane < 6
	if (pos.Turn() == White) == movingWhite {
		return countereval.Board{}, fmt.Errorf("%w: Counter move did not toggle side", errWorkerEvaluator)
	}
	return counterBoardFromPosition(pos)
}

func counterNonPawnMaterial(pos *Position) int64 {
	minor := pos.Board.GetBitboardOf(WhiteKnight) | pos.Board.GetBitboardOf(WhiteBishop) |
		pos.Board.GetBitboardOf(BlackKnight) | pos.Board.GetBitboardOf(BlackBishop)
	rooks := pos.Board.GetBitboardOf(WhiteRook) | pos.Board.GetBitboardOf(BlackRook)
	queens := pos.Board.GetBitboardOf(WhiteQueen) | pos.Board.GetBitboardOf(BlackQueen)
	return 4*int64(bits.OnesCount64(minor)) + 6*int64(bits.OnesCount64(rooks)) +
		12*int64(bits.OnesCount64(queens))
}

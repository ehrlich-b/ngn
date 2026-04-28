package engine

import (
	"math"
	"math/bits"
	"sync/atomic"
	"time"
)

const (
	INFINITY                = int(math.MaxInt32)
	MATE_VALUE              = 30000
	MATE_IN_MAX             = MATE_VALUE - 1000
	NULL_MOVE_R             = 3   // Null move reduction depth
	NULL_MOVE_MIN_DEPTH     = 3   // Minimum depth for null move pruning
	LMR_FULL_DEPTH          = 3   // Minimum depth for LMR
	LMR_MOVE_THRESHOLD      = 3   // Start reducing after this many moves
	FUTILITY_MARGIN         = 200 // Futility pruning margin
	FUTILITY_MAX_DEPTH      = 3   // Extended futility pruning depth
	REVERSE_FUTILITY_MARGIN = 120 // Reverse futility margin
	RAZORING_MARGIN         = 400 // Razoring margin (higher = less aggressive)
	RAZORING_MAX_DEPTH      = 3   // Maximum depth for razoring (lower = less aggressive)
	DELTA_MARGIN            = 100 // Delta pruning margin in quiescence
	SINGULAR_DEPTH          = 6   // Minimum depth for singular extensions
	SINGULAR_MARGIN         = 64  // Margin for singular move detection
)

// globalStopRequested is set by UCI's stop handler and observed by the search
// hot paths. It exists because TimeManager alone cannot interrupt search:
// - For `go infinite` it never asks to stop.
// - For other modes, stop must be honored even before the soft/hard time hits.
// Cleared at the start of every search; set by RequestStop().
var globalStopRequested atomic.Bool

// RequestStop signals running searches to stop as soon as practical.
func RequestStop() {
	globalStopRequested.Store(true)
}

// ClearStop clears any pending stop request (called at search entry).
func ClearStop() {
	globalStopRequested.Store(false)
}

// IsStopRequested reports whether a stop has been requested by an external
// caller (e.g. UCI `stop`). Cheap atomic load, safe to poll in hot paths.
func IsStopRequested() bool {
	return globalStopRequested.Load()
}

// Late Move Pruning thresholds by depth
// At depth d, prune quiet moves after lmpThreshold[d] moves
var lmpThreshold = [8]int{0, 5, 8, 12, 17, 23, 30, 38}

// scoreToTT converts a search score to a value safe to store in the TT, with
// mate scores adjusted to be node-relative ("mate-distance-from-this-node")
// rather than root-relative. Without this, a TT hit retrieved at a different
// ply via the same hash would report the wrong distance to mate.
func scoreToTT(score, ply int) int16 {
	if score >= MATE_IN_MAX {
		return int16(score + ply)
	}
	if score <= -MATE_IN_MAX {
		return int16(score - ply)
	}
	return int16(score)
}

// scoreFromTT inverts scoreToTT: takes a node-relative TT score and returns a
// root-relative score for the current node's ply.
func scoreFromTT(score int16, ply int) int {
	s := int(score)
	if s >= MATE_IN_MAX {
		return s - ply
	}
	if s <= -MATE_IN_MAX {
		return s + ply
	}
	return s
}

// Static Exchange Evaluation (SEE) implementation
// Returns the material balance after a series of exchanges on the target square
func staticExchangeEvaluation(pos *Position, move Move) int {
	// If not a capture, return 0
	if !move.IsCapture() && !move.IsEnPassant() {
		return 0
	}

	from := move.Source()
	to := move.Destination()
	targetSquare := to

	// Handle en passant specially
	if move.IsEnPassant() {
		// For en passant, the captured pawn is on a different square
		if move.MovingPiece().Color() == White {
			targetSquare = Square(int(to) - 8) // Black pawn is one rank down
		} else {
			targetSquare = Square(int(to) + 8) // White pawn is one rank up
		}
	}

	// Get the initial captured piece value
	capturedValue := move.CapturedPiece().seeWeight()

	// Get all attackers of the target square
	attackers := getAttackers(pos, targetSquare)

	// Remove the moving piece from attackers (it just moved)
	movingPiece := move.MovingPiece()
	attackers &^= SquareMask[int(from)]

	// Initialize the exchange - use stack array to avoid allocation
	var gain [32]int // Exchange gains on stack
	depth := 0
	color := movingPiece.Color().Other() // Side to move after initial capture
	occupied := pos.Board.GetWhitePieces() | pos.Board.GetBlackPieces()

	// Initial capture
	gain[depth] = int(capturedValue)

	for {
		depth++
		if depth >= 32 {
			break // Prevent infinite loops
		}

		// Find the least valuable attacker for the current side
		attackerSquare, attackerPiece := findLeastValuableAttacker(pos, attackers, color, occupied, targetSquare)
		if attackerSquare == NoSquare {
			break // No more attackers
		}

		// Remove this attacker from the occupied bitboard
		occupied &^= SquareMask[int(attackerSquare)]

		// Update attackers (remove pieces that can no longer attack)
		attackers = getAttackersAfterMove(pos, attackers, occupied, targetSquare, attackerSquare)

		// Add the gain/loss for this exchange
		gain[depth] = int(attackerPiece.seeWeight()) - gain[depth-1]

		// Switch sides
		color = color.Other()
	}

	// Evaluate the exchange from the end backwards
	for depth > 0 {
		depth--
		if gain[depth] > -gain[depth+1] {
			gain[depth] = -gain[depth+1]
		}
	}

	return gain[0]
}

// getAttackers returns a bitboard of all pieces attacking the given square
func getAttackers(pos *Position, square Square) uint64 {
	attackers := uint64(0)

	// Pawn attacks
	whitePawns := pos.Board.GetBitboardOf(WhitePawn)
	blackPawns := pos.Board.GetBitboardOf(BlackPawn)

	// White pawn attacks
	if square.Rank() > Rank1 {
		if square.File() > FileA {
			attackSq := int(square) - 7
			if attackSq >= 0 && attackSq < 64 {
				attackers |= (whitePawns & SquareMask[attackSq])
			}
		}
		if square.File() < FileH {
			attackSq := int(square) - 9
			if attackSq >= 0 && attackSq < 64 {
				attackers |= (whitePawns & SquareMask[attackSq])
			}
		}
	}

	// Black pawn attacks
	if square.Rank() < Rank8 {
		if square.File() > FileA {
			attackSq := int(square) + 9
			if attackSq >= 0 && attackSq < 64 {
				attackers |= (blackPawns & SquareMask[attackSq])
			}
		}
		if square.File() < FileH {
			attackSq := int(square) + 7
			if attackSq >= 0 && attackSq < 64 {
				attackers |= (blackPawns & SquareMask[attackSq])
			}
		}
	}

	// Knight attacks
	whiteKnights := pos.Board.GetBitboardOf(WhiteKnight)
	blackKnights := pos.Board.GetBitboardOf(BlackKnight)
	knightAttacks := KnightAttacks[square]
	attackers |= (whiteKnights | blackKnights) & knightAttacks

	// King attacks
	whiteKing := pos.Board.GetBitboardOf(WhiteKing)
	blackKing := pos.Board.GetBitboardOf(BlackKing)
	kingAttacks := KingAttacks[square]
	attackers |= (whiteKing | blackKing) & kingAttacks

	// Bishop/Queen diagonal attacks
	whiteBishops := pos.Board.GetBitboardOf(WhiteBishop)
	blackBishops := pos.Board.GetBitboardOf(BlackBishop)
	whiteQueens := pos.Board.GetBitboardOf(WhiteQueen)
	blackQueens := pos.Board.GetBitboardOf(BlackQueen)

	diagonalAttackers := whiteBishops | blackBishops | whiteQueens | blackQueens
	attackers |= diagonalAttackers & seeGetBishopAttacks(square, pos.Board.GetWhitePieces()|pos.Board.GetBlackPieces())

	// Rook/Queen straight attacks
	whiteRooks := pos.Board.GetBitboardOf(WhiteRook)
	blackRooks := pos.Board.GetBitboardOf(BlackRook)

	straightAttackers := whiteRooks | blackRooks | whiteQueens | blackQueens
	attackers |= straightAttackers & seeGetRookAttacks(square, pos.Board.GetWhitePieces()|pos.Board.GetBlackPieces())

	return attackers
}

// seeGetBishopAttacks returns squares attacked by bishops/queens on diagonals
// Uses magic bitboards for O(1) lookup
func seeGetBishopAttacks(square Square, occupied uint64) uint64 {
	return GetBishopAttacks(int(square), occupied)
}

// seeGetRookAttacks returns squares attacked by rooks/queens on ranks/files
// Uses magic bitboards for O(1) lookup
func seeGetRookAttacks(square Square, occupied uint64) uint64 {
	return GetRookAttacks(int(square), occupied)
}

// findLeastValuableAttacker finds the least valuable piece attacking the square
func findLeastValuableAttacker(pos *Position, attackers uint64, color Color, occupied uint64, square Square) (Square, Piece) {
	// Order: Pawn, Knight, Bishop, Rook, Queen, King
	pieceTypes := []PieceType{Pawn, Knight, Bishop, Rook, Queen, King}

	for _, pieceType := range pieceTypes {
		piece := GetPiece(pieceType, color)
		pieceBB := pos.Board.GetBitboardOf(piece) & attackers

		for pieceBB != 0 {
			sq := Square(bits.TrailingZeros64(pieceBB))

			// Verify this piece can actually attack the square
			if canPieceAttackSquare(pos, sq, square, occupied) {
				return sq, piece
			}

			pieceBB &^= SquareMask[int(sq)]
		}
	}

	return NoSquare, NoPiece
}

// canPieceAttackSquare checks if a piece on fromSq can attack toSq
func canPieceAttackSquare(pos *Position, fromSq Square, toSq Square, occupied uint64) bool {
	piece := pos.Board.PieceAt(fromSq)

	switch piece.Type() {
	case Pawn:
		// Check if pawn can attack the target square diagonally
		color := piece.Color()
		fromFile := int(fromSq) % 8
		toFile := int(toSq) % 8
		fileDiff := fromFile - toFile
		if fileDiff < 0 {
			fileDiff = -fileDiff
		}
		if fileDiff != 1 {
			return false // Pawns only attack diagonally (1 file difference)
		}
		if color == White {
			return int(toSq)-int(fromSq) == 7 || int(toSq)-int(fromSq) == 9
		}
		return int(fromSq)-int(toSq) == 7 || int(fromSq)-int(toSq) == 9
	case Knight:
		return (KnightAttacks[fromSq] & SquareMask[int(toSq)]) != 0
	case Bishop:
		return (seeGetBishopAttacks(fromSq, occupied) & SquareMask[int(toSq)]) != 0
	case Rook:
		return (seeGetRookAttacks(fromSq, occupied) & SquareMask[int(toSq)]) != 0
	case Queen:
		return ((seeGetBishopAttacks(fromSq, occupied) | seeGetRookAttacks(fromSq, occupied)) & SquareMask[int(toSq)]) != 0
	case King:
		return (KingAttacks[fromSq] & SquareMask[int(toSq)]) != 0
	}

	return false
}

// getAttackersAfterMove updates attackers after a piece has moved
func getAttackersAfterMove(pos *Position, attackers uint64, occupied uint64, targetSquare Square, movedFrom Square) uint64 {
	// For simplicity, recalculate attackers. In a full implementation,
	// this could be optimized by updating incrementally.
	// But for SEE, this is acceptable since it's not performance-critical.

	// Create a temporary position with updated occupancy
	tempAttackers := uint64(0)

	// Pawn attacks
	whitePawns := pos.Board.GetBitboardOf(WhitePawn) & occupied
	blackPawns := pos.Board.GetBitboardOf(BlackPawn) & occupied

	// White pawn attacks
	if targetSquare.Rank() > Rank1 {
		if targetSquare.File() > FileA {
			attackSq := int(targetSquare) - 7
			if attackSq >= 0 && attackSq < 64 {
				tempAttackers |= (whitePawns & SquareMask[attackSq])
			}
		}
		if targetSquare.File() < FileH {
			attackSq := int(targetSquare) - 9
			if attackSq >= 0 && attackSq < 64 {
				tempAttackers |= (whitePawns & SquareMask[attackSq])
			}
		}
	}

	// Black pawn attacks
	if targetSquare.Rank() < Rank8 {
		if targetSquare.File() > FileA {
			attackSq := int(targetSquare) + 9
			if attackSq >= 0 && attackSq < 64 {
				tempAttackers |= (blackPawns & SquareMask[attackSq])
			}
		}
		if targetSquare.File() < FileH {
			attackSq := int(targetSquare) + 7
			if attackSq >= 0 && attackSq < 64 {
				tempAttackers |= (blackPawns & SquareMask[attackSq])
			}
		}
	}

	// Knight attacks
	whiteKnights := pos.Board.GetBitboardOf(WhiteKnight) & occupied
	blackKnights := pos.Board.GetBitboardOf(BlackKnight) & occupied
	knightAttacks := KnightAttacks[targetSquare]
	tempAttackers |= (whiteKnights | blackKnights) & knightAttacks

	// King attacks
	whiteKing := pos.Board.GetBitboardOf(WhiteKing) & occupied
	blackKing := pos.Board.GetBitboardOf(BlackKing) & occupied
	kingAttacks := KingAttacks[targetSquare]
	tempAttackers |= (whiteKing | blackKing) & kingAttacks

	// Bishop/Queen diagonal attacks
	whiteBishops := pos.Board.GetBitboardOf(WhiteBishop) & occupied
	blackBishops := pos.Board.GetBitboardOf(BlackBishop) & occupied
	whiteQueens := pos.Board.GetBitboardOf(WhiteQueen) & occupied
	blackQueens := pos.Board.GetBitboardOf(BlackQueen) & occupied

	diagonalAttackers := whiteBishops | blackBishops | whiteQueens | blackQueens
	tempAttackers |= diagonalAttackers & seeGetBishopAttacks(targetSquare, occupied)

	// Rook/Queen straight attacks
	whiteRooks := pos.Board.GetBitboardOf(WhiteRook) & occupied
	blackRooks := pos.Board.GetBitboardOf(BlackRook) & occupied

	straightAttackers := whiteRooks | blackRooks | whiteQueens | blackQueens
	tempAttackers |= straightAttackers & seeGetRookAttacks(targetSquare, occupied)

	return tempAttackers
}

// Search debug logging removed for production

type SearchInfo struct {
	Nodes       uint64
	Depth       int
	BestMove    Move
	BestScore   int
	Stopped     bool
	TimeManager *TimeManager
	PV          [MaximumDepth]Move // Principal variation array
	PVLength    int                // Length of current PV

	// Zero-allocation buffers passed through recursion
	MoveBuffer    *[256]Move // For move generation
	CaptureBuffer *[64]Move  // For capture generation
	OrderedBuffer *[256]Move // For move ordering
	SEEGains      *[32]int   // For SEE evaluation

	// Pruning and search efficiency statistics
	FutilityPrunes      uint64 // Moves pruned by futility pruning
	NullMoveCutoffs     uint64 // Beta cutoffs from null move pruning
	LMRReductions       uint64 // Moves reduced by Late Move Reduction
	LMPPrunes           uint64 // Moves pruned by Late Move Pruning
	ProbcutPrunes       uint64 // Positions pruned by Probcut
	TTHits              uint64 // Transposition table hits
	TTCutoffs           uint64 // Beta cutoffs from TT
	BetaCutoffs         uint64 // Total beta cutoffs
	QNodes              uint64 // Quiescence search nodes
	CheckExtensions      uint64 // Check extensions applied
	SingularExtensions   uint64 // Singular extensions applied
	RecaptureExtensions  uint64 // Recapture extensions applied
	PassedPawnExtensions uint64 // Passed pawn extensions applied
	ExcludedMove         Move   // Move to exclude during singular search

	// Repetition detection during search
	RepStack    [512]uint64 // Stack of Zobrist hashes on current search path
	RepStackLen int         // Current depth in repetition stack
}

func Search(pos *Position, depth int) *SearchInfo {
	return SearchIterativeDeepening(pos, depth, nil)
}

// DepthCallback is called after each depth completes during iterative deepening
type DepthCallback func(depth int, score int, bestMove Move, nodes uint64)

func SearchIterativeDeepening(pos *Position, maxDepth int, timeManager *TimeManager) *SearchInfo {
	return SearchIterativeDeepeningWithCallback(pos, maxDepth, timeManager, nil)
}

func SearchIterativeDeepeningWithCallback(pos *Position, maxDepth int, timeManager *TimeManager, callback DepthCallback) *SearchInfo {
	return searchIterativeDeepeningUnsafe(pos, maxDepth, timeManager, callback)
}

func searchIterativeDeepeningUnsafe(pos *Position, maxDepth int, timeManager *TimeManager, callback DepthCallback) *SearchInfo {
	// Clear any stale stop request from a previous search.
	ClearStop()

	// Advance the age of the transposition table for each search
	TranspositionTable.AdvanceAge()

	// Allocate buffers once for the entire search tree
	var moveBuffer [256]Move
	var captureBuffer [64]Move
	var orderedBuffer [256]Move
	var seeGains [32]int

	info := &SearchInfo{
		Nodes:         0,
		Depth:         maxDepth,
		BestMove:      EmptyMove,
		BestScore:     -INFINITY,
		Stopped:       false,
		TimeManager:   timeManager,
		MoveBuffer:    &moveBuffer,
		CaptureBuffer: &captureBuffer,
		OrderedBuffer: &orderedBuffer,
		SEEGains:      &seeGains,
	}

	// Generate root moves without heap allocations (pseudo-legal; legality handled in search loop)
	var rootGen [256]Move
	numGen := GenerateMovesIntoBuffer(pos, rootGen[:])
	moves := rootGen[:numGen]
	if len(moves) == 0 {
		if isInCheck(pos, pos.Turn()) {
			info.BestScore = -MATE_VALUE
		} else {
			info.BestScore = 0
		}
		return info
	}

	// Iterative deepening - start from depth 1 and go up to maxDepth
	previousScore := 0 // Track previous iteration score for aspiration windows

	for currentDepth := 1; currentDepth <= maxDepth && !info.Stopped; currentDepth++ {
		// Honor an external UCI stop request even when no time manager is set
		// (e.g. `go infinite`).
		if IsStopRequested() {
			info.Stopped = true
			break
		}

		// Mark the start of this iteration so the soft-stop heuristic in
		// ShouldStopSearch can measure iteration time.
		if info.TimeManager != nil {
			info.TimeManager.NewIteration()
		}

		// Check time before starting each new depth iteration
		if info.TimeManager != nil {
			if info.TimeManager.ShouldStopSearch(currentDepth, time.Now()) {
				info.Stopped = true
				break
			}
		}

		// Optimized Aspiration Windows: focus on performance over complexity
		var alpha, beta int
		var windowSize int
		if currentDepth <= 3 {
			// Use full window for shallow depths (avoids re-searches)
			alpha = -INFINITY
			beta = INFINITY
		} else {
			// Simple, efficient window sizing
			windowSize = 50 // Fixed 50 centipawn window - good balance of efficiency vs accuracy

			// Only widen for extreme positions to minimize re-searches
			scoreAbs := previousScore
			if scoreAbs < 0 {
				scoreAbs = -scoreAbs
			}

			if scoreAbs > 800 { // Only for very extreme positions (near mate)
				windowSize = 100
			}

			alpha = previousScore - windowSize
			beta = previousScore + windowSize
		}

		// Aspiration window re-search loop
		iterationComplete := false
		attempts := 0
		maxAttempts := 3 // Reduced from 4 to prevent excessive re-searches

		for !iterationComplete && attempts < maxAttempts {
			attempts++

			// Check time before attempting search
			if info.TimeManager != nil && info.TimeManager.ShouldStopSearch(currentDepth, time.Now()) {
				info.Stopped = true
				break
			}

			// Snapshot the window for this attempt before the move loop mutates alpha.
			// Classification at the end must compare against the original window, not
			// the alpha that was raised by successful moves during the loop.
			windowAlpha := alpha
			windowBeta := beta

			// Root search with aspiration window
			var orderedRoot [256]Move
			orderedCount := orderMovesIntoBufferWithDepth(moves, currentDepth, orderedRoot[:], pos)
			iterationBestScore := -INFINITY
			iterationBestMove := EmptyMove

			for oi := 0; oi < orderedCount; oi++ {
				move := orderedRoot[oi]
				// Check time before each move
				if info.TimeManager != nil && info.TimeManager.ShouldStopSearch(currentDepth, time.Now()) {
					info.Stopped = true
					break
				}

				// Use GameMakeMove at root to track repetition history.
				// Thread lastMovePlayed so the counter-move heuristic sees the
				// actual predecessor inside search, not a stale global.
				savedLast := GetLastMovePlayed()
				ep, tag, hc, _ := pos.GameMakeMove(move)
				if isInCheck(pos, move.MovingPiece().Color()) {
					pos.GameUnMakeMove(move, tag, ep, hc)
					continue
				}
				SetLastMovePlayed(move)

				var score int
				if pos.IsFIDEDrawRule() {
					score = 0 // Draw
				} else {
					score = -alphaBetaPV(pos, currentDepth-1, 1, -beta, -alpha, true, info)
				}
				pos.GameUnMakeMove(move, tag, ep, hc)
				SetLastMovePlayed(savedLast)

				info.Nodes++

				if score > iterationBestScore {
					iterationBestScore = score
					iterationBestMove = move
				}

				if score > alpha {
					alpha = score
				}

				// Beta cutoff
				if alpha >= beta {
					break
				}
			}

			// Classify against the original window (windowAlpha/windowBeta), not the
			// alpha that was mutated during the move loop. Otherwise every successful
			// iteration looks like a fail-low because alpha was just raised to
			// iterationBestScore.
			if iterationBestScore <= windowAlpha && currentDepth > 3 {
				// Fail low: widen alpha downward, keep original beta
				alpha = -INFINITY
				beta = windowBeta
			} else if iterationBestScore >= windowBeta && currentDepth > 3 {
				// Fail high: widen beta upward, keep original alpha
				alpha = windowAlpha
				beta = INFINITY
			} else {
				// Search completed successfully within the window
				info.BestScore = iterationBestScore
				info.BestMove = iterationBestMove
				previousScore = iterationBestScore
				iterationComplete = true
			}
		}

		// Only commit the new depth and emit UCI info when the iteration actually
		// completed. If we exited the inner loop on info.Stopped or exhausted
		// maxAttempts without success, keep the previous completed depth/best move.
		if iterationComplete {
			info.Depth = currentDepth
			if callback != nil && info.BestMove != EmptyMove {
				callback(currentDepth, info.BestScore, info.BestMove, info.Nodes)
			}
		}
	}

	return info
}

// ExtractPVFromTT reconstructs the principal variation by walking the
// transposition table from the current position. firstMove is applied at the
// root (since the iterative-deepening root loop doesn't write a TT entry there);
// subsequent moves are read from Exact (PV) TT entries until a miss, a
// non-Exact bound, an illegal move, a cycle, or maxLength is reached.
//
// The position is mutated during the walk and restored before return. Caller
// must run this only after a search iteration has completed (no concurrent
// search writes).
func ExtractPVFromTT(pos *Position, firstMove Move, maxLength int) []Move {
	if maxLength <= 0 || firstMove == EmptyMove {
		return nil
	}
	type undoEntry struct {
		move Move
		ep   Square
		tag  PositionTag
		hc   uint8
	}
	pv := make([]Move, 0, maxLength)
	undoStack := make([]undoEntry, 0, maxLength)

	apply := func(m Move) bool {
		legal := false
		for _, lm := range GenerateLegalMoves(pos) {
			if lm == m {
				legal = true
				break
			}
		}
		if !legal {
			return false
		}
		ep, tag, hc, ok := pos.MakeMove(m)
		if !ok {
			return false
		}
		undoStack = append(undoStack, undoEntry{m, ep, tag, hc})
		pv = append(pv, m)
		return true
	}

	if !apply(firstMove) {
		return nil
	}

	seen := make(map[uint64]struct{}, maxLength)
	for len(pv) < maxLength {
		hash := pos.Hash()
		if _, exists := seen[hash]; exists {
			break
		}
		seen[hash] = struct{}{}

		ttMove, _, _, ttNodeType, ttHit := TranspositionTable.Get(hash)
		if !ttHit || ttMove == EmptyMove || ttNodeType != Exact {
			break
		}
		if !apply(ttMove) {
			break
		}
	}

	for i := len(undoStack) - 1; i >= 0; i-- {
		e := undoStack[i]
		pos.UnMakeMove(e.move, e.tag, e.ep, e.hc)
	}
	return pv
}

func SearchFixed(pos *Position, depth int, timeManager *TimeManager) *SearchInfo {
	return searchFixedUnsafe(pos, depth, timeManager)
}

func searchFixedUnsafe(pos *Position, depth int, timeManager *TimeManager) *SearchInfo {
	// Clear any stale stop request from a previous search.
	ClearStop()

	// Allocate buffers once for the entire search tree
	var moveBuffer [256]Move
	var captureBuffer [64]Move
	var orderedBuffer [256]Move
	var seeGains [32]int

	info := &SearchInfo{
		Nodes:         0, // Fresh node count for each depth
		Depth:         depth,
		BestMove:      EmptyMove,
		BestScore:     -INFINITY,
		Stopped:       false,
		TimeManager:   timeManager,
		MoveBuffer:    &moveBuffer,
		CaptureBuffer: &captureBuffer,
		OrderedBuffer: &orderedBuffer,
		SEEGains:      &seeGains,
	}

	alpha := -INFINITY
	beta := INFINITY

	var rootGen [256]Move
	numGen := GenerateMovesIntoBuffer(pos, rootGen[:])
	moves := rootGen[:numGen]
	if len(moves) == 0 {
		if isInCheck(pos, pos.Turn()) {
			info.BestScore = -MATE_VALUE
		} else {
			info.BestScore = 0
		}
		return info
	}

	var orderedRoot [256]Move
	orderedCount := orderMovesIntoBufferWithDepth(moves, depth, orderedRoot[:], pos)

	for oi := 0; oi < orderedCount; oi++ {
		move := orderedRoot[oi]
		// Check if search was stopped
		if info.Stopped {
			break
		}

		// Check time before each move in fixed depth search
		if info.TimeManager != nil && info.TimeManager.ShouldStopSearch(depth, time.Now()) {
			info.Stopped = true
			break
		}

		savedLast := GetLastMovePlayed()
		ep, tag, hc, _ := pos.MakeMove(move)
		movingColor := move.MovingPiece().Color()
		if isInCheck(pos, movingColor) {
			pos.UnMakeMove(move, tag, ep, hc)
			continue
		}
		SetLastMovePlayed(move)

		score := -alphaBetaPV(pos, depth-1, 1, -beta, -alpha, true, info)

		// Mate threat extension: if we found a very high score, search deeper to find mate
		if score >= MATE_IN_MAX && depth <= 2 {
			score = -alphaBetaPV(pos, depth+2, 1, -beta, -alpha, true, info)
			info.Nodes++
		}

		pos.UnMakeMove(move, tag, ep, hc)
		SetLastMovePlayed(savedLast)
		info.Nodes++

		if score > info.BestScore {
			info.BestScore = score
			info.BestMove = move
		}

		if score > alpha {
			alpha = score
		}

		if alpha >= beta {
			break
		}
	}

	// Workaround: If SearchFixed returns INFINITY but found a move, use a reasonable score
	if info.BestScore == INFINITY && info.BestMove != EmptyMove {
		// Use a high positive score to indicate a strong position
		info.BestScore = 2000
	}

	return info
}

func alphaBeta(pos *Position, depth int, alpha int, beta int, info *SearchInfo) int {
	return alphaBetaPV(pos, depth, 0, alpha, beta, true, info) // Root is always PV
}

func alphaBetaPV(pos *Position, depth int, ply int, alpha int, beta int, isPV bool, info *SearchInfo) int {
	// Safety checks - reasonable limits
	if info.Nodes > 50000000 { // 50M nodes
		info.Stopped = true
		return EvaluateForPlayer(&pos.Board, pos.Turn())
	}

	// External stop request (UCI `stop`) - honor it even before the time limit hits.
	// Cheap atomic load; sampled per node is fine.
	if IsStopRequested() {
		info.Stopped = true
		return EvaluateForPlayer(&pos.Board, pos.Turn())
	}

	// Check time limit if we have a time manager (now cheap to call frequently)
	if info.TimeManager != nil {
		if info.TimeManager.ShouldStopSearch(depth, time.Now()) {
			info.Stopped = true
			return EvaluateForPlayer(&pos.Board, pos.Turn())
		}
	}

	// Safety check for excessive depth (use MaximumDepth constant)
	if depth > MaximumDepth {
		return EvaluateForPlayer(&pos.Board, pos.Turn())
	}

	// Safety check for negative depth (bug prevention)
	if depth < 0 {
		return EvaluateForPlayer(&pos.Board, pos.Turn())
	}

	// Compute hash once and reuse for repetition detection and TT probe.
	hash := pos.Hash()

	// Draw detection: 50-move rule and repetition
	{
		// 50-move rule
		if pos.HalfMoveClock >= 100 {
			return 0
		}

		// Check game history for repetition (count >= 2 means 3-fold with this occurrence)
		if pos.Positions != nil {
			pos.positionsMutex.RLock()
			count := pos.Positions[hash]
			pos.positionsMutex.RUnlock()
			if count >= 2 {
				return 0
			}
		}

		// Check search path for repetition (2-fold in search = draw)
		for i := info.RepStackLen - 1; i >= 0; i-- {
			if info.RepStack[i] == hash {
				return 0
			}
		}

		// Push current hash onto repetition stack
		if info.RepStackLen < len(info.RepStack) {
			info.RepStack[info.RepStackLen] = hash
			info.RepStackLen++
		}
	}
	// Pop repetition stack on function exit
	defer func() { info.RepStackLen-- }()

	if depth == 0 {
		return quiescence(pos, alpha, beta, ply, info)
	}

	// Probe transposition table
	origAlpha := alpha

	ttMove, ttEvalRaw, ttDepth, ttNodeType, ttHit := TranspositionTable.Get(hash)
	ttEval := scoreFromTT(ttEvalRaw, ply)

	if ttHit && !isPV && ttDepth >= int8(depth) {
		info.TTHits++
		switch ttNodeType {
		case Exact:
			return ttEval
		case LowerBound:
			if ttEval >= beta {
				info.TTCutoffs++
				return ttEval
			}
		case UpperBound:
			if ttEval <= alpha {
				return ttEval
			}
		}
	}

	// Internal Iterative Deepening (IID)
	// When we have no TT move at high depth, do a reduced search to find one
	if ttMove == EmptyMove && depth >= 4 && isPV {
		// Search at reduced depth to get a good move for ordering
		iidDepth := depth - 2
		if iidDepth < 1 {
			iidDepth = 1
		}
		alphaBetaPV(pos, iidDepth, ply, alpha, beta, true, info)
		// Re-probe TT to get the move we just found
		ttMove, _, _, _, _ = TranspositionTable.Get(hash)
	}

	// Mate distance pruning - don't look for mates that are too far away
	currentDepth := info.Depth - depth
	if currentDepth > 1000 {
		currentDepth = 1000 // Clamp to prevent overflow
	}
	mateAlpha := -MATE_VALUE + currentDepth
	mateBeta := MATE_VALUE - currentDepth - 1

	if alpha < mateAlpha {
		alpha = mateAlpha
	}
	if beta > mateBeta {
		beta = mateBeta
	}
	if alpha >= beta {
		return alpha
	}

	// Check if we're in check before attempting null move
	inCheck := isInCheck(pos, pos.Turn())

	// Extended futility pruning (depths 1-3) and reverse futility pruning
	var futilityPrune bool
	var staticEval int
	var evalComputed bool

	// Reverse futility pruning - if position is very good, return early
	if depth <= 3 && !inCheck && beta-alpha == 1 {
		staticEval = EvaluateForPlayer(&pos.Board, pos.Turn())
		evalComputed = true
		margin := REVERSE_FUTILITY_MARGIN * depth
		if staticEval-margin >= beta {
			return staticEval - margin
		}
	}

	// PV-aware extended futility pruning for shallow depths
	if depth <= FUTILITY_MAX_DEPTH && !inCheck && !isPV { // Don't prune in PV nodes
		if !evalComputed {
			staticEval = EvaluateForPlayer(&pos.Board, pos.Turn())
			evalComputed = true
		}
		margin := FUTILITY_MARGIN + (depth-1)*50 // Increasing margin with depth
		if staticEval+margin <= alpha {
			futilityPrune = true
		}
	}

	// Note: Razoring pruning removed - futility pruning already provides effective pruning
	// with better performance characteristics for this engine architecture

	// PV-aware null move pruning - more conservative in PV nodes.
	// Guard against zugzwang: in pawn-only endgames, passing can be the best
	// available option for the opponent, so null-move cutoffs are unsound.
	if !inCheck && depth >= NULL_MOVE_MIN_DEPTH && beta-alpha > 1 && (!isPV || depth >= 6) &&
		hasNonPawnMaterial(&pos.Board, pos.Turn()) {
		// Set lastMovePlayed to EmptyMove so children don't pick spurious
		// counter-move replies keyed on our own prior move.
		savedLast := GetLastMovePlayed()
		oldEP := pos.MakeNullMove()
		SetLastMovePlayed(EmptyMove)

		score := -alphaBetaPV(pos, depth-NULL_MOVE_R-1, ply+1, -beta, -beta+1, false, info)

		pos.UnMakeNullMove(oldEP)
		SetLastMovePlayed(savedLast)

		if score >= beta {
			info.NullMoveCutoffs++
			return beta
		}
	}

	// Probcut pruning: if a reduced-depth search at a higher beta fails high,
	// we can prune because the full search would likely also fail high
	if !isPV && !inCheck && depth >= 5 && abs(beta) < MATE_VALUE-100 {
		probcutBeta := beta + 200 // Higher beta threshold

		// Generate captures only for probcut
		var capBuffer [64]Move
		capCount := 0
		var tempBuffer [256]Move
		tempCount := GenerateMovesIntoBuffer(pos, tempBuffer[:])
		for i := 0; i < tempCount && capCount < 64; i++ {
			if tempBuffer[i].IsCapture() {
				capBuffer[capCount] = tempBuffer[i]
				capCount++
			}
		}

		for i := 0; i < capCount; i++ {
			move := capBuffer[i]

			// Only consider captures with good SEE
			seeValue := staticExchangeEvaluation(pos, move)
			if seeValue < 0 {
				continue
			}

			savedLast := GetLastMovePlayed()
			ep, tag, hc, _ := pos.MakeMove(move)
			if isInCheck(pos, move.MovingPiece().Color()) {
				pos.UnMakeMove(move, tag, ep, hc)
				continue
			}
			SetLastMovePlayed(move)

			score := -quiescence(pos, -probcutBeta, -probcutBeta+1, ply+1, info)

			if score >= probcutBeta {
				score = -alphaBetaPV(pos, depth-4, ply+1, -probcutBeta, -probcutBeta+1, false, info)
			}

			pos.UnMakeMove(move, tag, ep, hc)
			SetLastMovePlayed(savedLast)

			if score >= probcutBeta {
				info.ProbcutPrunes++
				return beta
			}
		}
	}

	// Singular extension detection
	// If we have a TT move that's clearly best, extend its search
	singularMove := EmptyMove
	if ttHit && ttMove != EmptyMove && depth >= SINGULAR_DEPTH &&
		info.ExcludedMove == EmptyMove && // Not already in singular search
		(ttNodeType == Exact || ttNodeType == LowerBound) &&
		ttDepth >= int8(depth-3) {
		singularMove = ttMove
	}

	// Use pseudo-legal moves for performance, check legality only when trying moves
	var moveBuffer [256]Move // Stack-allocated move buffer
	numMoves := GenerateMovesIntoBuffer(pos, moveBuffer[:])
	moves := moveBuffer[:numMoves]
	if numMoves == 0 {
		if inCheck {
			// Checkmate - clamp mate distance to prevent overflow
			mateDistance := info.Depth - depth
			if mateDistance > 1000 {
				mateDistance = 1000
			}
			return -MATE_VALUE + mateDistance
		}
		// Stalemate - check if we're throwing away a winning position
		eval := Evaluate(&pos.Board)
		if pos.Turn() == Black {
			eval = -eval // Flip evaluation for black's perspective
		}

		// If we're winning by a significant margin, heavily penalize stalemate
		if eval > 300 { // We have significant advantage (about 3 pawns)
			return -500 // Penalize stalemate when winning
		}
		return 0 // Otherwise, stalemate is still a draw
	}

	// Order moves, prioritizing TT move if available
	var ordered [256]Move
	orderedCount := orderMovesIntoBufferWithDepthPreviousAndTTMove(moves, depth, GetLastMovePlayed(), ttMove, ttHit, ordered[:], pos)
	bestScore := -INFINITY
	bestMove := EmptyMove
	moveCount := 0
	legalTried := 0

	// Track quiet moves searched for history penalization
	var searchedQuiets [64]Move
	searchedQuietCount := 0

	for oi := 0; oi < orderedCount; oi++ {
		move := ordered[oi]
		moveCount++

		// Skip excluded move during singular search
		if move == info.ExcludedMove {
			continue
		}

		// Check time before evaluating each move (now cheap to call)
		if info.TimeManager != nil && info.TimeManager.ShouldStopSearch(depth, time.Now()) {
			info.Stopped = true
			break
		}

		// Futility pruning - skip quiet moves if position is hopeless (but never prune TT move)
		if futilityPrune && !move.IsCapture() && move != ttMove {
			info.FutilityPrunes++
			continue
		}

		// Late Move Pruning (LMP) - skip very late quiet moves at low depths
		// Only apply in non-PV nodes, not in check, and for quiet moves
		// Use legalTried (not moveCount) since moveCount includes illegal moves
		if !isPV && !inCheck && depth > 0 && depth < 8 &&
			!move.IsCapture() && move != ttMove && legalTried > 0 {
			threshold := lmpThreshold[depth]
			if legalTried > threshold {
				info.LMPPrunes++
				continue
			}
		}

		// SEE pruning for bad captures in main search
		// Skip obviously losing captures at low depths (not in PV, not TT move)
		if !isPV && !inCheck && depth <= 4 && move.IsCapture() && move != ttMove {
			seeValue := staticExchangeEvaluation(pos, move)
			if seeValue < -100 { // Skip captures losing more than a pawn
				continue
			}
		}

		// History pruning: skip quiet moves with very bad history at low depths
		if !isPV && !inCheck && depth <= 3 && !move.IsCapture() && move != ttMove {
			histScore := GetHistoryScore(move)
			if histScore < -1000 && legalTried > 3 { // Very bad history and not early move
				continue
			}
		}

		// Fast legality: special-case castling path, otherwise do make/check/unmake
		if move.IsCastle() {
			// Cannot castle out of check
			if inCheck {
				continue
			}
			color := move.MovingPiece().Color()
			var inter, fin Square
			if move.IsKingSideCastle() {
				if color == White {
					inter, fin = F1, G1
				} else {
					inter, fin = F8, G8
				}
			} else { // Queen-side
				if color == White {
					inter, fin = D1, C1
				} else {
					inter, fin = D8, C8
				}
			}
			if isSquareAttacked(pos, inter, color.Other()) || isSquareAttacked(pos, fin, color.Other()) {
				continue
			}
		}

		// Capture the predecessor move (the one that brought us into this node)
		// so children see the correct lastMovePlayed for counter-move lookups,
		// and extension logic uses the true parent move.
		savedLast := GetLastMovePlayed()
		ep, tag, hc, _ := pos.MakeMove(move)
		movingColor := move.MovingPiece().Color()
		if isInCheck(pos, movingColor) {
			pos.UnMakeMove(move, tag, ep, hc)
			continue
		}
		SetLastMovePlayed(move)
		legalTried++

		// Track quiet moves for history penalization
		if !move.IsCapture() && move.PromoType() == NoType && searchedQuietCount < 64 {
			searchedQuiets[searchedQuietCount] = move
			searchedQuietCount++
		}

		// Refined Check Extension logic - balance tactical strength with performance
		givesCheck := isInCheck(pos, pos.Turn())

		// Compute next depth with check extension
		nextDepth := depth - 1
		if givesCheck && depth > 1 {
			nextDepth++
			info.CheckExtensions++
		}

		// Recapture extension: extend when recapturing on the same square.
		// Use savedLast (the true previous move), not the global which now
		// holds our just-played move.
		if move.IsCapture() && savedLast.IsCapture() &&
			move.Destination() == savedLast.Destination() &&
			nextDepth+1 <= depth {
			nextDepth++
			info.RecaptureExtensions++
		}

		// Passed pawn extension: extend for advanced pawn pushes (near promotion)
		if move.MovingPiece().Type() == Pawn && !move.IsCapture() {
			destRank := int(move.Destination()) / 8
			color := move.MovingPiece().Color()
			// Extend if pawn reaches 6th or 7th rank (about to promote)
			if (color == White && destRank >= 5) || (color == Black && destRank <= 2) {
				if nextDepth+1 <= depth {
					nextDepth++
					info.PassedPawnExtensions++
				}
			}
		}

		// Singular Extension: extend if this move is clearly the best
		if move == singularMove && singularMove != EmptyMove {
			// Unmake to do singular search at the parent position.
			// Restore lastMove to savedLast so the singular search (which is
			// evaluating other replies from the parent node) sees the correct
			// predecessor.
			pos.UnMakeMove(move, tag, ep, hc)
			SetLastMovePlayed(savedLast)

			singularBeta := int(ttEval) - SINGULAR_MARGIN
			info.ExcludedMove = singularMove
			singularScore := alphaBetaPV(pos, depth/2, ply, singularBeta-1, singularBeta, false, info)
			info.ExcludedMove = EmptyMove

			// Re-make our move and restore the global to it for children.
			ep, tag, hc, _ = pos.MakeMove(move)
			SetLastMovePlayed(move)

			if singularScore < singularBeta {
				nextDepth++
				info.SingularExtensions++
			}
		}

		if nextDepth > depth+1 {
			// Guardrail: limit extensions to at most +1 beyond current depth
			nextDepth = depth + 1
		}

		var score int

		// Enhanced Principal Variation Search (PVS) with PV-node optimization
		if legalTried == 1 {
			// First move: search with full window and PV flag
			score = -alphaBetaPV(pos, nextDepth, ply+1, -beta, -alpha, isPV, info)
		} else {
			// Non-first moves: enhanced PVS with PV-aware reductions

			// PV-aware Late Move Reductions (use legalTried, not moveCount which includes illegal moves)
			reduction := 0
			if depth >= LMR_FULL_DEPTH && legalTried > LMR_MOVE_THRESHOLD &&
				!move.IsCapture() && !inCheck && !givesCheck {

				// Base reduction - less aggressive for PV nodes
				reduction = 1
				if !isPV {
					reduction++ // More aggressive reduction for non-PV nodes
				}

				// Extra reduction for later moves
				if legalTried > 6 {
					reduction++
				}
				if legalTried > 12 {
					reduction++
				}

				// Extra reduction for deeper searches in non-PV nodes
				if depth > 6 && !isPV {
					reduction++
				}

				// History-based LMR adjustment:
				// Reduce more for moves with bad history, less for moves with good history
				histScore := GetHistoryScore(move)
				if histScore < -500 {
					reduction++ // Bad history = reduce more
				} else if histScore > 1000 {
					reduction-- // Good history = reduce less
					if reduction < 0 {
						reduction = 0
					}
				}

				// Killer move reduction adjustment - don't reduce killer moves as much
				if IsKillerMove(move, depth) {
					reduction-- // Reduce less for killer moves
					if reduction < 0 {
						reduction = 0
					}
				}

				// Cap maximum reduction
				if reduction >= depth-1 {
					reduction = depth - 2
				}
				if reduction < 0 {
					reduction = 0
				}

				// Track LMR usage
				if reduction > 0 {
					info.LMRReductions++
				}
			}

			// Compute reduced depth (check extension already applied via nextDepth)
			reducedDepth := depth - 1 - reduction
			if reducedDepth >= depth {
				reducedDepth = depth - 1 - reduction
			}

			// Search with null window first (not PV)
			score = -alphaBetaPV(pos, reducedDepth, ply+1, -alpha-1, -alpha, false, info)

			// PVS re-search logic
			if score > alpha && score < beta {
				if reduction > 0 {
					// Re-search with full depth but still null window
					score = -alphaBetaPV(pos, nextDepth, ply+1, -alpha-1, -alpha, false, info)
				}
				// If still fails high, do full window search (PV if we're in PV)
				if score > alpha {
					score = -alphaBetaPV(pos, nextDepth, ply+1, -beta, -alpha, isPV, info)
				}
			}
		}

		pos.UnMakeMove(move, tag, ep, hc)
		// Restore predecessor for the next sibling (and for the caller if we
		// return via a beta cutoff or loop exit below).
		SetLastMovePlayed(savedLast)

		info.Nodes++

		if score > bestScore {
			bestScore = score
			bestMove = move
		}

		if score >= beta {
			info.BetaCutoffs++
			// Update history table for quiet moves that cause beta cutoffs
			UpdateHistoryTable(move, depth)
			// Update killer moves for quiet moves that cause beta cutoffs
			UpdateKillerMoves(move, depth)
			// Counter move: the cutoff move is a good reply to savedLast,
			// the move that actually brought us into this node.
			UpdateCounterMove(move, savedLast)

			// Penalize all quiet moves that were tried but didn't cause the cutoff
			for i := 0; i < searchedQuietCount; i++ {
				quietMove := searchedQuiets[i]
				if quietMove != move {
					PenalizeHistoryTable(quietMove, depth)
				}
			}

			// Store in transposition table as Lower Bound (Cut-Node)
			TranspositionTable.Set(hash, move, scoreToTT(score, ply), int8(depth), LowerBound)
			return score
		}

		if score > alpha {
			alpha = score
		}
	}

	// Check for checkmate/stalemate: no legal moves were found
	if legalTried == 0 {
		if inCheck {
			// Checkmate - return mate score adjusted for distance from root
			mateDistance := info.Depth - depth
			if mateDistance < 0 {
				mateDistance = 0
			}
			if mateDistance > 1000 {
				mateDistance = 1000
			}
			return -MATE_VALUE + mateDistance
		}
		// Stalemate is a draw
		return 0
	}

	// Store in transposition table
	var nodeType NodeType
	if bestScore <= origAlpha {
		// No move improved alpha - Upper Bound (All-Node)
		nodeType = UpperBound
	} else {
		// Best move improved alpha - Exact (PV-Node)
		nodeType = Exact
	}

	TranspositionTable.Set(hash, bestMove, scoreToTT(bestScore, ply), int8(depth), nodeType)
	return bestScore
}

func quiescence(pos *Position, alpha int, beta int, ply int, info *SearchInfo) int {
	return quiescenceWithDepth(pos, alpha, beta, ply, info, 0)
}

func quiescenceWithDepth(pos *Position, alpha int, beta int, ply int, info *SearchInfo, qDepth int) int {
	// Prevent infinite recursion - reduce depth limit for stability
	if qDepth >= 6 { // More conservative quiescence depth
		return EvaluateForPlayer(&pos.Board, pos.Turn())
	}

	// External stop request - bail out without searching deeper.
	if IsStopRequested() {
		info.Stopped = true
		return EvaluateForPlayer(&pos.Board, pos.Turn())
	}

	// TT cutoff probe. Qsearch is depth 0; any cached entry at depth >= 0
	// covers us (main search stores at depth >= 1 only). Mate scores need
	// converting from node-relative TT encoding back to root-relative.
	{
		_, ttEvalRaw, _, ttNodeType, ttHit := TranspositionTable.Get(pos.Hash())
		if ttHit {
			ttEval := scoreFromTT(ttEvalRaw, ply)
			switch ttNodeType {
			case Exact:
				return ttEval
			case LowerBound:
				if ttEval >= beta {
					return ttEval
				}
			case UpperBound:
				if ttEval <= alpha {
					return ttEval
				}
			}
		}
	}

	info.Nodes++
	info.QNodes++

	inCheck := isInCheck(pos, pos.Turn())

	// In-check branch: must search all evasions, stand-pat is illegal here.
	if inCheck {
		var moveBuffer [256]Move
		moveCount := GenerateMovesIntoBuffer(pos, moveBuffer[:])

		bestScore := -INFINITY
		legalMoves := 0

		for i := 0; i < moveCount; i++ {
			move := moveBuffer[i]
			ep, tag, hc, _ := pos.MakeMove(move)
			if isInCheck(pos, move.MovingPiece().Color()) {
				pos.UnMakeMove(move, tag, ep, hc)
				continue
			}
			legalMoves++

			score := -quiescenceWithDepth(pos, -beta, -alpha, ply+1, info, qDepth+1)
			pos.UnMakeMove(move, tag, ep, hc)

			if score > bestScore {
				bestScore = score
			}
			if score >= beta {
				return beta
			}
			if score > alpha {
				alpha = score
			}
		}

		if legalMoves == 0 {
			// Checkmate
			mateDistance := info.Depth + qDepth
			if mateDistance > 1000 || mateDistance < 0 {
				mateDistance = 1
			}
			return -MATE_VALUE + mateDistance
		}

		return bestScore
	}

	// Not in check. Detect stalemate at qDepth==0 (rare; preserves prior behavior).
	if qDepth == 0 {
		var moveBuffer [256]Move
		moveCount := GenerateMovesIntoBuffer(pos, moveBuffer[:])
		hasLegalMove := false
		for i := 0; i < moveCount && !hasLegalMove; i++ {
			move := moveBuffer[i]
			ep, tag, hc, _ := pos.MakeMove(move)
			if !isInCheck(pos, move.MovingPiece().Color()) {
				hasLegalMove = true
			}
			pos.UnMakeMove(move, tag, ep, hc)
		}
		if !hasLegalMove {
			// Stalemate is a draw regardless of material balance.
			return 0
		}
	}

	// Stand-pat
	standPat := EvaluateForPlayer(&pos.Board, pos.Turn())

	if standPat >= beta {
		return beta
	}

	bestScore := standPat
	if standPat > alpha {
		alpha = standPat
	}

	// Captures only. Quiet checks are dropped because moveGivesCheck misses
	// discovered, en-passant, and promotion checks; an incomplete quiet-check
	// path is worse than captures-only at the horizon.
	captureCount := GenerateCapturesIntoBuffer(pos, info.CaptureBuffer[:])
	var captures [64]Move
	copy(captures[:], (*info.CaptureBuffer)[:captureCount])

	for i := 0; i < captureCount; i++ {
		move := captures[i]

		// Delta pruning: skip captures that cannot possibly raise alpha
		gain := int(move.CapturedPiece().seeWeight())
		if standPat+gain+DELTA_MARGIN <= alpha {
			continue
		}

		// SEE pruning: skip losing captures
		if staticExchangeEvaluation(pos, move) < 0 {
			continue
		}

		ep, tag, hc, _ := pos.MakeMove(move)
		if isInCheck(pos, move.MovingPiece().Color()) {
			pos.UnMakeMove(move, tag, ep, hc)
			continue
		}

		score := -quiescenceWithDepth(pos, -beta, -alpha, ply+1, info, qDepth+1)
		pos.UnMakeMove(move, tag, ep, hc)

		if score > bestScore {
			bestScore = score
		}
		if score >= beta {
			return beta
		}
		if score > alpha {
			alpha = score
		}
	}

	return bestScore
}

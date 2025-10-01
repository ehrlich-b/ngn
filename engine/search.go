package engine

import (
	"math"
	"math/bits"
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
)

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
func seeGetBishopAttacks(square Square, occupied uint64) uint64 {
	attacks := uint64(0)

	for _, dir := range bishopDirections {
		for i := 1; i < 8; i++ {
			to := int(square) + dir*i
			if to < 0 || to > 63 {
				break
			}

			toFile := to % 8
			fromFile := int(square) % 8
			if abs(toFile-fromFile) != i {
				break
			}

			attacks |= SquareMask[to]
			if (occupied & SquareMask[to]) != 0 {
				break
			}
		}
	}

	return attacks
}

// seeGetRookAttacks returns squares attacked by rooks/queens on ranks/files
func seeGetRookAttacks(square Square, occupied uint64) uint64 {
	attacks := uint64(0)

	for _, dir := range rookDirections {
		for i := 1; i < 8; i++ {
			to := int(square) + dir*i
			if to < 0 || to > 63 {
				break
			}

			if dir == -1 || dir == 1 {
				fromFile := int(square) % 8
				toFile := to % 8
				if abs(toFile-fromFile) != i {
					break
				}
			}
			if dir == 8 || dir == -8 {
				if to%8 != int(square)%8 {
					break
				}
			}

			attacks |= SquareMask[to]
			if (occupied & SquareMask[to]) != 0 {
				break
			}
		}
	}

	return attacks
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
		// Pawn attacks are handled specially in getAttackers
		return false // We don't use this for pawns in SEE
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
	FutilityPrunes  uint64 // Moves pruned by futility pruning
	NullMoveCutoffs uint64 // Beta cutoffs from null move pruning
	LMRReductions   uint64 // Moves reduced by Late Move Reduction
	TTHits          uint64 // Transposition table hits
	TTCutoffs       uint64 // Beta cutoffs from TT
	BetaCutoffs     uint64 // Total beta cutoffs
	QNodes          uint64 // Quiescence search nodes
	CheckExtensions uint64 // Check extensions applied
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

				ep, tag, hc, _ := pos.MakeMove(move)
				// Check if move is legal (doesn't leave king in check)
				if isInCheck(pos, move.MovingPiece().Color()) {
					pos.UnMakeMove(move, tag, ep, hc)
					continue // Skip illegal move
				}

				score := -alphaBetaPV(pos, currentDepth-1, -beta, -alpha, true, info)
				pos.UnMakeMove(move, tag, ep, hc)

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

			// Simple window widening to minimize re-searches
			if iterationBestScore <= alpha && currentDepth > 3 {
				// Fail low - immediately use full window to avoid multiple re-searches
				alpha = -INFINITY
			} else if iterationBestScore >= beta && currentDepth > 3 {
				// Fail high - immediately use full window to avoid multiple re-searches
				beta = INFINITY
			} else {
				// Search completed successfully within the window
				info.BestScore = iterationBestScore
				info.BestMove = iterationBestMove
				previousScore = iterationBestScore
				iterationComplete = true
			}
		}

		// Store the current depth result - this gives us a valid answer
		// even if the search is interrupted at higher depths
		info.Depth = currentDepth

		// Call the callback if provided (for UCI info reporting)
		if callback != nil && info.BestMove != EmptyMove {
			callback(currentDepth, info.BestScore, info.BestMove, info.Nodes)
		}
	}

	return info
}

func SearchFixed(pos *Position, depth int, timeManager *TimeManager) *SearchInfo {
	return searchFixedUnsafe(pos, depth, timeManager)
}

func searchFixedUnsafe(pos *Position, depth int, timeManager *TimeManager) *SearchInfo {

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

		ep, tag, hc, _ := pos.MakeMove(move)
		// Check if move is legal (doesn't leave our king in check)
		movingColor := move.MovingPiece().Color()
		if isInCheck(pos, movingColor) {
			pos.UnMakeMove(move, tag, ep, hc)
			continue // Skip illegal move
		}

		score := -alphaBetaPV(pos, depth-1, -beta, -alpha, true, info)

		// Mate threat extension: if we found a very high score, search deeper to find mate
		if score >= MATE_IN_MAX && depth <= 2 {
			// This looks like a mate, search deeper to confirm
			score = -alphaBetaPV(pos, depth+2, -beta, -alpha, true, info)
			info.Nodes++
		}

		pos.UnMakeMove(move, tag, ep, hc)
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
	return alphaBetaPV(pos, depth, alpha, beta, true, info) // Root is always PV
}

func alphaBetaPV(pos *Position, depth int, alpha int, beta int, isPV bool, info *SearchInfo) int {
	// Safety checks - reasonable limits
	if info.Nodes > 50000000 { // 50M nodes
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

	if depth == 0 {
		return quiescence(pos, alpha, beta, info)
	}

	// Probe transposition table
	origAlpha := alpha
	hash := pos.Hash()

	// NOTE: Threefold repetition checking removed from hot path for performance
	// This should only be checked at root level, not during hypothetical search nodes
	ttMove, ttEval, ttDepth, ttNodeType, ttHit := TranspositionTable.Get(hash)

	if ttHit && ttDepth >= int8(depth) {
		info.TTHits++
		switch ttNodeType {
		case Exact:
			return int(ttEval)
		case LowerBound:
			if int(ttEval) >= beta {
				info.TTCutoffs++
				return int(ttEval)
			}
		case UpperBound:
			if int(ttEval) <= alpha {
				return int(ttEval)
			}
		}
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

	// PV-aware null move pruning - more conservative in PV nodes
	if !inCheck && depth >= NULL_MOVE_MIN_DEPTH && beta-alpha > 1 && (!isPV || depth >= 6) {
		// Make the null move
		oldEP := pos.MakeNullMove()

		// Search with reduced depth
		score := -alphaBetaPV(pos, depth-NULL_MOVE_R-1, -beta, -beta+1, false, info)

		// Unmake the null move
		pos.UnMakeNullMove(oldEP)

		// If null move search caused a beta cutoff, we can prune
		if score >= beta {
			info.NullMoveCutoffs++
			return beta
		}
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

	for oi := 0; oi < orderedCount; oi++ {
		move := ordered[oi]
		moveCount++

		// Check time before evaluating each move (now cheap to call)
		if info.TimeManager != nil && info.TimeManager.ShouldStopSearch(depth, time.Now()) {
			info.Stopped = true
			break
		}

		// Futility pruning - skip quiet moves if position is hopeless
		if futilityPrune && !move.IsCapture() {
			info.FutilityPrunes++
			continue
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

		ep, tag, hc, _ := pos.MakeMove(move)
		// Check if move is legal (doesn't leave our king in check)
		movingColor := move.MovingPiece().Color()
		if isInCheck(pos, movingColor) {
			pos.UnMakeMove(move, tag, ep, hc)
			continue
		}
		legalTried++

		// Refined Check Extension logic - balance tactical strength with performance
		givesCheck := isInCheck(pos, pos.Turn())

		// Compute next depth with check extension
		nextDepth := depth - 1
		if givesCheck && depth > 1 {
			nextDepth++ // Extend search when move gives check
			info.CheckExtensions++
		}
		if nextDepth >= depth {
			// Guardrail: never allow recursion with the same depth
			nextDepth = depth - 1
		}

		var score int

		// Enhanced Principal Variation Search (PVS) with PV-node optimization
		if moveCount == 1 {
			// First move: search with full window and PV flag
			score = -alphaBetaPV(pos, nextDepth, -beta, -alpha, isPV, info)
		} else {
			// Non-first moves: enhanced PVS with PV-aware reductions

			// PV-aware Late Move Reductions
			reduction := 0
			if depth >= LMR_FULL_DEPTH && moveCount > LMR_MOVE_THRESHOLD &&
				!move.IsCapture() && !inCheck && !givesCheck {

				// Base reduction - less aggressive for PV nodes
				reduction = 1
				if !isPV {
					reduction++ // More aggressive reduction for non-PV nodes
				}

				// Extra reduction for later moves
				if moveCount > 6 {
					reduction++
				}
				if moveCount > 12 {
					reduction++
				}

				// Extra reduction for deeper searches in non-PV nodes
				if depth > 6 && !isPV {
					reduction++
				}

				// Cap maximum reduction
				if reduction >= depth-1 {
					reduction = depth - 2
				}
				if reduction < 1 {
					reduction = 1
				}

				// Track LMR usage
				if reduction > 0 {
					info.LMRReductions++
				}
			}

			// Compute reduced depth with check extension
			reducedDepth := depth - 1 - reduction
			if givesCheck && reducedDepth+1 < depth {
				reducedDepth++
			}
			if reducedDepth >= depth {
				reducedDepth = depth - 1 - reduction
			}

			// Search with null window first (not PV)
			score = -alphaBetaPV(pos, reducedDepth, -alpha-1, -alpha, false, info)

			// PVS re-search logic
			if score > alpha && score < beta {
				if reduction > 0 {
					// Re-search with full depth but still null window
					score = -alphaBetaPV(pos, nextDepth, -alpha-1, -alpha, false, info)
				}
				// If still fails high, do full window search (PV if we're in PV)
				if score > alpha {
					score = -alphaBetaPV(pos, nextDepth, -beta, -alpha, isPV, info)
				}
			}
		}

		pos.UnMakeMove(move, tag, ep, hc)

		info.Nodes++

		if score > bestScore {
			bestScore = score
			bestMove = move
		}

		// No legal moves from this node
		if legalTried == 0 {
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

		if score >= beta {
			info.BetaCutoffs++
			// Update history table for quiet moves that cause beta cutoffs
			UpdateHistoryTable(move, depth)
			// Update killer moves for quiet moves that cause beta cutoffs
			UpdateKillerMoves(move, depth)
			// Update counter move for quiet moves that cause beta cutoffs
			UpdateCounterMove(move, GetLastMovePlayed())

			// Store in transposition table as Lower Bound (Cut-Node)
			TranspositionTable.Set(hash, move, int16(score), int8(depth), LowerBound)
			return score
		}

		if score > alpha {
			alpha = score
		}
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

	TranspositionTable.Set(hash, bestMove, int16(bestScore), int8(depth), nodeType)
	return bestScore
}

func quiescence(pos *Position, alpha int, beta int, info *SearchInfo) int {
	return quiescenceWithDepth(pos, alpha, beta, info, 0)
}

func quiescenceWithDepth(pos *Position, alpha int, beta int, info *SearchInfo, qDepth int) int {
	// Prevent infinite recursion - reduce depth limit for stability
	if qDepth >= 6 { // More conservative quiescence depth
		return EvaluateForPlayer(&pos.Board, pos.Turn())
	}

	// MATE DETECTION: Check for checkmate/stalemate at quiescence entry
	// This is critical for detecting mate-in-1 positions
	if qDepth == 0 {
		// Generate all pseudo-legal moves
		var moveBuffer [256]Move
		moveCount := GenerateMovesIntoBuffer(pos, moveBuffer[:])

		// Check if any move is actually legal by trying them
		hasLegalMove := false
		for i := 0; i < moveCount && !hasLegalMove; i++ {
			move := moveBuffer[i]
			oldEP, oldTag, oldHalfClock, _ := pos.MakeMove(move)

			// Check if this move leaves our king in check (illegal)
			ourKingInCheck := isInCheck(pos, pos.Turn().Other())

			pos.UnMakeMove(move, oldTag, oldEP, oldHalfClock)

			if !ourKingInCheck {
				hasLegalMove = true
			}
		}

		if !hasLegalMove {
			inCheck := pos.IsInCheck()
			if inCheck {
				// Checkmate - we're mated, clamp mate distance to prevent overflow
				mateDistance := info.Depth + qDepth
				if mateDistance > 1000 || mateDistance < 0 {
					mateDistance = 1 // Safe fallback for mate-in-1
				}
				return -MATE_VALUE + mateDistance
			}
			// Stalemate - check if we're throwing away a winning position
			// If we have a material advantage, stalemating the opponent is bad for us
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
	}

	// Stand pat - evaluate current position
	standPat := EvaluateForPlayer(&pos.Board, pos.Turn())
	info.Nodes++
	info.QNodes++

	if standPat >= beta {
		return beta
	}

	// BUG FIX: We must track the best score, not just update alpha
	bestScore := standPat

	if standPat > alpha {
		alpha = standPat
	}

	// Only consider captures and checks in quiescence search
	// Use buffers from SearchInfo to avoid allocations
	captureCount := GenerateCapturesIntoBuffer(pos, info.CaptureBuffer[:])
	var capturesAndChecks [84]Move // Small stack buffer for combined captures + checks
	capturesAndChecksCount := captureCount
	copy(capturesAndChecks[:], (*info.CaptureBuffer)[:captureCount])

	// Also consider moves that give check (but only if not too deep to avoid explosion)
	// Allow deeper check search when in mate-threatening positions
	maxCheckDepth := 2
	if standPat >= MATE_IN_MAX || standPat <= -MATE_IN_MAX {
		maxCheckDepth = 6 // Search deeper when mate is on the board
	}
	if qDepth < maxCheckDepth {
		// PERFORMANCE OPTIMIZATION: Use direct check detection instead of make/unmake cycle
		// Use MoveBuffer from SearchInfo to avoid allocation
		numMoves := GenerateMovesIntoBuffer(pos, info.MoveBuffer[:])
		moves := (*info.MoveBuffer)[:numMoves]

		for _, move := range moves {
			// Skip if already a capture
			if move.IsCapture() {
				continue
			}

			// OPTIMIZED: Check if move gives check without make/unmake
			if moveGivesCheck(pos, move) && capturesAndChecksCount < len(capturesAndChecks) {
				capturesAndChecks[capturesAndChecksCount] = move
				capturesAndChecksCount++
			}
		}
	}

	// Search captures and checks (with delta pruning for captures)
	for i := 0; i < capturesAndChecksCount; i++ {
		move := capturesAndChecks[i]

		// Delta pruning: skip captures that cannot possibly raise alpha
		if move.IsCapture() {
			gain := int(move.CapturedPiece().seeWeight())
			if standPat+gain+DELTA_MARGIN <= alpha {
				continue
			}
		}

		// SEE-based pruning: skip losing captures
		if move.IsCapture() {
			seeValue := staticExchangeEvaluation(pos, move)
			if seeValue < 0 {
				continue // Skip losing captures
			}
		}

		ep, tag, hc, _ := pos.MakeMove(move)

		// Check if move is legal (doesn't leave our king in check)
		// After making a move, we need to check if OUR king (opposite of current turn) is in check
		if isInCheck(pos, move.MovingPiece().Color()) {
			pos.UnMakeMove(move, tag, ep, hc)
			continue // Skip illegal move
		}

		// Recursively search (negamax)
		score := -quiescenceWithDepth(pos, -beta, -alpha, info, qDepth+1)
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

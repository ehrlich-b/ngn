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
	EXTENSION_BUDGET        = 24  // Max net extensions on a single root-to-leaf path (anti-explosion)
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

// globalMaxNodes caps total nodes for the current search (0 = unlimited). Set from
// UCI `go nodes N` and observed in the search hot paths, which trip the stop flag
// when the budget is spent. This makes self-play deterministic at an equal node
// budget — fast (small cap), cool (no timing, so concurrency is free), and able to
// credit ordering/efficiency changes that fixed-depth and movetime both miss.
var globalMaxNodes atomic.Uint64

// SetMaxNodes sets the per-search node budget (0 = unlimited). Called by UCI before
// each search and overwritten every `go`, so it needs no separate clear.
func SetMaxNodes(n uint64) {
	globalMaxNodes.Store(n)
}

// MaxNodes returns the current node budget (0 = unlimited).
func MaxNodes() uint64 {
	return globalMaxNodes.Load()
}

// Late Move Pruning thresholds, indexed [improving][depth] (S4). When our static
// eval is NOT improving vs two plies ago, prune late quiets ~2x sooner; when it is
// improving, keep the original (more generous) counts. Depth 0 is unused (the LMP
// guard requires depth > 0).
var lmpThreshold = [2][8]int{
	{0, 3, 4, 6, 9, 12, 15, 19},   // not improving — prune sooner
	{0, 5, 8, 12, 17, 23, 30, 38}, // improving — original schedule
}

// SearchToggles enables/disables individual pruning & reduction techniques for
// the correctness-ablation harness (TestPruningAblation). All default ON, so
// normal play is unchanged; the test flips one OFF at a time — a technique whose
// removal makes the engine solve MORE tactics is over-pruning real refutations
// (a bug), not just trading accuracy for speed.
var SearchToggles = struct {
	NullMove  bool
	Futility  bool
	RFP       bool
	Probcut   bool
	LMP       bool
	SEEPrune  bool
	LMR       bool
	HistPrune bool
	IID       bool
	Singular  bool
}{true, true, true, true, true, true, true, true, true, true}

// lmrTable[depth][legalTried] precomputes the Stockfish-style log-based LMR
// base reduction: floor(log(depth) * log(legalTried) / 2). Smoother gradient
// than the prior piecewise tiers (legalTried thresholds at 6/12 plus a
// depth>6 bump), and reduces less at typical mid-depth/mid-moveNum nodes
// where discrete +1 jumps tended to over-bite. PV/non-PV split, history,
// killer adjustments, and the depth cap remain on top.
var lmrTable [64][64]int

func init() {
	for d := 1; d < 64; d++ {
		for m := 1; m < 64; m++ {
			lmrTable[d][m] = int(math.Log(float64(d)) * math.Log(float64(m)) / 2.0)
		}
	}
}

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
	// Castling is a king move with no exchange — SEE is meaningless and the king's
	// infinite weight would poison the swap. Quiet (non-capture) moves DO flow
	// through: gain[0] = 0 (nothing captured) and the swap then evaluates whether
	// the moving piece survives on its destination square. This is what quiet-move
	// SEE pruning reads; capture/EP/promotion paths are unchanged.
	if move.IsCastle() {
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
	occupied &^= SquareMask[int(from)] // mover left its square -> reveals x-ray attackers behind it

	// gain[0] is the value of the piece initially captured. pieceOnSquare tracks
	// the value of whatever piece now stands on the target square (initially the
	// moving piece) -- that is what the NEXT recapture wins. For a promotion-
	// capture the pawn becomes the promoted piece, both for the material gain and
	// as the piece now standing on the square.
	gain[depth] = int(capturedValue)
	pieceOnSquare := int(movingPiece.seeWeight())
	if move.PromoType() != NoType {
		promoPiece := GetPiece(move.PromoType(), movingPiece.Color())
		gain[depth] += int(promoPiece.seeWeight()) - 100 // 100 = pawn seeWeight
		pieceOnSquare = int(promoPiece.seeWeight())
	}

	for {
		depth++
		if depth >= 32 {
			break // Prevent overflow of the gain array
		}

		// Find the least valuable attacker for the current side
		attackerSquare, attackerPiece := findLeastValuableAttacker(pos, attackers, color, occupied, targetSquare)
		if attackerSquare == NoSquare {
			break // No more attackers; last valid swap entry is gain[depth-1]
		}

		// Remove this attacker from the occupied bitboard
		occupied &^= SquareMask[int(attackerSquare)]

		// Update attackers (reveal sliders that were behind the moved attacker)
		attackers = getAttackersAfterMove(pos, attackers, occupied, targetSquare, attackerSquare)

		// This recapture wins the piece currently standing on the square; the
		// recapturing piece then becomes the piece on the square for the next ply.
		gain[depth] = pieceOnSquare - gain[depth-1]
		pieceOnSquare = int(attackerPiece.seeWeight())

		// Switch sides
		color = color.Other()
	}

	// Negamax the swap list backward over the valid entries [0 .. depth-1]. Each
	// side, working back, declines a recapture that would leave it worse off.
	for depth--; depth > 0; depth-- {
		if gain[depth-1] > -gain[depth] {
			gain[depth-1] = -gain[depth]
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

	// White pawn attacks. A white pawn attacking `square` sits one rank below it:
	// at square-9 (its file one LEFT of the target, valid when target file > A) or
	// square-7 (one file RIGHT, valid when target file < H). The file guards must
	// pair with the matching offset or SEE misses edge-file recapturers and invents
	// wrap-around attackers (off-by-file bug fixed 2026-05-30).
	if square.Rank() > Rank1 {
		if square.File() > FileA {
			attackSq := int(square) - 9
			if attackSq >= 0 && attackSq < 64 {
				attackers |= (whitePawns & SquareMask[attackSq])
			}
		}
		if square.File() < FileH {
			attackSq := int(square) - 7
			if attackSq >= 0 && attackSq < 64 {
				attackers |= (whitePawns & SquareMask[attackSq])
			}
		}
	}

	// Black pawn attacks. A black pawn attacking `square` sits one rank above it:
	// at square+7 (file one LEFT, valid when target file > A) or square+9 (one file
	// RIGHT, valid when target file < H).
	if square.Rank() < Rank8 {
		if square.File() > FileA {
			attackSq := int(square) + 7
			if attackSq >= 0 && attackSq < 64 {
				attackers |= (blackPawns & SquareMask[attackSq])
			}
		}
		if square.File() < FileH {
			attackSq := int(square) + 9
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
	// Try attackers cheapest-first: Pawn, Knight, Bishop, Rook, Queen, King — exactly the
	// PieceType order (Pawn=1..King=6). Enemy pieces are the White constants plus a color
	// offset, so hoist the offset instead of a per-call slice literal + per-iteration GetPiece.
	var base Piece
	if color == Black {
		base = 6
	}
	for pt := Pawn; pt <= King; pt++ {
		piece := Piece(pt) + base
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

	// White pawn attacks (file guards paired with the matching offset — see getAttackers).
	if targetSquare.Rank() > Rank1 {
		if targetSquare.File() > FileA {
			attackSq := int(targetSquare) - 9
			if attackSq >= 0 && attackSq < 64 {
				tempAttackers |= (whitePawns & SquareMask[attackSq])
			}
		}
		if targetSquare.File() < FileH {
			attackSq := int(targetSquare) - 7
			if attackSq >= 0 && attackSq < 64 {
				tempAttackers |= (whitePawns & SquareMask[attackSq])
			}
		}
	}

	// Black pawn attacks (file guards paired with the matching offset).
	if targetSquare.Rank() < Rank8 {
		if targetSquare.File() > FileA {
			attackSq := int(targetSquare) + 7
			if attackSq >= 0 && attackSq < 64 {
				tempAttackers |= (blackPawns & SquareMask[attackSq])
			}
		}
		if targetSquare.File() < FileH {
			attackSq := int(targetSquare) + 9
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

// searchFrame holds one alphaBetaPV invocation's move buffers, reused across the search
// via a recursion-frame pool (SearchInfo.Frames) instead of being stack-allocated and
// heap-escaping per node — cuts search allocations ~75% (docs/10 T1). Indexed by a
// recursion-FRAME counter, NOT ply: IID and singular verification re-enter at the same
// ply, so a ply-indexed buffer would clobber a live parent (docs/09 J3).
type searchFrame struct {
	moveBuffer [256]Move // raw pseudo-legal generation target (transient: gen -> order)
	ordered    [256]Move // ordered moves, iterated across the recursive move loop
	scores     [256]int  // ordering scores, parallel to ordered, for incremental selection
	quiets     [64]Move  // quiet moves tried (history penalties on a cutoff)
	captures   [64]Move  // captures tried (capture-history penalties)
}

// searchFramePoolSize bounds the reusable frame pool; a recursion depth beyond it falls
// back to a stack-allocated frame (still correct, just not allocation-free). Sized well
// above MaximumDepth(100) + extensions + same-ply re-entrant (IID/singular) restacking.
const searchFramePoolSize = 256

type SearchInfo struct {
	Nodes       uint64
	Depth       int
	SelDepth    int // Maximum ply reached including quiescence
	RootDepth   int // Depth of the current ID iteration; the reference for the per-path extension budget
	BestMove    Move
	BestScore   int
	Stopped     bool
	TimeManager *TimeManager
	PV          [MaximumDepth]Move // Principal variation array
	PVLength    int                // Length of current PV

	// Static eval per ply (white-to-move perspective of the node) for the
	// improving heuristic; indexed by ply like PV.
	StaticEvalStack [MaximumDepth]int

	// Zero-allocation buffers passed through recursion
	MoveBuffer    *[256]Move // For move generation
	CaptureBuffer *[64]Move  // For capture generation
	OrderedBuffer *[256]Move // For move ordering
	SEEGains      *[32]int   // For SEE evaluation

	// Pruning and search efficiency statistics
	FutilityPrunes       uint64 // Moves pruned by futility pruning
	NullMoveCutoffs      uint64 // Beta cutoffs from null move pruning
	LMRReductions        uint64 // Moves reduced by Late Move Reduction
	NullWindowScouts     uint64 // Non-first moves searched with a null window (re-search denominator)
	LMRReSearches        uint64 // Reduced fail-highs re-searched at full depth (LMR verification churn)
	PVReSearches         uint64 // PV nodes re-searched with the full window after a null-window fail-high
	LMPPrunes            uint64 // Moves pruned by Late Move Pruning
	ProbcutPrunes        uint64 // Positions pruned by Probcut
	SEEQuietPrunes       uint64 // Quiet moves pruned by SEE (soft hangs)
	TTProbes             uint64 // Transposition table probes (denominator for hit/cutoff rate)
	TTHits               uint64 // Transposition table hits
	TTCutoffs            uint64 // Beta cutoffs from TT
	BetaCutoffs          uint64 // Total beta cutoffs
	FirstMoveCutoffs     uint64 // Beta cutoffs on the first legal move tried (move-ordering quality)
	QNodes               uint64 // Quiescence search nodes
	QSearchTTStores      uint64 // Quiescence results written to the TT (S3)
	CheckExtensions      uint64 // Check extensions applied
	SingularExtensions   uint64 // Singular extensions applied
	RecaptureExtensions  uint64 // Recapture extensions applied
	PassedPawnExtensions uint64 // Passed pawn extensions applied
	ExcludedMove         Move   // Move to exclude during singular search
	ExcludedPly          int    // ply at which ExcludedMove applies (the singular verification root)

	// Repetition detection during search
	RepStack    [512]uint64 // Stack of Zobrist hashes on current search path
	RepStackLen int         // Current depth in repetition stack

	// Reusable per-invocation move buffers (docs/10 T1). Frames is allocated once per
	// search; FrameDepth is the recursion-frame counter, pushed/popped in lockstep with
	// RepStackLen so each live alphaBetaPV invocation owns a distinct frame.
	Frames     []searchFrame
	FrameDepth int
}

func Search(pos *Position, depth int) *SearchInfo {
	return SearchIterativeDeepening(pos, depth, nil)
}

// DepthCallback is called after each depth completes during iterative deepening
type DepthCallback func(info *SearchInfo)

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
		Frames:        make([]searchFrame, searchFramePoolSize),
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

		// Reset seldepth at the top of each iteration so UCI info reports the
		// deepest reach for THIS iteration, not the max across all prior ones.
		info.SelDepth = currentDepth

		// Record this iteration's root depth as the reference for the per-path
		// extension budget (search.go ~1470): a node's net extensions along the
		// current path == depth + ply - RootDepth, which we cap to keep a
		// non-decaying checking line from recursing toward MaximumDepth.
		info.RootDepth = currentDepth

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
			// S6: search the previous iteration's best move first — a good alpha early
			// tightens pruning and gives time management a best-move stability signal.
			// info.BestMove holds the last COMPLETED depth's best move.
			if info.BestMove != EmptyMove {
				for i := 1; i < orderedCount; i++ {
					if orderedRoot[i] == info.BestMove {
						orderedRoot[0], orderedRoot[i] = orderedRoot[i], orderedRoot[0]
						break
					}
				}
			}
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
					score = -alphaBetaPV(pos, currentDepth-1, 1, -beta, -alpha, true, true, info)
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

			// If the clock interrupted this iteration mid-flight, its results are
			// partial: child searches were cut off and returned unreliable scores, so
			// iterationBestMove may be a half-searched (often blundering) move. Discard
			// the interrupted iteration entirely and keep the last fully-completed
			// iteration's best move. Without this guard a partial iteration can still
			// land in the in-window branch below and overwrite the good move — the root
			// cause of catastrophic time-pressure blunders.
			if info.Stopped {
				break
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
				// Search completed successfully within the window. Feed the time
				// manager the decision-stability signal: did the best move change vs
				// the last completed iteration, and did the window hold on the first
				// attempt (attempts==1 ⇒ no fail-high/fail-low re-search). info.BestMove
				// still holds the PREVIOUS completed iteration's move at this point.
				bestMoveChanged := iterationBestMove != info.BestMove
				info.BestScore = iterationBestScore
				info.BestMove = iterationBestMove
				previousScore = iterationBestScore
				iterationComplete = true
				if info.TimeManager != nil {
					info.TimeManager.ReportCompletedIteration(bestMoveChanged, attempts == 1)
				}
			}
		}

		// Only commit the new depth and emit UCI info when the iteration actually
		// completed. If we exited the inner loop on info.Stopped or exhausted
		// maxAttempts without success, keep the previous completed depth/best move.
		if iterationComplete {
			info.Depth = currentDepth
			if callback != nil && info.BestMove != EmptyMove {
				callback(info)
			}

			// Forced mate proven: stop deepening. Deeper iterations only re-derive the
			// same mate instantly via the TT, spinning uselessly to maxDepth (the
			// "depth 100 in 13ms" pathology in the logs). The mate move is already
			// chosen and mate-distance scoring already prefers the longest defense.
			if info.BestScore >= MATE_IN_MAX || info.BestScore <= -MATE_IN_MAX {
				break
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
		Frames:        make([]searchFrame, searchFramePoolSize),
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

		score := -alphaBetaPV(pos, depth-1, 1, -beta, -alpha, true, true, info)

		// Mate threat extension: if we found a very high score, search deeper to find mate
		if score >= MATE_IN_MAX && depth <= 2 {
			score = -alphaBetaPV(pos, depth+2, 1, -beta, -alpha, true, true, info)
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
	return alphaBetaPV(pos, depth, 0, alpha, beta, true, true, info) // Root is always PV
}

func alphaBetaPV(pos *Position, depth int, ply int, alpha int, beta int, isPV bool, canNull bool, info *SearchInfo) int {
	if ply > info.SelDepth {
		info.SelDepth = ply
	}
	// Safety checks - reasonable limits
	if info.Nodes > 50000000 { // 50M nodes
		info.Stopped = true
		return EvaluateForPlayer(&pos.Board, pos.Turn())
	}

	// Fixed-node budget (UCI `go nodes`): trip the stop flag when the budget is
	// spent, reusing the unwind/discard machinery below.
	if m := MaxNodes(); m != 0 && info.Nodes >= m {
		RequestStop()
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

	// Safety check for excessive ply: no legal search should recurse past
	// MaximumDepth plies. A check extension restores depth (nextDepth == depth
	// when givesCheck), so a non-decaying checking line — perpetual check or a
	// king hunt — never bottoms out and recurses until a ply-indexed array
	// (e.g. PV[MaximumDepth]) overflows. The clock unwinds it in real games;
	// fixed-depth analysis has no clock, so it crashes outright.
	if ply >= MaximumDepth {
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
	// Reserve this invocation's reusable move-buffer frame (docs/10 T1). frameIdx is the
	// recursion-FRAME index (not ply): IID/singular re-enter at the same ply, so each gets
	// a distinct frame here, never clobbering a live parent's buffers (docs/09 J3). Balanced
	// by the same defer as the repetition stack — no extra per-node defer.
	frameIdx := info.FrameDepth
	info.FrameDepth++
	// Pop repetition stack and release the buffer frame on function exit.
	defer func() { info.RepStackLen--; info.FrameDepth-- }()

	if depth == 0 {
		return quiescence(pos, alpha, beta, ply, info)
	}

	// Probe transposition table
	origAlpha := alpha

	ttMove, ttEvalRaw, ttDepth, ttNodeType, ttHit := TranspositionTable.Get(hash)
	info.TTProbes++
	ttEval := scoreFromTT(ttEvalRaw, ply)

	// Singular verification root: this node is being re-searched with one move
	// excluded, so the cached entry (which reflects the full search INCLUDING the
	// excluded move) must not short-circuit it — otherwise the verification just
	// returns the original score and never proves singularity. Probe still runs
	// for move ordering; only the cutoff and the (corrupting) store are skipped.
	inSingular := info.ExcludedMove != EmptyMove && ply == info.ExcludedPly

	if ttHit && !isPV && ttDepth >= int8(depth) && !inSingular {
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
	if SearchToggles.IID && ttMove == EmptyMove && depth >= 4 && isPV {
		// Search at reduced depth to get a good move for ordering
		iidDepth := depth - 2
		if iidDepth < 1 {
			iidDepth = 1
		}
		// This re-enters alphaBetaPV on the SAME position (no move made). The
		// current node already pushed its hash onto info.RepStack above, so the
		// re-entry would otherwise hit its own hash in the search-path repetition
		// scan and return a bogus draw (0). Hide our entry for the duration.
		info.RepStackLen--
		alphaBetaPV(pos, iidDepth, ply, alpha, beta, true, true, info)
		info.RepStackLen++
		// Re-probe TT to get the move we just found
		ttMove, _, _, _, _ = TranspositionTable.Get(hash)
	}

	// Mate distance pruning - don't look for mates that are too far away.
	// Use ply (distance from root) directly. Earlier code computed
	// `info.Depth - depth`, but info.Depth is only set after iteration
	// completion (line 618), so during the tree walk it was either maxDepth
	// (first iteration) or the prior completed depth — i.e. wrong.
	mateAlpha := -MATE_VALUE + ply
	mateBeta := MATE_VALUE - ply - 1

	if alpha < mateAlpha {
		alpha = mateAlpha
	}
	if beta > mateBeta {
		beta = mateBeta
	}
	if alpha >= beta {
		return alpha
	}

	// Check if we're in check before attempting null move. The InCheck tag is
	// maintained for the side to move on every entry path (ParseFEN at the root,
	// makeMoveHelper after a move, preserved across IID/singular re-entry, and
	// clear after a null move since NMP only fires when !inCheck), so read it
	// instead of rescanning (N2, node-identical).
	inCheck := pos.IsInCheck()

	// Static eval for this node, computed once and reused by RFP/futility and the
	// improving heuristic. Undefined in check, so inherit the value from two plies
	// ago to keep the improving chain sane across checking plies.
	stm := pos.Turn()
	var staticEval int
	var corrIdx int // pawn-correction slot; valid only when !inCheck
	if inCheck {
		if ply >= 2 {
			staticEval = info.StaticEvalStack[ply-2]
		} else {
			staticEval = -INFINITY
		}
	} else {
		corrIdx = pawnCorrectionIndex(pos.Board.pieces[WhitePawn], pos.Board.pieces[BlackPawn])
		staticEval = EvaluateForPlayer(&pos.Board, stm) + correctionValue(stm, corrIdx)
		// S7: a search-refined TT score is a sharper static estimate than a fresh
		// eval for the margin heuristics below (RFP/futility/NMP/improving). Use it
		// when the stored bound brackets it. Skip mate scores (would distort margins)
		// and the singular-verification root (keep its pruning behaviour unchanged).
		if ttHit && !inSingular && abs(ttEval) < MATE_IN_MAX {
			if ttNodeType == Exact ||
				(ttNodeType == LowerBound && ttEval > staticEval) ||
				(ttNodeType == UpperBound && ttEval < staticEval) {
				staticEval = ttEval
			}
		}
	}
	info.StaticEvalStack[ply] = staticEval

	// improving: static eval better than two plies ago (our previous turn). When
	// the position is trending up we reduce one less ply in LMR below.
	improving := !inCheck && ply >= 2 && staticEval > info.StaticEvalStack[ply-2]

	// Extended futility pruning (depths 1-3) and reverse futility pruning
	var futilityPrune bool

	// Reverse futility pruning - if position is very good, return early
	if SearchToggles.RFP && depth <= 3 && !inCheck && beta-alpha == 1 {
		margin := REVERSE_FUTILITY_MARGIN * depth
		if staticEval-margin >= beta {
			return staticEval - margin
		}
	}

	// PV-aware extended futility pruning for shallow depths
	if SearchToggles.Futility && depth <= FUTILITY_MAX_DEPTH && !inCheck && !isPV { // Don't prune in PV nodes
		margin := FUTILITY_MARGIN + (depth-1)*50 // Increasing margin with depth
		if staticEval+margin <= alpha {
			futilityPrune = true
		}
	}

	// Note: Razoring pruning removed - futility pruning already provides effective pruning
	// with better performance characteristics for this engine architecture

	// Null-move pruning at cut nodes only. C3 fix: the prior gate required a full
	// window (beta-alpha>1), but PVS scouts every non-PV node with a null window, so
	// NMP fired only on the PV spine — almost never. Gate on !isPV (every cut node) +
	// staticEval>=beta (standard soundness/efficiency guard) + canNull (no two null
	// moves in a row, replacing the implicit beta-alpha>1 brake). Zugzwang guard:
	// in pawn-only endgames passing can be best, so require non-pawn material.
	if SearchToggles.NullMove && !inCheck && depth >= NULL_MOVE_MIN_DEPTH && canNull && !isPV && staticEval >= beta &&
		hasNonPawnMaterial(&pos.Board, pos.Turn()) {
		// Set lastMovePlayed to EmptyMove so children don't pick spurious
		// counter-move replies keyed on our own prior move.
		savedLast := GetLastMovePlayed()
		oldEP := pos.MakeNullMove()
		SetLastMovePlayed(EmptyMove)

		// Floor the null child depth at 0. At depth==NULL_MOVE_MIN_DEPTH (3) the
		// reduction depth-NULL_MOVE_R-1 = -1 trips the negative-depth guard and
		// verifies the null move with a raw static eval (tactically blind) instead
		// of quiescence; flooring routes it through qsearch like the depth>=4 path.
		nullDepth := depth - NULL_MOVE_R - 1
		if nullDepth < 0 {
			nullDepth = 0
		}
		score := -alphaBetaPV(pos, nullDepth, ply+1, -beta, -beta+1, false, false, info)

		pos.UnMakeNullMove(oldEP)
		SetLastMovePlayed(savedLast)

		if score >= beta {
			info.NullMoveCutoffs++
			return beta
		}
	}

	// Acquire this invocation's reusable buffers from the frame pool (used by probcut
	// below AND the main move loop); fall back to a fresh stack frame only when the pool
	// is exhausted by deep re-entrant nesting (correct, just not allocation-free).
	// Kept ABOVE probcut so probcut's generation is allocation-free too. docs/10 T1.
	var frame *searchFrame
	if info.Frames != nil && frameIdx < len(info.Frames) {
		frame = &info.Frames[frameIdx]
	} else {
		frame = new(searchFrame)
	}

	// Probcut pruning: if a reduced-depth search at a higher beta fails high,
	// we can prune because the full search would likely also fail high
	if SearchToggles.Probcut && !isPV && !inCheck && depth >= 5 && abs(beta) < MATE_VALUE-100 {
		probcutBeta := beta + 200 // Higher beta threshold

		// Generate into the reused frame buffer (NOT a per-node [256]Move that escapes
		// to the heap via the non-inlined GenerateMovesIntoBuffer — that was 34MB/run).
		var capBuffer [64]Move
		capCount := 0
		tempCount := GenerateMovesIntoBuffer(pos, frame.moveBuffer[:])
		for i := 0; i < tempCount && capCount < 64; i++ {
			if frame.moveBuffer[i].IsCapture() {
				capBuffer[capCount] = frame.moveBuffer[i]
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
				score = -alphaBetaPV(pos, depth-4, ply+1, -probcutBeta, -probcutBeta+1, false, true, info)
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
	if SearchToggles.Singular && ttHit && ttMove != EmptyMove && depth >= SINGULAR_DEPTH &&
		info.ExcludedMove == EmptyMove && // Not already in singular search
		(ttNodeType == Exact || ttNodeType == LowerBound) &&
		ttDepth >= int8(depth-3) {
		singularMove = ttMove
	}

	// Use pseudo-legal moves for performance, check legality only when trying moves
	// (frame acquired above, before probcut, so all generation here is allocation-free).
	numMoves := GenerateMovesIntoBuffer(pos, frame.moveBuffer[:])
	moves := frame.moveBuffer[:numMoves]
	if numMoves == 0 {
		if inCheck {
			// Checkmate - distance from root is `ply` directly.
			mateDistance := ply
			if mateDistance > 1000 {
				mateDistance = 1000
			}
			return -MATE_VALUE + mateDistance
		}
		// Stalemate is a draw regardless of material balance. Scoring it -500
		// when "winning" breaks the zero-sum invariant — the opponent would read
		// the same drawn position as +500 and steer into it.
		return 0
	}

	// Order moves, prioritizing TT move if available. Scores are written eagerly; the sort
	// itself is driven incrementally inside the loop (selectNextMove) so the selection tail
	// past a beta cutoff is never paid. Node-identical to a full sort (docs/10 T2a).
	ordered := frame.ordered[:]
	scores := frame.scores[:]
	orderedCount := scoreMovesIntoBuffer(moves, depth, GetLastMovePlayed(), ttMove, ttHit, ordered, scores, pos)
	bestScore := -INFINITY
	bestMove := EmptyMove
	moveCount := 0
	legalTried := 0

	// Track quiet/capture moves searched for history penalization (reused frame buffers)
	searchedQuiets := frame.quiets[:]
	searchedQuietCount := 0
	searchedCaptures := frame.captures[:]
	searchedCaptureCount := 0

	for oi := 0; oi < orderedCount; oi++ {
		// Deferred selection sort: pull the next-best move into ordered[oi]. Doing this
		// per-iteration (rather than fully up front) skips the sort tail after a cutoff.
		selectNextMove(ordered, scores, orderedCount, oi)
		move := ordered[oi]
		moveCount++

		// Skip excluded move at the singular verification root only (ply-scoped so
		// a coincidentally-identical move deeper in the verification subtree isn't
		// also skipped).
		if inSingular && move == info.ExcludedMove {
			continue
		}

		// Check time before evaluating each move (now cheap to call)
		if info.TimeManager != nil && info.TimeManager.ShouldStopSearch(depth, time.Now()) {
			info.Stopped = true
			break
		}

		// Futility pruning - skip quiet moves if position is hopeless (but never prune
		// the TT move or a queen promotion — a queen promo gains ~800cp, far beyond any
		// futility margin, so the "this quiet move can't reach alpha" premise is false).
		if futilityPrune && !move.IsCapture() && move != ttMove && move.PromoType() != Queen {
			info.FutilityPrunes++
			continue
		}

		// Late Move Pruning (LMP) - skip very late quiet moves at low depths
		// Only apply in non-PV nodes, not in check, and for quiet moves
		// Use legalTried (not moveCount) since moveCount includes illegal moves
		if SearchToggles.LMP && !isPV && !inCheck && depth > 0 && depth < 8 &&
			!move.IsCapture() && move != ttMove && legalTried > 0 {
			impRow := 0
			if improving {
				impRow = 1
			}
			threshold := lmpThreshold[impRow][depth]
			if legalTried > threshold {
				info.LMPPrunes++
				continue
			}
		}

		// SEE pruning for bad captures in main search
		// Skip obviously losing captures at low depths (not in PV, not TT move)
		if SearchToggles.SEEPrune && !isPV && !inCheck && depth <= 4 && move.IsCapture() && move != ttMove {
			seeValue := staticExchangeEvaluation(pos, move)
			if seeValue < -100 { // Skip captures losing more than a pawn
				continue
			}
		}

		// SEE pruning of QUIET moves: at shallow depth, skip a quiet move whose
		// piece can be captured on its destination for a net material loss (a
		// "soft hang"). Threshold scales with depth so deeper nodes prune only
		// larger losses. Exclude the TT move, promotions (tactical), and king
		// moves — the king's infinite SEE weight poisons the swap, and a pinned
		// enemy "attacker" of a king square may not be a legal capture.
		if SearchToggles.SEEPrune && !isPV && !inCheck && depth <= 4 &&
			!move.IsCapture() && move != ttMove && move.PromoType() == NoType {
			if mp := move.MovingPiece(); mp != WhiteKing && mp != BlackKing {
				if staticExchangeEvaluation(pos, move) < -80*depth {
					info.SEEQuietPrunes++
					continue
				}
			}
		}

		// History pruning: skip quiet moves with very bad history at low depths
		if SearchToggles.HistPrune && !isPV && !inCheck && depth <= 3 && !move.IsCapture() && move != ttMove {
			histScore := GetHistoryScore(move, GetLastMovePlayed())
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
		// Track captures for capture-history penalization
		if move.IsCapture() && searchedCaptureCount < 64 {
			searchedCaptures[searchedCaptureCount] = move
			searchedCaptureCount++
		}

		// Refined Check Extension logic - balance tactical strength with performance.
		// givesCheck == "is the side now to move in check" == exactly what makeMoveHelper
		// already computed and cached in the InCheck tag (position.go:263), so read the
		// cached tag instead of rescanning the board (N2, node-identical).
		givesCheck := pos.IsInCheck()

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

		// Passed pawn extension: extend an advanced push of an ACTUALLY PASSED pawn
		// (near promotion). Was: any non-capture push to the 6th/7th rank, which in
		// endgames extended every pawn shuffle and stormed the tree ~20x/knode vs the
		// middlegame (autopsy) — most of those pawns are not passers and not real
		// promotion threats. pos is post-MakeMove, so the pushed pawn sits on its
		// destination; it is passed iff its front span is clear of enemy pawns.
		if move.MovingPiece().Type() == Pawn && !move.IsCapture() {
			dest := int(move.Destination())
			destRank := dest / 8
			color := move.MovingPiece().Color()
			if (color == White && destRank >= 5) || (color == Black && destRank <= 2) {
				var oppPawns uint64
				if color == White {
					oppPawns = pos.Board.GetBitboardOf(BlackPawn)
				} else {
					oppPawns = pos.Board.GetBitboardOf(WhitePawn)
				}
				if passedPawnFrontSpan[color][dest]&oppPawns == 0 && nextDepth+1 <= depth {
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
			info.ExcludedPly = ply // the verification re-searches THIS node's position
			singularDepth := depth - 4
			if singularDepth < 1 {
				singularDepth = 1
			}
			// The verification search re-enters alphaBetaPV on the SAME (parent)
			// position with the singular move excluded. Hide this node's own
			// repetition-stack entry so the re-entry doesn't match its own hash
			// and bail out with a bogus draw (0) before doing any work.
			info.RepStackLen--
			singularScore := alphaBetaPV(pos, singularDepth, ply, singularBeta-1, singularBeta, false, true, info)
			info.RepStackLen++
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

		// Per-path extension budget (anti-explosion). The per-node guardrail above
		// caps a single node at +1, but a check extension only RESTORES the decrement
		// (nextDepth == depth), so a non-decaying checking line keeps depth flat and
		// can recurse toward MaximumDepth (the seldepth-111, ~20x-node explosion).
		// Bound the CUMULATIVE net extension on this root-to-leaf path: the child's
		// extensions == nextDepth + (ply+1) - RootDepth, so cap nextDepth to keep that
		// <= EXTENSION_BUDGET. Once the budget is spent, checks decrement normally and
		// the line terminates. Generous enough to leave normal tactical search
		// untouched (verified node-identical on Kiwipete d10-12), finite enough that
		// the C4/C5 LMR-rescan vectors can no longer recurse without bound.
		if maxNextDepth := info.RootDepth - (ply + 1) + EXTENSION_BUDGET; nextDepth > maxNextDepth {
			nextDepth = maxNextDepth
		}

		var score int

		// Enhanced Principal Variation Search (PVS) with PV-node optimization
		if legalTried == 1 {
			// First move: search with full window and PV flag
			score = -alphaBetaPV(pos, nextDepth, ply+1, -beta, -alpha, isPV, true, info)
		} else {
			// Non-first moves: enhanced PVS with PV-aware reductions

			// PV-aware Late Move Reductions (use legalTried, not moveCount which includes illegal moves)
			reduction := 0
			if SearchToggles.LMR && depth >= LMR_FULL_DEPTH && legalTried > LMR_MOVE_THRESHOLD &&
				!move.IsCapture() && !inCheck && !givesCheck {

				// Stockfish-style log-based base reduction; non-PV gets +1
				d := depth
				if d > 63 {
					d = 63
				}
				m := legalTried
				if m > 63 {
					m = 63
				}
				reduction = lmrTable[d][m]
				if !isPV {
					reduction++
				}

				// Reduce one less ply when our position is improving.
				if !improving {
					reduction++
				}

				// History-based LMR adjustment:
				// Reduce more for moves with bad history, less for moves with good history
				histScore := GetHistoryScore(move, savedLast)
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
			info.NullWindowScouts++
			score = -alphaBetaPV(pos, reducedDepth, ply+1, -alpha-1, -alpha, false, true, info)

			// LMR re-search (C4): a reduced search that beats alpha may have
			// under-counted the move, so re-verify it at full depth with the null
			// window. This must NOT be gated by score<beta — in a non-PV node
			// beta==alpha+1, so that guard was never satisfiable and reduced fail-highs
			// went UNVERIFIED in ~99% of the tree (reduced cutoffs taken on faith, and
			// the check extension in nextDepth never actually applied off the PV). The
			// per-path extension budget bounds the check recursion this re-search opens.
			if reduction > 0 && score > alpha {
				info.LMRReSearches++
				score = -alphaBetaPV(pos, nextDepth, ply+1, -alpha-1, -alpha, false, true, info)
			}
			// PVS full-window re-search: only a PV node has a window wider than the null
			// window, so only there does an in-window score need its exact value.
			if isPV && score > alpha && score < beta {
				info.PVReSearches++
				score = -alphaBetaPV(pos, nextDepth, ply+1, -beta, -alpha, isPV, true, info)
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
			if legalTried == 1 {
				info.FirstMoveCutoffs++
			}
			// Update history table for quiet moves that cause beta cutoffs
			UpdateHistoryTable(move, savedLast, depth)
			// Update killer moves for quiet moves that cause beta cutoffs
			UpdateKillerMoves(move, depth)
			// Counter move: the cutoff move is a good reply to savedLast,
			// the move that actually brought us into this node.
			UpdateCounterMove(move, savedLast)

			// Penalize all quiet moves that were tried but didn't cause the cutoff
			for i := 0; i < searchedQuietCount; i++ {
				quietMove := searchedQuiets[i]
				if quietMove != move {
					PenalizeHistoryTable(quietMove, savedLast, depth)
				}
			}

			// Capture history: reward a cutting capture, penalize tried captures that
			// didn't cut (whether the cutoff move was a capture or a quiet).
			if move.IsCapture() {
				UpdateCaptureHistory(move, depth)
			}
			for i := 0; i < searchedCaptureCount; i++ {
				if searchedCaptures[i] != move {
					PenalizeCaptureHistory(searchedCaptures[i], depth)
				}
			}

			// Learn the pawn-structure correction from this fail-high before returning.
			maybeUpdatePawnCorrection(stm, corrIdx, staticEval, bestScore, beta, depth, bestMove, inCheck)

			// Store in transposition table as Lower Bound (Cut-Node). Skip at the
			// singular verification root: its result is a move-excluded search and
			// would corrupt the real entry for this position.
			if !inSingular {
				TranspositionTable.Set(hash, move, scoreToTT(score, ply), int8(depth), LowerBound)
			}
			return score
		}

		if score > alpha {
			alpha = score
		}
	}

	// Check for checkmate/stalemate: no legal moves were found
	if legalTried == 0 {
		if inCheck {
			// Checkmate - distance from root is `ply` directly.
			mateDistance := ply
			if mateDistance > 1000 {
				mateDistance = 1000
			}
			return -MATE_VALUE + mateDistance
		}
		// Stalemate is a draw
		return 0
	}

	// Learn the pawn-structure correction from the completed search of this node.
	maybeUpdatePawnCorrection(stm, corrIdx, staticEval, bestScore, beta, depth, bestMove, inCheck)

	// Store in transposition table
	var nodeType NodeType
	if bestScore <= origAlpha {
		// No move improved alpha - Upper Bound (All-Node)
		nodeType = UpperBound
	} else {
		// Best move improved alpha - Exact (PV-Node)
		nodeType = Exact
	}

	// Skip the store at the singular verification root (move-excluded result must
	// not overwrite the real entry for this position).
	if !inSingular {
		TranspositionTable.Set(hash, bestMove, scoreToTT(bestScore, ply), int8(depth), nodeType)
	}
	return bestScore
}

func quiescence(pos *Position, alpha int, beta int, ply int, info *SearchInfo) int {
	return quiescenceWithDepth(pos, alpha, beta, ply, info, 0)
}

func quiescenceWithDepth(pos *Position, alpha int, beta int, ply int, info *SearchInfo, qDepth int) int {
	if d := ply + qDepth; d > info.SelDepth {
		info.SelDepth = d
	}
	// Prevent infinite recursion - reduce depth limit for stability
	if qDepth >= 6 { // More conservative quiescence depth
		return EvaluateForPlayer(&pos.Board, pos.Turn())
	}

	// Fixed-node budget: trip the stop flag when spent (qsearch nodes count too).
	if m := MaxNodes(); m != 0 && info.Nodes >= m {
		RequestStop()
	}

	// External stop request - bail out without searching deeper.
	if IsStopRequested() {
		info.Stopped = true
		return EvaluateForPlayer(&pos.Board, pos.Turn())
	}

	// TT cutoff probe. Qsearch is depth 0; any cached entry at depth >= 0
	// covers us (main search stores at depth >= 1 only). Mate scores need
	// converting from node-relative TT encoding back to root-relative.
	// Also keep ttMove for move ordering even when no cutoff is taken.
	ttMove, ttEvalRaw, _, ttNodeType, ttHit := TranspositionTable.Get(pos.Hash())
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

	info.Nodes++
	info.QNodes++

	// Reusable move buffer from the recursion-frame pool (docs/10 T1b): replaces the
	// per-qnode stack `[256]Move` that escaped via the non-inlined GenerateMovesIntoBuffer
	// (~7GB allocated/search here, 96.9% of all search allocs). Indexed by the FrameDepth
	// counter so the in-check buffer survives the recursive qsearch call; stack fallback on
	// pool exhaustion. Only `moveBuffer` is needed here — the in-check and stalemate-probe
	// branches are mutually exclusive per call, and `captures [64]Move` below stays on the
	// stack (inlined copy, never escaped).
	qframeIdx := info.FrameDepth
	info.FrameDepth++
	defer func() { info.FrameDepth-- }()
	var qframe *searchFrame
	if info.Frames != nil && qframeIdx < len(info.Frames) {
		qframe = &info.Frames[qframeIdx]
	} else {
		qframe = new(searchFrame)
	}

	// Node-in-check: read the maintained InCheck tag rather than rescanning
	// (N2, node-identical) — qsearch is only entered from alphaBetaPV (same pos)
	// or recursively after a capture make (tag set by makeMoveHelper).
	inCheck := pos.IsInCheck()

	// In-check branch: must search all evasions, stand-pat is illegal here.
	if inCheck {
		moveBuffer := qframe.moveBuffer[:]
		moveCount := GenerateMovesIntoBuffer(pos, moveBuffer)

		// Move ttMove to the front if it's among the legal moves. Cheap O(n)
		// pass that pays off when the TT suggestion produces an early cutoff.
		if ttMove != EmptyMove {
			for i := 0; i < moveCount; i++ {
				if moveBuffer[i] == ttMove {
					moveBuffer[0], moveBuffer[i] = moveBuffer[i], moveBuffer[0]
					break
				}
			}
		}

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
			// Checkmate at this node — distance from root is `ply` directly.
			// Earlier code used info.Depth + qDepth, which was wrong because
			// info.Depth is the last completed iteration depth, not the current
			// search distance.
			return -MATE_VALUE + ply
		}

		return bestScore
	}

	// Not in check. Detect stalemate at qDepth==0 (rare; preserves prior behavior).
	if qDepth == 0 {
		moveBuffer := qframe.moveBuffer[:]
		moveCount := GenerateMovesIntoBuffer(pos, moveBuffer)
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

	// Stand-pat (lazy: skip the expensive eval extras when the cheap core already
	// proves the fail-high — see evaluateLazyStandPat / qsearchLazyMargin).
	standPat, lazyFailHigh := evaluateLazyStandPat(&pos.Board, pos.Turn(), beta)
	if lazyFailHigh || standPat >= beta {
		return beta
	}

	bestScore := standPat
	bestMove := EmptyMove
	if standPat > alpha {
		alpha = standPat
	}

	// Captures only. Quiet checks are dropped because moveGivesCheck misses
	// discovered, en-passant, and promotion checks; an incomplete quiet-check
	// path is worse than captures-only at the horizon.
	captureCount := GenerateCapturesIntoBuffer(pos, info.CaptureBuffer[:])
	var captures [64]Move
	copy(captures[:], (*info.CaptureBuffer)[:captureCount])

	// Swap ttMove to index 0 if it's a capture in this position. Avoids the
	// regression of upfront MVV/LVA sort (commit 2f61d3a, -129 ELO) while
	// still letting the TT-suggested capture cut early.
	if ttMove != EmptyMove {
		for i := 0; i < captureCount; i++ {
			if captures[i] == ttMove {
				captures[0], captures[i] = captures[i], captures[0]
				break
			}
		}
	}

	for i := 0; i < captureCount; i++ {
		move := captures[i]

		// Delta pruning: skip captures that cannot possibly raise alpha. For
		// promotion-captures, include the promotion gain so we don't prune a
		// winning Pxf8=Q that would actually deliver a queen.
		gain := int(move.CapturedPiece().seeWeight())
		if move.PromoType() != NoType {
			promoPiece := GetPiece(move.PromoType(), move.MovingPiece().Color())
			gain += int(promoPiece.seeWeight()) - 100
		}
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
			bestMove = move
		}
		if score >= beta {
			// Fail-high: store a lower bound (the cutting capture) at depth 0 (S3).
			// Only fail-highs are stored — the common non-failing-high case would
			// flood the TT with depth-0 entries and evict deeper main-search ones.
			TranspositionTable.Set(pos.Hash(), bestMove, scoreToTT(beta, ply), 0, LowerBound)
			info.QSearchTTStores++
			return beta
		}
		if score > alpha {
			alpha = score
		}
	}

	return bestScore
}

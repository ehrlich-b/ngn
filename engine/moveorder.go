package engine

// History table for move ordering heuristic, indexed by [piece][to].
// Piece is `Move.MovingPiece()` (1..12 mapping WhitePawn..BlackKing; 0 is unused
// NoPiece sentinel). Indexing by piece-type rather than from-square is strictly
// more discriminative: a knight retreating to e4 and a queen trading on e4 are
// stored separately, so a strong knight-to-e4 history doesn't elevate weak
// queen-to-e4 moves. Same total bookkeeping cost; smaller table (13×64 = 832
// entries) than the prior [64][64] = 4096.
var historyTable [13][64]int

// Continuation history: indexed by [prev-piece][prev-to][curr-piece][curr-to].
// Captures move sequences that work well together — e.g. given that opponent
// just moved knight to e5, our rook lifting to e3 may be especially good.
// 13×64×13×64 = 692,224 ints (~5.5MB). Read alongside the main history table
// in move ordering and at history-pruning / LMR-history sites; updated at the
// same beta-cutoff and quiet-penalty points.
var continuationHistory [13][64][13][64]int

// Pawn-structure correction history (Stockfish concept): learns a per-pawn-structure
// adjustment to the static eval from the gap between static eval and the search result.
// Static eval is structurally blind to some recurring pawn shapes; this nudges it toward
// what search actually finds for positions sharing a pawn skeleton. Indexed by
// [side-to-move][hashed-pawn-structure]; collisions just share a slot (acceptable noise).
const (
	corrHistSize  = 16384 // 2^14 slots per side
	corrHistMask  = corrHistSize - 1
	corrHistLimit = 1024 // clamp on the stored EMA entry (Stockfish CORRECTION_HISTORY_LIMIT)
)

var pawnCorrectionHistory [2][corrHistSize]int

// pawnCorrectionIndex hashes both colors' pawn bitboards into a correction-table slot.
func pawnCorrectionIndex(whitePawns, blackPawns uint64) int {
	key := whitePawns*0x9E3779B97F4A7C15 ^ blackPawns*0xC2B2AE3D27D4EB4F
	key ^= key >> 29
	return int(key & corrHistMask)
}

// correctionValue returns the centipawn adjustment to add to the static eval. The
// 6245/131072 weight mirrors Stockfish's pawn-correction term, so a saturated entry
// (±corrHistLimit) yields at most ±49 cp.
func correctionValue(stm Color, idx int) int {
	return pawnCorrectionHistory[stm][idx] * 6245 / 131072
}

// updatePawnCorrection nudges a slot toward the observed (searchScore - staticEval) gap
// with a depth-weighted, gravity-damped EMA (Stockfish StatsEntry form) that keeps the
// entry bounded in roughly [-corrHistLimit, corrHistLimit].
func updatePawnCorrection(stm Color, idx, diff, depth int) {
	bonus := diff * depth / 8
	if bonus > corrHistLimit/4 {
		bonus = corrHistLimit / 4
	} else if bonus < -corrHistLimit/4 {
		bonus = -corrHistLimit / 4
	}
	mag := bonus
	if mag < 0 {
		mag = -mag
	}
	e := &pawnCorrectionHistory[stm][idx]
	*e += bonus - (*e)*mag/corrHistLimit
}

// maybeUpdatePawnCorrection applies the correction update at a completed node when the
// search result is a consistent directional signal: worse than static eval (a true upper
// bound) or better with a real best move. Capture-driven results are skipped as noisy.
func maybeUpdatePawnCorrection(stm Color, idx, staticEval, bestScore, beta, depth int, bestMove Move, inCheck bool) {
	if inCheck {
		return
	}
	if bestMove != EmptyMove && bestMove.IsCapture() {
		return
	}
	if !((bestScore < staticEval && bestScore < beta) || (bestScore > staticEval && bestMove != EmptyMove)) {
		return
	}
	updatePawnCorrection(stm, idx, bestScore-staticEval, depth)
}

// Killer moves table for move ordering heuristic
// killerMoves[depth][slot] stores killer moves at each depth (2 killer moves per depth)
var killerMoves [MaximumDepth][2]Move

// Counter moves table for move ordering heuristic
// counterMoves[from][to] stores the best counter move to moves from 'from' to 'to'
var counterMoves [64][64]Move

// Track the last move played for counter move heuristic
var lastMovePlayed Move

// UpdateHistoryTable increases the history score for a move that caused a cutoff
func UpdateHistoryTable(move Move, prevMove Move, depth int) {
	if move.IsCapture() || move.PromoType() != NoType {
		return
	}
	piece := move.MovingPiece()
	to := move.Destination()
	bonus := depth * depth
	historyTable[piece][to] += bonus
	if historyTable[piece][to] > 8192 {
		ageHistoryTable()
	}
	if prevMove != EmptyMove {
		prevPiece := prevMove.MovingPiece()
		prevTo := prevMove.Destination()
		continuationHistory[prevPiece][prevTo][piece][to] += bonus
		if continuationHistory[prevPiece][prevTo][piece][to] > 8192 {
			continuationHistory[prevPiece][prevTo][piece][to] /= 2
		}
	}
}

// PenalizeHistoryTable decreases the history score for moves that didn't cause cutoff
func PenalizeHistoryTable(move Move, prevMove Move, depth int) {
	if move.IsCapture() || move.PromoType() != NoType {
		return
	}
	piece := move.MovingPiece()
	to := move.Destination()
	penalty := depth * depth
	historyTable[piece][to] -= penalty
	if historyTable[piece][to] < -8192 {
		historyTable[piece][to] = -8192
	}
	if prevMove != EmptyMove {
		prevPiece := prevMove.MovingPiece()
		prevTo := prevMove.Destination()
		continuationHistory[prevPiece][prevTo][piece][to] -= penalty
		if continuationHistory[prevPiece][prevTo][piece][to] < -8192 {
			continuationHistory[prevPiece][prevTo][piece][to] = -8192
		}
	}
}

// GetHistoryScore returns the history score for a move, summing the main
// [piece][to] history with the continuation [prev-piece][prev-to][piece][to]
// when prevMove is available. Continuation gets full weight — same magnitude
// updates means same ELO scale.
func GetHistoryScore(move Move, prevMove Move) int {
	if move.IsCapture() || move.PromoType() != NoType {
		return 0
	}
	piece := move.MovingPiece()
	to := move.Destination()
	score := historyTable[piece][to]
	if prevMove != EmptyMove {
		score += continuationHistory[prevMove.MovingPiece()][prevMove.Destination()][piece][to]
	}
	return score
}

// Capture history: indexed by [moving-piece][to][captured-piece]. The quiet history
// tables above skip captures, so without this captures are ordered by SEE alone. This
// adds a learned tie-breaker — captures that repeatedly cause cutoffs in this search are
// tried earlier among captures of similar SEE. Gravity-clamped like the quiet tables.
var captureHistory [13][64][13]int

func captureHistoryScore(move Move) int {
	return captureHistory[move.MovingPiece()][move.Destination()][move.CapturedPiece()]
}

// UpdateCaptureHistory rewards a capture that caused a beta cutoff.
func UpdateCaptureHistory(move Move, depth int) {
	e := &captureHistory[move.MovingPiece()][move.Destination()][move.CapturedPiece()]
	*e += depth * depth
	if *e > 8192 {
		*e /= 2
	}
}

// PenalizeCaptureHistory decreases a tried capture that did not cause the cutoff.
func PenalizeCaptureHistory(move Move, depth int) {
	e := &captureHistory[move.MovingPiece()][move.Destination()][move.CapturedPiece()]
	*e -= depth * depth
	if *e < -8192 {
		*e = -8192
	}
}

// ClearHistoryTable resets both history tables (used between games)
func ClearHistoryTable() {
	for i := range historyTable {
		for j := range historyTable[i] {
			historyTable[i][j] = 0
		}
	}
	for i := range continuationHistory {
		for j := range continuationHistory[i] {
			for k := range continuationHistory[i][j] {
				for l := range continuationHistory[i][j][k] {
					continuationHistory[i][j][k][l] = 0
				}
			}
		}
	}
	for i := range pawnCorrectionHistory {
		for j := range pawnCorrectionHistory[i] {
			pawnCorrectionHistory[i][j] = 0
		}
	}
	for i := range captureHistory {
		for j := range captureHistory[i] {
			for k := range captureHistory[i][j] {
				captureHistory[i][j][k] = 0
			}
		}
	}
}

// ageHistoryTable divides all history values by 2 to prevent overflow
func ageHistoryTable() {
	for i := range historyTable {
		for j := range historyTable[i] {
			historyTable[i][j] /= 2
		}
	}
}

// UpdateKillerMoves stores a move that caused a beta cutoff as a killer move
func UpdateKillerMoves(move Move, depth int) {
	// Only store quiet moves as killer moves
	if !move.IsCapture() && move.PromoType() == NoType && depth >= 0 && depth < MaximumDepth {
		// Check if this move is already stored as first killer
		if killerMoves[depth][0] == move {
			return
		}

		// Shift moves down: second killer becomes first, new move becomes second
		killerMoves[depth][1] = killerMoves[depth][0]
		killerMoves[depth][0] = move
	}
}

// IsKillerMove checks if a move is a killer move at the given depth
func IsKillerMove(move Move, depth int) bool {
	if depth < 0 || depth >= MaximumDepth {
		return false
	}
	return killerMoves[depth][0] == move || killerMoves[depth][1] == move
}

// ClearKillerMoves resets the killer moves table
func ClearKillerMoves() {
	for i := range killerMoves {
		killerMoves[i][0] = EmptyMove
		killerMoves[i][1] = EmptyMove
	}
}

// UpdateCounterMove stores a move as the best counter to the given previous move
func UpdateCounterMove(counterMove Move, previousMove Move) {
	// Only store quiet moves as counter moves
	if !counterMove.IsCapture() && counterMove.PromoType() == NoType && previousMove != EmptyMove {
		from := previousMove.Source()
		to := previousMove.Destination()
		counterMoves[from][to] = counterMove
	}
}

// GetCounterMove returns the counter move for the given previous move
func GetCounterMove(previousMove Move) Move {
	if previousMove == EmptyMove {
		return EmptyMove
	}
	from := previousMove.Source()
	to := previousMove.Destination()
	return counterMoves[from][to]
}

// IsCounterMove checks if a move is the counter move to the previous move
func IsCounterMove(move Move, previousMove Move) bool {
	return GetCounterMove(previousMove) == move && move != EmptyMove
}

// SetLastMovePlayed sets the last move played for counter move heuristic
func SetLastMovePlayed(move Move) {
	lastMovePlayed = move
}

// GetLastMovePlayed returns the last move played
func GetLastMovePlayed() Move {
	return lastMovePlayed
}

// ClearCounterMoves resets the counter moves table
func ClearCounterMoves() {
	for i := range counterMoves {
		for j := range counterMoves[i] {
			counterMoves[i][j] = EmptyMove
		}
	}
	lastMovePlayed = EmptyMove
}

// Single-pass move ordering with enhanced SEE integration
// Orders moves in one pass using scores, with captures ordered by SEE value
func orderMovesSinglePass(moves []Move, depth int, previousMove Move, ttMove Move, ttHit bool, pos *Position) []Move {
	if len(moves) == 0 {
		return moves
	}

	// Use zero-allocation buffer version internally
	var buffer [256]Move
	count := orderMovesSinglePassIntoBuffer(moves, depth, previousMove, ttMove, ttHit, buffer[:], pos)

	// Return a copy of the ordered moves (only allocation, but tests need owned slice)
	result := make([]Move, count)
	copy(result, buffer[:count])
	return result
}

// Zero-allocation single-pass ordering into caller-provided buffer. Scores every move then
// fully sorts; used at the root and wherever the entire ordered list is consumed.
func orderMovesSinglePassIntoBuffer(moves []Move, depth int, previousMove Move, ttMove Move, ttHit bool, out []Move, pos *Position) int {
	// Use a fixed-size array for scoring to avoid allocations
	var scores [256]int
	n := scoreMovesIntoBuffer(moves, depth, previousMove, ttMove, ttHit, out, scores[:], pos)
	for i := 0; i < n-1; i++ {
		selectNextMove(out, scores[:], n, i)
	}
	return n
}

// scoreMovesIntoBuffer writes each move and its ordering score into the parallel out/scores
// buffers WITHOUT sorting. The search drives the sort incrementally via selectNextMove so the
// O(n^2) selection tail past a beta cutoff is never paid. Returns the number of moves written.
func scoreMovesIntoBuffer(moves []Move, depth int, previousMove Move, ttMove Move, ttHit bool, out []Move, scores []int, pos *Position) int {
	n := len(moves)
	if n == 0 || len(out) < n || len(scores) < n {
		return 0
	}

	// Hoist the continuation-history sub-table for previousMove out of the per-move
	// loop (previousMove is constant for the whole list), so each quiet move pays one
	// 2-D index instead of the full 4-D continuationHistory index. Bit-identical to
	// the GetHistoryScore path below.
	usePrev := previousMove != EmptyMove
	var contHist *[13][64]int
	if usePrev {
		contHist = &continuationHistory[previousMove.MovingPiece()][previousMove.Destination()]
	}

	// Hoist the loop-invariant counter move and killer slots out of the per-move loop.
	// previousMove and depth are constant for the whole list, so each quiet move becomes
	// two equality tests instead of re-indexing counterMoves[from][to] and the killer
	// table (with its bounds check) every iteration. EmptyMove never matches a real
	// generated quiet move, so the guards below are bit-identical to IsCounterMove /
	// IsKillerMove.
	counterMove := GetCounterMove(previousMove)
	var killer0, killer1 Move
	if depth >= 0 && depth < MaximumDepth {
		killer0 = killerMoves[depth][0]
		killer1 = killerMoves[depth][1]
	}

	// Single pass: score all moves
	for i, move := range moves {
		score := 0

		// TT move gets highest priority
		if ttHit && move == ttMove {
			score = 100000
		} else {
			// Calculate score based on move type and SEE
			if move.IsCapture() || move.IsEnPassant() {
				// Use SEE to order captures precisely
				seeValue := staticExchangeEvaluation(pos, move)
				// Capture-history tie-breaker, scaled down to stay secondary to SEE and
				// inside each SEE band (band gap is 60000, this is at most ±1024).
				chScore := captureHistoryScore(move) / 8
				if seeValue >= 0 {
					// Winning/equal captures: base score + SEE value for fine ordering
					score = 80000 + seeValue + 1000 + chScore
				} else {
					// Losing captures: below killers/counters but above unscored quiet moves
					score = 20000 + seeValue + 1000 + chScore
				}
			} else if move.PromoType() != NoType {
				promoWeight := int(GetPiece(move.PromoType(), move.MovingPiece().Color()).Weight())
				if move.PromoType() == Queen {
					// Non-capture queen promotion: gains ~800cp; rank above winning
					// captures (which max around 82000) so we explore it before any
					// capture or quiet alternative.
					score = 90000 + promoWeight
				} else {
					// Under-promotions are rare and usually inferior; keep them
					// in the original priority band, ordered by promoted piece value.
					score = 70000 + promoWeight
				}
			} else if move.IsCastle() {
				// Castling: moderate priority
				score = 50000
			} else {
				// Quiet moves: use heuristics
				score = 0

				// Counter moves get highest priority among quiet moves
				if move == counterMove && counterMove != EmptyMove {
					score += 30000
				}

				// Killer moves get high priority
				if move == killer0 || move == killer1 {
					score += 25000
				}

				// History heuristic for remaining quiet moves; continuation
				// history adds context-aware ordering when previousMove != EmptyMove.
				// Inlined GetHistoryScore using the hoisted contHist: this branch only
				// runs for quiet moves, so that function's capture/promo guard (which
				// would return 0) is always false here — identical result.
				piece := move.MovingPiece()
				to := move.Destination()
				hs := historyTable[piece][to]
				if usePrev {
					hs += contHist[piece][to]
				}
				score += hs
			}
		}

		scores[i] = score
		out[i] = move
	}

	return n
}

// selectNextMove performs one selection-sort step: it scans out[i:n] for the highest score
// and swaps that entry (and its score) into position i. Calling it for i = 0,1,2,... yields
// moves in the exact order a full selection sort produces, so the search can stop as soon as
// a beta cutoff fires instead of sorting the whole list. Strict > keeps ties at their earlier
// index, matching the full-sort order — so the incremental path is node-identical.
func selectNextMove(out []Move, scores []int, n, i int) {
	maxIdx := i
	maxScore := scores[i] // hoist out of the loop: avoids re-loading scores[maxIdx] each iteration
	for j := i + 1; j < n; j++ {
		if scores[j] > maxScore {
			maxIdx = j
			maxScore = scores[j]
		}
	}
	if maxIdx != i {
		scores[i], scores[maxIdx] = scores[maxIdx], scores[i]
		out[i], out[maxIdx] = out[maxIdx], out[i]
	}
}

func orderMoves(moves []Move, pos *Position) []Move {
	return orderMovesWithDepth(moves, 0, pos)
}

func orderMovesWithDepth(moves []Move, depth int, pos *Position) []Move {
	return orderMovesWithDepthAndPrevious(moves, depth, GetLastMovePlayed(), pos)
}

func orderMovesWithDepthAndTTMove(moves []Move, depth int, ttMove Move, ttHit bool, pos *Position) []Move {
	return orderMovesWithDepthPreviousAndTTMove(moves, depth, GetLastMovePlayed(), ttMove, ttHit, pos)
}

func orderMovesWithDepthPreviousAndTTMove(moves []Move, depth int, previousMove Move, ttMove Move, ttHit bool, pos *Position) []Move {
	return orderMovesSinglePass(moves, depth, previousMove, ttMove, ttHit, pos)
}

// Zero-allocation ordering into caller-provided buffer
// Returns number of moves written to out (equals len(moves))
func orderMovesIntoBufferWithDepthPreviousAndTTMove(moves []Move, depth int, previousMove Move, ttMove Move, ttHit bool, out []Move, pos *Position) int {
	return orderMovesSinglePassIntoBuffer(moves, depth, previousMove, ttMove, ttHit, out, pos)
}

func orderMovesIntoBufferWithDepth(moves []Move, depth int, out []Move, pos *Position) int {
	return orderMovesIntoBufferWithDepthPreviousAndTTMove(moves, depth, GetLastMovePlayed(), EmptyMove, false, out, pos)
}

func orderMovesWithDepthAndPrevious(moves []Move, depth int, previousMove Move, pos *Position) []Move {
	return orderMovesWithDepthPreviousAndTTMove(moves, depth, previousMove, EmptyMove, false, pos)
}

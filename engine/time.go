package engine

import (
	"math"
	"sync/atomic"
	"time"
)

// TimeControl represents different types of time controls
type TimeControl int

const (
	FixedDepth TimeControl = iota
	FixedTime
	TimePerMove
	Tournament
	Infinite
)

// TimeManager handles time allocation and management during search
type TimeManager struct {
	timeControl   TimeControl
	allocatedTime time.Duration // Time allocated for this move (== hardTime for Tournament)
	softTime      time.Duration // Tournament soft budget: target before starting a new ID iteration
	hardTime      time.Duration // Tournament hard ceiling: never exceeded (the anti-flag guarantee)
	startTime     time.Time
	emergencyTime time.Duration // Reserve time for emergencies
	moveOverhead  time.Duration // Network/GUI overhead per move

	// Tournament time control parameters
	baseTime  time.Duration // Base time available
	increment time.Duration // Increment per move
	movesToGo int           // Moves until next time control

	// Game state for time allocation
	gamePhase   GamePhase
	movesPlayed int
	isEndgame   bool

	// Dynamic adjustment factors
	lastScoreChange    int
	positionComplexity float64

	// Internal rate limiting for frequent time checks
	checkCounter  uint64    // Atomic counter for time checks
	lastTimeCheck time.Time // Last time we actually checked the clock
	shouldStop    bool      // Cached result of last time check

	// iterationStartTime is set by NewIteration at the top of each iterative
	// deepening depth and used by the soft-stop heuristic to estimate the
	// next iteration's cost. Previously each caller passed time.Now() in this
	// slot, which made iterationTime always ~0 and disabled soft-stop.
	iterationStartTime time.Time
	// lastIterationTime is the wall time the most recently COMPLETED iteration
	// took; the soft-stop projects the next iteration as a multiple of it.
	lastIterationTime time.Duration
}

// GamePhase represents the current phase of the game
type GamePhase int

const (
	Opening GamePhase = iota
	Middlegame
	Endgame
)

// NewTimeManager creates a new time manager
func NewTimeManager() *TimeManager {
	return &TimeManager{
		timeControl:   FixedDepth,
		emergencyTime: 100 * time.Millisecond, // Reserve 100ms
		moveOverhead:  50 * time.Millisecond,  // 50ms overhead
	}
}

// SetTimeControl configures the time manager for different time controls
func (tm *TimeManager) SetTimeControl(params SearchParams, isWhite bool) {
	tm.startTime = time.Now()
	tm.lastTimeCheck = tm.startTime
	tm.shouldStop = false
	tm.iterationStartTime = time.Time{} // Cleared until NewIteration is called
	tm.lastIterationTime = 0
	atomic.StoreUint64(&tm.checkCounter, 0)

	// Determine time control type and allocate time accordingly
	if params.Infinite {
		tm.timeControl = Infinite
		tm.allocatedTime = time.Hour * 24 // Effectively infinite
	} else if params.MoveTime > 0 {
		tm.timeControl = TimePerMove
		tm.allocatedTime = time.Duration(params.MoveTime) * time.Millisecond
	} else if params.Depth > 0 && (params.WhiteTime == 0 && params.BlackTime == 0) {
		tm.timeControl = FixedDepth
		tm.allocatedTime = time.Second * 30 // Max 30 seconds for fixed depth
	} else {
		// Tournament time control
		tm.timeControl = Tournament

		var myTime, myInc int
		if isWhite {
			myTime = params.WhiteTime
			myInc = params.WhiteInc
		} else {
			myTime = params.BlackTime
			myInc = params.BlackInc
		}

		tm.baseTime = time.Duration(myTime) * time.Millisecond
		tm.increment = time.Duration(myInc) * time.Millisecond
		tm.movesToGo = params.MovesToGo

		// Tournament computes its own soft/hard budgets (with built-in safety) and
		// returns; the shared cap/overhead below is for the simpler modes only.
		tm.computeTournamentLimits()
		return
	}

	// Non-tournament modes (TimePerMove/FixedDepth/Infinite): cap and account for
	// overhead. (Tournament returned above with its own remaining-relative caps.)
	tm.allocatedTime = tm.ensureSafeTimeAllocation()

	// Global maximum time limit for any move
	// Keep a 30 second maximum for expediency
	maxTime := time.Second * 30
	if tm.allocatedTime > maxTime {
		tm.allocatedTime = maxTime
	}
}

// computeTournamentLimits sets the soft and hard per-move budgets from an
// increment-aware, remaining-relative formula. The old allocation stacked
// game-phase × complexity × score × time multipliers on top of an even share,
// so it could plan several× a sustainable per-move slice and bleed the bank far
// faster than the increment refilled. This version is sustainable at any clock:
//   - soft = (usable bank / moves-to-go) + 0.8·increment  — the normal target;
//     spending ~the increment each move keeps the bank roughly flat.
//   - hard = min(4·soft, 0.3·usable bank)                 — the rare critical-move
//     ceiling; the 0.3 cap means one move can never blow the bank (anti-flag).
func (tm *TimeManager) computeTournamentLimits() {
	moves := tm.movesToGo
	if moves <= 0 {
		moves = tm.estimateMovesRemaining()
	}
	if moves < 1 {
		moves = 1
	}
	// Reserve so we never plan into the emergency buffer or the GUI/move overhead.
	usable := tm.baseTime - tm.emergencyTime - tm.moveOverhead
	if usable < 0 {
		usable = 0
	}
	uf := float64(usable)

	soft := uf/float64(moves) + 0.8*float64(tm.increment)
	hard := math.Min(soft*4, uf*0.3)
	if hard < soft {
		hard = soft // tiny banks: don't let the 0.3 cap fall below the soft target
	}

	tm.softTime = time.Duration(soft)
	tm.hardTime = time.Duration(hard)
	tm.allocatedTime = tm.softTime // the normal per-move target (hard is the rare ceiling)
}

// estimateMovesRemaining estimates how many moves are left in the game
func (tm *TimeManager) estimateMovesRemaining() int {
	// Be much more optimistic about remaining moves to allow more thinking time
	remaining := 40 - tm.movesPlayed // Assume shorter games for better time usage
	switch tm.gamePhase {
	case Opening:
		remaining = 30 - tm.movesPlayed // Shorter estimate in opening
	case Middlegame:
		remaining = 25 - tm.movesPlayed // Focus time on critical middle game
	case Endgame:
		remaining = 20 - tm.movesPlayed // Endgame needs precise calculation
	}

	// Never estimate more than 40 moves remaining, even early in game
	if remaining > 40 {
		remaining = 40
	}
	// Always assume at least 10 moves remaining for safety
	if remaining < 10 {
		remaining = 10
	}

	return remaining
}

// ensureSafeTimeAllocation ensures we don't allocate too much time
func (tm *TimeManager) ensureSafeTimeAllocation() time.Duration {
	if tm.timeControl == Tournament {
		// Scale maximum allocation with available time
		var maxAllocation time.Duration
		if tm.baseTime > time.Minute*10 {
			// With lots of time, can use up to 1/2 of remaining time
			maxAllocation = tm.baseTime / 2
		} else if tm.baseTime > time.Minute*5 {
			// With moderate time, use up to 2/5 of remaining time
			maxAllocation = tm.baseTime * 2 / 5
		} else {
			// With little time, be conservative - 1/3 of remaining time
			maxAllocation = tm.baseTime / 3
		}
		if tm.allocatedTime > maxAllocation {
			tm.allocatedTime = maxAllocation
		}

		// Always keep emergency reserve
		if tm.allocatedTime > tm.baseTime-tm.emergencyTime {
			tm.allocatedTime = tm.baseTime - tm.emergencyTime
		}
	}

	// Account for move overhead
	if tm.allocatedTime > tm.moveOverhead {
		tm.allocatedTime -= tm.moveOverhead
	}

	return tm.allocatedTime
}

// ShouldStopSearch determines if search should stop based on time
// This method can be called frequently without performance penalty - it uses
// internal rate limiting to only do expensive time checks every N calls
// NewIteration is called by iterative deepening at the top of each new depth
// so the soft-stop heuristic can measure how long the current iteration took
// and project the next one. Without this, ShouldStopSearch's nextIterationEstimate
// would be zero.
func (tm *TimeManager) NewIteration() {
	// Record how long the iteration that just finished took, so the soft-stop can
	// project the next one. NewIteration runs at the top of each ID depth, so when
	// called for depth N the previous (N-1) iteration's elapsed is measured here.
	if !tm.iterationStartTime.IsZero() {
		tm.lastIterationTime = time.Since(tm.iterationStartTime)
	}
	tm.iterationStartTime = time.Now()
}

func (tm *TimeManager) ShouldStopSearch(depth int, _ time.Time) bool {
	// Increment counter atomically - this is very fast
	count := atomic.AddUint64(&tm.checkCounter, 1)

	// Only do expensive time checking every 1024 calls (or immediately if we should stop)
	if tm.shouldStop || count%1024 == 0 {
		now := time.Now()
		elapsed := now.Sub(tm.startTime)
		tm.lastTimeCheck = now

		switch tm.timeControl {
		case FixedDepth:
			tm.shouldStop = false // Time doesn't matter for fixed depth
		case Infinite:
			tm.shouldStop = false // Never stop for infinite search
		case TimePerMove, FixedTime:
			// Hard time limit - stop when time is up
			tm.shouldStop = elapsed >= tm.allocatedTime
		case Tournament:
			tm.shouldStop = tm.shouldStopTournamentSearch(depth, elapsed)
		}
	}

	return tm.shouldStop
}

// shouldStopTournamentSearch decides, from the soft/hard budgets set by
// computeTournamentLimits, whether to stop searching. The OLD version layered
// absolute minThinkTime (0.5-8s) and minDepth (4-12) floors that ignored the bank
// — the bullet-killers that forced spending a fixed wall-time regardless of how
// little time was left. This version is purely bank-relative.
func (tm *TimeManager) shouldStopTournamentSearch(depth int, elapsed time.Duration) bool {
	// Hard ceiling — never exceeded. computeTournamentLimits caps this at 30% of the
	// usable bank, so a single move can never flag us (the anti-flag guarantee).
	if elapsed >= tm.hardTime {
		return true
	}
	// Emergency: down to the reserve, stop regardless of the budget arithmetic.
	if tm.baseTime-elapsed <= tm.emergencyTime {
		return true
	}
	// Always finish at least one iteration so the move is real (never a depth-0
	// blunder); the search's clock-interrupt guard keeps the last completed depth
	// if a later iteration is cut off mid-flight.
	if depth < 1 {
		return false
	}
	// Soft limit: don't START another iteration we can't finish within the soft
	// budget. Project the next iteration as ~2x the last completed one (typical PVS
	// growth with good ordering). Before any iteration timing exists, fall back to a
	// plain elapsed-vs-soft check.
	if tm.lastIterationTime == 0 {
		return elapsed >= tm.softTime
	}
	return elapsed+tm.lastIterationTime*2 > tm.softTime
}

// UpdateGameState updates the time manager with current game state
func (tm *TimeManager) UpdateGameState(movesPlayed int, lastScore, currentScore int, position *Position) {
	tm.movesPlayed = movesPlayed
	tm.lastScoreChange = currentScore - lastScore

	// Determine game phase
	tm.gamePhase = tm.determineGamePhase(position)

	// Calculate position complexity
	tm.positionComplexity = tm.calculatePositionComplexity(position)
}

// determineGamePhase determines the current game phase
func (tm *TimeManager) determineGamePhase(position *Position) GamePhase {
	if position == nil {
		return Middlegame // Default
	}

	// Count material to determine phase
	totalMaterial := 0
	queens := PopCount(position.Board.GetBitboardOf(WhiteQueen)) + PopCount(position.Board.GetBitboardOf(BlackQueen))
	rooks := PopCount(position.Board.GetBitboardOf(WhiteRook)) + PopCount(position.Board.GetBitboardOf(BlackRook))
	minors := PopCount(position.Board.GetBitboardOf(WhiteBishop)) + PopCount(position.Board.GetBitboardOf(BlackBishop)) +
		PopCount(position.Board.GetBitboardOf(WhiteKnight)) + PopCount(position.Board.GetBitboardOf(BlackKnight))

	totalMaterial = queens*9 + rooks*5 + minors*3

	if tm.movesPlayed < 15 && totalMaterial > 60 {
		return Opening
	} else if totalMaterial < 20 {
		return Endgame
	}

	return Middlegame
}

// calculatePositionComplexity calculates a complexity score for the position
func (tm *TimeManager) calculatePositionComplexity(position *Position) float64 {
	if position == nil {
		return 0.5 // Default complexity
	}

	allPiecesBitboard := position.Board.GetWhitePieces() | position.Board.GetBlackPieces()
	totalPieces := PopCount(allPiecesBitboard)
	if totalPieces == 0 {
		return 0.0
	}

	complexity := 0.0

	// Piece count: middlegame (15-25 pieces) is most complex, opening and endgame less so.
	// Triangle peak at 20 pieces.
	pieceCountFactor := 1.0 - math.Abs(float64(totalPieces)-20.0)/20.0
	if pieceCountFactor < 0 {
		pieceCountFactor = 0
	}
	complexity += pieceCountFactor * 0.3

	// More piece-type variety = more complex
	pieceTypes := 0
	for _, piece := range []Piece{WhiteQueen, BlackQueen, WhiteRook, BlackRook,
		WhiteBishop, BlackBishop, WhiteKnight, BlackKnight} {
		if PopCount(position.Board.GetBitboardOf(piece)) > 0 {
			pieceTypes++
		}
	}
	complexity += float64(pieceTypes) / 8.0 * 0.3

	return math.Min(complexity, 1.0)
}

// GetTimeControlString returns a string describing the current time control
func (tm *TimeManager) GetTimeControlString() string {
	switch tm.timeControl {
	case FixedDepth:
		return "Fixed Depth"
	case FixedTime:
		return "Fixed Time"
	case TimePerMove:
		return "Time Per Move"
	case Tournament:
		return "Tournament"
	case Infinite:
		return "Infinite"
	}
	return "Unknown"
}

// GetAllocatedTime returns the time allocated for the current move
func (tm *TimeManager) GetAllocatedTime() time.Duration {
	return tm.allocatedTime
}

// GetElapsedTime returns the time elapsed since search started
func (tm *TimeManager) GetElapsedTime() time.Duration {
	return time.Since(tm.startTime)
}

// IsEmergencyTime returns true if we're in emergency time
func (tm *TimeManager) IsEmergencyTime() bool {
	if tm.timeControl != Tournament {
		return false
	}

	elapsed := time.Since(tm.startTime)
	remaining := tm.baseTime - elapsed

	return remaining <= tm.emergencyTime*2 // Emergency when 2x emergency time left
}

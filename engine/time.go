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
	allocatedTime time.Duration // Time allocated for this move
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

		tm.allocatedTime = tm.calculateTimeAllocation()
	}

	// Ensure we don't allocate more time than we have
	tm.allocatedTime = tm.ensureSafeTimeAllocation()

	// Global maximum time limit for any move
	// Keep a 30 second maximum for expediency
	maxTime := time.Second * 30
	if tm.allocatedTime > maxTime {
		tm.allocatedTime = maxTime
	}
}

// calculateTimeAllocation calculates optimal time allocation for tournament play
func (tm *TimeManager) calculateTimeAllocation() time.Duration {
	baseAllocation := tm.calculateBaseAllocation()

	// Apply game phase multiplier
	phaseMultiplier := tm.getGamePhaseMultiplier()

	// Apply complexity multiplier
	complexityMultiplier := tm.getComplexityMultiplier()

	// Apply score change multiplier (spend more time when score drops)
	scoreMultiplier := tm.getScoreChangeMultiplier()

	allocation := float64(baseAllocation) * phaseMultiplier * complexityMultiplier * scoreMultiplier

	return time.Duration(allocation)
}

// calculateBaseAllocation calculates the base time allocation
func (tm *TimeManager) calculateBaseAllocation() time.Duration {
	if tm.movesToGo > 0 {
		// Fixed number of moves until time control
		movesLeft := float64(tm.movesToGo)
		timePerMove := float64(tm.baseTime)/movesLeft + float64(tm.increment)

		// Use 80% of calculated time, save 20% for later moves
		return time.Duration(timePerMove * 0.8)
	}

	// No moves to go specified - estimate moves remaining
	estimatedMovesLeft := tm.estimateMovesRemaining()
	timePerMove := float64(tm.baseTime) / float64(estimatedMovesLeft)

	// Add increment to available time
	availableTime := timePerMove + float64(tm.increment)

	// Use a percentage based on game phase - be more aggressive with lots of time
	var usagePercent float64
	timeMultiplier := 1.0
	if tm.baseTime > time.Minute*5 {
		timeMultiplier = 1.8 // Use significantly more time when we have lots available
	} else if tm.baseTime > time.Minute*2 {
		timeMultiplier = 1.4 // Use moderately more time
	}

	switch tm.gamePhase {
	case Opening:
		usagePercent = 0.8 * timeMultiplier // Still less in opening, but not too little
	case Middlegame:
		usagePercent = 1.2 * timeMultiplier // More time in complex middlegame
	case Endgame:
		usagePercent = 1.4 * timeMultiplier // Most time in precise endgame
	}

	return time.Duration(availableTime * usagePercent)
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

// getGamePhaseMultiplier returns time multiplier based on game phase
func (tm *TimeManager) getGamePhaseMultiplier() float64 {
	switch tm.gamePhase {
	case Opening:
		return 0.8 // Spend less time in opening
	case Middlegame:
		return 1.2 // Spend more time in complex middlegame
	case Endgame:
		return 1.0 // Normal time in endgame
	}
	return 1.0
}

// getComplexityMultiplier returns time multiplier based on position complexity
func (tm *TimeManager) getComplexityMultiplier() float64 {
	// Simple complexity heuristic - can be improved later
	if tm.positionComplexity > 0.8 {
		return 1.3 // Spend more time on complex positions
	} else if tm.positionComplexity < 0.3 {
		return 0.8 // Spend less time on simple positions
	}
	return 1.0
}

// getScoreChangeMultiplier returns time multiplier based on score changes
func (tm *TimeManager) getScoreChangeMultiplier() float64 {
	if tm.lastScoreChange < -50 {
		return 1.4 // Score dropped significantly, think more
	} else if tm.lastScoreChange < -20 {
		return 1.2 // Score dropped, think a bit more
	} else if tm.lastScoreChange > 30 {
		return 0.9 // Score improved, can think less
	}
	return 1.0
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

// shouldStopTournamentSearch determines if tournament search should stop
func (tm *TimeManager) shouldStopTournamentSearch(depth int, elapsed time.Duration) bool {
	// Hard limit - never exceed allocated time
	if elapsed >= tm.allocatedTime {
		return true
	}

	// Compute remaining time dynamically for this decision point
	remainingTime := tm.baseTime - elapsed

	// Dynamic depth/time floors: don't stop before we have thought "enough"
	// unless we're in an emergency. Tiers are based on remaining time so
	// they adapt if we spend time during this move.
	// Keep conservative to prevent long searches
	var minDepthFloor int
	var minThinkTime time.Duration
	switch {
	case remainingTime >= 10*time.Minute:
		minDepthFloor = 12             // CRITICAL: Increased from 5 to competitive level
		minThinkTime = 8 * time.Second // Increased from 3s to allow deeper search
	case remainingTime >= 5*time.Minute:
		minDepthFloor = 10             // Increased from 4 to competitive level
		minThinkTime = 6 * time.Second // Increased from 2s to allow deeper search
	case remainingTime >= 2*time.Minute:
		minDepthFloor = 8              // Increased from 4 to competitive level
		minThinkTime = 4 * time.Second // Increased from 1.5s to allow deeper search
	case remainingTime >= 60*time.Second:
		minDepthFloor = 6              // Increased from 3 to competitive level
		minThinkTime = 2 * time.Second // Increased from 1s to allow deeper search
	case remainingTime >= 30*time.Second:
		minDepthFloor = 5              // Increased from 3 to competitive level
		minThinkTime = 1 * time.Second // Increased from 500ms to allow deeper search
	default:
		minDepthFloor = 4                     // Increased from 2 for emergency situations
		minThinkTime = 500 * time.Millisecond // Increased from 200ms for emergency
	}

	// Emergency time check - if we're down to emergency time, stop regardless
	if remainingTime <= tm.emergencyTime {
		return true
	}

	// Respect floors: if we haven't met either floor, keep searching
	if elapsed < minThinkTime || depth < minDepthFloor {
		return false
	}

	// Soft time management - don't start new iteration if likely to exceed time.
	// iterationStartTime must be set via NewIteration at the start of each ID depth;
	// if it was never set we conservatively skip the soft-stop check rather than
	// using an undefined zero time.
	if tm.iterationStartTime.IsZero() {
		return false
	}
	iterationTime := time.Since(tm.iterationStartTime)
	// Estimate time for next iteration (typically 3-4x current iteration)
	nextIterationEstimate := iterationTime * 4

	// Only stop if next iteration would exceed allocated time AND we're running low on total time
	allocatedUsed := float64(elapsed) / float64(tm.allocatedTime)

	// If we've used less than 50% of allocated time, always continue (unless total time very low)
	if allocatedUsed < 0.5 && remainingTime > 10*time.Second {
		return false
	}

	// If we have lots of total time remaining, be very lenient
	if remainingTime > 60*time.Second {
		return elapsed+nextIterationEstimate > tm.allocatedTime*2 // Allow double allocated time
	} else if remainingTime > 30*time.Second {
		return elapsed+nextIterationEstimate > time.Duration(float64(tm.allocatedTime)*1.5) // Allow 50% over
	}
	return elapsed+nextIterationEstimate > tm.allocatedTime // Normal limit when time is tight
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

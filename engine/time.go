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
	timeControl    TimeControl
	allocatedTime  time.Duration // Time allocated for this move
	startTime      time.Time
	emergencyTime  time.Duration // Reserve time for emergencies
	moveOverhead   time.Duration // Network/GUI overhead per move
	
	// Tournament time control parameters
	baseTime       time.Duration // Base time available
	increment      time.Duration // Increment per move
	movesToGo      int          // Moves until next time control
	
	// Game state for time allocation
	gamePhase      GamePhase
	movesPlayed    int
	isEndgame      bool
	
	// Dynamic adjustment factors
	lastScoreChange int
	positionComplexity float64
	
	// Internal rate limiting for frequent time checks
	checkCounter     uint64 // Atomic counter for time checks
	lastTimeCheck    time.Time // Last time we actually checked the clock
	shouldStop       bool   // Cached result of last time check
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
	
	// Global maximum time limit for any move - 30 seconds
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
		timePerMove := float64(tm.baseTime) / movesLeft
		
		// Use 80% of calculated time, save 20% for later moves
		return time.Duration(timePerMove * 0.8)
	}
	
	// No moves to go specified - estimate moves remaining
	estimatedMovesLeft := tm.estimateMovesRemaining()
	timePerMove := float64(tm.baseTime) / float64(estimatedMovesLeft)
	
	// Add increment to available time
	availableTime := timePerMove + float64(tm.increment)
	
	// Use a percentage based on game phase
	var usagePercent float64
	switch tm.gamePhase {
	case Opening:
		usagePercent = 0.6 // Use less time in opening
	case Middlegame:
		usagePercent = 1.0 // Normal time usage
	case Endgame:
		usagePercent = 1.2 // More time in endgame if available
	}
	
	return time.Duration(availableTime * usagePercent)
}

// estimateMovesRemaining estimates how many moves are left in the game
func (tm *TimeManager) estimateMovesRemaining() int {
	switch tm.gamePhase {
	case Opening:
		return 60 - tm.movesPlayed // Games typically 60-80 moves
	case Middlegame:
		return 45 - tm.movesPlayed
	case Endgame:
		return 30 - tm.movesPlayed
	}
	return 40 // Default estimate
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
		// Never use more than 1/3 of remaining time on a single move
		maxAllocation := tm.baseTime / 3
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
func (tm *TimeManager) ShouldStopSearch(depth int, iterationStartTime time.Time) bool {
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
			tm.shouldStop = tm.shouldStopTournamentSearch(depth, iterationStartTime, elapsed)
		}
	}
	
	return tm.shouldStop
}

// shouldStopTournamentSearch determines if tournament search should stop
func (tm *TimeManager) shouldStopTournamentSearch(depth int, iterationStartTime time.Time, elapsed time.Duration) bool {
	// Hard limit - never exceed allocated time
	if elapsed >= tm.allocatedTime {
		return true
	}
	
	// Soft time management - don't start new iteration if likely to exceed time
	if depth >= 4 { // Only apply soft limits after reasonable depth
		iterationTime := time.Since(iterationStartTime)
		
		// Estimate time for next iteration (typically 3-4x current iteration)
		nextIterationEstimate := iterationTime * 4
		
		// If next iteration would exceed our time budget, stop now
		if elapsed+nextIterationEstimate > tm.allocatedTime {
			return true
		}
	}
	
	// Emergency time check - if we're down to emergency time, stop
	remainingTime := tm.baseTime - elapsed
	if remainingTime <= tm.emergencyTime {
		return true
	}
	
	return false
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
	
	complexity := 0.0
	
	// More pieces = more complex
	allPiecesBitboard := position.Board.GetWhitePieces() | position.Board.GetBlackPieces()
	totalPieces := PopCount(allPiecesBitboard)
	complexity += float64(totalPieces) / 32.0 * 0.3
	
	// More piece types = more complex
	pieceTypes := 0
	for _, piece := range []Piece{WhiteQueen, BlackQueen, WhiteRook, BlackRook, 
		WhiteBishop, BlackBishop, WhiteKnight, BlackKnight} {
		if PopCount(position.Board.GetBitboardOf(piece)) > 0 {
			pieceTypes++
		}
	}
	complexity += float64(pieceTypes) / 8.0 * 0.3
	
	// Tactical positions (many pieces attacking each other) are complex
	// This is a simplified heuristic - count pieces that can capture
	// Only calculate if we have pieces on the board to avoid crashes
	if totalPieces > 2 { // Need at least 2 kings + 1 other piece
		captures := len(GenerateLegalMoves(position))
		complexity += math.Min(float64(captures)/50.0, 1.0) * 0.4
	}
	
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
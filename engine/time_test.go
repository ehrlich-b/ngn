package engine

import (
	"testing"
	"time"
)

func TestTimeManagerCreation(t *testing.T) {
	tm := NewTimeManager()

	if tm == nil {
		t.Fatal("TimeManager should not be nil")
	}

	if tm.emergencyTime != 100*time.Millisecond {
		t.Errorf("Expected emergency time 100ms, got %v", tm.emergencyTime)
	}

	if tm.moveOverhead != 50*time.Millisecond {
		t.Errorf("Expected move overhead 50ms, got %v", tm.moveOverhead)
	}
}

func TestFixedDepthTimeControl(t *testing.T) {
	tm := NewTimeManager()
	params := SearchParams{Depth: 6}

	tm.SetTimeControl(params, true)

	if tm.timeControl != FixedDepth {
		t.Errorf("Expected FixedDepth, got %v", tm.timeControl)
	}

	// Should not stop for fixed depth (time doesn't matter)
	if tm.ShouldStopSearch(5, time.Now()) {
		t.Error("Fixed depth search should not stop due to time")
	}
}

func TestInfiniteTimeControl(t *testing.T) {
	tm := NewTimeManager()
	params := SearchParams{Infinite: true}

	tm.SetTimeControl(params, true)

	if tm.timeControl != Infinite {
		t.Errorf("Expected Infinite, got %v", tm.timeControl)
	}

	// Should never stop for infinite search
	if tm.ShouldStopSearch(10, time.Now()) {
		t.Error("Infinite search should never stop due to time")
	}
}

func TestTimePerMoveControl(t *testing.T) {
	tm := NewTimeManager()
	params := SearchParams{MoveTime: 5000} // 5 seconds

	tm.SetTimeControl(params, true)

	if tm.timeControl != TimePerMove {
		t.Errorf("Expected TimePerMove, got %v", tm.timeControl)
	}

	expectedTime := 5000*time.Millisecond - tm.moveOverhead
	if tm.allocatedTime != expectedTime {
		t.Errorf("Expected allocated time %v, got %v", expectedTime, tm.allocatedTime)
	}
}

func TestTournamentTimeControl(t *testing.T) {
	tm := NewTimeManager()
	params := SearchParams{
		WhiteTime: 300000, // 5 minutes
		WhiteInc:  5000,   // 5 second increment
		MovesToGo: 0,      // No moves to go specified
	}

	tm.SetTimeControl(params, true)

	if tm.timeControl != Tournament {
		t.Errorf("Expected Tournament, got %v", tm.timeControl)
	}

	if tm.baseTime != 300*time.Second {
		t.Errorf("Expected base time 300s, got %v", tm.baseTime)
	}

	if tm.increment != 5*time.Second {
		t.Errorf("Expected increment 5s, got %v", tm.increment)
	}
}

func TestBlackTimeControl(t *testing.T) {
	tm := NewTimeManager()
	params := SearchParams{
		WhiteTime: 300000, // 5 minutes
		BlackTime: 250000, // 4 minutes 10 seconds
		WhiteInc:  5000,   // 5 second increment
		BlackInc:  3000,   // 3 second increment
	}

	// Test for black player
	tm.SetTimeControl(params, false)

	if tm.baseTime != 250*time.Second {
		t.Errorf("Expected base time 250s for black, got %v", tm.baseTime)
	}

	if tm.increment != 3*time.Second {
		t.Errorf("Expected increment 3s for black, got %v", tm.increment)
	}
}

func TestMovesToGoTimeControl(t *testing.T) {
	tm := NewTimeManager()
	params := SearchParams{
		WhiteTime: 300000, // 5 minutes
		WhiteInc:  0,      // No increment
		MovesToGo: 30,     // 30 moves to next time control
	}

	tm.SetTimeControl(params, true)

	// Time allocation should be reasonable but consider safety limits
	// Base calculation: 300s / 30 moves = 10s per move * 0.8 = 8s
	// But limited by safety (max 1/3 of total = 100s) and emergency reserve

	// Should allocate less than the naive calculation due to safety features
	naiveTime := time.Duration(float64(300*time.Second) / 30.0 * 0.8) // 8s
	if tm.allocatedTime >= naiveTime {
		t.Errorf("Time allocation should be conservative, got %v which is >= naive %v", tm.allocatedTime, naiveTime)
	}

	// But should still allocate a reasonable amount (at least 3 seconds)
	minTime := 3 * time.Second
	if tm.allocatedTime < minTime {
		t.Errorf("Time allocation too low, got %v, expected at least %v", tm.allocatedTime, minTime)
	}
}

func TestGamePhaseDetection(t *testing.T) {
	tm := NewTimeManager()

	// Test opening position
	pos := newUCIStartingPosition()
	tm.movesPlayed = 5
	phase := tm.determineGamePhase(pos)
	if phase != Opening {
		t.Errorf("Expected Opening phase, got %v", phase)
	}

	// Test endgame position - create a position with minimal material
	pos = &Position{}
	pos.Board = Bitboard{}
	pos.Board.UpdateSquare(E1, WhiteKing, NoPiece)
	pos.Board.UpdateSquare(E8, BlackKing, NoPiece)
	pos.Board.UpdateSquare(A7, WhitePawn, NoPiece)
	pos.Board.UpdateSquare(H7, BlackPawn, NoPiece)

	tm.movesPlayed = 50
	phase = tm.determineGamePhase(pos)
	if phase != Endgame {
		t.Errorf("Expected Endgame phase, got %v", phase)
	}
}

func TestComplexityCalculation(t *testing.T) {
	tm := NewTimeManager()

	// Test starting position complexity
	pos := newUCIStartingPosition()
	complexity := tm.calculatePositionComplexity(pos)

	if complexity < 0.3 || complexity > 0.8 {
		t.Errorf("Starting position complexity should be moderate, got %v", complexity)
	}

	// Test empty position
	pos = &Position{}
	pos.Board = Bitboard{}
	complexity = tm.calculatePositionComplexity(pos)

	if complexity != 0.0 {
		t.Errorf("Empty position should have zero complexity, got %v", complexity)
	}
}

func TestEmergencyTimeDetection(t *testing.T) {
	tm := NewTimeManager()
	params := SearchParams{WhiteTime: 1000} // 1 second

	tm.SetTimeControl(params, true)

	// Initially should not be emergency time
	if tm.IsEmergencyTime() {
		t.Error("Should not be emergency time initially")
	}

	// Simulate time passing
	time.Sleep(500 * time.Millisecond)
	tm.startTime = tm.startTime.Add(-600 * time.Millisecond) // Fake elapsed time

	// Now should be emergency time
	if !tm.IsEmergencyTime() {
		t.Error("Should be emergency time with little time remaining")
	}
}

func TestTimeAllocationSafety(t *testing.T) {
	tm := NewTimeManager()
	params := SearchParams{WhiteTime: 10000} // 10 seconds

	tm.SetTimeControl(params, true)

	// Should never allocate more than 1/3 of total time
	maxAllowed := tm.baseTime / 3
	if tm.allocatedTime > maxAllowed {
		t.Errorf("Allocated time %v exceeds maximum allowed %v", tm.allocatedTime, maxAllowed)
	}

	// Should always keep emergency reserve
	emergencyReserve := tm.emergencyTime
	if tm.allocatedTime > tm.baseTime-emergencyReserve {
		t.Errorf("Allocated time %v doesn't leave emergency reserve %v", tm.allocatedTime, emergencyReserve)
	}
}

func TestTimeControlStringRepresentation(t *testing.T) {
	tm := NewTimeManager()

	testCases := []struct {
		params   SearchParams
		expected string
	}{
		{SearchParams{Depth: 6}, "Fixed Depth"},
		{SearchParams{Infinite: true}, "Infinite"},
		{SearchParams{MoveTime: 5000}, "Time Per Move"},
		{SearchParams{WhiteTime: 300000}, "Tournament"},
	}

	for _, tc := range testCases {
		tm.SetTimeControl(tc.params, true)
		result := tm.GetTimeControlString()
		if result != tc.expected {
			t.Errorf("Expected time control string %s, got %s", tc.expected, result)
		}
	}
}

func TestScoreChangeMultiplier(t *testing.T) {
	tm := NewTimeManager()

	// Test large score drop
	tm.lastScoreChange = -60
	multiplier := tm.getScoreChangeMultiplier()
	if multiplier <= 1.0 {
		t.Error("Should spend more time when score drops significantly")
	}

	// Test score improvement
	tm.lastScoreChange = 40
	multiplier = tm.getScoreChangeMultiplier()
	if multiplier >= 1.0 {
		t.Error("Should spend less time when score improves")
	}

	// Test neutral score change
	tm.lastScoreChange = 5
	multiplier = tm.getScoreChangeMultiplier()
	if multiplier != 1.0 {
		t.Error("Should maintain normal time for neutral score changes")
	}
}

func TestGameStateUpdate(t *testing.T) {
	tm := NewTimeManager()
	pos := newUCIStartingPosition()

	tm.UpdateGameState(10, 50, 30, pos) // Score dropped from 50 to 30

	if tm.movesPlayed != 10 {
		t.Errorf("Expected moves played 10, got %d", tm.movesPlayed)
	}

	if tm.lastScoreChange != -20 {
		t.Errorf("Expected score change -20, got %d", tm.lastScoreChange)
	}

	if tm.gamePhase != Opening {
		t.Errorf("Expected Opening phase, got %v", tm.gamePhase)
	}
}

func BenchmarkTimeAllocation(b *testing.B) {
	tm := NewTimeManager()
	params := SearchParams{WhiteTime: 300000, WhiteInc: 5000}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.SetTimeControl(params, true)
	}
}

func BenchmarkShouldStopSearch(b *testing.B) {
	tm := NewTimeManager()
	params := SearchParams{MoveTime: 5000}
	tm.SetTimeControl(params, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tm.ShouldStopSearch(6, time.Now())
	}
}

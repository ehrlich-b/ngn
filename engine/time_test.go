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
	if tm.ShouldStopSearch(5) {
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
	if tm.ShouldStopSearch(10) {
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

	// Soft target ~ the even share of the bank over the moves to go (300s/30 = 10s):
	// the sustainable per-move budget. Allow a band around it.
	evenShare := tm.baseTime / time.Duration(params.MovesToGo) // 10s
	if tm.softTime < evenShare/2 || tm.softTime > evenShare*6/5 {
		t.Errorf("soft time %v not a sane fraction of the even share %v", tm.softTime, evenShare)
	}

	// Hard ceiling must never exceed 30%% of the bank — the anti-flag guarantee.
	if tm.hardTime > tm.baseTime*3/10 {
		t.Errorf("hard time %v exceeds 30%% of bank %v (flag risk)", tm.hardTime, tm.baseTime)
	}

	// And still allocate a reasonable amount (at least 3 seconds).
	if tm.allocatedTime < 3*time.Second {
		t.Errorf("Time allocation too low, got %v", tm.allocatedTime)
	}
}

func TestTournamentSoftStop(t *testing.T) {
	tm := NewTimeManager()

	// ReportCompletedIteration still TRACKS decision stability (a changed best move or
	// a broken window resets it); the counter is kept for diagnostics even though the
	// soft budget no longer shrinks on it.
	tm.ReportCompletedIteration(true, true) // best move changed
	if tm.stableIters != 0 {
		t.Fatalf("a changed best move must reset stability, got %d", tm.stableIters)
	}
	tm.ReportCompletedIteration(false, false) // window broke (re-search)
	if tm.stableIters != 0 {
		t.Fatalf("a window break must reset stability, got %d", tm.stableIters)
	}
	for i := 0; i < 6; i++ {
		tm.ReportCompletedIteration(false, true) // settled iterations
	}
	if tm.stableIters != 6 {
		t.Fatalf("six settled iterations should give stability 6, got %d", tm.stableIters)
	}

	// The soft budget is NOT shrunk by stability anymore: the old stableIters soft/2
	// (and *0.7) shrink fired on nearly every quiet position, so NGN played whole games
	// at ~25% of its clock. Now a settled and a volatile decision stop at the SAME soft
	// budget; only the ~1x next-iteration projection and the hard ceiling gate the stop.
	tm.timeControl = Tournament
	tm.baseTime = 120 * time.Second
	tm.softTime = 4000 * time.Millisecond
	tm.hardTime = 16000 * time.Millisecond
	tm.lastIterationTime = 500 * time.Millisecond
	const depth = 10

	// elapsed+lastIter = 2700 < soft 4000: keep searching, regardless of stability.
	for _, si := range []int{0, 6, 100} {
		tm.stableIters = si
		if tm.shouldStopTournamentSearch(depth, 2200*time.Millisecond) {
			t.Errorf("stableIters=%d: should keep searching at 2200ms (soft 4000, no stability shrink)", si)
		}
	}

	// elapsed+lastIter = 4200 > soft 4000: the projection stops it, regardless of stability.
	for _, si := range []int{0, 6} {
		tm.stableIters = si
		if !tm.shouldStopTournamentSearch(depth, 3700*time.Millisecond) {
			t.Errorf("stableIters=%d: projection should stop near the soft budget", si)
		}
	}

	// The hard ceiling must NOT move with stability — the anti-flag guarantee.
	tm.stableIters = 100
	if !tm.shouldStopTournamentSearch(depth, tm.hardTime) {
		t.Error("hard ceiling must still stop the search regardless of stability")
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
		tm.ShouldStopSearch(6)
	}
}

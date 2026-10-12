package engine

import (
	"testing"
	"time"
)

func TestTimeHorizonRetainsBankInLateGame(t *testing.T) {
	for _, phase := range []GamePhase{Opening, Middlegame, Endgame} {
		tm := NewTimeManager()
		tm.gamePhase = phase
		tm.movesPlayed = 30
		tm.SetTimeControl(SearchParams{WhiteTime: 120000, WhiteInc: 1000}, true)
		// At 120+1, preserve a later-game bank with a target below six seconds.
		// The old ten-move horizon allocated 12.785 seconds in this case.
		if tm.softTime >= 6*time.Second || tm.softTime < 5*time.Second {
			t.Fatalf("phase %v: late-game soft target %v does not retain the bank", phase, tm.softTime)
		}
		if tm.hardTime > (tm.baseTime-tm.emergencyTime-tm.moveOverhead)*3/10 {
			t.Fatalf("phase %v: hard ceiling exceeds the existing bank cap", phase)
		}
	}
}

func TestTimeHorizonHonorsExplicitMovesToGo(t *testing.T) {
	for _, moves := range []int{1, 3, 10, 40} {
		tm := NewTimeManager()
		tm.gamePhase = Endgame
		tm.movesPlayed = 60
		tm.SetTimeControl(SearchParams{BlackTime: 120000, BlackInc: 1000, MovesToGo: moves}, false)
		usable := tm.baseTime - tm.emergencyTime - tm.moveOverhead
		want := time.Duration(float64(usable)/float64(moves) + 0.8*float64(tm.increment))
		if tm.softTime != want {
			t.Fatalf("moves-to-go %d: target %v, want %v", moves, tm.softTime, want)
		}
	}
}

func TestTimeHorizonPreservesInitialOpeningTarget(t *testing.T) {
	tm := NewTimeManager()
	tm.gamePhase = Opening
	tm.SetTimeControl(SearchParams{WhiteTime: 120000, WhiteInc: 1000}, true)
	if tm.softTime != 4795*time.Millisecond {
		t.Fatalf("initial opening target changed: %v", tm.softTime)
	}
}

func TestTimeHorizonReducedBankRemainsIncrementAware(t *testing.T) {
	tm := NewTimeManager()
	tm.gamePhase = Endgame
	tm.movesPlayed = 60
	tm.SetTimeControl(SearchParams{WhiteTime: 5150, WhiteInc: 1000}, true)
	// Five usable seconds should add 200ms to the unchanged 800ms increment share.
	if tm.softTime != time.Second {
		t.Fatalf("reduced bank target %v, want 1s", tm.softTime)
	}
}

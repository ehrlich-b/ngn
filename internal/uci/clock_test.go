package uci

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTournamentClockLedger(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX engine fixture")
	}
	for _, newWhite := range []bool{true, false} {
		for _, flag := range []bool{false, true} {
			t.Run(fmt.Sprintf("newWhite=%v/flag=%v", newWhite, flag), func(t *testing.T) {
				dir := t.TempDir()
				commands := filepath.Join(dir, "commands")
				t.Setenv("NGN_UCI_CLOCK_LOG", commands)
				engines := make([]*Engine, 2)
				for color, first := range []string{"e2e4", "e7e5"} {
					delay := "0.02"
					if flag && color == 0 {
						delay = "0.4"
					}
					path := filepath.Join(dir, fmt.Sprintf("engine%d", color))
					script := fmt.Sprintf(`#!/bin/sh
i=0
while IFS= read -r line; do
 printf '%%s\n' "$line" >> "$NGN_UCI_CLOCK_LOG"
 case "$line" in
 uci) printf 'uciok\n' ;;
 isready) printf 'readyok\n' ;;
 ucinewgame) i=0 ;;
 'go movetime 200') printf 'bestmove e2e4\n' ;;
 'go wtime '*)
   sleep %s
   if [ "$i" -eq 0 ]; then printf 'bestmove %s\n';
   else printf 'bestmove g1f3\n'; fi
   i=$((i+1)) ;;
 quit) exit 0 ;;
 esac
done
`, delay, first)
					if err := os.WriteFile(path, []byte(script), 0700); err != nil {
						t.Fatal(err)
					}
					engines[color] = StartWithOptions(path, "clock", false, nil)
					if engines[color] == nil {
						t.Fatal("startup")
					}
					defer Stop(engines[color])
				}
				bank := 1000
				if flag {
					bank = 1
				}
				a, b := engines[0], engines[1]
				if !newWhite {
					a, b = b, a
				}
				var clocks []ClockSample
				var moves []string
				result := PlayGame(a, b, []string{"d2d4", "d7d5"}, newWhite, GameConfig{
					TCTimeMs: bank, TCIncMs: 100, MaxMoves: 5, MovesOut: &moves, ClockOut: &clocks,
				})
				wantN, wantReason := 3, "max-moves"
				if flag {
					wantN, wantReason = 1, "time-forfeit"
					want := Win
					if newWhite {
						want = Loss
					}
					if result.Res != want {
						t.Fatalf("flag POV: %+v", result)
					}
				}
				if len(clocks) != wantN || result.Reason != wantReason {
					t.Fatalf("result %+v; ledger %+v", result, clocks)
				}
				logged, err := os.ReadFile(commands)
				if err != nil {
					t.Fatal(err)
				}
				whiteMs, blackMs := bank, bank
				for i, row := range clocks {
					if row.Ply != i+3 || row.White != (i%2 == 0) || row.ElapsedMs < 1 ||
						row.WhiteBeforeMs != whiteMs || row.BlackBeforeMs != blackMs {
						t.Fatalf("opening/bank continuity: %+v", row)
					}
					wantGo := fmt.Sprintf("go wtime %d btime %d winc 100 binc 100", whiteMs, blackMs)
					if row.Go != wantGo || !strings.Contains(string(logged), wantGo+"\n") {
						t.Fatalf("actual command mismatch: %+v; %s", row, logged)
					}
					if row.White {
						whiteMs -= row.ElapsedMs
						whiteMs += row.IncrementAppliedMs
					} else {
						blackMs -= row.ElapsedMs
						blackMs += row.IncrementAppliedMs
					}
					if row.WhiteAfterMs != whiteMs || row.BlackAfterMs != blackMs || row.Flagged != flag {
						t.Fatalf("bank conservation: %+v", row)
					}
					if flag {
						if row.IncrementAppliedMs != 0 || row.WhiteAfterMs >= -300 {
							t.Fatalf("increment after flag: %+v", row)
						}
					} else if row.IncrementAppliedMs != 100 || row.Move != moves[row.Ply-1] {
						t.Fatalf("move/increment: %+v", row)
					}
				}
			})
		}
	}
}

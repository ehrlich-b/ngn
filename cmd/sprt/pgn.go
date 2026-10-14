package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ehrlich-b/ngn/engine"
	"github.com/ehrlich-b/ngn/internal/uci"
)

// The collector is the sole writer. Replay happens after the clocked game.
func writeGamePGN(out io.Writer, game gameOutcome, cfg runConfig) error {
	fen := uci.StartFEN
	if len(game.game.opening) == 1 && strings.Contains(game.game.opening[0], "/") {
		fen = game.game.opening[0]
	}
	pos, err := engine.ParseFEN(fen)
	if err != nil {
		return err
	}
	white, black := "candidate", "base"
	if !game.game.newIsWhite {
		white, black = black, white
	}
	result := "1/2-1/2"
	if game.res.Res != uci.Draw {
		whiteWins := (game.res.Res == uci.Win) == game.game.newIsWhite
		result = "0-1"
		if whiteWins {
			result = "1-0"
		}
	}
	var text strings.Builder
	for _, tag := range [][2]string{
		{"Event", "NGN campaign"}, {"Site", "rig"}, {"Date", time.Now().Format("2006.01.02")},
		{"Round", fmt.Sprintf("%d.%d", game.pairID+1, 1+btoi(!game.game.newIsWhite))},
		{"White", white}, {"Black", black}, {"Result", result}, {"Termination", game.res.Reason},
		{"TimeControl", fmt.Sprintf("%g+%g", float64(cfg.tcTimeMs)/1000, float64(cfg.tcIncMs)/1000)},
	} {
		fmt.Fprintf(&text, "[%s %q]\n", tag[0], tag[1])
	}
	if fen != uci.StartFEN {
		fmt.Fprintf(&text, "[SetUp \"1\"]\n[FEN %q]\n", fen)
	}
	text.WriteByte('\n')
	fields := strings.Fields(fen)
	moveNumber := 1
	fmt.Sscan(fields[5], &moveNumber)
	for i, u := range game.moves {
		move, err := engine.ParseUCIMove(pos, u)
		if err != nil {
			return fmt.Errorf("pair%d move%d %s: %w", game.pairID, i, u, err)
		}
		whiteTurn := pos.Turn() == engine.White
		if whiteTurn {
			fmt.Fprintf(&text, "%d. ", moveNumber)
		} else if i == 0 {
			fmt.Fprintf(&text, "%d... ", moveNumber)
		}
		text.WriteString(engine.MoveToSAN(move, pos))
		text.WriteByte(' ')
		if _, _, _, ok := pos.GameMakeMove(move); !ok {
			return fmt.Errorf("PGN replay rejected %s", u)
		}
		if !whiteTurn {
			moveNumber++
		}
	}
	fmt.Fprintf(&text, "%s\n\n", result)
	_, err = io.WriteString(out, text.String())
	return err
}

func btoi(value bool) int {
	if value {
		return 1
	}
	return 0
}

package engine

import (
	"reflect"
	"strings"
	"testing"
)

// Reflect ranks and swap colors: the material exchange must be unchanged.
func seeGeometryReflection(fen, uci string) (string, string) {
	parts := strings.Fields(fen)
	ranks := strings.Split(parts[0], "/")
	for i, j := 0, len(ranks)-1; i < j; i, j = i+1, j-1 {
		ranks[i], ranks[j] = ranks[j], ranks[i]
	}
	parts[0] = strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		if r >= 'a' && r <= 'z' {
			return r - ('a' - 'A')
		}
		return r
	}, strings.Join(ranks, "/"))
	parts[1] = "b"
	if parts[3] != "-" {
		parts[3] = string([]byte{parts[3][0], '9' - parts[3][1] + '0'})
	}
	move := []byte(uci)
	move[1], move[3] = '9'-move[1]+'0', '9'-move[3]+'0'
	return strings.Join(parts, " "), string(move)
}

func TestSEEPostMoveGeometry(t *testing.T) {
	cases := []struct {
		name, fen, uci string
		want           int
	}{
		{"source reveals rook", "7k/8/8/3p4/3R4/3r4/8/K7 w - - 0 1", "d4d5", -400},
		{"quiet source reveals rook", "7k/8/8/8/3R4/3r4/8/K7 w - - 0 1", "d4d5", -500},
		{"source reveals bishop", "7k/8/8/4p3/3B4/2b5/K7/8 w - - 0 1", "d4e5", -240},
		{"EP ignores attack on captured square", "7k/8/4p3/3pP3/8/8/8/K7 w - d6 0 1", "e5d6", 100},
		{"EP destination defended by pawn", "7k/2p5/8/3pP3/8/8/8/K7 w - d6 0 1", "e5d6", 0},
		{"EP removal reveals rook", "7k/8/8/3pP3/8/3r4/8/K7 w - d6 0 1", "e5d6", 0},
	}
	for _, c := range cases {
		for _, reflected := range []bool{false, true} {
			fen, uci := c.fen, c.uci
			name := c.name + "/white"
			if reflected {
				fen, uci = seeGeometryReflection(fen, uci)
				name = c.name + "/black"
			}
			t.Run(name, func(t *testing.T) {
				pos := n3cPosition(t, fen)
				move := adapterLegalMove(t, pos, uci)
				before := snapshotPVPosition(pos)
				got := staticExchangeEvaluation(pos, move)
				t.Logf("SEE(%s)=%d, want %d; %s", uci, got, c.want, fen)
				if got != c.want {
					t.Errorf("SEE=%d, want %d", got, c.want)
				}
				if !reflect.DeepEqual(before, snapshotPVPosition(pos)) {
					t.Fatal("SEE changed the position")
				}
			})
		}
	}
}

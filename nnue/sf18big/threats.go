package sf18big

import (
	"errors"
	"fmt"
	"slices"

	base "github.com/ehrlich-b/ngn/nnue"
)

const maxActiveThreats = 128

var ErrEvaluation = errors.New("invalid Stockfish 18 BIG evaluation input")

// ThreatIndexList is the allocation-free logical-order FullThreats feature set
// for one perspective. Only Indices[:Count] is live.
type ThreatIndexList struct {
	Count   uint8
	Indices [maxActiveThreats]uint32
}

// ThreatDiff is the simple sorted-set reference transition used by the N1b
// feasibility gate. Production fused/double-update machinery is deliberately
// outside this contract.
type ThreatDiff struct {
	Removed         ThreatIndexList
	Added           ThreatIndexList
	RequiresRefresh bool
}

type boardPiece struct {
	piece base.PieceType
	color base.Color
	set   bool
}

var threatTargetMap = [6][6]int8{
	{0, 1, -1, 2, -1, -1},
	{0, 1, 2, 3, 4, 5},
	{0, 1, 2, 3, -1, 4},
	{0, 1, 2, 3, -1, 4},
	{0, 1, 2, 3, 4, 5},
	{0, 1, 2, 3, -1, -1},
}

var threatValidTargets = [6]int{6, 12, 10, 10, 12, 8}

type threatPieceOffsets struct {
	perTarget int
	base      int
	from      [64]int
}

var threatOffsets = buildThreatOffsets()

func buildThreatOffsets() [2][6]threatPieceOffsets {
	var result [2][6]threatPieceOffsets
	baseOffset := 0
	for color := base.White; color <= base.Black; color++ {
		for piece := base.Pawn; piece <= base.King; piece++ {
			perTarget := 0
			var fromOffsets [64]int
			for square := base.Square(0); square < 64; square++ {
				fromOffsets[square] = perTarget
				if piece != base.Pawn || square >= 8 && square < 56 {
					perTarget += popcount64(pseudoThreatAttacks(piece, color, square))
				}
			}
			result[color][piece] = threatPieceOffsets{
				perTarget: perTarget,
				base:      baseOffset,
				from:      fromOffsets,
			}
			baseOffset += threatValidTargets[piece] * perTarget
		}
	}
	if baseOffset != threatInputFeatures {
		panic(fmt.Sprintf("sf18big: FullThreats dimensions %d, expected %d", baseOffset, threatInputFeatures))
	}
	return result
}

// ActiveThreats returns Stockfish 18 FullThreats indices for one perspective.
// The returned live prefix is sorted to make before/after transition diffs
// explicit and deterministic.
func ActiveThreats(position base.Position, perspective base.Color) (ThreatIndexList, error) {
	board, kings, occupied, _, err := validatePosition(position)
	if err != nil {
		return ThreatIndexList{}, err
	}
	if perspective > base.Black {
		return ThreatIndexList{}, fmt.Errorf("%w: perspective %d", ErrEvaluation, perspective)
	}

	var active ThreatIndexList
	for relativeColor := base.White; relativeColor <= base.Black; relativeColor++ {
		color := perspective ^ relativeColor
		for piece := base.Pawn; piece <= base.King; piece++ {
			for from, attacker := range board {
				if !attacker.set || attacker.color != color || attacker.piece != piece {
					continue
				}
				attacks := occupiedThreatAttacks(piece, color, base.Square(from), occupied)
				for attacks != 0 {
					to := base.Square(trailingZeros64(attacks))
					attacks &= attacks - 1
					index, ok := threatIndex(perspective, attacker, base.Square(from), board[to], to, kings[perspective])
					if !ok {
						continue
					}
					if int(active.Count) >= len(active.Indices) {
						return ThreatIndexList{}, fmt.Errorf("%w: more than %d active threats", ErrEvaluation, maxActiveThreats)
					}
					active.Indices[active.Count] = uint32(index)
					active.Count++
				}
			}
		}
	}
	slices.Sort(active.Indices[:active.Count])
	return active, nil
}

// DiffThreats computes a reference sorted-set transition. It accepts two
// validated positions rather than a move-specific dirty-threat encoding, which
// keeps this feasibility oracle independent of Stockfish's update machinery.
func DiffThreats(before, after base.Position, perspective base.Color) (ThreatDiff, error) {
	_, beforeKings, _, _, err := validatePosition(before)
	if err != nil {
		return ThreatDiff{}, err
	}
	_, afterKings, _, _, err := validatePosition(after)
	if err != nil {
		return ThreatDiff{}, err
	}
	beforeActive, err := ActiveThreats(before, perspective)
	if err != nil {
		return ThreatDiff{}, err
	}
	afterActive, err := ActiveThreats(after, perspective)
	if err != nil {
		return ThreatDiff{}, err
	}

	diff := ThreatDiff{
		RequiresRefresh: beforeKings[perspective]&4 != afterKings[perspective]&4,
	}
	diff.Removed, diff.Added = diffThreatIndexLists(beforeActive, afterActive)
	return diff, nil
}

func diffThreatIndexLists(beforeActive, afterActive ThreatIndexList) (ThreatIndexList, ThreatIndexList) {
	var removed ThreatIndexList
	var added ThreatIndexList
	i, j := 0, 0
	for i < int(beforeActive.Count) || j < int(afterActive.Count) {
		switch {
		case i == int(beforeActive.Count):
			added.Indices[added.Count] = afterActive.Indices[j]
			added.Count++
			j++
		case j == int(afterActive.Count):
			removed.Indices[removed.Count] = beforeActive.Indices[i]
			removed.Count++
			i++
		case beforeActive.Indices[i] < afterActive.Indices[j]:
			removed.Indices[removed.Count] = beforeActive.Indices[i]
			removed.Count++
			i++
		case afterActive.Indices[j] < beforeActive.Indices[i]:
			added.Indices[added.Count] = afterActive.Indices[j]
			added.Count++
			j++
		default:
			i++
			j++
		}
	}
	return removed, added
}

func threatIndex(
	perspective base.Color,
	attacker boardPiece,
	from base.Square,
	attacked boardPiece,
	to base.Square,
	king base.Square,
) (int, bool) {
	orientation := base.Square(56 * perspective)
	if king&7 >= 4 {
		orientation ^= 7
	}
	fromOriented := from ^ orientation
	toOriented := to ^ orientation
	attackerColor := attacker.color ^ perspective
	attackedColor := attacked.color ^ perspective

	group := int(threatTargetMap[attacker.piece][attacked.piece])
	if group < 0 {
		return 0, false
	}
	enemy := attackerColor != attackedColor
	semiExcluded := attacker.piece == attacked.piece && (enemy || attacker.piece != base.Pawn)
	if semiExcluded && fromOriented < toOriented {
		return 0, false
	}

	offsets := threatOffsets[attackerColor][attacker.piece]
	group += int(attackedColor) * (threatValidTargets[attacker.piece] / 2)
	attacks := pseudoThreatAttacks(attacker.piece, attackerColor, fromOriented)
	toBit := uint64(1) << toOriented
	if attacks&toBit == 0 {
		return 0, false
	}
	rank := popcount64(attacks & (toBit - 1))
	index := offsets.base + group*offsets.perTarget + offsets.from[fromOriented] + rank
	if index < 0 || index >= threatInputFeatures {
		return 0, false
	}
	return index, true
}

func pseudoThreatAttacks(piece base.PieceType, color base.Color, from base.Square) uint64 {
	switch piece {
	case base.Pawn:
		return pawnThreatAttacks(color, from)
	case base.Knight:
		return leaperThreatAttacks(from, [][2]int{{1, 2}, {2, 1}, {2, -1}, {1, -2}, {-1, -2}, {-2, -1}, {-2, 1}, {-1, 2}})
	case base.Bishop:
		return slidingThreatAttacks(from, 0, [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}})
	case base.Rook:
		return slidingThreatAttacks(from, 0, [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}})
	case base.Queen:
		return slidingThreatAttacks(from, 0, [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}})
	case base.King:
		return leaperThreatAttacks(from, [][2]int{{1, 1}, {1, 0}, {1, -1}, {0, 1}, {0, -1}, {-1, 1}, {-1, 0}, {-1, -1}})
	default:
		return 0
	}
}

func occupiedThreatAttacks(piece base.PieceType, color base.Color, from base.Square, occupied uint64) uint64 {
	switch piece {
	case base.Bishop:
		return slidingThreatAttacks(from, occupied, [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}}) & occupied
	case base.Rook:
		return slidingThreatAttacks(from, occupied, [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}) & occupied
	case base.Queen:
		return slidingThreatAttacks(from, occupied, [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}}) & occupied
	default:
		return pseudoThreatAttacks(piece, color, from) & occupied
	}
}

func pawnThreatAttacks(color base.Color, from base.Square) uint64 {
	file, rank := int(from&7), int(from>>3)
	direction := 1
	if color == base.Black {
		direction = -1
	}
	toRank := rank + direction
	if toRank < 0 || toRank >= 8 {
		return 0
	}
	var attacks uint64
	for _, fileDelta := range []int{-1, 1} {
		toFile := file + fileDelta
		if toFile >= 0 && toFile < 8 {
			attacks |= uint64(1) << (toRank*8 + toFile)
		}
	}
	return attacks
}

func leaperThreatAttacks(from base.Square, deltas [][2]int) uint64 {
	file, rank := int(from&7), int(from>>3)
	var attacks uint64
	for _, delta := range deltas {
		toFile, toRank := file+delta[0], rank+delta[1]
		if toFile >= 0 && toFile < 8 && toRank >= 0 && toRank < 8 {
			attacks |= uint64(1) << (toRank*8 + toFile)
		}
	}
	return attacks
}

func slidingThreatAttacks(from base.Square, occupied uint64, directions [][2]int) uint64 {
	file, rank := int(from&7), int(from>>3)
	var attacks uint64
	for _, direction := range directions {
		for distance := 1; ; distance++ {
			toFile, toRank := file+direction[0]*distance, rank+direction[1]*distance
			if toFile < 0 || toFile >= 8 || toRank < 0 || toRank >= 8 {
				break
			}
			bit := uint64(1) << (toRank*8 + toFile)
			attacks |= bit
			if occupied&bit != 0 {
				break
			}
		}
	}
	return attacks
}

func validatePosition(position base.Position) ([64]boardPiece, [2]base.Square, uint64, int, error) {
	var board [64]boardPiece
	var kings [2]base.Square
	var kingCount [2]uint8
	if position.SideToMove > base.Black {
		return board, kings, 0, 0, fmt.Errorf("%w: side to move %d", ErrEvaluation, position.SideToMove)
	}
	if len(position.Pieces) > 32 {
		return board, kings, 0, 0, fmt.Errorf("%w: %d occupied squares, maximum 32", ErrEvaluation, len(position.Pieces))
	}
	var occupied uint64
	for _, piece := range position.Pieces {
		if piece.Color > base.Black || piece.Piece > base.King || piece.Square >= 64 {
			return board, kings, 0, 0, fmt.Errorf("%w: invalid piece %+v", ErrEvaluation, piece)
		}
		if board[piece.Square].set {
			return board, kings, 0, 0, fmt.Errorf("%w: duplicate square %d", ErrEvaluation, piece.Square)
		}
		board[piece.Square] = boardPiece{piece: piece.Piece, color: piece.Color, set: true}
		occupied |= uint64(1) << piece.Square
		if piece.Piece == base.King {
			kingCount[piece.Color]++
			kings[piece.Color] = piece.Square
		}
	}
	for color := base.White; color <= base.Black; color++ {
		if kingCount[color] != 1 {
			return board, kings, 0, 0, fmt.Errorf("%w: colour %d has %d kings", ErrEvaluation, color, kingCount[color])
		}
	}
	return board, kings, occupied, len(position.Pieces), nil
}

func popcount64(value uint64) int {
	value = value - ((value >> 1) & 0x5555555555555555)
	value = (value & 0x3333333333333333) + ((value >> 2) & 0x3333333333333333)
	return int((((value + (value >> 4)) & 0x0F0F0F0F0F0F0F0F) * 0x0101010101010101) >> 56)
}

func trailingZeros64(value uint64) int {
	if value == 0 {
		return 64
	}
	return popcount64((value & -value) - 1)
}

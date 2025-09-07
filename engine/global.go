package engine

const MaximumDepth = 100

// Precomputed attack tables
var KnightAttacks [64]uint64
var KingAttacks [64]uint64
var RankMasks [8]uint64
var FileMasks [8]uint64

func init() {
	initKnightAttacks()
	initKingAttacks()
	initRankMasks()
	initFileMasks()
}

// initKnightAttacks precomputes knight attack bitboards for all squares
func initKnightAttacks() {
	knightMoves := []int{-17, -15, -10, -6, 6, 10, 15, 17}

	for sq := 0; sq < 64; sq++ {
		attacks := uint64(0)
		file := sq % 8
		rank := sq / 8

		for _, move := range knightMoves {
			to := sq + move
			if to < 0 || to > 63 {
				continue
			}

			toFile := to % 8
			toRank := to / 8

			// Check if move is valid (not wrapping around board)
			fileDiff := abs(file - toFile)
			rankDiff := abs(rank - toRank)

			if (fileDiff == 2 && rankDiff == 1) || (fileDiff == 1 && rankDiff == 2) {
				attacks |= SquareMask[to]
			}
		}

		KnightAttacks[sq] = attacks
	}
}

// initKingAttacks precomputes king attack bitboards for all squares
func initKingAttacks() {
	kingMoves := []int{-9, -8, -7, -1, 1, 7, 8, 9}

	for sq := 0; sq < 64; sq++ {
		attacks := uint64(0)
		file := sq % 8

		for _, move := range kingMoves {
			to := sq + move
			if to < 0 || to > 63 {
				continue
			}

			toFile := to % 8

			// Check if move wraps around the board horizontally
			if abs(file-toFile) > 1 {
				continue
			}

			attacks |= SquareMask[to]
		}

		KingAttacks[sq] = attacks
	}
}

// initRankMasks precomputes rank bitboards
func initRankMasks() {
	for rank := 0; rank < 8; rank++ {
		mask := uint64(0)
		for file := 0; file < 8; file++ {
			sq := rank*8 + file
			mask |= SquareMask[sq]
		}
		RankMasks[rank] = mask
	}
}

// initFileMasks precomputes file bitboards
func initFileMasks() {
	for file := 0; file < 8; file++ {
		mask := uint64(0)
		for rank := 0; rank < 8; rank++ {
			sq := rank*8 + file
			mask |= SquareMask[sq]
		}
		FileMasks[file] = mask
	}
}

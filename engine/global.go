package engine

const MaximumDepth = 100

// Precomputed attack tables
var KnightAttacks [64]uint64
var KingAttacks [64]uint64
var RankMasks [8]uint64
var FileMasks [8]uint64

// Ray masks for sliding piece attacks - [square][direction]
// Directions: 0=N, 1=NE, 2=E, 3=SE, 4=S, 5=SW, 6=W, 7=NW  
var RayMasks [64][8]uint64

func init() {
	initKnightAttacks()
	initKingAttacks()
	initRankMasks()
	initFileMasks()
	initRayMasks()
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

// initRayMasks precomputes ray bitboards for sliding attacks
func initRayMasks() {
	// Direction offsets: N, NE, E, SE, S, SW, W, NW
	directions := []int{-8, -7, 1, 9, 8, 7, -1, -9}
	
	for sq := 0; sq < 64; sq++ {
		file := sq % 8
		rank := sq / 8
		
		for dir := 0; dir < 8; dir++ {
			mask := uint64(0)
			offset := directions[dir]
			
			for i := 1; i < 8; i++ {
				to := sq + offset*i
				
				if to < 0 || to > 63 {
					break
				}
				
				toFile := to % 8
				toRank := to / 8
				
				// Check for board wrap-around
				fileDiff := abs(file - toFile)
				rankDiff := abs(rank - toRank)
				
				// For diagonal moves, file and rank differences must be equal
				if dir == 1 || dir == 3 || dir == 5 || dir == 7 { // diagonal
					if fileDiff != rankDiff {
						break
					}
				}
				// For horizontal moves, rank must stay same
				if dir == 2 || dir == 6 { // horizontal
					if rankDiff != 0 || fileDiff != i {
						break
					}
				}
				// For vertical moves, file must stay same  
				if dir == 0 || dir == 4 { // vertical
					if fileDiff != 0 {
						break
					}
				}
				
				mask |= SquareMask[to]
			}
			
			RayMasks[sq][dir] = mask
		}
	}
}

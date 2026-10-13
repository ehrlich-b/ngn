package engine

// orderSMPHelperRootMoves preserves every ordering score and every tactical
// slot. Only quiet moves of exactly equal score rotate among their slots; the
// existing root loop still hoists its last completed best move afterward.
func (h *workerHistory) orderSMPHelperRootMoves(moves []Move, out []Move, pos *Position, worker int) int {
	var scores [256]int
	n, _ := h.scoreMovesIntoBuffer(moves, 0, h.GetLastMovePlayed(), EmptyMove, EmptyMove, false, out, scores[:], pos, false)
	for i := 0; i < n-1; i++ {
		selectNextMove(out, scores[:], n, i)
	}
	if worker <= 0 {
		return n
	}
	for first := 0; first < n; {
		end := first + 1
		for end < n && scores[end] == scores[first] {
			end++
		}
		var slots [256]int
		var quiets [256]Move
		count := 0
		for i := first; i < end; i++ {
			m := out[i]
			if !m.IsCapture() && !m.IsEnPassant() && m.PromoType() == NoType && !m.IsCastle() {
				slots[count], quiets[count] = i, m
				count++
			}
		}
		if count > 1 {
			shift := worker % count
			for i := 0; i < count; i++ {
				out[slots[i]] = quiets[(i+shift)%count]
			}
		}
		first = end
	}
	return n
}

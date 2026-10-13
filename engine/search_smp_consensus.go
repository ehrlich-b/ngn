package engine

// selectSMPConsensus considers only the last fully completed root decision of
// each joined worker. SearchInfo.Depth/BestMove/BestScore are committed together
// after a completed aspiration search; an interrupted iteration changes none of
// them. EmptyMove therefore also excludes workers that completed no iteration.
// Counters, clock ownership and worker-private search state stay with primary.
func selectSMPConsensus(pos *Position, primary *SearchInfo, helpers []*SearchInfo) *SearchInfo {
	if primary == nil || primary.Depth <= 0 || primary.BestMove == EmptyMove {
		return primary
	}
	// IsLegalMove assumes a generated pseudo-legal input. Validate the actual
	// packed root vote against generated legal moves, including its geometry.
	legalRoot := GenerateLegalMoves(pos)
	isLegal := func(move Move) bool {
		for _, legal := range legalRoot {
			if move == legal {
				return true
			}
		}
		return false
	}
	if !isLegal(primary.BestMove) {
		return primary
	}
	// Keep an already found winning mate. Other score classes vote only on the
	// move; numerical scores from selective searches are never averaged.
	if smpMateClass(primary.BestScore) > 0 {
		return primary
	}
	type group struct {
		move           Move
		count          int
		representative *SearchInfo
		worker         int
		mateClass      int
		mixedMateClass bool
	}
	groups := []group{{move: primary.BestMove, count: 1, representative: primary, worker: 0, mateClass: smpMateClass(primary.BestScore)}}
	for worker, info := range helpers {
		if info == nil || info.Depth < primary.Depth || info.BestMove == EmptyMove || !isLegal(info.BestMove) {
			continue
		}
		index := -1
		for i := range groups {
			if groups[i].move == info.BestMove {
				index = i
				break
			}
		}
		if index < 0 {
			groups = append(groups, group{move: info.BestMove, representative: info, worker: worker + 1, mateClass: smpMateClass(info.BestScore)})
			index = len(groups) - 1
		}
		g := &groups[index]
		g.count++
		g.mixedMateClass = g.mixedMateClass || g.mateClass != smpMateClass(info.BestScore)
		if info.Depth > g.representative.Depth || (info.Depth == g.representative.Depth && worker+1 < g.worker) {
			g.representative, g.worker = info, worker+1
		}
	}
	winner, tied := 0, false
	for i := 1; i < len(groups); i++ {
		if groups[i].count > groups[winner].count {
			winner, tied = i, false
		} else if groups[i].count == groups[winner].count {
			tied = true
		}
	}
	g := &groups[winner]
	if winner == 0 || tied || g.count < 2 || g.count <= groups[0].count || g.mixedMateClass {
		return primary
	}
	// Never replace an ordinary score with a forced loss. A majority escaping
	// the primary's losing mate is eligible; unanimous winning-mate votes are
	// eligible when the primary did not already find a winning mate.
	if g.mateClass < 0 && smpMateClass(primary.BestScore) >= 0 {
		return primary
	}
	return g.representative
}

func smpMateClass(score int) int {
	if score >= MATE_VALUE-MaximumDepth {
		return 1
	}
	if score <= -MATE_VALUE+MaximumDepth {
		return -1
	}
	return 0
}

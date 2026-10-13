package engine

// nextSMPHelperDepth preserves depths1–4, then keeps a helper's parity schedule
// for the whole search. A final clamped depth always reaches the requested cap;
// maxDepth+1 terminates the loop. Primary and every third helper keep all depths.
func nextSMPHelperDepth(current, maxDepth, worker int) int {
	if worker <= 0 || current < 4 {
		return current + 1
	}
	if current >= maxDepth {
		return maxDepth + 1
	}
	next := current + 1
	switch (worker - 1) % 3 {
	case 0:
		if next%2 != 0 {
			next++
		}
	case 1:
		if next%2 == 0 {
			next++
		}
	}
	if next > maxDepth {
		next = maxDepth
	}
	return next
}

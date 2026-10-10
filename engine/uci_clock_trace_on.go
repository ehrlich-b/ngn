//go:build ngnclocktrace

package engine

import (
	"encoding/json"
	"fmt"
	"io"
)

func startClockTrace(session *uciSearchSession) {
	session.timeManager.setSearchTerminationObserver(new(searchTerminationObserver))
}

// This diagnostic build runs the ordinary UCI/search path. It reports the
// existing observer after search finishes, before bestmove, without changing
// admission, active stops, root discard, history, the horizon or the net.
func completeClockTrace(session *uciSearchSession, info *SearchInfo, output io.Writer) {
	tm := session.timeManager
	o := tm.observer
	if o == nil {
		return
	}
	elapsed := tm.GetElapsedTime().Nanoseconds()
	last := int64(0)
	for _, iteration := range o.Iterations[:o.IterationCount] {
		if iteration.CompletedAtElapsed.Nanoseconds() > last {
			last = iteration.CompletedAtElapsed.Nanoseconds()
		}
	}
	tail := elapsed - last
	if tail < 0 {
		tail = 0
	}
	record := map[string]any{
		"session_id": session.id, "wall_ns": elapsed, "tail_ns": tail,
		"last_completed_ns": last, "first_reason": o.FirstReason.String(),
		"first_depth": o.FirstDepth, "first_elapsed_ns": o.FirstElapsed.Nanoseconds(),
		"iterations": o.Iterations[:o.IterationCount], "iteration_overflow": o.IterationOverflow,
		"actual_nodes": info.Nodes, "completed_depth": info.Depth, "stopped": info.Stopped,
	}
	bytes, err := json.Marshal(record)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(output, "info string ngn_clock %s\n", bytes)
}

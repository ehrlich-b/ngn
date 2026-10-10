//go:build ngnclocktrace

package engine

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestUCIClockTraceAccountsForUnfinishedTail(t *testing.T) {
	start := time.Unix(321, 0)
	clock := &terminationObserverClock{now: start}
	tm := newTimeManager(clock.Now)
	session := &uciSearchSession{id: 7, timeManager: tm}
	startClockTrace(session)
	tm.SetTimeControlAt(SearchParams{WhiteTime: 10000, WhiteInc: 100}, true, start)
	tm.NewIteration()
	clock.Advance(50 * time.Millisecond)
	tm.observeCompletedIteration(4, 1)
	clock.Advance(30 * time.Millisecond)
	tm.observeControlTermination(searchTerminationSoftAbort, 5)
	var out bytes.Buffer
	completeClockTrace(session, &SearchInfo{Nodes: 50000, Depth: 4, Stopped: true}, &out)
	var j struct {
		Wall   int64  `json:"wall_ns"`
		Tail   int64  `json:"tail_ns"`
		Reason string `json:"first_reason"`
		Nodes  int64  `json:"actual_nodes"`
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(out.String(), "info string ngn_clock ")), &j); err != nil {
		t.Fatal(err)
	}
	if j.Wall != int64(80*time.Millisecond) || j.Tail != int64(30*time.Millisecond) || j.Reason != "soft-abort" || j.Nodes != 50000 {
		t.Fatal(j)
	}
}

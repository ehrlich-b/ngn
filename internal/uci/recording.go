package uci

import (
	"bufio"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// TraceFunc records timestamped UCI traffic, stderr, and process exits. It may
// be called concurrently by the input, output and stderr readers.
type TraceFunc func(kind, text string)

type recordedEngine struct {
	trace  TraceFunc
	lines  chan string
	drain  chan struct{}
	exited chan struct{}
	stop   sync.Once
}

type traceWriter struct{ trace TraceFunc }

func (w traceWriter) Write(p []byte) (int, error) {
	w.trace("stderr", string(p))
	return len(p), nil
}

// StartRecorded starts exactly one diagnostic process. Unlike the normal
// tournament constructor, a failed attempt is retained and never retried.
// All stdout is read immediately, including lines emitted during shutdown.
func StartRecorded(path string, options []string, trace TraceFunc) *Engine {
	if trace == nil {
		panic("nil diagnostic recorder")
	}
	cmd := exec.Command(path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		trace("start-error", err.Error())
		return nil
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		trace("start-error", err.Error())
		return nil
	}
	cmd.Stderr = traceWriter{trace}
	if err = cmd.Start(); err != nil {
		trace("start-error", err.Error())
		return nil
	}
	r := &recordedEngine{trace: trace, lines: make(chan string, 1024), drain: make(chan struct{}), exited: make(chan struct{})}
	e := &Engine{cmd: cmd, stdin: bufio.NewWriter(stdin), recorded: r}
	trace("start", fmt.Sprintf("pid=%d path=%s", cmd.Process.Pid, path))
	outputDone := make(chan struct{})
	go func() {
		defer close(outputDone)
		defer close(r.lines)
		s := bufio.NewScanner(stdout)
		s.Buffer(make([]byte, 64*1024), 1024*1024)
		for s.Scan() {
			line := s.Text()
			trace("stdout", line)
			select {
			case r.lines <- line:
			case <-r.drain:
			}
		}
		if err := s.Err(); err != nil {
			trace("stdout-error", err.Error())
		}
	}()
	go func() {
		<-outputDone
		err := cmd.Wait() // stdout has drained; Wait also joins the stderr reader.
		if err != nil {
			trace("exit", err.Error())
		} else {
			trace("exit", "0")
		}
		close(r.exited)
	}()
	Send(e, "uci")
	if !WaitFor(e, "uciok", 15*time.Second) {
		Stop(e)
		return nil
	}
	for _, option := range options {
		Send(e, option)
	}
	Send(e, "isready")
	if !WaitFor(e, "readyok", 15*time.Second) {
		Stop(e)
		return nil
	}
	Send(e, "position startpos")
	if GetMove(e, "go movetime 200", 10*time.Second, 1500*time.Millisecond) == "" {
		Stop(e)
		return nil
	}
	return e
}

func nextLine(e *Engine, deadline time.Time) (string, bool) {
	if e.recorded == nil {
		if e.stdout.Scan() {
			return e.stdout.Text(), true
		}
		return "", false
	}
	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	select {
	case line, ok := <-e.recorded.lines:
		return line, ok
	case <-timer.C:
		return "", false
	}
}

func stopRecorded(e *Engine) {
	r := e.recorded
	r.stop.Do(func() {
		Send(e, "quit")
		close(r.drain)
		timer := time.NewTimer(2 * time.Second)
		defer timer.Stop()
		select {
		case <-r.exited:
		case <-timer.C:
			r.trace("forced-kill", "quit deadline exceeded")
			e.cmd.Process.Kill()
			<-r.exited
		}
	})
}

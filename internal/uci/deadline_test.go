package uci

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestReadDeadlinesWithSilentOutput(t *testing.T) {
	for _, mode := range []string{"ready", "analyze", "move"} {
		t.Run(mode, func(t *testing.T) {
			r, w := io.Pipe()
			defer w.Close()
			defer r.Close()
			var commands bytes.Buffer
			e := &Engine{stdin: bufio.NewWriter(&commands), stdout: bufio.NewScanner(r)}
			start := time.Now()
			switch mode {
			case "ready":
				if WaitFor(e, "readyok", 50*time.Millisecond) {
					t.Fatal("silent ready accepted")
				}
			case "analyze":
				if got := Analyze(e, "go depth 1", 50*time.Millisecond); len(got) != 0 {
					t.Fatal(got)
				}
			case "move":
				if got := GetMove(e, "go movetime 1", 50*time.Millisecond, 10*time.Millisecond); got != "" {
					t.Fatal(got)
				}
				if commands.String() != "go movetime 1\nstop\n" {
					t.Fatalf("missing timed stop: %q", commands.String())
				}
			}
			if elapsed := time.Since(start); elapsed < 40*time.Millisecond || elapsed > time.Second {
				t.Fatalf("deadline elapsed %s", elapsed)
			}
			r.Close()
			<-e.readerDone
		})
	}
}

func TestEOFReturnsWithoutWaitingForDeadline(t *testing.T) {
	e := &Engine{stdin: bufio.NewWriter(io.Discard), stdout: bufio.NewScanner(strings.NewReader(""))}
	start := time.Now()
	if WaitFor(e, "readyok", time.Minute) || GetMove(e, "go depth 1", time.Minute, 0) != "" || len(Analyze(e, "go depth 1", time.Minute)) != 0 {
		t.Fatal("EOF accepted")
	}
	if time.Since(start) > time.Second {
		t.Fatal("EOF waited for deadline")
	}
}

func fakeProcess(t *testing.T, tail string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixture")
	}
	p := filepath.Join(t.TempDir(), "fake-uci")
	script := `#!/bin/sh
while IFS= read -r line; do
 case "$line" in
  uci) printf 'uciok\n' ;;
  isready) printf 'readyok\n' ;;
  'go movetime 200') printf 'bestmove e2e4\n' ;;
` + tail + `
 esac
done
`
	if err := os.WriteFile(p, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestStopReapsChildAndRecordsAbnormalExit(t *testing.T) {
	p := fakeProcess(t, "'go depth 1') printf 'fixture failed\\n' >&2; exit 7 ;;\nquit) exit 0 ;;")
	e := Start(p, "exit fixture", false)
	if e == nil {
		t.Fatal("fixture startup failed")
	}
	if GetMove(e, "go depth 1", time.Second, 0) != "" {
		t.Fatal("dead child returned move")
	}
	Stop(e)
	Stop(e)
	d := e.Diagnostics()
	if !d.Exited || d.ExitError != "exit status 7" || !strings.Contains(d.Stderr, "fixture failed") {
		t.Fatalf("diagnostics %+v", d)
	}
	if e.cmd.ProcessState == nil || !e.cmd.ProcessState.Exited() {
		t.Fatal("child not reaped")
	}
}

func TestStopKillsChildThatIgnoresQuit(t *testing.T) {
	e := Start(fakeProcess(t, "quit) : ;;"), "ignore quit", false)
	if e == nil {
		t.Fatal("fixture startup failed")
	}
	start := time.Now()
	Stop(e)
	if time.Since(start) > time.Second {
		t.Fatal("cleanup hung")
	}
	if !e.Diagnostics().Exited || e.cmd.ProcessState == nil {
		t.Fatal("killed child not reaped")
	}
}

func TestStartupRetryRemainsAnOperationalFault(t *testing.T) {
	p := fakeProcess(t, "quit) exit 0 ;;")
	t.Setenv("NGN_UCI_FIRST_START", filepath.Join(t.TempDir(), "started"))
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	prefix := `#!/bin/sh
if [ ! -e "$NGN_UCI_FIRST_START" ]; then
 : > "$NGN_UCI_FIRST_START"
 printf 'first startup failed\n' >&2
 exit 7
fi
`
	if err := os.WriteFile(p, []byte(prefix+strings.TrimPrefix(string(data), "#!/bin/sh\n")), 0700); err != nil {
		t.Fatal(err)
	}
	e := Start(p, "retry fixture", false)
	if e == nil {
		t.Fatal("second startup failed")
	}
	defer Stop(e)
	if e.Diagnostics().StartupFailures != 1 {
		t.Fatalf("retry hidden: %+v", e.Diagnostics())
	}
}

func TestResetRejectsDeadEngine(t *testing.T) {
	e := &Engine{stdin: bufio.NewWriter(io.Discard), stdout: bufio.NewScanner(strings.NewReader(""))}
	if got := PlayGame(e, scriptedEngine(), []string{"7k/6Q1/6K1/8/8/8/8/8 b - - 0 1"}, true, GameConfig{MaxMoves: 1}); got.Reason != "engine-error" {
		t.Fatalf("dead engine scored terminal position: %+v", got)
	}
}

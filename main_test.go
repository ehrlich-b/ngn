package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunDefaultHCEIgnoresAdjacentModelsAndLocksEvaluator(t *testing.T) {
	var output, errorOutput bytes.Buffer
	input := "uci\neval\nsetoption name EvalFile value ngn.nnue\nsetoption name EvalBackend value ngn-v1\nucinewgame\neval\nquit\n"
	if exit := run(nil, strings.NewReader(input), &output, &errorOutput); exit != 0 || errorOutput.Len() != 0 {
		t.Fatalf("HCE startup exit=%d stderr=%q", exit, errorOutput.String())
	}
	for _, option := range []string{"option name EvalBackend", "option name EvalFile", "option name K4EvalScale"} {
		if strings.Contains(output.String(), option) {
			t.Fatalf("HCE startup advertised evaluator switching: %s", output.String())
		}
	}
	if strings.Count(output.String(), "info string eval backend hce ") != 2 {
		t.Fatalf("GUI options or new game changed HCE: %s", output.String())
	}
	if !strings.Contains(output.String(), "option name OwnBook type check default false\n") {
		t.Fatalf("default book was enabled: %s", output.String())
	}
}

func TestRunHCERejectsExplicitExternalEvaluatorBeforeUCI(t *testing.T) {
	for _, args := range [][]string{
		{"-eval-backend", "ngn-v1"},
		{"-eval-backend", "counter-5.5", "-eval-file", "missing.nn"},
		{"-eval-backend", "rodent-v1.2-default"},
		{"-eval-backend", "ngn-k4-768-v1"},
		{"-eval-file", "ngn.nnue"},
	} {
		var output, errorOutput bytes.Buffer
		if exit := run(args, strings.NewReader("uci\nquit\n"), &output, &errorOutput); exit != 2 || output.Len() != 0 ||
			!strings.Contains(errorOutput.String(), "HCE build requires") {
			t.Fatalf("args=%q exit=%d stdout=%q stderr=%q", args, exit, output.String(), errorOutput.String())
		}
	}
}

func TestRunHCEVersionAndExplicitBook(t *testing.T) {
	var output, errorOutput bytes.Buffer
	if exit := run([]string{"-version"}, strings.NewReader(""), &output, &errorOutput); exit != 0 ||
		output.String() != releaseVersion+"\n" || errorOutput.Len() != 0 {
		t.Fatalf("version exit=%d stdout=%q stderr=%q", exit, output.String(), errorOutput.String())
	}
	output.Reset()
	if exit := run([]string{"-own-book=true", "-eval-backend", "HCE", "-eval-file", "<empty>"},
		strings.NewReader("uci\nquit\n"), &output, &errorOutput); exit != 0 || errorOutput.Len() != 0 ||
		!strings.Contains(output.String(), "option name OwnBook type check default true\n") {
		t.Fatalf("explicit HCE exit=%d stdout=%q stderr=%q", exit, output.String(), errorOutput.String())
	}
}

func TestRunRetiredOwnedProfileFailsClosed(t *testing.T) {
	profile := releaseProfile
	t.Cleanup(func() { releaseProfile = profile })
	releaseProfile = "owned"
	var output, errorOutput bytes.Buffer
	if exit := run(nil, strings.NewReader("uci\nquit\n"), &output, &errorOutput); exit != 2 || output.Len() != 0 ||
		!strings.Contains(errorOutput.String(), "retired build profile") {
		t.Fatalf("retired profile exit=%d stdout=%q stderr=%q", exit, output.String(), errorOutput.String())
	}
}

func TestRunResearchRetainsRemainingExplicitBackends(t *testing.T) {
	profile := releaseProfile
	t.Cleanup(func() { releaseProfile = profile })
	releaseProfile = "research"
	var output, errorOutput bytes.Buffer
	if exit := run(nil, strings.NewReader("uci\nquit\n"), &output, &errorOutput); exit != 0 || errorOutput.Len() != 0 {
		t.Fatalf("research exit=%d stderr=%q", exit, errorOutput.String())
	}
	if !strings.Contains(output.String(), "option name EvalBackend type combo default hce var hce var ngn-v1 var sf18-big var counter-5.5\n") {
		t.Fatalf("remaining research options: %s", output.String())
	}
	for _, args := range [][]string{{"-eval-backend", "counter-5.5"}, {"-eval-backend", "rodent-v1.1-anand"},
		{"-eval-backend", "ngn-k4-768-v1"}, {"-eval-backend", "hce", "-eval-file", "missing.nn"}, {"stray"}} {
		output.Reset()
		errorOutput.Reset()
		if exit := run(args, strings.NewReader("uci\nquit\n"), &output, &errorOutput); exit != 2 || output.Len() != 0 {
			t.Fatalf("research rejection args=%q exit=%d stdout=%q stderr=%q", args, exit, output.String(), errorOutput.String())
		}
	}
}

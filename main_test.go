package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunDefaultHCEAdvertisesActualDefaults(t *testing.T) {
	var output, errorOutput bytes.Buffer
	if exit := run(nil, strings.NewReader("uci\nquit\n"), &output, &errorOutput); exit != 0 {
		t.Fatalf("run exit=%d stderr=%s", exit, errorOutput.String())
	}
	if errorOutput.Len() != 0 {
		t.Fatalf("default startup stderr=%q", errorOutput.String())
	}
	for _, want := range []string{
		"option name EvalBackend type combo default hce var hce var ngn-v1 var ngn-k4-768-v1 var sf18-big var counter-5.5 var rodent-v1.1-anand var rodent-v1.2-default\n",
		"option name EvalFile type string default <empty>\n",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("default UCI output missing %q:\n%s", want, output.String())
		}
	}
}

func TestRunInvalidExplicitEvaluatorFailsBeforeUCI(t *testing.T) {
	for _, test := range []struct {
		name, wantPrefix string
		args             []string
	}{
		{"missing model", "ngn: evaluator startup configuration failed: ", []string{"-eval-backend", "counter-5.5"}},
		{"unsupported backend", "ngn: evaluator startup configuration failed: ", []string{"-eval-backend", "stockfish"}},
		{"file with hce", "ngn: evaluator startup configuration failed: ", []string{"-eval-backend", "hce", "-eval-file", "model.nn"}},
		{"positional before explicit request", "ngn: unexpected positional arguments: ", []string{"stray", "-eval-backend", "counter-5.5"}},
		{"scale below range", "ngn: K4 score scale configuration failed: ", []string{"-k4-eval-scale", "9"}},
		{"scale above range", "ngn: K4 score scale configuration failed: ", []string{"-k4-eval-scale", "401"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output, errorOutput bytes.Buffer
			exit := run(test.args, strings.NewReader("uci\nquit\n"), &output, &errorOutput)
			if exit != 2 {
				t.Fatalf("run exit=%d, want 2", exit)
			}
			if output.Len() != 0 {
				t.Fatalf("failed startup reached UCI: %q", output.String())
			}
			if !strings.HasPrefix(errorOutput.String(), test.wantPrefix) {
				t.Fatalf("startup stderr=%q", errorOutput.String())
			}
		})
	}
}

func TestRunExplicitStartupScaleAndBookDefaults(t *testing.T) {
	var output, errorOutput bytes.Buffer
	input := "uci\nsetoption name K4EvalScale value 100\nsetoption name OwnBook value true\nuci\nquit\n"
	args := []string{"-k4-eval-scale", "60", "-own-book=false"}
	if exit := run(args, strings.NewReader(input), &output, &errorOutput); exit != 0 {
		t.Fatalf("run exit=%d stderr=%s", exit, errorOutput.String())
	}
	for _, want := range []string{
		"option name K4EvalScale type spin default 60 min 10 max 400\n",
		"option name OwnBook type check default false\n",
	} {
		if strings.Count(output.String(), want) != 2 {
			t.Fatalf("startup default %q changed after a runtime option:\n%s", want, output.String())
		}
	}
}

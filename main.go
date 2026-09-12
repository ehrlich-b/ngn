package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/ehrlich-b/ngn/engine"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, input io.Reader, output, errorOutput io.Writer) int {
	flags := flag.NewFlagSet("ngn", flag.ContinueOnError)
	flags.SetOutput(errorOutput)
	allowResignation := flags.Bool("allow-resignation", false, "Allow engine to resign in hopeless positions")
	evalBackend := flags.String("eval-backend", engine.EvaluatorBackendHCEName,
		"Evaluator backend: hce, ngn-v1, or counter-5.5 (non-HCE requires -eval-file)")
	evalFile := flags.String("eval-file", "", "Evaluator model path for a non-HCE backend")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(errorOutput, "ngn: unexpected positional arguments: %q\n", flags.Args())
		return 2
	}

	uciEngine := engine.NewUCIEngine()
	if err := uciEngine.ConfigureStartupEvaluator(*evalBackend, *evalFile); err != nil {
		fmt.Fprintf(errorOutput, "ngn: evaluator startup configuration failed: %v\n", err)
		return 2
	}
	uciEngine.SetResignationPolicy(*allowResignation)
	uciEngine.Run(input, output)
	return 0
}

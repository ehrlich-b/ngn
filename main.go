package main

import (
	"flag"
	"os"
	
	"github.com/ehrlich-b/ngn/engine"
)

func main() {
	// Parse command line flags
	allowResignation := flag.Bool("allow-resignation", false, "Allow engine to resign in hopeless positions")
	flag.Parse()
	
	uciEngine := engine.NewUCIEngine()
	uciEngine.SetResignationPolicy(*allowResignation)
	uciEngine.Run(os.Stdin, os.Stdout)
}

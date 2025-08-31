package main

import (
	"os"
	
	"github.com/ehrlich-b/ngn/engine"
)

func main() {
	uciEngine := engine.NewUCIEngine()
	uciEngine.Run(os.Stdin, os.Stdout)
}

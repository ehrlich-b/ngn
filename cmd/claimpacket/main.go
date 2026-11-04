// claimpacket verifies retained confirmation evidence offline. It has no game
// runner or prospective launch mode and never executes an engine or GPU probe.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ehrlich-b/ngn/internal/claim"
)

func run() error {
	receipt := flag.String("receipt", "", "retained confirmation receipt to verify with its admitted screen")
	openings := flag.String("positions", "", "UCI opening histories to canonicalize without engine search")
	flag.Parse()
	if (*receipt == "") == (*openings == "") {
		return fmt.Errorf("choose --receipt or --positions")
	}
	if *openings != "" {
		b, err := os.ReadFile(*openings)
		if err != nil {
			return err
		}
		keys, err := claim.OpeningPositionKeys(strings.Split(strings.TrimSpace(string(b)), "\n"))
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(keys)
	}
	a, err := claim.VerifyConfirmationInputs(*receipt)
	if err != nil {
		json.NewEncoder(os.Stdout).Encode(map[string]any{"status":"NO_GO", "error":err.Error(), "screen":a})
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "verified historical inputs; no games started; no prospective launch admission", "screen": a})
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

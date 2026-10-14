package main

import (
	"fmt"
	"strings"
)

// Fixed options select the network and game configuration; SPSA still sets its
// complete perturbed search vector before each pair.
type optionFlags []string

func (o *optionFlags) String() string { return strings.Join(*o, "; ") }

func (o *optionFlags) Set(value string) error {
	name, setting, ok := strings.Cut(value, "=")
	name = strings.TrimSpace(name)
	if !ok || name == "" || strings.ContainsAny(value, "\r\n") || strings.Contains(name, " value ") {
		return fmt.Errorf("UCI option must be NAME=VALUE on one line")
	}
	*o = append(*o, "setoption name "+name+" value "+setting)
	return nil
}

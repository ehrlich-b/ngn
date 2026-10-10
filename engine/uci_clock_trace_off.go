//go:build !ngnclocktrace

package engine

import "io"

// Ordinary builds inline these empty hooks; only the dedicated diagnostic build
// attaches an observer or serializes any clock information.
func startClockTrace(*uciSearchSession)                            {}
func completeClockTrace(*uciSearchSession, *SearchInfo, io.Writer) {}

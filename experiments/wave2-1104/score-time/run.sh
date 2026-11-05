#!/bin/sh
set -eu
# Execute from the repository root; optional -v retains the fake-clock trace.
exec python3 experiments/wave2-1104/score-time/proof.py go test -p 1 -count=1 ./engine -run '^TestWave2ScoreTime' -timeout 15s "$@"

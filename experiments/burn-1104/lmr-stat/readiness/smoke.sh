#!/usr/bin/env bash
set -euo pipefail
GOAMD64=v3 go build -p 2 -trimpath -o build/ngn .
go build -p 2 -o build/sprt ./cmd/sprt
nice -n 10 build/sprt -lowpower=false -new build/ngn -newoption T1104LMRFractional=1 -base build/ngn -nodes 1000 -openings experiments/burn-1104/lmr-stat/smoke-openings.txt -concurrency 1 -elo0 0 -elo1 5 -mingames 2 -maxgames 2 -maxmoves 20 -pgn ../evidence/replay-smoke.pgn | tee ../evidence/replay-smoke.txt
python3 ../acceptance.py smoke ../evidence/replay-smoke.txt ../evidence/replay-smoke.pgn
python3 ../acceptance.py artifacts

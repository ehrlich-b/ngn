#!/usr/bin/env bash
set -euo pipefail
baseline=$(mktemp -d "${TMPDIR:?}/lmr-baseline.XXXXXXXX")
git archive 6d3168b8c5f15532cc7a352d05465a77ab50f361 | tar -x -C "$baseline"
mkdir -p build
(cd "$baseline" && go build -p 2 -o "$OLDPWD/build/sp-base" ./cmd/speedprobe)
go build -p 2 -o build/sp-new ./cmd/speedprobe
nice -n 10 python3 scripts/speedcheck.py build/sp-base build/sp-new --net engine/nets/ngn-h512-81215229.nnue --cpu 3 --depth 10 --identity-only --out ../evidence/replay-identity.json
go test -p 2 -short ./engine -run LMRFractional

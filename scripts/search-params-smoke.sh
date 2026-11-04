#!/usr/bin/env bash
# Rebuild the pinned baseline and current tree, then check default identity and
# exercise the game harness. The reduced smoke is not the performance gate.
set -euo pipefail
scratch=$(mktemp -d "${TMPDIR:?}/search-params.XXXXXXXX")
mkdir "$scratch/base"
git archive faaf3741eaf0e2add133ffe6574a3c24db415907 | tar -x -C "$scratch/base"
(cd "$scratch/base" && go build -p 2 -o "$scratch/base-probe" ./cmd/speedprobe && go build -p 2 -o "$scratch/base-engine" .)
go build -p 2 -o "$scratch/new-probe" ./cmd/speedprobe
go build -p 2 -o "$scratch/new-engine" .
python3 scripts/speedcheck.py "$scratch/base-probe" "$scratch/new-probe" --net engine/nets/ngn-h512-81215229.nnue --cpu 3 --depth 6 --duration 1 --repeats 1 --identity-only --out "$scratch/identity.json"
SMOKE_ENGINE_A="$scratch/new-engine" SMOKE_ENGINE_B="$scratch/base-engine" go test -p 2 -count=1 -v -run '^TestFixedNodeSmoke$' ./internal/uci

#!/usr/bin/env bash
set -euo pipefail
S=/work/lanes/ngn-s54/spsa-final
export PATH=/home/ehrlich/.local/ngn-go1255/bin:$PATH
export GOCACHE=/work/cache/go/ngn-build GOMODCACHE=/work/cache/go/ngn-mod
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
cd "$S"
finish() { local rc=$?; echo "$rc" > checks.exit; date -Is > checks.done; }
trap finish EXIT
date -Is > checks.started
mkdir source
tar xf /work/lanes/ngn-s53/tt-refresh/base.tar -C source
cp final-live.go.txt source/engine/spsa_final_live_test.go
(cd source && gofmt -w engine/spsa_final_live_test.go && NGN_SPSA_FINAL_VECTOR="$S/vector.json" timeout 10m go test -p 2 -count=1 -run '^TestCampaignSPSAFinalLiveVector$' -v ./engine) > handler.log 2>&1
timeout 10m python3 final-live.py "$S" > probe.log 2>&1

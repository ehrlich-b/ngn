#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session91/threads8
cd "$D/source"
export PATH=/usr/local/go/bin:$PATH
export GOTOOLCHAIN=local GOPROXY=off GOAMD64=v3 GOMAXPROCS=8
export GOCACHE=$R/session17f/go-cache
export NGN_CAMPAIGN_NET=$R/session82/net-final115/candidate.nnue
cp "$D/current-net-test.go.txt" engine/campaign_threads8_test.go
cp "$D/uci-test.go.txt" engine/campaign_threads8_uci_test.go
gofmt -w engine/campaign_threads8_test.go engine/campaign_threads8_uci_test.go
go test -p 1 -short -race ./engine -run '^TestCampaignThreads8' -count=1 -v > "$D/threads8-race.log" 2>&1
go test -p 1 -short -race ./... -count=1 > "$D/full-race8.log" 2>&1
python3 "$D/uci-smoke.py" > "$D/uci-smoke.log" 2>&1

#!/usr/bin/env bash
set -euo pipefail
s=/home/ehrli/ngn-data/night-20261007/session64-final14
export PATH=/usr/local/go/bin:$PATH GOMAXPROCS=2 GOAMD64=v3 GOTOOLCHAIN=local GOPROXY=off
cd "$s"
finish() { local rc=$?; echo "$rc" > "$s/checks.exit"; date -Is > "$s/checks.done"; }
trap finish EXIT
date -Is > checks.started
cd candidate
gofmt -w engine/search.go
timeout 10m go test -p 2 -short ./engine -count=1 > ../engine-short.log 2>&1
timeout 10m go test -p 2 -short -race ./engine -count=1 > ../engine-race.log 2>&1
timeout 10m go test -p 2 -short ./... -count=1 > ../full-short.log 2>&1
timeout 10m go test -p 2 -short -race ./... -count=1 > ../full-race.log 2>&1
timeout 10m go vet -p 2 ./... > ../vet.log 2>&1
timeout 10m go build -p 2 -pgo="$s/candidate/default.pgo" -o "$s/candidate-ngn" . > ../candidate-build.log 2>&1
cd ../baseline
timeout 10m go build -p 2 -pgo="$s/baseline/default.pgo" -o "$s/baseline-ngn" . > ../baseline-build.log 2>&1
cd "$s"
timeout 10m python3 default-parity.py > default-parity.log 2>&1
sha256sum baseline-ngn candidate-ngn candidate/engine/search.go > build.sha256

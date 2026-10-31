#!/usr/bin/env bash
set -euo pipefail
D=/home/ehrli/ngn-data/night-20261007/session79-admission-see
cd "$D/source"
finish(){ rc=$?; echo "$rc" > "$D/checks.exit"; date -Is > "$D/checks.done"; }
trap finish EXIT
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOTOOLCHAIN=local GOPROXY=off GOMAXPROCS=2
export GOCACHE=/home/ehrli/ngn-data/night-20261007/session17f/go-cache
go version > "$D/environment.txt"
go test -p 1 -short ./engine -run '^TestIterationAdmission' -count=1 -v > "$D/focused.log" 2>&1
go test -p 1 -short ./engine -count=1 > "$D/engine-short.log" 2>&1
go test -p 1 -short -race ./engine -count=1 > "$D/engine-race.log" 2>&1
go test -p 1 -short ./... -count=1 > "$D/full-short.log" 2>&1
go test -p 1 -short -race ./... -count=1 > "$D/full-race.log" 2>&1
go vet -p 1 ./... > "$D/vet.log" 2>&1
go build -p 1 -pgo=default.pgo -o "$D/candidate-ngn" . > "$D/build.log" 2>&1
sha256sum engine/time.go engine/time_admission_test.go "$D/candidate-ngn" > "$D/candidate-inputs.sha256"

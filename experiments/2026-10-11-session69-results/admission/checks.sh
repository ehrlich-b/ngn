#!/usr/bin/env bash
set -euo pipefail
s=$1
taskset -pc 4-11 $$ > "$s/controller-affinity.txt"
cd "$s"
st() { printf '{"utc":"%s","stage":"session69-admission-final14","event":"%s","detail":"%s"}\n' "$(date -u +%FT%TZ)" "$1" "$2" >> "$s/status.jsonl"; }
finish() { local rc=$?; echo "$rc" > "$s/checks.exit"; date -Is > "$s/checks.done"; st terminal "exit $rc; no games launched"; }
trap finish EXIT
date -Is > checks.started
export PATH=/home/ehrlich/.local/ngn-go1255/bin:$PATH
export GOCACHE=/work/cache/go/ngn-build GOMODCACHE=/work/cache/go/mod
export GOMAXPROCS=2 GOAMD64=v3 GOTOOLCHAIN=local GOPROXY=off
go version > environment.txt
st start 'existing admission alone on f1d0a03/final14; taskset4-11/3GiB/swap0/nice10'
cd source
gofmt -w engine/time.go engine/time_admission_test.go
sha256sum engine/time.go engine/time_admission_test.go default.pgo > ../source-inputs.sha256
timeout 10m go test -p 2 -short ./engine -count=1 -run '^TestIterationAdmission' -v > ../narrow.log 2>&1
timeout 10m go test -p 2 -short ./engine -count=1 > ../engine-short.log 2>&1
timeout 10m go test -p 2 -short -race ./engine -count=1 > ../engine-race.log 2>&1
timeout 10m go test -p 2 -short ./... -count=1 > ../full-short.log 2>&1
timeout 10m go test -p 2 -short -race ./... -count=1 > ../full-race.log 2>&1
timeout 10m go vet -p 2 ./... > ../vet.log 2>&1
timeout 10m go build -p 2 -pgo=default.pgo -o ../candidate-ngn . > ../build.log 2>&1
cd "$s"
sha256sum candidate-ngn > binary.sha256
st checked 'focused/engine/full short/race/vet/PGO; no strength verdict'

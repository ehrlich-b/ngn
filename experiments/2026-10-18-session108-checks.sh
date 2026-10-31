#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session108-speed
export PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local GOPROXY=off GOAMD64=v3 GOMAXPROCS=2 GOCACHE=$R/session17f/go-cache
finish() { local rc=$?; echo "$rc" > "$D/checks.exit"; date -Is > "$D/checks.done"; }
trap finish EXIT
echo $$ > "$D/checks.pid"
mkdir "$D/candidate"
tar xf "$R/session107/base.tar" -C "$D/candidate"
cd "$D/candidate"
git apply "$D/candidate.patch"
gofmt -w engine/search.go
go test -p 1 -short ./engine -count=1 > "$D/engine-short.log" 2>&1
go test -p 1 -short -race ./engine -count=1 > "$D/engine-race.log" 2>&1
go test -p 1 -short ./... -count=1 > "$D/full-short.log" 2>&1
go test -p 1 -short -race ./... -count=1 > "$D/full-race.log" 2>&1
go vet -p 1 ./... > "$D/vet.log" 2>&1
go build -p 1 -pgo="$R/session100-speed/baseline/default.pgo" -o "$D/s4-ngn" . > "$D/build.log" 2>&1
go build -p 1 -pgo="$R/session100-speed/baseline/default.pgo" -o "$D/s4-probe" ./cmd/speedprobe > "$D/probe-build.log" 2>&1
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -p 1 -pgo="$R/session100-speed/baseline/default.pgo" -o "$D/s4-windows.exe" . > "$D/windows-build.log" 2>&1
python3 "$D/identity.py" > "$D/identity.log" 2>&1

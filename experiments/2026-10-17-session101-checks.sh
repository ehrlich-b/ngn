#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session101-speed
export PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local GOPROXY=off GOAMD64=v3 GOMAXPROCS=2 GOCACHE=$R/session17f/go-cache
finish() { local rc=$?; echo "$rc" > "$S/checks.exit"; date -Is > "$S/checks.done"; }
trap finish EXIT
cd "$S/source"
gofmt -w engine/moveorder.go engine/move_selection_test.go
go test -p 2 -short ./engine -run TestSelectNextMoveIdentity -count=1 -v > "$S/focused.log" 2>&1
go test -p 2 -short ./engine -count=1 > "$S/engine-short.log" 2>&1
go test -p 2 -short -race ./engine -count=1 > "$S/engine-race.log" 2>&1
go test -p 2 -short ./... -count=1 > "$S/full-short.log" 2>&1
go test -p 2 -short -race ./... -count=1 > "$S/full-race.log" 2>&1
go vet -p 2 ./... > "$S/vet.log" 2>&1
go build -p 2 -pgo="$R/session100-speed/baseline/default.pgo" -o "$S/s3-ngn" . > "$S/build.log" 2>&1
go build -p 2 -pgo="$R/session100-speed/baseline/default.pgo" -o "$S/s3-probe" ./cmd/speedprobe > "$S/probe-build.log" 2>&1
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -p 2 -pgo="$R/session100-speed/baseline/default.pgo" -o "$S/s3-windows.exe" . > "$S/windows-build.log" 2>&1
go build -p 2 -pgo="$R/session100-speed/baseline/default.pgo" -gcflags='github.com/ehrlich-b/ngn/engine=-S' -o "$S/s3-assembly-ngn" . > "$S/assembly-full.log" 2>&1
python3 "$S/identity.py" > "$S/identity.log" 2>&1

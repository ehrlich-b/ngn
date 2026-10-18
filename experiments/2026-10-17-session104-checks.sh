#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session104-speed
export PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local GOPROXY=off GOAMD64=v3 GOMAXPROCS=2 GOCACHE=$R/session17f/go-cache
finish() { local rc=$?; echo "$rc" > "$D/checks.exit"; date -Is > "$D/checks.done"; }
trap finish EXIT
for kind in s2 s1; do
    mkdir "$D/$kind"
    tar xf "$D/base.tar" -C "$D/$kind"
    cd "$D/$kind"
    git apply "$D/$kind.patch"
    go test -p 2 -short ./engine -count=1 > "$D/$kind-engine-short.log" 2>&1
    go test -p 2 -short -race ./engine -count=1 > "$D/$kind-engine-race.log" 2>&1
    go test -p 2 -short ./... -count=1 > "$D/$kind-full-short.log" 2>&1
    go test -p 2 -short -race ./... -count=1 > "$D/$kind-full-race.log" 2>&1
    go vet -p 2 ./... > "$D/$kind-vet.log" 2>&1
    go build -p 2 -pgo="$R/session100-speed/baseline/default.pgo" -o "$D/$kind-ngn" . > "$D/$kind-build.log" 2>&1
    go build -p 2 -pgo="$R/session100-speed/baseline/default.pgo" -o "$D/$kind-probe" ./cmd/speedprobe > "$D/$kind-probe-build.log" 2>&1
    GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -p 2 -pgo="$R/session100-speed/baseline/default.pgo" -o "$D/$kind-windows.exe" . > "$D/$kind-windows-build.log" 2>&1
done
python3 "$D/identity.py" > "$D/identity.log" 2>&1

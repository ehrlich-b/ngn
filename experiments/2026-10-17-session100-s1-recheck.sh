#!/usr/bin/env bash
set -euo pipefail
S=/home/ehrli/ngn-data/night-20261007/session100-speed
export PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local GOPROXY=off GOAMD64=v3 GOMAXPROCS=2 GOCACHE=/home/ehrli/ngn-data/night-20261007/session17f/go-cache
cd "$S"
finish() { local rc=$?; echo "$rc" > s1-recheck2.exit; date -Is > s1-recheck2.done; }
trap finish EXIT
mkdir s1-prefault
tar xf base.tar -C s1-prefault
tar xf s1.tar -C s1-prefault
cp cache_alloc_linux-prefault.go s1-prefault/engine/cache_alloc_linux.go
cd s1-prefault
gofmt -w engine/cache_alloc_linux.go engine/cache_alloc_other.go engine/cache_alloc_linux_test.go
go test -p 2 -short -run 'Test(TTAllocation|ClusterTT|IndependentCache|TT)' ./engine -count=1 -v > "$S/s1-prefault-focused.log" 2>&1
go test -p 2 -short ./engine -count=1 > "$S/s1-prefault-engine-short.log" 2>&1
go test -p 2 -short -race ./engine -count=1 > "$S/s1-prefault-engine-race.log" 2>&1
go test -p 2 -short ./... -count=1 > "$S/s1-prefault-full-short.log" 2>&1
go test -p 2 -short -race ./... -count=1 > "$S/s1-prefault-full-race.log" 2>&1
go vet -p 2 ./... > "$S/s1-prefault-vet.log" 2>&1
go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/s1-prefault-ngn" . > "$S/s1-prefault-build.log" 2>&1
go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/s1-prefault-probe" ./cmd/speedprobe > "$S/s1-prefault-probe-build.log" 2>&1
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/s1-prefault-windows.exe" . > "$S/s1-prefault-windows-build.log" 2>&1
cd "$S"
python3 s1-prefault-identity.py > s1-prefault-identity.log 2>&1

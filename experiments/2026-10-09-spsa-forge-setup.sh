#!/usr/bin/env bash
# Run inside Bryan's CPUs8-11/5G scope. This builds the frozen legacy source.
set -euo pipefail
R=/work/lanes/ngn-spsa-20261009
export PATH=/home/ehrlich/.local/ngn-go1255/bin:$PATH
export GOCACHE=/work/cache/go/ngn-build GOMODCACHE=/work/cache/go/ngn-mod
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local CGO_ENABLED=0
cd "$R"
finish() { local rc=$?; echo "$rc" > setup.exit; date -Is > setup.done; }
trap finish EXIT
date -Is > setup.started
go version > go-version.txt
(cd spsa/src && CGO_ENABLED=1 go test -p 2 -short -race -timeout 3m ./cmd/spsa ./internal/uci -count=1) > tool-tests.log 2>&1
(cd spsa/src && go build -p 2 -trimpath -pgo=off -o "$R/spsa-forge" ./cmd/spsa) > tuner-build.log 2>&1
(cd kb8bin/src && go build -p 2 -trimpath -pgo=off -o "$R/ngn-kb8" .) > engine-build.log 2>&1
sha256sum spsa-forge ngn-kb8 kb8bin/src/engine/nets/*.nnue > builds.sha256
go version -m ngn-kb8 > engine-build-info.txt
printf 'Frozen engine and bounded legacy tuner built; checkpoint transfer has not started.\n'

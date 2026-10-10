#!/usr/bin/env bash
# All checks share one scope with a total5G cap, after the tool build finishes.
set -euo pipefail
R=/work/lanes/ngn-spsa-20261009
export PATH=/home/ehrlich/.local/ngn-go1255/bin:$PATH
export GOCACHE=/work/cache/go/ngn-build GOMODCACHE=/work/cache/go/ngn-mod
export GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local
cd "$R"
finish() { local rc=$?; echo "$rc" > checks.exit; date -Is > checks.done; }
trap finish EXIT
date -Is > checks.started
mkdir baseline
tar -xf base.tar -C baseline
for spec in admission:2026-10-09-time-playing.patch:TestIterationAdmission horizon:2026-10-09-time-horizon.patch:TestTimeHorizon partial:2026-10-09-partial-root.patch:TestPartialRoot; do
 IFS=: read -r kind patch pattern <<< "$spec"
 mkdir "$kind"
 tar -xf base.tar -C "$kind"
 (cd "$kind" && git apply "../$patch")
 (cd "$kind" && timeout 10m go test -p 2 -run "^$pattern" -count=1 -v ./engine) > "$kind-narrow.log" 2>&1
 (cd "$kind" && timeout 20m go test -p 2 -short ./... -count=1) > "$kind-full-short.log" 2>&1
 (cd "$kind" && timeout 20m go test -p 2 -short -race ./... -count=1) > "$kind-full-race.log" 2>&1
 (cd "$kind" && timeout 10m go vet -p 2 ./...) > "$kind-vet.log" 2>&1
 (cd "$kind" && timeout 10m go build -p 2 -pgo="$R/baseline/default.pgo" -o "$R/$kind-ngn" .) > "$kind-build.log" 2>&1
 date -Is > "$kind.done"
done
python3 key-prepare.py
export GOMAXPROCS=1 NGN_KEY_NET=$R/kb8bin/src/engine/nets/ngn-kb8h512-be34532c.nnue
for kind in oldkey newkey; do
 (cd "$kind" && NGN_KEY_OUT="$R/$kind.json" timeout 10m go test -p 2 -run '^TestCampaignKeyDiagnostics$' -count=1 -v ./engine) > "$kind.log" 2>&1
done
printf 'All three isolated candidates checked; key diagnostics complete. No strength games.\n'

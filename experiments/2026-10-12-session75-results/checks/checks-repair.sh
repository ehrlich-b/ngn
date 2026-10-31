#!/usr/bin/env bash
set -euo pipefail
s=/home/ehrli/ngn-data/night-20261007/session75
cd "$s/source"
finish() { rc=$?; echo "$rc" > "$s/checks.exit"; date -Is > "$s/checks.done"; }
trap finish EXIT
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOTOOLCHAIN=local
export GOCACHE=/home/ehrli/ngn-data/night-20261007/session17f/go-cache
export NGN_CAMPAIGN_NET=/home/ehrli/ngn-data/night-20261007/session72-see-rig/base.nnue
gofmt -w engine/campaign_smp_test.go
for width in 2 4; do
 GOMAXPROCS=$width go test -p 1 -short -race ./engine -run '^TestCampaignCurrentNetSMP$' -count=1 -v > "$s/current-net-fixed-race-$width.log" 2>&1
done
GOMAXPROCS=2 go vet -p 1 ./... > "$s/vet-fixed.log" 2>&1
printf 'PASS; prior full-race-2/4 unchanged production checks reused\n' > "$s/checks-pass.txt"

#!/usr/bin/env bash
set -euo pipefail
D=/home/ehrli/ngn-data/night-20261007/session77-smp-panel
cd "$D/source"
finish(){ rc=$?; echo "$rc" > "$D/checks.exit"; date -Is > "$D/checks.done"; }
trap finish EXIT
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOTOOLCHAIN=local GOMAXPROCS=2
export GOCACHE=/home/ehrli/ngn-data/night-20261007/session17f/go-cache
export NGN_CAMPAIGN_NET=/home/ehrli/ngn-data/night-20261007/session77-see-rig/base.nnue
patch -p1 < "$D/diagnostic.patch" > "$D/patch.log"
gofmt -w engine/search.go engine/search_smp.go engine/campaign_s77_test.go
go test -p 1 -short ./engine -run '^(TestLazySMP|TestCampaignCurrentNetSMP)' -count=1 > "$D/controls.log" 2>&1
NGN_S77_PANEL="$D/panel.json" NGN_S77_SMOKE=1 NGN_S77_BLOCKS=1 NGN_S77_SAMPLING=1 go test -p 1 -short -race ./engine -run '^TestS77WorkerPanel$' -count=1 -v > "$D/smoke-race.log" 2>&1
go test -p 1 -c ./engine -o "$D/panel.test" > "$D/compile.log" 2>&1

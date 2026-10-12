#!/usr/bin/env bash
set -euo pipefail
s=/home/ehrli/ngn-data/night-20261007/session74-clock-panel
cd "$s"
finish() { local rc=$?; echo "$rc" > "$s/checks.exit"; date -Is > "$s/checks.done"; }
trap finish EXIT
export PATH=/home/ehrli/go/bin:/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2
export GOCACHE=/home/ehrli/ngn-data/night-20261007/session17f/go-cache
st() { python3 - "$s/status.jsonl" "$1" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f: f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),event=sys.argv[2]))+'\n')
PY
}
date -Is > checks.started
st checks-start
mkdir base horizon
tar -xf base.tar -C base
tar -xf base.tar -C horizon
(cd horizon && git apply ../horizon.patch)
cd horizon
go test -p 2 -short ./engine -run '^TestTimeHorizon' -count=1 -v > "$s/horizon-narrow.log" 2>&1
go test -p 2 -short ./engine -count=1 > "$s/horizon-engine-short.log" 2>&1
go test -p 2 -short -race ./engine -count=1 > "$s/horizon-engine-race.log" 2>&1
go test -p 2 -short ./... -count=1 > "$s/horizon-full-short.log" 2>&1
go test -p 2 -short -race ./... -count=1 > "$s/horizon-full-race.log" 2>&1
go vet -p 2 ./... > "$s/horizon-vet.log" 2>&1
go build -p 2 -pgo=default.pgo -o "$s/horizon-ngn" . > "$s/horizon-build.log" 2>&1
sha256sum engine/time.go engine/time_horizon_test.go "$s/horizon-ngn" > "$s/horizon-inputs.sha256"
st horizon-checks-pass

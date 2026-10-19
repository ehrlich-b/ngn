#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session115
finish() { local rc=$?; echo "$rc" > "$D/checks.exit"; date -Is > "$D/checks.done"; }
trap finish EXIT
export PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local GOPROXY=off GOAMD64=v3 GOMAXPROCS=2 GOCACHE=$R/session17f/go-cache
cd "$D/source"
gofmt -w engine/search.go
go test -p 1 -short ./engine -count=1 > "$D/engine-short.log" 2>&1
go test -p 1 -short -race ./engine -count=1 > "$D/engine-race.log" 2>&1
go test -p 1 -short ./... -count=1 > "$D/full-short.log" 2>&1
go test -p 1 -short -race ./... -count=1 > "$D/full-race.log" 2>&1
go vet -p 1 ./... > "$D/vet.log" 2>&1
go build -p 1 -pgo="$R/session100-speed/baseline/default.pgo" -o "$D/accepted-ngn" . > "$D/build.log" 2>&1
go build -p 1 -pgo="$R/session100-speed/baseline/default.pgo" -o "$D/accepted-probe" ./cmd/speedprobe > "$D/probe-build.log" 2>&1
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -p 1 -pgo="$R/session100-speed/baseline/default.pgo" -o "$D/accepted-windows.exe" . > "$D/windows-build.log" 2>&1
python3 "$D/default-parity.py" > "$D/default-parity.log" 2>&1
python3 - <<'PY'
import datetime,json
from pathlib import Path
r=Path('/home/ehrli/ngn-data/night-20261007');d=r/'session115'
p=json.loads((d/'default-parity.json').read_text());assert p['verdict']=='PASS'
result=dict(state='DEFAULTS_CHECKS_PASS',passed=['engine short/race','full short/race','vet','Linux PGO/Windows builds','24 exact default-option identities','14 UCI defaults'],parity=p,finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
(d/'checks-result.json').write_text(json.dumps(result,indent=2)+'\n')
with (r/'status.jsonl').open('a') as f:f.write(json.dumps(dict(utc=result['finished_utc'],stage='session115-defaults',event='complete',detail=dict(state=result['state'],binary_sha256=p['binary_sha256'])))+'\n')
PY

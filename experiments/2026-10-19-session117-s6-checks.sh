#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session117-speed/s6
export PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local GOPROXY=off GOAMD64=v3 GOMAXPROCS=2 GOCACHE=$R/session17f/go-cache
finish() { local rc=$?; echo "$rc" > "$D/checks.exit"; date -Is > "$D/checks.done"; }
trap finish EXIT
echo $$ > "$D/checks.pid"
mkdir "$D/candidate"
tar xf "$R/session117/base.tar" -C "$D/candidate"
cd "$D/candidate"
git apply "$D/candidate.patch"
go test -p 1 -short ./engine -run 'TestSEEIncremental|TestSEEPostMoveGeometry' -v -count=1 > "$D/focused.log" 2>&1
go test -p 1 -short ./engine -count=1 > "$D/engine-short.log" 2>&1
go test -p 1 -short -race ./engine -count=1 > "$D/engine-race.log" 2>&1
go test -p 1 -short ./... -count=1 > "$D/full-short.log" 2>&1
go test -p 1 -short -race ./... -count=1 > "$D/full-race.log" 2>&1
go vet -p 1 ./... > "$D/vet.log" 2>&1
go build -p 1 -pgo="$R/session100-speed/baseline/default.pgo" -o "$D/s6-ngn" . > "$D/build.log" 2>&1
go build -p 1 -pgo="$R/session100-speed/baseline/default.pgo" -o "$D/s6-probe" ./cmd/speedprobe > "$D/probe-build.log" 2>&1
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -p 1 -pgo="$R/session100-speed/baseline/default.pgo" -o "$D/s6-windows.exe" . > "$D/windows-build.log" 2>&1
python3 "$D/identity.py" > "$D/identity.log" 2>&1
python3 - <<'PY_DONE'
import datetime,json
from pathlib import Path
r=Path('/home/ehrli/ngn-data/night-20261007');d=r/'session117-speed/s6'
i=json.loads((d/'identity.json').read_text());assert i['verdict']=='PASS'
result=dict(state='S6_CHECKS_PASS_TIMING_UNARMED',base=(r/'session117/accepted-source.txt').read_text().strip(),vector='compiled warm2000',identity=i,passed=['frozen NGN complete SEE parity','randomized exact attacker masks','engine/full short/race','vet','Linux-PGO/Windows builds','60 exact identities'],finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),automatic_timing=False,automatic_adoption=False)
(d/'checks-result.json').write_text(json.dumps(result,indent=2)+'\n')
with (r/'status.jsonl').open('a') as f:f.write(json.dumps(dict(utc=result['finished_utc'],stage='session117-s6',event='checks-complete',detail=dict(state=result['state'],base=result['base'])))+'\n')
PY_DONE

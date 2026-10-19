#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session117-speed/s5
export PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local GOPROXY=off GOAMD64=v3 GOMAXPROCS=2 GOCACHE=$R/session17f/go-cache
finish() { local rc=$?; echo "$rc" > "$D/checks.exit"; date -Is > "$D/checks.done"; }
trap finish EXIT
echo $$ > "$D/checks.pid"
mkdir "$D/candidate"
tar xf "$R/session117/base.tar" -C "$D/candidate"
cd "$D/candidate"
git apply "$D/candidate.patch"
gofmt -w engine/search.go engine/search_epilogue_test.go
cp engine/search.go "$D/formatted-search.go"
cp engine/search_epilogue_test.go "$D/formatted-test.go"
go test -p 1 -short ./engine -run 'TestRecursiveSearchRestoresFrameState|TestStopped|TestProbcut|TestSingularVerification' -count=1 > "$D/focused.log" 2>&1
go test -p 1 -short ./engine -count=1 > "$D/engine-short.log" 2>&1
go test -p 1 -short -race ./engine -count=1 > "$D/engine-race.log" 2>&1
go test -p 1 -short ./... -count=1 > "$D/full-short.log" 2>&1
go test -p 1 -short -race ./... -count=1 > "$D/full-race.log" 2>&1
go vet -p 1 ./... > "$D/vet.log" 2>&1
go build -p 1 -gcflags='github.com/ehrlich-b/ngn/engine=-d=defer' -pgo="$R/session100-speed/baseline/default.pgo" -o "$D/s5-ngn" . > "$D/build.log" 2>&1
go build -p 1 -pgo="$R/session100-speed/baseline/default.pgo" -o "$D/s5-probe" ./cmd/speedprobe > "$D/probe-build.log" 2>&1
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -p 1 -pgo="$R/session100-speed/baseline/default.pgo" -o "$D/s5-windows.exe" . > "$D/windows-build.log" 2>&1
python3 "$D/identity.py" > "$D/identity.log" 2>&1
python3 - <<'PY_DONE'
import datetime,json
from pathlib import Path
r=Path('/home/ehrli/ngn-data/night-20261007');d=r/'session117-speed/s5'
i=json.loads((d/'identity.json').read_text());assert i['verdict']=='PASS'
compiler=(d/'build.log').read_text();assert compiler.count('open-coded defer')>=2
result=dict(state='S5_CHECKS_PASS',base=(r/'session117/accepted-source.txt').read_text().strip(),vector='compiled warm2000',identity=i,passed=['normal/panic cleanup regressions','engine/full short/race','vet','Linux-PGO/Windows builds','open-coded defer witnesses','60 exact identities'],finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
(d/'checks-result.json').write_text(json.dumps(result,indent=2)+'\n')
with (r/'status.jsonl').open('a') as f:f.write(json.dumps(dict(utc=result['finished_utc'],stage='session117-s5',event='checks-complete',detail=dict(state=result['state'],base=result['base'])))+'\n')
PY_DONE

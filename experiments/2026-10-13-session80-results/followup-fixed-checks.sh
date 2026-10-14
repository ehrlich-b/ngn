#!/bin/bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session80
finish() { local rc=$?; echo "$rc" > "$D/followup-checks.exit"; date -Is > "$D/followup-checks.done"; }
trap finish EXIT
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOTOOLCHAIN=local GOPROXY=off GOMAXPROCS=2 GOCACHE=$R/session17f/go-cache PYTHONDONTWRITEBYTECODE=1

C=$D/candidates/probcut
cd "$C/source"
go test -p 1 -short ./engine -count=1 > "$C/engine-short.log" 2>&1
go test -p 1 -short -race ./engine -count=1 > "$C/engine-race.log" 2>&1
go test -p 1 -short ./... -count=1 > "$C/full-short.log" 2>&1
go test -p 1 -short -race ./... -count=1 > "$C/full-race.log" 2>&1
go vet -p 1 ./... > "$C/vet.log" 2>&1
go build -p 1 -pgo=default.pgo -o "$C/candidate-ngn" . > "$C/build.log" 2>&1
python3 - "$C" <<'PY'
import datetime,hashlib,json,pathlib,sys
c=pathlib.Path(sys.argv[1]); sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
(c/'checks-result.json').write_text(json.dumps(dict(base='f203e2f',candidate='probcut',passed=['focused','engine short/race','full short/race','vet','PGO build'],binary_sha256=sha(c/'candidate-ngn'),completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat()),indent=2)+'\n')
PY
RODENT=$R/session19-rodent3/opponent-only/rodent-v12
cd "$R/session19-rodent3/opponent-only/rodent"
GOMAXPROCS=1 "$D/sprt" -new "$RODENT" -base "$RODENT" -newoption Threads=4 -baseoption Threads=1 -newoption Hash=256 -baseoption Hash=256 -newoption OwnBook=false -baseoption OwnBook=false -nodes 1000 -concurrency 1 -maxgames 4 -mingames 1000000 -maxmoves 24 -openings "$R/session79-admission-see/openings.txt" -lowpower=false -pgn "$D/rodent-smoke.pgn" > "$D/rodent-smoke.log" 2>&1
grep -q 'Games: 4' "$D/rodent-smoke.log"

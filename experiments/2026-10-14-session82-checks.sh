#!/bin/bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session82
finish() { local rc=$?; echo "$rc" > "$D/checks.exit"; date -Is > "$D/checks.done"; }
trap finish EXIT
echo $$ > "$D/checks.pid"
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOTOOLCHAIN=local GOPROXY=off GOMAXPROCS=2 GOCACHE=$R/session17f/go-cache PYTHONDONTWRITEBYTECODE=1
for name in qsearch continuation pawn probcut; do
 C=$D/candidates/$name
 mkdir -p "$C/source"
 tar -xf "$D/base.tar" -C "$C/source"
 cd "$C/source"
 git apply "$D/candidates/$name.patch"
 case "$name" in qsearch) pattern='^TestQStatic|^TestSingularCorrectionBounds$';; continuation) pattern='^TestContinuationCorrection|^TestSingularCorrectionBounds$';; pawn) pattern='^TestPawnQuiet|^TestSingularCorrectionBounds$';; probcut) pattern='^TestProbCut|^TestSingularCorrectionBounds$';; esac
 go test -p 1 -short ./engine -run "$pattern" -count=1 -v > "$C/focused.log" 2>&1
 go test -p 1 -short ./engine -count=1 > "$C/engine-short.log" 2>&1
 go test -p 1 -short -race ./engine -count=1 > "$C/engine-race.log" 2>&1
 go test -p 1 -short ./... -count=1 > "$C/full-short.log" 2>&1
 go test -p 1 -short -race ./... -count=1 > "$C/full-race.log" 2>&1
 go vet -p 1 ./... > "$C/vet.log" 2>&1
 go build -p 1 -pgo=default.pgo -o "$C/candidate-ngn" . > "$C/build.log" 2>&1
 python3 - "$C" <<'PY'
import datetime,hashlib,json,pathlib,sys
c=pathlib.Path(sys.argv[1]);sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
(c/'checks-result.json').write_text(json.dumps(dict(base='9ec5d31',candidate=c.name,passed=['focused including singular witness','engine short/race','full short/race','vet','PGO build'],binary_sha256=sha(c/'candidate-ngn'),patch_sha256=sha(c.parent/(c.name+'.patch')),completed_utc=datetime.datetime.now(datetime.timezone.utc).isoformat()),indent=2)+'\n')
print(c.name,'checks PASS',flush=True)
PY
done

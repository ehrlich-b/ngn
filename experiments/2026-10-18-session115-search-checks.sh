#!/usr/bin/env bash
# Retarget the four existing isolated search candidates to compiled warm2000.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session115-search
finish() { local rc=$?; echo "$rc" > "$D/checks.exit"; date -Is > "$D/checks.done"; }
trap finish EXIT
echo $$ > "$D/checks.pid"
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOTOOLCHAIN=local GOPROXY=off GOMAXPROCS=2 GOCACHE=$R/session17f/go-cache PYTHONDONTWRITEBYTECODE=1
for name in qsearch continuation pawn probcut; do
 C=$D/candidates/$name
 mkdir -p "$C/source"
 tar -xf "$R/session115/base.tar" -C "$C/source"
 cd "$C/source"
 git apply "$R/session107/candidates/$name.patch"
 case "$name" in qsearch) pattern='^TestQStatic|^TestSingularCorrectionBounds$';; continuation) pattern='^TestContinuationCorrection|^TestSingularCorrectionBounds$';; pawn) pattern='^TestPawnQuiet|^TestSingularCorrectionBounds$';; probcut) pattern='^TestProbCut|^TestSingularCorrectionBounds$';; esac
 go test -p 1 -short ./engine -run "$pattern" -count=1 -v > "$C/focused.log" 2>&1
 go test -p 1 -short ./engine -count=1 > "$C/engine-short.log" 2>&1
 go test -p 1 -short -race ./engine -count=1 > "$C/engine-race.log" 2>&1
 go test -p 1 -short ./... -count=1 > "$C/full-short.log" 2>&1
 go test -p 1 -short -race ./... -count=1 > "$C/full-race.log" 2>&1
 go vet -p 1 ./... > "$C/vet.log" 2>&1
 go build -p 1 -pgo="$R/session100-speed/baseline/default.pgo" -o "$C/candidate-ngn" . > "$C/build.log" 2>&1
 python3 "$D/nominate.py" "$name"
done
python3 - <<'PY'
import datetime,json
from pathlib import Path
r=Path('/home/ehrli/ngn-data/night-20261007');d=r/'session115-search'
results={name:json.loads((d/'candidates'/name/'checks-result.json').read_text()) for name in ['qsearch','continuation','pawn','probcut']}
result=dict(state='COMBINATION_CHECKS_PASS',base=(r/'session115/accepted-source.txt').read_text().strip(),vector='compiled warm2000',candidates=results,automatic_adoption=False,finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
(d/'checks-result.json').write_text(json.dumps(result,indent=2)+'\n')
with (r/'status.jsonl').open('a') as f:f.write(json.dumps(dict(utc=result['finished_utc'],stage='session115-search-checks',event='complete',detail=dict(base=result['base'],state=result['state'])))+'\n')
PY
# Retarget the sole pending speed candidate while the first SPRT is running.
bash "$R/session115-speed/checks.sh"

#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
D=$R/session117
export PATH=/usr/local/go/bin:/usr/lib/wsl/lib:$PATH GOTOOLCHAIN=local GOPROXY=off GOAMD64=v3 GOMAXPROCS=2 GOCACHE=$R/session17f/go-cache PYTHONDONTWRITEBYTECODE=1
finish() { local rc=$?; echo "$rc" > "$D/checks.exit"; date -Is > "$D/checks.done"; }
trap finish EXIT
echo $$ > "$D/checks.pid"
cd "$D/source"
before=$(sha256sum "$D/accepted-ngn" | cut -d' ' -f1)
go build -buildvcs=true -p 2 -pgo="$R/session100-speed/baseline/default.pgo" -o "$D/accepted-ngn" . > "$D/accepted-build-p2.log" 2>&1
after=$(sha256sum "$D/accepted-ngn" | cut -d' ' -f1)
[ "$before" = "$after" ]
python3 "$D/freeze.py" > "$D/freeze-retry.log" 2>&1
for name in continuation pawn probcut; do
 C=$R/session117-search/candidates/$name
 mkdir -p "$C/source"
 tar xf "$D/base.tar" -C "$C/source"
 cd "$C/source"
 git apply "$C/candidate.patch"
 case "$name" in continuation) pattern='^TestContinuationCorrection|^TestSingularCorrectionBounds$';; pawn) pattern='^TestPawnQuiet|^TestSingularCorrectionBounds$';; probcut) pattern='^TestProbCut|^TestSingularCorrectionBounds$';; esac
 go test -p 1 -short ./engine -run "$pattern" -count=1 -v > "$C/focused.log" 2>&1
 go test -p 1 -short ./engine -count=1 > "$C/engine-short.log" 2>&1
 go test -p 1 -short -race ./engine -count=1 > "$C/engine-race.log" 2>&1
 go test -p 1 -short ./... -count=1 > "$C/full-short.log" 2>&1
 go test -p 1 -short -race ./... -count=1 > "$C/full-race.log" 2>&1
 go vet -p 1 ./... > "$C/vet.log" 2>&1
 go build -p 1 -pgo="$R/session100-speed/baseline/default.pgo" -o "$C/candidate-ngn" . > "$C/build.log" 2>&1
 python3 "$D/nominate.py" "$name"
done
bash "$R/session117-speed/s5/checks.sh"
bash "$R/session117-speed/s6/checks.sh"
python3 - <<'PY'
import datetime,json
from pathlib import Path
r=Path('/home/ehrli/ngn-data/night-20261007');d=r/'session117'
result=dict(state='ALL_RETARGET_CHECKS_PASS',base=(d/'accepted-source.txt').read_text().strip(),
 search=['continuation','pawn','probcut'],speed=['s5','s6'],automatic_adoption=False,
 finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat())
(d/'checks-result.json').write_text(json.dumps(result,indent=2)+'\n')
with (r/'status.jsonl').open('a') as f:f.write(json.dumps(dict(utc=result['finished_utc'],stage='session117',event='checks-complete',detail=result))+'\n')
PY

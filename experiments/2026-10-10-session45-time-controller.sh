#!/usr/bin/env bash
# Launch once after a reviewed capture decision and concrete base nomination.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session45-time-admission
export PATH=/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2 GOPROXY=off GOTOOLCHAIN=local GOCACHE=$R/session17f/go-cache
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session45-time-admission',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; session review required; no automatic adoption"; }
trap finish EXIT
date -Is > controller.started
python3 - <<'PY'
import os
d=os.statvfs('/mnt/c');assert d.f_bavail*d.f_frsize>=25.5*1024**3
PY
sha256sum -c stage-inputs.sha256 > stage-input-check.txt
mkdir baseline candidate
tar -xf base.tar -C baseline
tar -xf base.tar -C candidate
(cd candidate && git apply ../candidate.patch)
st checks-start 'unchanged admission patch; actual source combination checks only'
(cd candidate && taskset -c 13,15 nice -n 19 go test -p 2 -run '^TestIterationAdmission' -count=1 -v ./engine) > narrow.log 2>&1
for mode in short race; do
 flags=(); [[ $mode != race ]] || flags=(-race)
 (cd candidate && taskset -c 13,15 nice -n 19 go test -p 2 -short "${flags[@]}" ./... -count=1) > "full-$mode.log" 2>&1
done
(cd candidate && taskset -c 13,15 nice -n 19 go vet -p 2 ./...) > vet.log 2>&1
for kind in baseline candidate; do
 (cd "$kind" && taskset -c 13,15 nice -n 19 go build -p 2 -pgo="$S/baseline/default.pgo" -o "$S/$kind-ngn" .) > "$kind-build.log" 2>&1
done
export GOMAXPROCS=1
python3 manifest.py
st aa-start 'fresh actual-base A/A40 at60+0.6/concurrency2/rig13,15'
COMMON=(-tc 60+0.6 -concurrency 2 -maxmoves 400 -gametimeout 900 -openings "$S/openings.txt" -lowpower=false)
NEW=(-newoption "EvalFile=$S/base.nnue" -newoption UseNNUE=true -newoption Hash=64 -newoption Threads=1 -newoption OwnBook=false)
BASE=(-baseoption "EvalFile=$S/base.nnue" -baseoption UseNNUE=true -baseoption Hash=64 -baseoption Threads=1 -baseoption OwnBook=false)
timeout --signal=TERM --kill-after=15s 2h taskset -c 13,15 nice -n 19 ./sprt -new ./baseline-ngn -base ./baseline-ngn "${COMMON[@]}" -mingames 1000000 -maxgames 40 "${NEW[@]}" "${BASE[@]}" > aa.log 2>&1
python3 aa-parser.py gate aa.log --gate-mode aa > aa-summary.json
sha256sum -c game-inputs.sha256 > input-check-after-aa.txt
st gate-start 'paired SPRT[0,+6]/min400/cap4000 at60+0.6; nominated source frozen; no automatic adoption'
timeout --signal=TERM --kill-after=15s 96h taskset -c 13,15 nice -n 19 ./sprt -new ./candidate-ngn -base ./baseline-ngn "${COMMON[@]}" -elo0 0 -elo1 6 -alpha .05 -beta .05 -mingames 400 -maxgames 4000 "${NEW[@]}" "${BASE[@]}" > gate.log 2>&1
python3 gate-review.py
sha256sum -c game-inputs.sha256 > input-check-after-games.txt
st gate-done "$(cat gate-summary.json)"

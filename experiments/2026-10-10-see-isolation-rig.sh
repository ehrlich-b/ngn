#!/usr/bin/env bash
# Fresh rig sample after the forge waiter was interrupted before any games.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session49-see-isolation
export GOMAXPROCS=1 GOAMD64=v3
cd "$S"
st() { python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session49-see-isolation-rig',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() {
 local rc=$?
 echo "$rc" > controller.exit
 date -Is > controller.done
 st end "rc $rc; diagnostic only; review complete sample; no adoption or pooling"
}
trap finish EXIT
date -Is > controller.started
sha256sum -c inputs.sha256 > input-check-before.txt
COMMON=(-tc 10+0.1 -concurrency 6 -mingames 1000000 -maxmoves 400 -gametimeout 180 -openings "$S/openings.txt" -lowpower=false)
NEW=(-newoption "EvalFile=$S/base.nnue" -newoption UseNNUE=true -newoption Hash=64 -newoption Threads=1 -newoption OwnBook=false)
BASE=(-baseoption "EvalFile=$S/base.nnue" -baseoption UseNNUE=true -baseoption Hash=64 -baseoption Threads=1 -baseoption OwnBook=false)
st aa-start 'rig0-11/c6/nice19/6GiB/swap0; fresh40 at10+0.1; trainer12,14 unchanged; SPSA stays on forge'
timeout --signal=TERM --kill-after=15s 1h taskset -c 0-11 nice -n 19 ./sprt -new ./baseline-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 40 "${NEW[@]}" "${BASE[@]}" > aa.log 2>&1
python3 isolation-parser.py aa aa.log > aa-summary.json
sha256sum -c inputs.sha256 > input-check-after-aa.txt
st gate-start 'exactly1000/500 reversed pairs SEE-only10+0.1; independent rig diagnosis/no pooling; data may use free forge lane'
timeout --signal=TERM --kill-after=15s 8h taskset -c 0-11 nice -n 19 ./sprt -new ./candidate-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 1000 "${NEW[@]}" "${BASE[@]}" > gate.log 2>&1
python3 isolation-parser.py gate gate.log > diagnostic-summary.json
sha256sum -c inputs.sha256 > input-check-after.txt
st gate-done "$(cat diagnostic-summary.json); no playing change; review before redesign"

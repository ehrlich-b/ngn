#!/usr/bin/env bash
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session51-tt-isolation
export GOMAXPROCS=1 GOAMD64=v3
cd "$S"
st() { python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session51-tt-isolation-rig',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; diagnostic only; review required; no adoption or pooling"; }
trap finish EXIT
date -Is > controller.started
sha256sum -c inputs.sha256 > input-check-before.txt
COMMON=(-tc 60+0.6 -concurrency 6 -mingames 1000000 -maxmoves 400 -gametimeout 900 -openings "$S/openings.txt" -lowpower=false)
NEW=(-newoption "EvalFile=$S/base.nnue" -newoption UseNNUE=true -newoption Hash=64 -newoption Threads=1 -newoption OwnBook=false)
BASE=(-baseoption "EvalFile=$S/base.nnue" -baseoption UseNNUE=true -baseoption Hash=64 -baseoption Threads=1 -baseoption OwnBook=false)
st aa-start 'fresh archived TT A/A40 on rig0-11/c6/nice19/3GiB; factorizer12,14/4GiB reserved'
timeout --signal=TERM --kill-after=15s 90m taskset -c 0-11 nice -n 19 ./sprt -new ./baseline-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 40 "${NEW[@]}" "${BASE[@]}" > aa.log 2>&1
python3 isolation-parser.py aa aa.log > aa-summary.json
sha256sum -c inputs.sha256 > input-check-after-aa.txt
st gate-start 'exactly400/200 pairs archived TT-only60+0.6; fresh independent rig sample'
timeout --signal=TERM --kill-after=15s 12h taskset -c 0-11 nice -n 19 ./sprt -new ./candidate-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 400 "${NEW[@]}" "${BASE[@]}" > gate.log 2>&1
python3 isolation-parser.py gate gate.log > diagnostic-summary.json
sha256sum -c inputs.sha256 > input-check-after.txt
st gate-done "$(cat diagnostic-summary.json); no playing change"

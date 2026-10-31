#!/usr/bin/env bash
# Manually launch only after checks and Rodent terminal review on unchanged base.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session53-tt-refresh-games
export GOMAXPROCS=1 GOAMD64=v3
cd "$S"
st() { python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session53-tt-refresh-games',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; review required; no automatic adoption"; }
trap finish EXIT
date -Is > controller.started
sha256sum -c inputs.sha256 > input-check-before.txt
COMMON=(-tc 60+0.6 -concurrency 6 -mingames 1000000 -maxmoves 400 -gametimeout 900 -openings "$S/openings.txt" -lowpower=false)
NEW=(-newoption "EvalFile=$S/base.nnue" -newoption UseNNUE=true -newoption Hash=64 -newoption Threads=1 -newoption OwnBook=false)
BASE=(-baseoption "EvalFile=$S/base.nnue" -baseoption UseNNUE=true -baseoption Hash=64 -baseoption Threads=1 -baseoption OwnBook=false)
parse() { python3 - "$1" "$2" <<'PY'
import importlib.util,json,sys
from pathlib import Path
spec=importlib.util.spec_from_file_location('gate_parser','gate-parser.py')
m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
j=m.gate(Path(sys.argv[2]).read_text(),sys.argv[1])
if Path('ngn_crashes.log').exists():
 assert all('Crash Handler Initialized' in line or 'Crash log:' in line for line in Path('ngn_crashes.log').read_text().splitlines())
print(json.dumps(j,indent=2))
PY
}
st aa-start 'fresh own-TT immediate-base A/A40; rig0-11/c6/nice19/3GiB'
timeout --signal=TERM --kill-after=15s 4h taskset -c 0-11 nice -n19 ./sprt -new ./baseline-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 40 "${NEW[@]}" "${BASE[@]}" > aa.log 2>&1
parse aa aa.log > aa-summary.json
sha256sum -c inputs.sha256 > input-check-after-aa.txt
st gate-start 'exactly400/200 pairs60+0.6; same-position refresh only vs9bfbae4/a4d90b41'
timeout --signal=TERM --kill-after=15s 12h taskset -c 0-11 nice -n19 ./sprt -new ./candidate-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 400 "${NEW[@]}" "${BASE[@]}" > gate.log 2>&1
parse gate gate.log > gate-summary.json
sha256sum -c inputs.sha256 > input-check-after.txt
st gate-done "$(cat gate-summary.json); manual review only"

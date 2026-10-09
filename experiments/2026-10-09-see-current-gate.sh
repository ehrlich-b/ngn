#!/usr/bin/env bash
# Fresh SEE-only provenance gate on the accepted 39eb128 runtime.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session30-see-gate
export GOMAXPROCS=1 GOAMD64=v3
cd "$S"
st() {
 python3 - "$R/status.jsonl" "$1" "${2:-}" <<'PY'
import datetime,json,sys
with open(sys.argv[1],'a') as f:
 f.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),stage='session30-see-gate',event=sys.argv[2],detail=sys.argv[3]))+'\n')
PY
}
finish() { local rc=$?; echo "$rc" > controller.exit; date -Is > controller.done; st end "rc $rc; review before adoption; no retry or extension"; }
trap finish EXIT
date -Is > controller.started
sha256sum -c inputs.sha256 > input-check-before.txt
python3 - <<'PY'
import os
d=os.statvfs('/mnt/c');assert d.f_bavail*d.f_frsize>=32*1024**3
PY
COMMON=(-tc 10+0.1 -concurrency 2 -mingames 1000000 -maxmoves 400 -gametimeout 180 -openings "$S/openings.txt" -lowpower=false)
NEW=(-newoption "EvalFile=$S/base.nnue" -newoption UseNNUE=true -newoption Hash=64 -newoption Threads=1 -newoption OwnBook=false)
BASE=(-baseoption "EvalFile=$S/base.nnue" -baseoption UseNNUE=true -baseoption Hash=64 -baseoption Threads=1 -baseoption OwnBook=false)
st aa-start 'SEE independent of failed key-order preflight; accepted39eb128/be34532c; fresh A/A40 on13,15'
timeout --signal=TERM --kill-after=15s 30m taskset -c 13,15 nice -n 19 ./sprt -new ./baseline-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 40 "${NEW[@]}" "${BASE[@]}" > aa.log 2>&1
python3 gate-parser.py gate aa.log --gate-mode aa > aa-summary.json
st gate-start 'exactly400 paired10+0.1 games; upper95>0 provenance rule; zero operational failures'
timeout --signal=TERM --kill-after=15s 3h taskset -c 13,15 nice -n 19 ./sprt -new ./candidate-ngn -base ./baseline-ngn "${COMMON[@]}" -maxgames 400 "${NEW[@]}" "${BASE[@]}" > gate.log 2>&1
python3 gate-parser.py gate gate.log --gate-mode gate > original-parser-summary.json
sha256sum -c inputs.sha256 > input-check-after.txt
python3 - <<'PY'
import json,re
from pathlib import Path
j=json.loads(Path('original-parser-summary.json').read_text())
assert not re.search(r'PANIC|RECOVERED|FATAL',Path('ngn_crashes.log').read_text())
j['verdict']='provisional-provenance-keep' if j['ci95'][1]>0 and j['exact_elo_ci95'][2]>0 else 'shelved'
Path('gate-summary.json').write_text(json.dumps(j,indent=2)+'\n')
PY
st gate-done "$(cat gate-summary.json); no automatic adoption; combined provenance certification remains required"

#!/usr/bin/env bash
# Fresh accepted-net screen, then longer H512 training and its fixed gate.
# Run detached on WSL; no 120+1 claim or automatic change of accepted base.
set -euo pipefail
R=/home/ehrli/ngn-data/night-20261007
S=$R/session1
WT=/home/ehrli/wt/t80-20261006
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
BIN=/home/ehrli/ngn-data/release-0.3.0/pub-ngn-linux
SPRT=/home/ehrli/ngn-data/t80-20261006/sprt
OPEN=/home/ehrli/ngn-data/net1-gates-20261006/openings.txt
BASE=$R/kb8h512/best.nnue
CAND=$R/kb8h512-long
SP=$R/spsa/kb8
TUNER_PAUSED=0
st() {
  python3 - "$R/status.jsonl" "$1" "$2" "${3:-}" <<'PY'
import datetime, json, sys
with open(sys.argv[1], 'a') as stream:
    stream.write(json.dumps(dict(utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),
                                stage=sys.argv[2], event=sys.argv[3], detail=sys.argv[4])) + '\n')
PY
}
pause_tuner() {
  local pid pgid
  pid=$(cat "$SP/spsa.pid")
  if kill -0 "$pid" 2>/dev/null; then
    pgid=$(ps -o pgid= -p "$pid" | tr -d ' ')
    [[ $pgid == "$pid" ]] || { st session1 fail "tuner has unexpected group $pgid"; return 1; }
    # SPSA has no signal handler; its last completed checkpoint is the resume boundary.
    kill -INT -- "-$pid" 2>/dev/null || true
    for ((i=0; i<30; i++)); do
      kill -0 -- "-$pid" 2>/dev/null || break
      sleep 1
    done
    if kill -0 -- "-$pid" 2>/dev/null; then
      kill -TERM -- "-$pid" 2>/dev/null || true
      sleep 2
    fi
    if ps -eo pgid=,stat= | awk -v p="$pid" '$1==p && $2 !~ /^Z/ { found=1 } END { exit !found }'; then
      st session1 fail "tuner group $pid did not drain"; return 1
    fi
  fi
  TUNER_PAUSED=1
  st spsa pause-session1 "pid $pid; resume from last JSONL checkpoint"
}
resume_tuner() {
  if [[ $TUNER_PAUSED == 1 ]]; then
    cd "$SP"
    setsid nohup bash "$S/spsa-resume.cmd" >> "$SP/spsa.log" 2>&1 < /dev/null &
    echo $! > "$SP/spsa.pid"
    TUNER_PAUSED=0
    st spsa resume-session1 "pid $(cat "$SP/spsa.pid")"
  fi
}
cleanup() { local rc=$?; resume_tuner; [[ $rc == 0 ]] || st session1 fail "controller rc $rc"; }
trap cleanup EXIT
st session1 start "controller pid $$"
sha256sum -c "$S/frozen-inputs.sha256" > "$S/input-check.log"
pause_tuner
export NGN_SCREEN_OUT=$R/screen-kb8h512-session1
export NGN_WSL_INTEROP=/run/WSL/552_interop
if bash "$S/night-screen.sh" kb8h512 > "$S/screen.log" 2>&1; then
  st screen kb8h512-session1 "complete; $NGN_SCREEN_OUT/screen/analysis.json"
else
  rc=$?
  st screen kb8h512-session1-fail "rc $rc; preserve attempt, see $S/screen.log"
fi
resume_tuner

cd "$WT"
export PYTHONPATH=$WT OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2
export GOAMD64=v3 GOTOOLCHAIN=local GOPROXY=off GOMAXPROCS=2
bash "$S/kb8h512-long.cmd" > "$S/train.log" 2>&1 < /dev/null &
TRAIN_PID=$!
echo "$TRAIN_PID" > "$S/train.pid"
st train-long start "pid $TRAIN_PID; 116 epochs from scratch"
wait "$TRAIN_PID"
python3 - "$CAND" <<'PY'
import json, sys
from pathlib import Path
p=Path(sys.argv[1]); history=json.loads((p/'history.json').read_text())
assert len(history)==116 and history[-1]['epoch']==116
assert (p/'best.nnue').stat().st_size>0
PY
st train-long done "116 epochs; $(sha256sum "$CAND/best.nnue" | cut -c1-64)"

pause_tuner
cd "$WT"
taskset -c 12,14 nice -n 19 "$PY" - > "$S/parity.json" 2> "$S/parity.err" <<'PY'
import importlib.util, json
from pathlib import Path
import numpy as np
from trainer.ngnp import RECORD_DTYPE
spec=importlib.util.spec_from_file_location('rt', '/home/ehrli/wt/t80-20261006/experiments/2026-10-06-net1-recipe-trials.py')
m=importlib.util.module_from_spec(spec); spec.loader.exec_module(m)
selected=np.fromfile('/home/ehrli/ngn-data/t80-20261006/parity-records.ngnp', dtype=RECORD_DTYPE)
print(json.dumps(m.parity(Path('/home/ehrli/ngn-data/night-20261007/kb8h512-long'), selected)))
PY
st parity-long done "$(cat "$S/parity.json")"
sha256sum -c "$S/frozen-inputs.sha256" > "$S/pre-gate-input-check.log"
export GOMAXPROCS=1
COMMON=(-tc 10+0.1 -concurrency 8 -mingames 1000000 -maxmoves 400 -gametimeout 180 -openings "$OPEN" -lowpower=false)
NEW_OPTS=(-newoption "EvalFile=$BASE" -newoption UseNNUE=true -newoption Hash=64 -newoption Threads=1 -newoption OwnBook=false)
BASE_OPTS=(-baseoption "EvalFile=$BASE" -baseoption UseNNUE=true -baseoption Hash=64 -baseoption Threads=1 -baseoption OwnBook=false)
taskset -c 0-11 "$SPRT" -new "$BIN" -base "$BIN" "${COMMON[@]}" -maxgames 40 "${NEW_OPTS[@]}" "${BASE_OPTS[@]}" > "$S/aa.log" 2>&1
python3 "$S/check-gate.py" aa "$S/aa.log" > "$S/aa-summary.json"
st aa-long done "$(cat "$S/aa-summary.json")"
NEW_OPTS[1]="EvalFile=$CAND/best.nnue"
taskset -c 0-11 "$SPRT" -new "$BIN" -base "$BIN" "${COMMON[@]}" -maxgames 400 "${NEW_OPTS[@]}" "${BASE_OPTS[@]}" > "$S/gate.log" 2>&1
python3 "$S/check-gate.py" gate "$S/gate.log" > "$S/gate-summary.json"
sha256sum -c "$S/frozen-inputs.sha256" > "$S/post-gate-input-check.log"
st gate-long done "$(cat "$S/gate-summary.json"); review before promotion"
resume_tuner
st session1 complete "long-net verdict awaits log review; accepted base remains be34532c"

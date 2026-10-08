#!/bin/bash
# Follow-up to night-chain.sh after kb8h512 was accepted: wait for kb8h1024 training, pause the SPSA tuner,
# parity, A/A, 400-game gate kb8h1024 vs kb8h512, v2 screens (kb8h512, then kb8h1024 if its gate CI lower > 0),
# then resume the tuner. No trainer may run during a screen (claimtool NoOtherMatch), so none is started here.
set -u
R=/home/ehrli/ngn-data/night-20261007
T=/home/ehrli/ngn-data/t80-20261006
WT=/home/ehrli/wt/t80-20261006
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
BIN=/home/ehrli/ngn-data/release-0.3.0/pub-ngn-linux
SPRT=$T/sprt
BASE=$R/kb8h512/best.nnue
OPEN=/home/ehrli/ngn-data/net1-gates-20261006/openings.txt
CAND=$R/kb8h1024
SP=$R/spsa/kb8
st() { printf '{"utc":"%s","stage":"%s","event":"%s","detail":"%s"}\n' "$(date -u +%FT%TZ)" "$1" "$2" "$3" >> $R/status.jsonl; }
opts() { for o in "EvalFile=$2" UseNNUE=true Hash=64 Threads=1 OwnBook=false; do printf -- '-%soption %s ' "$1" "$o"; done; }

st controller2 start "pid $$"
PID=$(cat $R/kb8h1024.pid)
while kill -0 "$PID" 2>/dev/null; do sleep 60; done
if [ "$(grep -c '"epoch": 58,' $R/kb8h1024.log)" -lt 1 ] || [ ! -s $CAND/best.nnue ]; then st train2 fail "no epoch 58 or no best.nnue"; fi
[ -s $CAND/best.nnue ] && st train2 done "best.nnue $(sha256sum $CAND/best.nnue | cut -c1-64)"

SPID=$(cat $SP/spsa.pid)
kill -INT "$SPID" 2>/dev/null
for i in $(seq 60); do kill -0 "$SPID" 2>/dev/null || break; sleep 1; done
kill -0 "$SPID" 2>/dev/null && kill -TERM -- -"$SPID"
sleep 5
st spsa pause "$(tail -1 $SP/spsa.log | cut -c1-60)"

GATE=0
if [ -s $CAND/best.nnue ]; then
  cd $WT || exit 1
  export PYTHONPATH=$WT OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2 GOAMD64=v3 GOTOOLCHAIN=local GOPROXY=off GOMAXPROCS=2
  if taskset -c 0-7 $PY - > $R/parity-kb8h1024.json 2> $R/parity-kb8h1024.err <<'P'
import importlib.util, json
from pathlib import Path
import numpy as np
from trainer.ngnp import RECORD_DTYPE
spec = importlib.util.spec_from_file_location('rt', '/home/ehrli/wt/t80-20261006/experiments/2026-10-06-net1-recipe-trials.py')
m = importlib.util.module_from_spec(spec); spec.loader.exec_module(m)
sel = np.fromfile('/home/ehrli/ngn-data/t80-20261006/parity-records.ngnp', dtype=RECORD_DTYPE)
print(json.dumps(m.parity(Path('/home/ehrli/ngn-data/night-20261007/kb8h1024'), sel)))
P
  then st parity2 done "$(tr -d '"' < $R/parity-kb8h1024.json | head -c 400)"; GATE=1
  else st parity2 fail "see parity-kb8h1024.err"; fi
fi

if [ $GATE = 1 ]; then
  export GOMAXPROCS=1
  COMMON="-tc 10+0.1 -concurrency 8 -mingames 1000000 -maxmoves 400 -gametimeout 180 -openings $OPEN -lowpower=false"
  cd $R
  taskset -c 0-11 $SPRT -new $BIN -base $BIN $COMMON -maxgames 40 $(opts new $BASE) $(opts base $BASE) > $R/aa-kb8h1024.log 2>&1
  st aa2 done "$(grep -h 'Penta Elo\|Flag-outs' $R/aa-kb8h1024.log | tr '\n' ' ')"
  taskset -c 0-11 $SPRT -new $BIN -base $BIN $COMMON -maxgames 400 $(opts new $CAND/best.nnue) $(opts base $BASE) > $R/gate-kb8h1024.log 2>&1
  st gate2 done "$(grep -h 'W-D-L\|Penta Elo\|Flag-outs' $R/gate-kb8h1024.log | tr '\n' ' ')"
fi

bash $R/night-screen.sh kb8h512 > $R/screen-kb8h512.log 2>&1
st screen kb8h512 "rc $? $(tail -1 $R/screen-kb8h512.log | tr -d '"')"
LOW=$(grep -h 'Penta Elo' $R/gate-kb8h1024.log 2>/dev/null | sed -n 's/.*CI \[\([-+0-9]*\),.*/\1/p')
if [ -n "$LOW" ] && [ "$LOW" -gt 0 ] && grep -q 'Flag-outs (lost on time): new 0, base 0' $R/gate-kb8h1024.log; then
  bash $R/night-screen.sh kb8h1024 > $R/screen-kb8h1024.log 2>&1
  st screen kb8h1024 "rc $? $(tail -1 $R/screen-kb8h1024.log | tr -d '"')"
fi

cd $SP
setsid nohup bash -c "$(cat $SP/spsa.cmd) -resume" >> $SP/spsa.log 2>&1 < /dev/null &
echo $! > $SP/spsa.pid
st spsa resume "pid $(cat $SP/spsa.pid)"
st controller2 idle ""

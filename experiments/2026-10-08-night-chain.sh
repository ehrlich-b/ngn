#!/bin/bash
# Overnight 2026-10-07/08 net chain: wait for kb8h512 training, parity, start kb8h1024, A/A, 400-game gate.
set -u
R=/home/ehrli/ngn-data/night-20261007
T=/home/ehrli/ngn-data/t80-20261006
WT=/home/ehrli/wt/t80-20261006
PY=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
BIN=/home/ehrli/ngn-data/release-0.3.0/pub-ngn-linux
SPRT=$T/sprt
BASE=$T/h512/best.nnue
OPEN=/home/ehrli/ngn-data/net1-gates-20261006/openings.txt
CAND=$R/kb8h512
st() { printf '{"utc":"%s","stage":"%s","event":"%s","detail":"%s"}\n' "$(date -u +%FT%TZ)" "$1" "$2" "$3" >> $R/status.jsonl; }
opts() { for o in "EvalFile=$2" UseNNUE=true Hash=64 Threads=1 OwnBook=false; do printf -- '-%soption %s ' "$1" "$o"; done; }

st controller start "pid $$"
PID=$(cat $R/kb8h512.pid)
while kill -0 "$PID" 2>/dev/null; do sleep 60; done
if [ "$(grep -c '"epoch": 58,' $R/kb8h512.log)" -lt 1 ] || [ ! -s $CAND/best.nnue ]; then st train fail "no epoch 58 or no best.nnue"; exit 1; fi
st train done "best.nnue $(sha256sum $CAND/best.nnue | cut -c1-64)"

cd $WT || exit 1
export PYTHONPATH=$WT OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2 GOAMD64=v3 GOTOOLCHAIN=local GOPROXY=off GOMAXPROCS=2
if ! taskset -c 0-7 $PY - > $R/parity-kb8h512.json 2> $R/parity-kb8h512.err <<'P'
import importlib.util, json
from pathlib import Path
import numpy as np
from trainer.ngnp import RECORD_DTYPE
spec = importlib.util.spec_from_file_location('rt', '/home/ehrli/wt/t80-20261006/experiments/2026-10-06-net1-recipe-trials.py')
m = importlib.util.module_from_spec(spec); spec.loader.exec_module(m)
sel = np.fromfile('/home/ehrli/ngn-data/t80-20261006/parity-records.ngnp', dtype=RECORD_DTYPE)
print(json.dumps(m.parity(Path('/home/ehrli/ngn-data/night-20261007/kb8h512'), sel)))
P
then st parity fail "see parity-kb8h512.err"; exit 1; fi
st parity done "$(tr -d '"' < $R/parity-kb8h512.json | head -c 400)"

sed -e "s|--out $R/kb8h512|--out $R/kb8h1024|" -e 's|--hidden 512|--hidden 1024|' $R/kb8h512.cmd > $R/kb8h1024.cmd
setsid nohup bash $R/kb8h1024.cmd > $R/kb8h1024.log 2>&1 < /dev/null &
echo $! > $R/kb8h1024.pid
st train2 start "kb8h1024 pid $(cat $R/kb8h1024.pid)"

export GOMAXPROCS=1
COMMON="-tc 10+0.1 -concurrency 8 -mingames 1000000 -maxmoves 400 -gametimeout 180 -openings $OPEN -lowpower=false"
taskset -c 0-11 $SPRT -new $BIN -base $BIN $COMMON -maxgames 40 $(opts new $BASE) $(opts base $BASE) > $R/aa-kb8h512.log 2>&1
st aa done "$(grep -h 'Penta Elo\|Flag-outs' $R/aa-kb8h512.log | tr '\n' ' ')"
taskset -c 0-11 $SPRT -new $BIN -base $BIN $COMMON -maxgames 400 $(opts new $CAND/best.nnue) $(opts base $BASE) > $R/gate-kb8h512.log 2>&1
st gate done "$(grep -h 'W-D-L\|Penta Elo\|Flag-outs' $R/gate-kb8h512.log | tr '\n' ' ')"
st controller idle ""

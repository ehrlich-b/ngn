#!/usr/bin/env bash
set -euo pipefail
s=/home/ehrli/ngn-data/night-20261007/session68-sparse-preflight
cd "$s"
finish() { local rc=$?; echo "$rc" > "$s/preflight.exit"; date -Is > "$s/preflight.done"; }
trap finish EXIT
date -Is > preflight.started
export PYTHONPATH=/home/ehrli/ngn-data/night-20261007/session62-loader-index/source
export OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2 PYTHONDONTWRITEBYTECODE=1
taskset -c 12,14 nice -n 19 timeout --signal=TERM --kill-after=15s 1200 /home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python "$s/preflight.py"

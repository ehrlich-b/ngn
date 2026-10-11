#!/usr/bin/env bash
set -euo pipefail
cd /home/ehrli/ngn-data/night-20261007/session66-data-diagnostic
finish() { local rc=$?; echo "$rc" > diagnostic.exit; date -Is > diagnostic.done; }
trap finish EXIT
export PYTHONPATH=/home/ehrli/ngn-data/night-20261007/session62-loader-index/source
export OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2
python=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
"$python" -m py_compile diagnostic.py
date -Is > diagnostic.started
timeout --signal=TERM --kill-after=15s 1800 "$python" diagnostic.py

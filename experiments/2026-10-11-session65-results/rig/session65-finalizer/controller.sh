#!/usr/bin/env bash
set -euo pipefail
s=/home/ehrli/ngn-data/night-20261007/session65-finalizer
export PATH=/usr/local/go/bin:$PATH GOMAXPROCS=2 GOAMD64=v3 GOTOOLCHAIN=local GOPROXY=off
export PYTHONPATH=/home/ehrli/ngn-data/night-20261007/session62-loader-index/source
export OMP_NUM_THREADS=2 MKL_NUM_THREADS=2 OPENBLAS_NUM_THREADS=2
export NGN_TRAIN_PYTHON=/home/ehrli/ngn-original-nnue-night-20261001/venv/bin/python
export NGN_PARITY_RECORDS=/home/ehrli/ngn-data/t80-20261006/parity-records.ngnp
finish() { local rc=$?; echo "$rc" > "$s/controller.exit"; date -Is > "$s/controller.done"; }
trap finish EXIT
python3 "$s/wait.py"
